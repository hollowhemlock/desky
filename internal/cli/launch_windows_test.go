package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hollowhemlock/desky/internal/launch"
)

func TestLaunchReceipt(t *testing.T) {
	if os.Getenv("DESKY_TEST_PROCESS") != "1" {
		return
	}
	for i, a := range os.Args {
		if a == "--" {
			cwd, _ := os.Getwd()
			if cwd != os.Args[i+2] {
				os.Exit(90)
			}
			if os.WriteFile(os.Args[i+1]+".tmp", []byte(cwd), 0600) != nil {
				os.Exit(91)
			}
			if os.Rename(os.Args[i+1]+".tmp", os.Args[i+1]) != nil {
				os.Exit(93)
			}
			os.Exit(0)
		}
	}
	os.Exit(92)
}
func TestEntryAcrossCLIProcesses(t *testing.T) {
	root, l := cliFixture(t)
	exe, _ := os.Executable()
	receipt := filepath.Join(l.Home, "receipt")
	if err := os.MkdirAll(filepath.Dir(l.ConfigFile), 0700); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf("schema_version=1\n[launchers.editor]\nexecutable=%q\nargs=['-test.run=^TestLaunchReceipt$','--',%q,'{path}']\n", exe, receipt)
	if err := os.WriteFile(l.ConfigFile, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	b, err := process(t, root, l, "open", root, "--dry-run", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v", b, err)
	}
	var dry struct{ Data launch.Result }
	if err = json.Unmarshal(b, &dry); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(l.StateDir); !os.IsNotExist(err) {
		t.Fatal("dry run created state")
	}
	b, err = process(t, root, l, root, "--json").CombinedOutput()
	if err == nil {
		t.Fatal("noninteractive entry bypassed consent")
	}
	if _, err = os.Stat(receipt); !os.IsNotExist(err) {
		t.Fatal("unapproved entry dispatched")
	}
	b, err = process(t, root, l, root, "--trust", dry.Data.Plan.Digest, "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v", b, err)
	}
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err = os.Stat(receipt); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatal("helper was not dispatched", err)
	}
	if err = os.Remove(receipt); err != nil {
		t.Fatal(err)
	}
	b, err = process(t, root, l, "open", root, "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("stored consent not reused: %s %v", b, err)
	}
	// Observe completion of the repeated detached helper before fixture cleanup.
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if _, err = os.Stat(receipt); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatal("repeated helper was not dispatched", err)
	}
}
