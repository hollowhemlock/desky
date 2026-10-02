package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hollowhemlock/desky/internal/workspace"
)

func cliFixture(t *testing.T) (string, workspace.Locations) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "project")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root, workspace.Locations{Home: base, ConfigFile: filepath.Join(base, "config", "config.toml"), StateDir: filepath.Join(base, "state"), PersonalDir: filepath.Join(base, "personal")}
}

func invoke(t *testing.T, root string, l workspace.Locations, args ...string) (int, string, string) {
	t.Helper()
	var out, err bytes.Buffer
	exit := run(args, &out, &err, func() (string, error) { return root, nil }, func() (workspace.Locations, error) { return l, nil })
	return exit, out.String(), err.String()
}

func TestCLIJSONAndErrors(t *testing.T) {
	root, l := cliFixture(t)
	cases := []struct {
		args []string
		exit int
		code string
	}{
		{[]string{"list", "--json"}, 0, ""},
		{[]string{"info", "--json"}, 0, ""},
		{[]string{"info", "./missing", "--json"}, 3, "workspace_not_found"},
		{[]string{"info", "--unknown", "--json"}, 2, "usage"},
		{[]string{"list", "--name", "x", "--json"}, 2, "usage"},
		{[]string{"init", "--name=", "--json"}, 2, "usage"},
		{[]string{"url", "add", "https://example.com", "--json"}, 2, "not_implemented"},
		{[]string{"open", "--json"}, 2, "usage"},
		{[]string{"--json"}, 2, "not_implemented"},
	}
	for _, tt := range cases {
		exit, out, stderr := invoke(t, root, l, tt.args...)
		if exit != tt.exit || stderr != "" {
			t.Fatalf("%v: exit %d stderr %q", tt.args, exit, stderr)
		}
		var e struct {
			Schema int              `json:"schema_version"`
			OK     bool             `json:"ok"`
			Error  *workspace.Error `json:"error"`
		}
		if err := json.Unmarshal([]byte(out), &e); err != nil || e.Schema != 1 || e.OK != (exit == 0) {
			t.Fatalf("invalid envelope: %s %v", out, err)
		}
		if tt.code != "" && (e.Error == nil || e.Error.Code != tt.code) {
			t.Fatalf("wrong error: %s", out)
		}
	}
	if _, err := os.Stat(l.StateDir); !os.IsNotExist(err) {
		t.Fatal("read-only/error commands created state")
	}
}

func TestCLIReservedNamesAndTerminalEscapes(t *testing.T) {
	root, l := cliFixture(t)
	exit, _, stderr := invoke(t, root, l, "init", "--name", "config\x1b[31m\nattack")
	if exit != 0 {
		t.Fatalf("init: %s", stderr)
	}
	exit, out, _ := invoke(t, root, l, "info")
	if exit != 0 || strings.Contains(out, "\x1b") || strings.Contains(out, "\nattack") {
		t.Fatalf("unsafe output %q", out)
	}
	exit, _, _ = invoke(t, root, l, "config", "path")
	if exit != 0 {
		t.Fatal("reserved command not recognized")
	}
	put := filepath.Join(root, "--json")
	os.Mkdir(put, 0700)
	exit, out, _ = invoke(t, root, l, "info", "--", "--json")
	if exit != 0 || strings.HasPrefix(out, "{") {
		t.Fatal("--json after -- was parsed as flag")
	}
}

// The test executable supplies the same process boundary as main while allowing
// explicit fixture locations on every OS, without any real profile data access.
func TestCLIProcess(t *testing.T) {
	if os.Getenv("DESKY_TEST_PROCESS") != "1" {
		return
	}
	var l workspace.Locations
	if err := json.Unmarshal([]byte(os.Getenv("DESKY_TEST_LOCATIONS")), &l); err != nil {
		os.Exit(99)
	}
	args := os.Args
	for i, a := range args {
		if a == "--" {
			args = args[i+1:]
			break
		}
	}
	os.Exit(run(args, os.Stdout, os.Stderr, os.Getwd, func() (workspace.Locations, error) { return l, nil }))
}

func process(t *testing.T, root string, l workspace.Locations, args ...string) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
	cmd.Dir = root
	data, _ := json.Marshal(l)
	cmd.Env = append(os.Environ(), "DESKY_TEST_PROCESS=1", "DESKY_TEST_LOCATIONS="+string(data))
	return cmd
}

func TestSubprocessPersistenceAndConcurrentWriters(t *testing.T) {
	root, l := cliFixture(t)
	first, err := process(t, root, l, "init", "--name", "first", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v", first, err)
	}
	second, err := process(t, root, l, "info", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v", second, err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("fresh process changed identity:\n%s\n%s", first, second)
	}
	const n = 4
	var wg sync.WaitGroup
	results := make(chan string, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(filepath.Dir(root), string(rune('a'+i)))
		os.Mkdir(p, 0700)
		cmd := process(t, p, l, "init", "--json")
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, e := cmd.CombinedOutput()
			if e != nil {
				results <- string(b) + e.Error()
			}
		}()
	}
	wg.Wait()
	close(results)
	for failure := range results {
		t.Error(failure)
	}
	b, err := process(t, root, l, "list", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	var result struct {
		Data []workspace.ListedCheckout `json:"data"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Data) != n+1 {
		t.Fatalf("lost process registrations: %d", len(result.Data))
	}
	cmd := process(t, root, l, "init", "--json")
	b, err = cmd.CombinedOutput()
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 5 {
		t.Fatalf("repeated init exit: %v %s", err, b)
	}
}
