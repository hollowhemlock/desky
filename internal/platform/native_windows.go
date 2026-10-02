package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/hollowhemlock/desky/internal/workspace"
	"golang.org/x/sys/windows"
)

func (Native) Resolve(kind string, p workspace.Profile, root string) (workspace.Profile, bool, error) {
	if strings.HasPrefix(root, `\\`) {
		return p, false, workspace.Failure(7, "unsupported_action", "network checkout launching is not supported")
	}
	if kind == "url" {
		return p, false, nil
	}
	wait := false
	if p.Executable == "" {
		switch kind {
		case "editor":
			for _, base := range []string{os.Getenv("LOCALAPPDATA"), os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
				if !filepath.IsAbs(base) {
					continue
				}
				for _, suffix := range []string{"Programs/Microsoft VS Code/Code.exe", "Microsoft VS Code/Code.exe"} {
					candidate := filepath.Join(base, filepath.FromSlash(suffix))
					candidate, _ = filepath.EvalSymlinks(candidate)
					if st, e := os.Stat(candidate); e == nil && st.Mode().IsRegular() && !inside(root, candidate) {
						p.Executable = candidate
						break
					}
				}
				if p.Executable != "" {
					break
				}
			}
			p.Args = []string{"--reuse-window", "{path}"}
		case "terminal":
			for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
				if !filepath.IsAbs(dir) {
					continue
				}
				real, e := filepath.EvalSymlinks(dir)
				if e != nil || inside(root, real) {
					continue
				}
				candidate := filepath.Join(real, "wt.exe")
				if st, e := os.Lstat(candidate); e == nil && st.Mode()&os.ModeSymlink != 0 {
					candidate, e = filepath.EvalSymlinks(candidate)
					if e != nil || inside(root, candidate) {
						continue
					}
				}
				if _, e := os.Stat(candidate); e == nil {
					p.Executable = candidate
					break
				}
			}
			// wt parses semicolons as its own command separator, even without a shell.
			// Resolve '.' via the child's CWD rather than interpolating a project path.
			p.Args = []string{"new-tab", "-d", "."}
			wait = true
		}
	}
	if !filepath.IsAbs(p.Executable) || !strings.EqualFold(filepath.Ext(p.Executable), ".exe") {
		return p, false, workspace.Failure(7, "missing_launcher", "configure an absolute native executable for "+kind)
	}
	st, err := os.Stat(p.Executable)
	if err != nil || st.IsDir() || (!st.Mode().IsRegular() && !strings.EqualFold(filepath.Base(p.Executable), "wt.exe")) {
		return p, false, workspace.Failure(7, "missing_launcher", "configured "+kind+" executable is unavailable")
	}
	return p, wait, nil
}

func inside(root, path string) bool {
	r, e := filepath.Rel(root, path)
	return e == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}

func (Native) Dispatch(a Action) error {
	if a.Type == "url" {
		done := make(chan error, 1)
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			// S_FALSE (1) also succeeds and must be paired with CoUninitialize.
			if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err != nil && err != syscall.Errno(1) {
				done <- err
				return
			}
			defer windows.CoUninitialize()
			u, err := windows.UTF16PtrFromString(a.Target)
			if err != nil {
				done <- err
				return
			}
			d, err := windows.UTF16PtrFromString(a.Directory)
			if err != nil {
				done <- err
				return
			}
			v, _ := windows.UTF16PtrFromString("open")
			done <- windows.ShellExecute(0, v, u, nil, d, windows.SW_SHOWNORMAL)
		}()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			return fmt.Errorf("URL dispatch timed out; outcome unknown, do not retry automatically")
		}
	}
	cmd := exec.Command(a.Executable, a.Args...)
	cmd.Dir = a.Directory
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP}
	if err := cmd.Start(); err != nil {
		return err
	}
	if !a.Wait {
		return cmd.Process.Release()
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("dispatch helper timed out; outcome unknown, do not retry automatically")
	}
}
