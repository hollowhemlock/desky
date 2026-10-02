package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hollowhemlock/desky/internal/resources"
	"github.com/hollowhemlock/desky/internal/workspace"
)

func TestURLSubprocessPersistenceAndConcurrency(t *testing.T) {
	root, l := cliFixture(t)
	var wg sync.WaitGroup
	results := make(chan []byte, 4)
	for j := 0; j < 4; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, err := process(t, root, l, "url", "add", "https://example.com/", "--title", "Guide", "--json").CombinedOutput()
			if err != nil {
				t.Errorf("save: %s %v", b, err)
			}
			results <- b
		}()
	}
	wg.Wait()
	close(results)
	id := ""
	for b := range results {
		var e struct {
			OK   bool
			Data resources.Resource
		}
		if err := json.Unmarshal(b, &e); err != nil || !e.OK {
			t.Fatalf("save: %s %v", b, err)
		}
		if id != "" && id != e.Data.ID {
			t.Fatal("concurrent duplicate ID")
		}
		id = e.Data.ID
	}
	for _, command := range []string{"pin", "unpin", "archive", "restore"} {
		b, err := process(t, root, l, "url", command, id, "--json").CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %s %v", command, b, err)
		}
	}
	b, err := process(t, root, l, "url", "list", "--json").CombinedOutput()
	var e struct{ Data []resources.Resource }
	if err != nil || json.Unmarshal(b, &e) != nil || len(e.Data) != 1 || e.Data[0].ID != id || e.Data[0].Title != "Guide" || e.Data[0].Status != "saved" {
		t.Fatalf("restart: %s %v", b, err)
	}
	// Human output escapes titles and does not dump saved URLs by default.
	exit, out, stderr := invoke(t, root, l, "url", "add", "https://example.com/private?secret=value", "--title", "Guide\x1b[31m")
	if exit != 0 || stderr != "" || strings.Contains(out, "\x1b") || strings.Contains(out, "secret=value") {
		t.Fatalf("unsafe output: %q %q", out, stderr)
	}
}

func TestPickerQueriesNumbersAndNoninteractive(t *testing.T) {
	root, l := cliFixture(t)
	w := workspace.New(l)
	_, err := w.Init(root, "same")
	if err != nil {
		t.Fatal(err)
	}
	second := filepath.Join(filepath.Dir(root), "second")
	if err := os.Mkdir(second, 0700); err != nil {
		t.Fatal(err)
	}
	_, err = w.Init(second, "same")
	if err != nil {
		t.Fatal(err)
	}
	// Recency is owned by workspace, tested through its actual entry transaction.
	i, _ := w.Info(second, ".")
	if err := w.VisitEntry(i, func(workspace.Info) bool { return true }); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ input, want string }{{"\n", second}, {"2\n", root}, {"project\n\n", root}, {"absent\n/\n1\n", second}, {"999\n1\n", second}} {
		var out bytes.Buffer
		got, err := pick(w, bufio.NewReader(strings.NewReader(tc.input)), &out, true, nil)
		if err != nil || got != tc.want {
			t.Fatalf("%q: %s %v", tc.input, got, err)
		}
		if !strings.Contains(out.String(), root) || !strings.Contains(out.String(), second) {
			t.Fatal("duplicate names hide paths")
		}
	}
	for _, tc := range []struct {
		args        []string
		input       string
		interactive bool
		want        int
	}{
		{nil, ":q\n", true, 130},
		{nil, "1\ny\n", false, 2},
		{[]string{"--json"}, "1\ny\n", true, 2},
		{[]string{"info", "same"}, "2\n", true, 0},
		{[]string{"info", "same", "--json"}, "2\n", true, 4},
	} {
		var out, diagnostics bytes.Buffer
		exit := runInteractive(tc.args, &out, &diagnostics, func() (string, error) { return l.Home, nil }, func() (workspace.Locations, error) { return l, nil }, strings.NewReader(tc.input), tc.interactive)
		if exit != tc.want {
			t.Fatalf("%v: %d %s %s", tc.args, exit, &out, &diagnostics)
		}
		if (!tc.interactive || wantsJSON(tc.args)) && strings.Contains(diagnostics.String(), "Number or query") {
			t.Fatal("noninteractive prompt")
		}
	}
}
