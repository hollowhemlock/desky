package launch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hollowhemlock/desky/internal/resources"
	"github.com/hollowhemlock/desky/internal/workspace"
)

// BenchmarkLargeWorkspace excludes process startup, dispatch and fixture writes.
// Both operations see 1,000 real checkout directories and 1,000 current URLs.
func BenchmarkLargeWorkspace(b *testing.B) {
	base := b.TempDir()
	l := workspace.Locations{Home: base, ConfigFile: filepath.Join(base, "config.toml"), StateDir: filepath.Join(base, "state"), PersonalDir: filepath.Join(base, "personal")}
	checkouts := make([]workspace.Checkout, 0, 1000)
	wid, _ := workspace.NewID()
	now := time.Now().UTC()
	write := func(path string, data []byte) {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			b.Fatal(err)
		}
	}
	for j := 0; j < 1000; j++ {
		root := filepath.Join(base, "projects", fmt.Sprintf("project-%04d", j))
		if err := os.MkdirAll(root, 0700); err != nil {
			b.Fatal(err)
		}
		cid, _ := workspace.NewID()
		checkouts = append(checkouts, workspace.Checkout{CheckoutID: cid, WorkspaceID: wid, RootPath: root, IdentitySource: "directory", Name: fmt.Sprintf("project-%04d", j), LastEnteredAt: &now, EntryCount: 1})
		rid, _ := workspace.NewID()
		rev, _ := workspace.NewID()
		r := resources.Revision{SchemaVersion: 1, WorkspaceID: wid, ResourceID: rid, RevisionID: rev, Parents: []string{}, Type: "url", URL: fmt.Sprintf("https://example.com/%d", j), Title: "Page", Status: "pinned", CreatedAt: now.Format(time.RFC3339Nano), UpdatedAt: now.Format(time.RFC3339Nano)}
		data, _ := json.Marshal(r)
		write(filepath.Join(l.PersonalDir, "workspaces", wid, "resources", rid, rev+".json"), data)
	}
	data, _ := json.Marshal(struct {
		SchemaVersion int                  `json:"schema_version"`
		Checkouts     []workspace.Checkout `json:"checkouts"`
	}{1, checkouts})
	write(filepath.Join(l.StateDir, "registry.json"), data)
	s := &Service{workspace.New(l), &fakePlatform{}}
	b.Run("list-1000", func(b *testing.B) {
		for b.Loop() {
			if _, err := s.Workspace.List(); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("dry-run-1000", func(b *testing.B) {
		for b.Loop() {
			if _, err := s.Open(checkouts[0].RootPath, Request{Selector: ".", DryRun: true}, nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}
