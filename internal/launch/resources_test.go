package launch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hollowhemlock/desky/internal/resources"
	"github.com/hollowhemlock/desky/internal/workspace"
)

func TestConflictingURLsSkipOnlyAffectedResources(t *testing.T) {
	s, f, root := fixture(t)
	recipe(t, root)
	u := resources.New(s.Workspace)
	r, err := u.SaveURL(root, "", "https://conflict.example/", "Conflict", true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := u.SaveURL(root, "", "https://healthy.example/", "Healthy", true); err != nil {
		t.Fatal(err)
	}
	if _, err := u.SaveURL(root, "", "https://example.com:443/docs?q=one#two", "Shared duplicate", true); err != nil {
		t.Fatal(err)
	}
	if _, err := u.SaveURL(root, "", "https://saved.example/", "Saved", false); err != nil {
		t.Fatal(err)
	}
	first, err := s.Open(root, Request{Selector: "."}, func(Plan) bool { return true })
	if err != nil || len(f.actions) != 5 {
		t.Fatalf("initial entry: %d %v", len(f.actions), err)
	}
	// Simulate an independently edited root revision delivered by another device.
	other := r.Heads[0]
	other.RevisionID, _ = workspace.NewID()
	other.Status = "archived"
	b, _ := json.Marshal(other)
	path := filepath.Join(s.Workspace.Locations.PersonalDir, "workspaces", other.WorkspaceID, "resources", r.ID, "provider conflict copy.json")
	put(t, path, string(b))
	before := snapshot(t, s.Workspace.Locations.PersonalDir)
	f.actions = nil
	plan, err := s.Prepare(root, Request{Selector: "."})
	if err != nil || plan.Digest != first.Plan.Digest || !plan.Trusted || len(plan.Skipped) != 1 {
		t.Fatalf("conflict plan: %+v %v", plan, err)
	}
	result, err := s.Open(root, Request{Selector: "."}, nil)
	wantCode(t, err, "resource_conflict")
	if len(f.actions) != 4 || len(result.Items) != 5 || result.Items[0].Status != "skipped" || result.Items[0].ID != r.ID {
		t.Fatalf("unaffected entry: %+v %+v", f.actions, result)
	}
	for _, a := range f.actions {
		if a.Target == r.URL || a.Target == "https://saved.example/" {
			t.Fatal("conflicted/saved resource launched")
		}
	}
	if !reflect.DeepEqual(before, snapshot(t, s.Workspace.Locations.PersonalDir)) {
		t.Fatal("entry changed conflicting data")
	}
	info, _ := s.Workspace.Info(root, ".")
	if info.EntryCount != 2 {
		t.Fatalf("partial entry recency: %+v", info)
	}
	inspected := s.InspectTrust(info)
	if len(inspected.ResourceWarnings) != 1 {
		t.Fatal("info hides conflict")
	}
	// A pin/status change does not require repository reapproval.
	if _, err := u.SetURLStatus(root, "", r.ID, "pinned", true); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.Prepare(root, Request{Selector: "."})
	if err != nil || !resolved.Trusted || resolved.Digest != first.Plan.Digest || len(resolved.Items) != 5 {
		t.Fatalf("resolved plan: %+v %v", resolved, err)
	}
}

func TestMalformedPersonalDataAndEmptyPlan(t *testing.T) {
	s, f, root := fixture(t)
	u := resources.New(s.Workspace)
	r, err := u.SaveURL(root, "", "https://example.com/", "Page", true)
	if err != nil {
		t.Fatal(err)
	}
	rev := r.Heads[0]
	path := filepath.Join(s.Workspace.Locations.PersonalDir, "workspaces", rev.WorkspaceID, "resources", r.ID, "bad.json")
	put(t, path, "{")
	result, err := s.Open(root, Request{Selector: "."}, func(Plan) bool { return true })
	wantCode(t, err, "resource_conflict")
	if len(f.actions) != 1 || f.actions[0].Type != "editor" || len(result.Plan.Skipped) != 1 {
		t.Fatal("malformed resource blocked healthy editor")
	}
	// With no independent resources there is no dispatch or recency update.
	put(t, filepath.Join(root, "workspace.toml"), "schema_version=1\n[workspace]\n")
	f.actions = nil
	_, err = s.Open(root, Request{Selector: "."}, nil)
	wantCode(t, err, "resource_conflict")
	if len(f.actions) != 0 {
		t.Fatal("unexpected dispatch")
	}
	info, _ := s.Workspace.Info(root, ".")
	if info.EntryCount != 1 {
		t.Fatal("empty plan recorded entry")
	}
	if b, _ := os.ReadFile(path); string(b) != "{" {
		t.Fatal("malformed data overwritten")
	}
}

func TestPersonalChangesDuringConsentAreReRead(t *testing.T) {
	s, f, root := fixture(t)
	u := resources.New(s.Workspace)
	_, err := u.SaveURL(root, "", "https://before.example/", "Before", true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Open(root, Request{Selector: "."}, func(Plan) bool {
		_, err := u.SaveURL(root, "", "https://after.example/", "After", true)
		if err != nil {
			t.Fatal(err)
		}
		return true
	})
	if err != nil || len(f.actions) != 3 {
		t.Fatalf("personal re-read: %+v %v", f.actions, err)
	}
}
