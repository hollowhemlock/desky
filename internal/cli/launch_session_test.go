package cli

import (
	"os"
	"strings"
	"testing"
)

func TestHeadlessEntryPreservesInspectionAndSaving(t *testing.T) {
	root, l := cliFixture(t)
	t.Setenv("SSH_CONNECTION", "fixture")
	code, out, _ := invoke(t, root, l, "open", root, "--dry-run", "--json")
	if code != 7 || !strings.Contains(out, "unsupported_action") {
		t.Fatalf("headless entry: %d %s", code, out)
	}
	code, out, _ = invoke(t, root, l, "info", "--json")
	if code != 0 || !strings.Contains(out, "not_evaluated") {
		t.Fatalf("headless inspection: %d %s", code, out)
	}
	if _, err := os.Stat(l.StateDir); !os.IsNotExist(err) {
		t.Fatal("headless preflight or inspection wrote state")
	}
	code, out, _ = invoke(t, root, l, "url", "add", "https://example.com/saved", "--json")
	if code != 0 {
		t.Fatalf("headless save: %d %s", code, out)
	}
	code, out, _ = invoke(t, root, l, "url", "list", "--json")
	if code != 0 || !strings.Contains(out, "https://example.com/saved") {
		t.Fatalf("headless retrieval: %d %s", code, out)
	}
}
