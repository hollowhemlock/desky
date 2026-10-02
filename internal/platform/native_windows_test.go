package platform

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hollowhemlock/desky/internal/workspace"
)

// This subprocess exits immediately after dispatch. Its delayed child proves
// argument/CWD preservation and survival after the launching process exits.
func TestNativeProcess(t *testing.T) {
	if os.Getenv("DESKY_NATIVE_HELPER") != "1" {
		return
	}
	var args []string
	for i, a := range os.Args {
		if a == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	if args[0] == "parent" {
		exe, _ := os.Executable()
		err := (Native{}).Dispatch(Action{Type: "editor", Executable: exe, Directory: args[1], Args: append([]string{"-test.run=^TestNativeProcess$", "--", "child"}, args[2:]...)})
		if err != nil {
			os.Exit(98)
		}
		os.Exit(0)
	}
	time.Sleep(300 * time.Millisecond)
	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(97)
	}
	b, _ := json.Marshal(struct {
		CWD  string
		Args []string
	}{cwd, args[2:]})
	if os.WriteFile(args[1]+".tmp", b, 0600) != nil {
		os.Exit(96)
	}
	if os.Rename(args[1]+".tmp", args[1]) != nil {
		os.Exit(95)
	}
	os.Exit(0)
}

func TestNativeArgumentsCWDAndDetachment(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "-雪 & echo INJECTED; %TEMP% $(touch nope)")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	root, _ = filepath.EvalSymlinks(root)
	receipt := filepath.Join(base, "receipt.json")
	want := []string{root, "", `a"b`, `trailing\`, "& echo hacked > marker", "; touch marker", "--leading"}
	exe, _ := os.Executable()
	cmd := exec.Command(exe, append([]string{"-test.run=^TestNativeProcess$", "--", "parent", root, receipt}, want...)...)
	cmd.Env = append(os.Environ(), "DESKY_NATIVE_HELPER=1")
	start := time.Now()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("parent: %v %s", err, out)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("launcher waited for child")
	}
	var b []byte
	var err error
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		b, err = os.ReadFile(receipt)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatal("child did not survive parent exit:", err)
	}
	var got struct {
		CWD  string
		Args []string
	}
	if err = json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.CWD != root || !reflect.DeepEqual(got.Args, want) {
		t.Fatalf("got %+v want %q %q", got, root, want)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 0 {
		t.Fatal("argument injection created files")
	}
}

func TestNativePreflightAndLookup(t *testing.T) {
	root := t.TempDir()
	directoryExecutable := filepath.Join(t.TempDir(), "wt.exe")
	if err := os.Mkdir(directoryExecutable, 0700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := (Native{}).Resolve("terminal", workspace.Profile{Executable: directoryExecutable}, root); err == nil {
		t.Fatal("accepted directory as terminal executable")
	}
	for _, path := range []string{"relative.exe", filepath.Join(root, "missing.exe"), filepath.Join(root, "code.cmd")} {
		_, _, err := (Native{}).Resolve("editor", workspace.Profile{Executable: path}, root)
		if err == nil {
			t.Fatal("accepted missing/unsafe launcher", path)
		}
	}
	_, _, err := (Native{}).Resolve("url", workspace.Profile{}, `\\server\share`)
	if err == nil {
		t.Fatal("accepted network checkout")
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+".")
	if err = os.WriteFile(filepath.Join(root, "wt.exe"), []byte("untrusted"), 0600); err != nil {
		t.Fatal(err)
	}
	_, _, err = (Native{}).Resolve("terminal", workspace.Profile{}, root)
	if err == nil {
		t.Fatal("resolved checkout executable")
	}
	t.Run("checkout alias", func(t *testing.T) {
		alias := filepath.Join(t.TempDir(), "alias")
		if err := os.Symlink(root, alias); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, _, err := (Native{}).Resolve("terminal", workspace.Profile{}, alias); err == nil {
			t.Fatal("resolved executable through checkout alias")
		}
	})
	installed := t.TempDir()
	t.Setenv("PATH", installed)
	if err = os.WriteFile(filepath.Join(installed, "wt.exe"), []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	p, wait, err := (Native{}).Resolve("terminal", workspace.Profile{}, root)
	if err != nil || !wait || strings.Join(p.Args, "|") != "new-tab|-d|." {
		t.Fatalf("terminal default: %+v %v %v", p, wait, err)
	}
}
