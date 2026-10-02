package resources

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hollowhemlock/desky/internal/fileio"
	"github.com/hollowhemlock/desky/internal/workspace"
)

func TestFailedSaveRetryLeavesUsableState(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "first-save", true: "existing-data"}[existing], func(t *testing.T) {
			s, root := fixture(t)
			wantCount := 1
			if existing {
				saved(t, s, root)
				wantCount++
			}
			before := history(t, s.Workspace.Locations.PersonalDir)
			s.write = func(string, []byte, bool) error { return errors.New("injected publication failure") }
			if _, err := s.SaveURL(root, "", "https://retry.example/", "Retry", true); err == nil {
				t.Fatal("write failure hidden")
			}
			registered, err := s.Workspace.Info(root, ".")
			if err != nil || !registered.Registered || registered.EntryCount != 0 || registered.WorkspaceID == "" {
				t.Fatalf("failed save identity: %+v %v", registered, err)
			}
			// Recreate both services so recovery cannot depend on in-memory state.
			s = New(workspace.New(s.Workspace.Locations))
			r, err := s.SaveURL(root, "", "https://retry.example/", "Retry", true)
			if err != nil || r.Status != "pinned" || r.Heads[0].WorkspaceID != registered.WorkspaceID {
				t.Fatalf("retry: %+v %v", r, err)
			}
			list, err := s.ListResources(root, "", true)
			if err != nil || len(list) != wantCount {
				t.Fatalf("retry left orphan resources: %+v %v", list, err)
			}
			for _, item := range list {
				if item.Problem != "" {
					t.Fatalf("retry left conflict: %+v", item)
				}
			}
			if _, err := s.SetURLStatus(root, "", r.ID, "archived", false); err != nil {
				t.Fatalf("retried resource is unusable: %v", err)
			}
			after := history(t, s.Workspace.Locations.PersonalDir)
			for path, contents := range before {
				if after[path] != contents {
					t.Fatalf("existing data changed: %s", path)
				}
			}
			info, err := s.Workspace.Info(root, ".")
			if err != nil || info.CheckoutID != registered.CheckoutID || info.WorkspaceID != registered.WorkspaceID || info.EntryCount != 0 {
				t.Fatalf("retry changed registration: %+v %v", info, err)
			}
			if _, err := os.Stat(filepath.Join(s.Workspace.Locations.StateDir, "trust.json")); !os.IsNotExist(err) {
				t.Fatalf("save wrote trust: %v", err)
			}
		})
	}
}

func TestFailedPublishPreservesPreexistingDirectories(t *testing.T) {
	s, root := fixture(t)
	r := saved(t, s, root).Heads[0]
	for _, empty := range []bool{false, true} {
		if empty {
			r.ResourceID, _ = workspace.NewID()
		}
		path := filepath.Join(s.Workspace.Locations.PersonalDir, "workspaces", r.WorkspaceID, "resources", r.ResourceID)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		before := history(t, s.Workspace.Locations.PersonalDir)
		err := publish(s.Workspace.Locations.PersonalDir, r, func(string, []byte, bool) error {
			return errors.New("injected failure")
		})
		if err == nil {
			t.Fatal("write failure hidden")
		}
		if st, err := os.Stat(path); err != nil || !st.IsDir() {
			t.Fatalf("preexisting directory removed: %v", err)
		}
		if !reflect.DeepEqual(before, history(t, s.Workspace.Locations.PersonalDir)) {
			t.Fatal("preexisting data changed")
		}
		if empty {
			items, err := s.ListResources(root, "", true)
			if err != nil || len(items) != 2 {
				t.Fatalf("incomplete delivery hidden: %+v %v", items, err)
			}
			for _, item := range items {
				if item.ID == r.ResourceID && item.Problem == "" {
					t.Fatal("empty synchronized resource no longer diagnosed")
				}
			}
		}
	}
}

func TestFailedSavePreservesFilesArrivingDuringPublication(t *testing.T) {
	for _, kind := range []string{"published", "provider-copy", "temporary", "replacement-file"} {
		t.Run(kind, func(t *testing.T) {
			s, root := fixture(t)
			var keptPath string
			var keptData []byte
			s.write = func(path string, data []byte, replace bool) error {
				keptPath, keptData = path, data
				switch kind {
				case "provider-copy":
					keptPath = filepath.Join(filepath.Dir(path), "provider conflict copy.json")
				case "temporary":
					keptPath, keptData = filepath.Join(filepath.Dir(path), "provider.tmp"), []byte("partial delivery")
				case "replacement-file":
					keptPath, keptData = filepath.Dir(path), []byte("external replacement")
					if err := os.Remove(keptPath); err != nil {
						t.Fatal(err)
					}
				}
				if err := fileio.Write(keptPath, keptData, replace); err != nil {
					t.Fatal(err)
				}
				return errors.New("injected failure after file arrived")
			}
			if _, err := s.SaveURL(root, "", "https://retry.example/", "Original", true); err == nil {
				t.Fatal("write failure hidden")
			}
			s = New(workspace.New(s.Workspace.Locations))
			if _, err := s.SaveURL(root, "", "https://retry.example/", "Retry", false); err != nil {
				t.Fatal(err)
			}
			if data, err := os.ReadFile(keptPath); err != nil || string(data) != string(keptData) {
				t.Fatalf("arriving data lost: %q %v", data, err)
			}
			items, err := s.ListResources(root, "", true)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "published" || kind == "provider-copy" {
				if len(items) != 1 || items[0].Problem != "" || items[0].Title != "Original" || items[0].Status != "pinned" {
					t.Fatalf("retry did not reuse published data: %+v", items)
				}
			} else if len(items) != 2 {
				t.Fatalf("externally supplied incomplete data hidden: %+v", items)
			}
		})
	}
}
