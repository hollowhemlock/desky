package platform

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
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
		err := (Native{}).Dispatch(Action{Type: "editor", Executable: exe, Directory: args[1], Args: append([]string{"-test.run=^TestNativeProcess$", "--", "gated"}, args[2:]...)})
		if err != nil {
			os.Exit(98)
		}
		os.Exit(0)
	}
	if args[0] == "fail" {
		os.Exit(42)
	}
	if args[0] == "gated" {
		for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(20 * time.Millisecond) {
			if _, err := os.Stat(args[1] + ".release"); err == nil {
				break
			}
			if time.Now().After(deadline) {
				os.Exit(94)
			}
		}
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, append([]string{"-test.run=^TestNativeProcess$", "--", "parent", root, receipt}, want...)...)
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), "DESKY_NATIVE_HELPER=1")
	start := time.Now()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("parent: %v %s", err, out)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("launcher waited for child")
	}
	// The child can finish only after the parent and its output pipes close.
	// Attached standard handles would keep CombinedOutput blocked above.
	if err := os.WriteFile(receipt+".release", nil, 0600); err != nil {
		t.Fatal(err)
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

func TestDispatchHelperFailureAndTimeout(t *testing.T) {
	exe, _ := os.Executable()
	root := t.TempDir()
	cmd := exec.Command(exe, "-test.run=^TestNativeProcess$", "--", "fail")
	cmd.Env = append(os.Environ(), "DESKY_NATIVE_HELPER=1")
	if err := dispatchProcess(cmd, true, 5*time.Second); err == nil {
		t.Fatal("failed helper reported success")
	}
	receipt := filepath.Join(root, "late.json")
	cmd = exec.Command(exe, "-test.run=^TestNativeProcess$", "--", "gated", receipt)
	cmd.Env = append(os.Environ(), "DESKY_NATIVE_HELPER=1")
	if err := dispatchProcess(cmd, true, 30*time.Millisecond); err == nil || !strings.Contains(err.Error(), "outcome unknown") {
		t.Fatalf("timeout: %v", err)
	}
	if err := os.WriteFile(receipt+".release", nil, 0600); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err := os.Stat(receipt); err == nil {
			return
		}
	}
	t.Fatal("timeout killed helper or prevented completion")
}
