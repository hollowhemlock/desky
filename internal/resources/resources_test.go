package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/hollowhemlock/desky/internal/fileio"
	"github.com/hollowhemlock/desky/internal/workspace"
)

func fixture(t *testing.T) (*Service, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "project")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	w := workspace.New(workspace.Locations{Home: base, ConfigFile: filepath.Join(base, "config.toml"), PersonalDir: filepath.Join(base, "personal"), StateDir: filepath.Join(base, "state")})
	return New(w), root
}

func saved(t *testing.T, s *Service, root string) Resource {
	t.Helper()
	r, err := s.SaveURL(root, "", "https://EXAMPLE.com:443/page?q=secret#fragment", "Page", false)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func history(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !d.IsDir() {
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			rel, _ := filepath.Rel(dir, p)
			out[rel] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestStatusHistoryDedupAndDirectoryIdentity(t *testing.T) {
	s, root := fixture(t)
	r := saved(t, s, root)
	initial := history(t, s.Workspace.Locations.PersonalDir)
	duplicate, err := s.SaveURL(root, "", "https://example.com/page?q=secret#fragment", "Changed", true)
	if err != nil || !reflect.DeepEqual(r, duplicate) {
		t.Fatalf("duplicate changed resource: %+v %v", duplicate, err)
	}
	for _, status := range []string{"pinned", "saved", "archived", "saved"} {
		changed, err := s.SetURLStatus(root, "", r.ID, status, false)
		if err != nil || changed.ID != r.ID || changed.Status != status {
			t.Fatalf("status: %+v %v", changed, err)
		}
		list, err := New(workspace.New(s.Workspace.Locations)).ListResources(root, "", false)
		if err != nil || (status == "archived" && len(list) != 0) || (status != "archived" && (len(list) != 1 || list[0].Status != status)) {
			t.Fatalf("restart list: %+v %v", list, err)
		}
	}
	all := history(t, s.Workspace.Locations.PersonalDir)
	if len(all) != 5 {
		t.Fatalf("expected retained revisions: %d", len(all))
	}
	for p, contents := range initial {
		if all[p] != contents {
			t.Fatal("initial revision changed")
		}
	}
	i, err := s.Workspace.Init(root, "explicit")
	if err != nil || i.WorkspaceID != r.Heads[0].WorkspaceID || i.EntryCount != 0 {
		t.Fatalf("init lost identity: %+v %v", i, err)
	}
	for _, contents := range all {
		if strings.Contains(contents, root) || strings.Contains(contents, "checkout_id") || strings.Contains(contents, "digest") {
			t.Fatal("device state leaked into personal data")
		}
	}
}

func TestReplicaUnionConflictAndResolution(t *testing.T) {
	a, rootA := fixture(t)
	i, err := a.Workspace.Init(rootA, "same")
	if err != nil {
		t.Fatal(err)
	}
	r := saved(t, a, rootA)
	b, rootB := fixture(t)
	if err := os.CopyFS(b.Workspace.Locations.PersonalDir, os.DirFS(a.Workspace.Locations.PersonalDir)); err != nil {
		t.Fatal(err)
	}
	cfg, _ := os.ReadFile(filepath.Join(rootA, "workspace.toml"))
	if err := os.WriteFile(filepath.Join(rootB, "workspace.toml"), cfg, 0600); err != nil {
		t.Fatal(err)
	}
	list, err := b.ListResources(rootB, "", true)
	if err != nil || len(list) != 1 || list[0].ID != r.ID {
		t.Fatalf("reconnect: %+v %v", list, err)
	}
	if _, err := os.Stat(b.Workspace.Locations.StateDir); !os.IsNotExist(err) {
		t.Fatal("copy/inspection wrote registry")
	}
	left, err := a.SetURLStatus(rootA, "", r.ID, "pinned", false)
	if err != nil {
		t.Fatal(err)
	}
	right, err := b.SetURLStatus(rootB, "", r.ID, "archived", false)
	if err != nil {
		t.Fatal(err)
	}
	// Different arrival order and timestamps never choose a winner.
	if err := publish(a.Workspace.Locations.PersonalDir, right.Heads[0], fileio.Write); err != nil {
		t.Fatal(err)
	}
	if err := publish(b.Workspace.Locations.PersonalDir, left.Heads[0], fileio.Write); err != nil {
		t.Fatal(err)
	}
	la, _ := a.ListResources(rootA, "", false)
	lb, _ := b.ListResources(rootB, "", false)
	if !reflect.DeepEqual(la, lb) || len(la) != 1 || len(la[0].Heads) != 2 || la[0].Status != "conflict" {
		t.Fatalf("nonconvergent union: %+v %+v", la, lb)
	}
	before := history(t, a.Workspace.Locations.PersonalDir)
	if _, err := a.SetURLStatus(rootA, "", r.ID, "saved", false); err == nil {
		t.Fatal("conflict silently resolved")
	}
	if !reflect.DeepEqual(before, history(t, a.Workspace.Locations.PersonalDir)) {
		t.Fatal("failed mutation wrote files")
	}
	resolved, err := a.SetURLStatus(rootA, "", r.ID, "saved", true)
	if err != nil || len(resolved.Heads[0].Parents) != 2 {
		t.Fatalf("resolution: %+v %v", resolved, err)
	}
	if err := publish(b.Workspace.Locations.PersonalDir, resolved.Heads[0], fileio.Write); err != nil {
		t.Fatal(err)
	}
	la, _ = a.ListResources(rootA, "", true)
	lb, _ = b.ListResources(rootB, "", true)
	if !reflect.DeepEqual(la, lb) {
		t.Fatal("resolution did not converge")
	}
	// An exact UUID can address only copied personal data, without a checkout.
	c, rootC := fixture(t)
	if err := os.CopyFS(c.Workspace.Locations.PersonalDir, os.DirFS(a.Workspace.Locations.PersonalDir)); err != nil {
		t.Fatal(err)
	}
	if list, err := c.ListResources(rootC, i.WorkspaceID, true); err != nil || len(list) != 1 {
		t.Fatalf("remote exact identity: %v %v", list, err)
	}
	if list, err := c.ListResources(rootC, "", true); err != nil || len(list) != 0 {
		t.Fatal("directory identity merged by name")
	}
}

func TestCorruptionPreservedAndIsolated(t *testing.T) {
	for _, kind := range []string{"missing-parent", "cycle", "schema", "malformed", "different-copy", "renamed-copy", "filename", "duplicate-key", "null", "temporary", "different-title"} {
		t.Run(kind, func(t *testing.T) {
			s, root := fixture(t)
			r := saved(t, s, root)
			rev := r.Heads[0]
			path := filepath.Join(s.Workspace.Locations.PersonalDir, "workspaces", rev.WorkspaceID, "resources", r.ID)
			name := "provider conflict copy.json"
			switch kind {
			case "missing-parent":
				rev.RevisionID, _ = workspace.NewID()
				parent, _ := workspace.NewID()
				rev.Parents = []string{parent}
			case "cycle":
				rev.Parents = []string{rev.RevisionID}
			case "schema":
				rev.SchemaVersion = 2
			case "different-copy":
				rev.Status = "archived"
			case "filename":
				id, _ := workspace.NewID()
				name = id + ".json"
			case "different-title":
				rev.RevisionID, _ = workspace.NewID()
				rev.Title = "other"
			case "temporary":
				name = "partial.tmp"
			}
			data, _ := json.Marshal(rev)
			if kind == "malformed" || kind == "temporary" {
				data = []byte("{")
			}
			if kind == "duplicate-key" {
				data = append([]byte(`{"status":"archived",`), data[1:]...)
			}
			if kind == "null" {
				data = []byte(strings.Replace(string(data), `"parents":[]`, `"parents":null`, 1))
			}
			if err := os.WriteFile(filepath.Join(path, name), data, 0600); err != nil {
				t.Fatal(err)
			}
			before := history(t, s.Workspace.Locations.PersonalDir)
			list, err := s.ListResources(root, "", true)
			if err != nil || len(list) != 1 {
				t.Fatalf("list: %+v %v", list, err)
			}
			if kind == "renamed-copy" || kind == "temporary" {
				if list[0].Problem != "" {
					t.Fatal(list[0].Problem)
				}
			} else {
				if list[0].Problem == "" {
					t.Fatal("bad data hidden")
				}
				if _, err := s.SetURLStatus(root, "", r.ID, "pinned", true); err == nil {
					t.Fatal("bad data repaired automatically")
				}
			}
			if !reflect.DeepEqual(before, history(t, s.Workspace.Locations.PersonalDir)) {
				t.Fatal("data changed")
			}
			if _, err := s.SaveURL(root, "", "https://healthy.example/", "Healthy", true); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestConcurrentSavesAndWriteFailure(t *testing.T) {
	s, root := fixture(t)
	var wg sync.WaitGroup
	errorsOut := make(chan error, 8)
	for j := 0; j < 8; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.SaveURL(root, "", "https://example.com/", "One", false)
			errorsOut <- err
		}()
	}
	wg.Wait()
	close(errorsOut)
	for err := range errorsOut {
		if err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.ListResources(root, "", true)
	if err != nil || len(list) != 1 {
		t.Fatalf("concurrent dedup: %v %v", list, err)
	}
	before := history(t, s.Workspace.Locations.PersonalDir)
	s.write = func(string, []byte, bool) error { return errors.New("injected disk failure") }
	if _, err := s.SetURLStatus(root, "", list[0].ID, "archived", false); err == nil {
		t.Fatal("write failure hidden")
	}
	if !reflect.DeepEqual(before, history(t, s.Workspace.Locations.PersonalDir)) {
		t.Fatal("failed write changed data")
	}
}

func TestUnavailableStorageAndUnknownIdentity(t *testing.T) {
	s, root := fixture(t)
	id, _ := workspace.NewID()
	if _, err := s.SaveURL(root, id, "https://example.com/", "", false); err == nil {
		t.Fatal("unknown identity accepted")
	}
	config := fmt.Sprintf("schema_version=1\npersonal_data_dir=%q\n", filepath.Join(s.Workspace.Locations.Home, "missing"))
	if err := os.WriteFile(s.Workspace.Locations.ConfigFile, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveURL(root, "", "https://example.com/", "", false); err == nil {
		t.Fatal("unavailable storage accepted")
	}
	if _, err := os.Stat(s.Workspace.Locations.PersonalDir); !os.IsNotExist(err) {
		t.Fatal("fell back to default personal directory")
	}
}

func TestLinkedResourceAndDeviceResetPreservePersonalData(t *testing.T) {
	s, root := fixture(t)
	if _, err := s.Workspace.Init(root, "explicit"); err != nil {
		t.Fatal(err)
	}
	r := saved(t, s, root)
	wid := r.Heads[0].WorkspaceID
	rid, _ := workspace.NewID()
	external := t.TempDir()
	sentinel := filepath.Join(external, "keep.json")
	if err := os.WriteFile(sentinel, []byte("external content"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(s.Workspace.Locations.PersonalDir, "workspaces", wid, "resources", rid)
	if err := os.Symlink(external, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	list, err := s.ListResources(root, "", true)
	if err != nil || len(list) != 2 {
		t.Fatalf("linked resource: %v %v", list, err)
	}
	for _, item := range list {
		if item.ID == rid && item.Problem == "" {
			t.Fatal("followed linked resource directory")
		}
	}
	if _, err := s.SetURLStatus(root, "", rid, "pinned", true); err == nil {
		t.Fatal("wrote through linked resource")
	}
	if data, _ := os.ReadFile(sentinel); string(data) != "external content" {
		t.Fatal("external data changed")
	}
	// A new empty device-state location models clearing/recreating local state.
	l := s.Workspace.Locations
	l.StateDir = filepath.Join(l.Home, "fresh-device")
	restarted := New(workspace.New(l))
	list, err = restarted.ListResources(root, "", true)
	if err != nil || len(list) != 2 {
		t.Fatalf("reset lost personal data: %v %v", list, err)
	}
	if _, err := os.Stat(l.StateDir); !os.IsNotExist(err) {
		t.Fatal("inspection recreated device state")
	}
}
