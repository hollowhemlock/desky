package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hollowhemlock/desky/internal/workspace"
)

func TestNativePreflightAndLookup(t *testing.T) {
	root := t.TempDir()
	directoryExecutable := filepath.Join(t.TempDir(), "wt.exe")
	if err := os.Mkdir(directoryExecutable, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveWindows("terminal", workspace.Profile{Executable: directoryExecutable}, root); err == nil {
		t.Fatal("accepted directory as terminal executable")
	}
	for _, path := range []string{"relative.exe", filepath.Join(root, "missing.exe"), filepath.Join(root, "code.cmd")} {
		_, _, err := resolveWindows("editor", workspace.Profile{Executable: path}, root)
		if err == nil {
			t.Fatal("accepted missing/unsafe launcher", path)
		}
	}
	_, _, err := resolveWindows("url", workspace.Profile{}, `\\server\share`)
	if err == nil {
		t.Fatal("accepted network checkout")
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+".")
	if err = os.WriteFile(filepath.Join(root, "wt.exe"), []byte("untrusted"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err = resolveWindows("terminal", workspace.Profile{}, root)
	if err == nil {
		t.Fatal("resolved checkout executable")
	}
	t.Run("checkout alias", func(t *testing.T) {
		alias := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(root, alias); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, _, err := resolveWindows("terminal", workspace.Profile{}, alias); err == nil {
			t.Fatal("resolved executable through checkout alias")
		}
	})
	installed := t.TempDir()
	t.Setenv("PATH", installed)
	if err = os.WriteFile(filepath.Join(installed, "wt.exe"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	p, wait, err := resolveWindows("terminal", workspace.Profile{}, root)
	if err != nil || !wait || strings.Join(p.Args, "|") != "new-tab|-d|." {
		t.Fatalf("terminal default: %+v %v %v", p, wait, err)
	}
}
