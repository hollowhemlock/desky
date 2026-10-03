//go:build linux || darwin

package platform

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/hollowhemlock/desky/internal/workspace"
)

func (Native) Resolve(kind string, p workspace.Profile, root string) (workspace.Profile, bool, error) {
	if err := requireDesktop(desktopSession); err != nil {
		return p, false, err
	}
	return resolveUnix(runtime.GOOS, kind, p, root)
}

// Keep profile resolution independent of session detection so native CI can
// exercise lookup and defaults without pretending to qualify a desktop.
func resolveUnix(goos, kind string, p workspace.Profile, root string) (workspace.Profile, bool, error) {
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return p, false, workspace.Failure(7, "unsupported_action", "checkout directory is unavailable")
	}
	wait := false
	if kind == "url" {
		p = workspace.Profile{}
		if goos == "darwin" {
			p.Executable = "/usr/bin/open"
		} else {
			p.Executable = lookupUnix("xdg-open", root)
		}
		wait = true
	} else if p.Executable == "" {
		switch kind {
		case "editor":
			p = workspace.Profile{Executable: lookupUnix("code", root), Args: []string{"--reuse-window", "{path}"}}
			wait = true
		case "terminal":
			if goos != "darwin" {
				return p, false, workspace.Failure(7, "missing_launcher", "configure launchers.terminal with an absolute executable and one {path} argument on Linux")
			}
			p = workspace.Profile{Executable: "/usr/bin/open", Args: []string{"-a", "Terminal", "{path}"}}
		}
	}
	if !unixExecutable(p.Executable) {
		return p, false, workspace.Failure(7, "missing_launcher", "configure an absolute executable with execute permission for "+kind)
	}
	// open acknowledges LaunchServices dispatch; do not wait for the GUI app.
	if goos == "darwin" {
		real, _ := filepath.EvalSymlinks(p.Executable)
		wait = wait || real == "/usr/bin/open"
	}
	return p, wait, nil
}

func unixExecutable(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular() && st.Mode().Perm()&0111 != 0
}

func lookupUnix(name, root string) string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if !filepath.IsAbs(dir) {
			continue
		}
		realDir, err := filepath.EvalSymlinks(dir)
		if err != nil || inside(root, realDir) {
			continue
		}
		candidate, err := filepath.EvalSymlinks(filepath.Join(realDir, name))
		if err != nil || inside(root, candidate) || !unixExecutable(candidate) {
			continue
		}
		return candidate
	}
	return ""
}

func inside(root, path string) bool {
	r, err := filepath.Rel(root, path)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) && !filepath.IsAbs(r)
}

func (Native) Dispatch(a Action) error {
	args := a.Args
	if a.Type == "url" {
		if _, err := workspace.NormalizeURL(a.Target); err != nil {
			return err
		}
		args = []string{a.Target}
	}
	cmd := exec.Command(a.Executable, args...)
	cmd.Dir = a.Directory
	// nil standard handles map to /dev/null. A new session disconnects the
	// controlling terminal; no shell, signal relay or application teardown.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return dispatchProcess(cmd, a.Wait, 5*time.Second)
}
