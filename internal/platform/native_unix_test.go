//go:build linux || darwin

package platform

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/hollowhemlock/desky/internal/workspace"
)

func executableFixture(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUnixLookupAndProfiles(t *testing.T) {
	root, installed := t.TempDir(), t.TempDir()
	root, _ = filepath.EvalSymlinks(root)
	installed, _ = filepath.EvalSymlinks(installed)
	untrusted := executableFixture(t, root, "code", "exit 99")
	alias := filepath.Join(t.TempDir(), "checkout")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(installed, "code")
	if err := os.Symlink(untrusted, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", strings.Join([]string{".", "relative", root, alias, installed}, string(os.PathListSeparator)))
	if _, _, err := resolveUnix(runtime.GOOS, "editor", workspace.Profile{}, alias); err == nil {
		t.Fatal("selected executable in checkout through PATH or symlink")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	code := executableFixture(t, installed, "code", "exit 0")
	for _, goos := range []string{"linux", "darwin"} {
		p, wait, err := resolveUnix(goos, "editor", workspace.Profile{}, root)
		if err != nil || !wait || p.Executable != code || !reflect.DeepEqual(p.Args, []string{"--reuse-window", "{path}"}) {
			t.Fatalf("%s editor: %+v %v %v", goos, p, wait, err)
		}
	}
	for _, bad := range []string{"code", root, filepath.Join(root, "missing")} {
		if _, _, err := resolveUnix(runtime.GOOS, "app", workspace.Profile{Executable: bad}, root); err == nil {
			t.Fatalf("accepted unavailable executable %s", bad)
		}
	}
	if err := os.Chmod(code, 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveUnix(runtime.GOOS, "editor", workspace.Profile{Executable: code}, root); err == nil {
		t.Fatal("accepted nonexecutable file")
	}
	// Explicit device profiles intentionally take precedence over default lookup.
	explicit := workspace.Profile{Executable: untrusted, Args: []string{"literal", "{path}"}}
	p, wait, err := resolveUnix(runtime.GOOS, "terminal", explicit, root)
	if err != nil || wait || !reflect.DeepEqual(p, explicit) {
		t.Fatalf("explicit profile: %+v %v %v", p, wait, err)
	}
	if _, _, err := resolveUnix("linux", "terminal", workspace.Profile{}, root); err == nil {
		t.Fatal("Linux silently supplied a terminal")
	}
	if runtime.GOOS == "darwin" {
		p, wait, err := resolveUnix("darwin", "terminal", workspace.Profile{}, root)
		if err != nil || !wait || p.Executable != "/usr/bin/open" || !reflect.DeepEqual(p.Args, []string{"-a", "Terminal", "{path}"}) {
			t.Fatalf("macOS terminal: %+v %v %v", p, wait, err)
		}
		p, wait, err = resolveUnix("darwin", "url", workspace.Profile{}, root)
		if err != nil || !wait || p.Executable != "/usr/bin/open" {
			t.Fatalf("macOS URL: %+v %v %v", p, wait, err)
		}
	}
	t.Run("checkout directory linking outside", func(t *testing.T) {
		dir := filepath.Join(root, "bin")
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"code", "xdg-open"} {
			target := executableFixture(t, installed, name, "exit 0")
			if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", filepath.Join(alias, "bin"))
			if got := lookupUnix(name, root); got != "" {
				t.Fatalf("lookup accepted checkout-owned PATH entry: %s", got)
			}
		}
	})
}

func TestUnixDefaultEditorFailure(t *testing.T) {
	root, installed := t.TempDir(), t.TempDir()
	executableFixture(t, installed, "code", "exit 42")
	t.Setenv("PATH", installed)
	p, wait, err := resolveUnix(runtime.GOOS, "editor", workspace.Profile{}, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := (Native{}).Dispatch(Action{Type: "editor", Executable: p.Executable, Args: []string{"--reuse-window", root}, Directory: root, Wait: wait}); err == nil {
		t.Fatal("default editor helper failure reported success")
	}
}

func TestUnixURLDispatch(t *testing.T) {
	root, installed, home := t.TempDir(), t.TempDir(), t.TempDir()
	home, _ = filepath.EvalSymlinks(home)
	t.Setenv("PATH", installed)
	if _, _, err := resolveUnix("linux", "url", workspace.Profile{}, root); err == nil {
		t.Fatal("accepted missing URL helper")
	}
	// A user-installed shebang launcher is supported, with no shell fallback.
	executableFixture(t, installed, "xdg-open", "printf '%s\\n' \"$#\" \"$1\" \"$PWD\" > receipt")
	p, wait, err := resolveUnix("linux", "url", workspace.Profile{}, root)
	if err != nil || !wait {
		t.Fatalf("URL preflight: %+v %v %v", p, wait, err)
	}
	url := "https://example.com/a?q=$(touch%20marker)&b=1#part"
	a := Action{Type: "url", Target: url, Executable: p.Executable, Directory: home, Wait: wait}
	if err := (Native{}).Dispatch(a); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(home, "receipt"))
	if err != nil || string(b) != "1\n"+url+"\n"+home+"\n" {
		t.Fatalf("URL argument/CWD: %q %v", b, err)
	}
	executableFixture(t, installed, "xdg-open", "exit 4")
	if err := (Native{}).Dispatch(a); err == nil {
		t.Fatal("URL helper failure reported success")
	}
	a.Target = "file:///etc/passwd"
	if err := (Native{}).Dispatch(a); err == nil {
		t.Fatal("accepted unsafe URL")
	}
	broken := filepath.Join(installed, "broken")
	if err := os.WriteFile(broken, []byte("touch marker\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := (Native{}).Dispatch(Action{Type: "app", Executable: broken, Directory: root}); err == nil {
		t.Fatal("script without shebang silently fell back to shell")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("shell injection created a marker")
	}
}

func TestUnixUnsupportedSession(t *testing.T) {
	for _, key := range []string{"SSH_CONNECTION", "SSH_TTY", "WSL_DISTRO_NAME", "WSL_INTEROP"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "fixture")
			_, _, err := (Native{}).Resolve("url", workspace.Profile{}, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), "local desktop") {
				t.Fatalf("unsupported session: %v", err)
			}
		})
	}
	if runtime.GOOS == "linux" {
		t.Setenv("DISPLAY", "")
		t.Setenv("WAYLAND_DISPLAY", "")
		if _, _, err := (Native{}).Resolve("url", workspace.Profile{}, t.TempDir()); err == nil {
			t.Fatal("accepted headless launch")
		}
	}
}
