package launch

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

	"github.com/hollowhemlock/desky/internal/platform"
	"github.com/hollowhemlock/desky/internal/workspace"
)

type fakePlatform struct {
	actions []platform.Action
	fail    string
	after   func()
}

func (f *fakePlatform) Resolve(kind string, p workspace.Profile, root string) (workspace.Profile, bool, error) {
	if p.Executable == "" && kind != "url" {
		p = workspace.Profile{Executable: filepath.Join(filepath.VolumeName(root)+string(filepath.Separator), "installed", kind+".exe"), Args: []string{"{path}"}}
	}
	return p, false, nil
}
func (f *fakePlatform) Dispatch(a platform.Action) error {
	f.actions = append(f.actions, a)
	if f.after != nil {
		f.after()
	}
	if a.Type == f.fail || f.fail == "all" {
		return errors.New("simulated dispatch failure")
	}
	return nil
}
func fixture(t *testing.T) (*Service, *fakePlatform, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "projects", "-雪 & echo marker; %TEMP% $()")
	if e := os.MkdirAll(root, 0700); e != nil {
		t.Fatal(e)
	}
	root, _ = filepath.EvalSymlinks(root)
	l := workspace.Locations{Home: base, ConfigFile: filepath.Join(base, "config", "config.toml"), StateDir: filepath.Join(base, "state"), PersonalDir: filepath.Join(base, "personal")}
	f := &fakePlatform{}
	return &Service{workspace.New(l), f}, f, root
}
func put(t *testing.T, path, data string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, []byte(data), 0600); e != nil {
		t.Fatal(e)
	}
}
func wantCode(t *testing.T, err error, code string) {
	t.Helper()
	var e *workspace.Error
	if !errors.As(err, &e) || e.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
}
func recipe(t *testing.T, root string) {
	put(t, filepath.Join(root, "workspace.toml"), `schema_version=1
[workspace]
id="225116ad-90e9-42cd-9770-65d4b5d467ab"
[[resource]]
id="edit"
type="editor"
[[resource]]
id="term"
type="terminal"
[[resource]]
id="web"
type="url"
url="https://EXAMPLE.com:443/docs?q=one#two"
[[resource]]
id="duplicate"
type="url"
url="https://example.com/docs?q=one#two"
`)
}
func snapshot(t *testing.T, dir string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, e error) error {
		if os.IsNotExist(e) {
			return nil
		}
		if e != nil {
			return e
		}
		if !d.IsDir() {
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			m[p] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func TestConsentAndDryRunAreReadOnly(t *testing.T) {
	s, f, root := fixture(t)
	before := snapshot(t, s.Workspace.Locations.Home)
	result, err := s.Open(root, Request{Selector: root, DryRun: true}, nil)
	if err != nil || result.Plan.Digest == "" || result.Plan.Trusted {
		t.Fatalf("dry run: %+v %v", result, err)
	}
	_, err = s.Open(root, Request{Selector: root}, nil)
	wantCode(t, err, "trust_required")
	_, err = s.Open(root, Request{Selector: root, Trust: "stale"}, nil)
	wantCode(t, err, "trust_mismatch")
	_, err = s.Open(root, Request{Selector: root}, func(Plan) bool { return false })
	wantCode(t, err, "cancelled")
	if len(f.actions) != 0 || !reflect.DeepEqual(before, snapshot(t, s.Workspace.Locations.Home)) {
		t.Fatal("read-only flow had effects")
	}
	if _, e := os.Stat(s.Workspace.Locations.StateDir); !os.IsNotExist(e) {
		t.Fatal("created state directory")
	}
	if result.Plan.Items[0].Args[0] != root {
		t.Fatal("target argument changed")
	}
}
func TestApprovalRepeatAndMutation(t *testing.T) {
	s, f, root := fixture(t)
	recipe(t, root)
	p, err := s.Prepare(root, Request{Selector: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 3 {
		t.Fatal("URL deduplication failed")
	}
	r, err := s.Open(root, Request{Selector: root, Trust: p.Digest}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Items) != 3 || len(f.actions) != 3 {
		t.Fatal(r)
	}
	i, err := s.Workspace.Info(root, root)
	if err != nil || i.EntryCount != 1 || i.LastEnteredAt == nil {
		t.Fatalf("recency: %+v %v", i, err)
	}
	if _, err = s.Open(root, Request{Selector: root}, nil); err != nil {
		t.Fatal(err)
	}
	i, _ = s.Workspace.Info(root, root)
	if i.EntryCount != 2 {
		t.Fatal("repeat count")
	}
	other := filepath.Join(filepath.Dir(root), "other")
	recipe(t, other)
	_, err = s.Open(root, Request{Selector: other}, nil)
	wantCode(t, err, "trust_required")
	if len(f.actions) != 6 {
		t.Fatal("unapproved worktree dispatched")
	}
	put(t, s.Workspace.Locations.ConfigFile, fmt.Sprintf("schema_version=1\n[launchers.editor]\nexecutable=%q\nargs=['--new-window','{path}']\n", filepath.Join(s.Workspace.Locations.Home, "editor.exe")))
	_, err = s.Open(root, Request{Selector: root}, nil)
	wantCode(t, err, "trust_required")
	_, err = s.Open(root, Request{Selector: root, Trust: p.Digest}, nil)
	wantCode(t, err, "trust_mismatch")
}
func TestPreflightAndApprovalRace(t *testing.T) {
	for _, change := range []string{"path", "url", "argv", "profile", "approval"} {
		t.Run(change, func(t *testing.T) {
			s, f, root := fixture(t)
			recipe(t, root)
			path := filepath.Join(root, "workspace.toml")
			b, _ := os.ReadFile(path)
			switch change {
			case "path":
				b = append(b, []byte("\n[[resource]]\nid='escape'\ntype='editor'\npath='..'\n")...)
			case "url":
				b = []byte(strings.ReplaceAll(string(b), "https://EXAMPLE.com:443/docs?q=one#two", "javascript:alert(1)"))
			case "argv":
				b = append(b, []byte("args=['bad']\n")...)
			case "profile":
				b = append(b, []byte("\n[[resource]]\nid='app'\ntype='app'\nprofile='missing'\n")...)
			}
			if change != "approval" {
				put(t, path, string(b))
			}
			_, err := s.Open(root, Request{Selector: root}, func(Plan) bool {
				put(t, path, strings.ReplaceAll(string(b), "q=one", "q=changed"))
				return true
			})
			if err == nil || len(f.actions) != 0 {
				t.Fatalf("preflight effects: %v %+v", err, f.actions)
			}
		})
	}
}
func TestPartialDispatchAndStateFailure(t *testing.T) {
	for _, mode := range []string{"partial", "all", "state", "both"} {
		t.Run(mode, func(t *testing.T) {
			s, f, root := fixture(t)
			if _, err := s.Workspace.Init(root, "test"); err != nil {
				t.Fatal(err)
			}
			recipeBytes, _ := os.ReadFile(filepath.Join(root, "workspace.toml"))
			put(t, filepath.Join(root, "workspace.toml"), string(recipeBytes)+"\n[[resource]]\nid='term'\ntype='terminal'\n[[resource]]\nid='web'\ntype='url'\nurl='https://example.com/'\n")
			if mode == "partial" || mode == "both" {
				f.fail = "terminal"
			}
			if mode == "all" {
				f.fail = "all"
			}
			if mode == "state" || mode == "both" {
				f.after = func() { os.Mkdir(filepath.Join(s.Workspace.Locations.StateDir, "registry.json.bak"), 0700) }
			}
			r, err := s.Open(root, Request{Selector: root}, func(Plan) bool { return true })
			if err == nil || len(f.actions) != 3 || len(r.Items) != 3 {
				t.Fatalf("dispatch results: %+v %v", r, err)
			}
			var failure *workspace.Error
			errors.As(err, &failure)
			if mode == "state" {
				if failure.Exit != 9 {
					t.Fatal(err)
				}
			} else if failure.Exit != 8 {
				t.Fatal(err)
			}
			i, _ := s.Workspace.Info(root, root)
			want := uint64(0)
			if mode == "partial" {
				want = 1
			}
			if i.EntryCount != want {
				t.Fatalf("count %d want %d", i.EntryCount, want)
			}
			if mode == "state" || mode == "both" {
				if !strings.Contains(r.StateWarning, "applications were dispatched") || r.Items[0].Status != "dispatched" {
					t.Fatal(r)
				}
			}
		})
	}
}
func TestExactSelectorsAndIdentityChange(t *testing.T) {
	s, _, root := fixture(t)
	recipe(t, root)
	open := func(path string) {
		t.Helper()
		if _, e := s.Open(root, Request{Selector: path}, func(Plan) bool { return true }); e != nil {
			t.Fatal(e)
		}
	}
	open(root)
	other := filepath.Join(filepath.Dir(root), "other")
	recipe(t, other)
	open(other)
	i, _ := s.Workspace.Info(root, root)
	_, err := s.Prepare(s.Workspace.Locations.Home, Request{WorkspaceID: i.WorkspaceID})
	wantCode(t, err, "ambiguous_checkout")
	p, err := s.Prepare(root, Request{WorkspaceID: i.WorkspaceID})
	if err != nil || p.Workspace.RootPath != root {
		t.Fatal(err)
	}
	p, err = s.Prepare(root, Request{WorkspaceID: i.WorkspaceID, Checkout: other})
	if err != nil || p.Workspace.RootPath != other {
		t.Fatal(err)
	}
	path := filepath.Join(root, "workspace.toml")
	b, _ := os.ReadFile(path)
	newID := "335116ad-90e9-42cd-9770-65d4b5d467ab"
	put(t, path, strings.ReplaceAll(string(b), i.WorkspaceID, newID))
	_, err = s.Prepare(root, Request{Selector: root})
	wantCode(t, err, "identity_changed")
	p, err = s.Prepare(root, Request{Selector: root, AcceptIdentityChange: true})
	if err != nil || p.Trusted || p.Workspace.PreviousWorkspaceID != i.WorkspaceID {
		t.Fatalf("rebind: %+v %v", p, err)
	}
	_, err = s.Open(root, Request{Selector: root, AcceptIdentityChange: true, Trust: p.Digest}, nil)
	if err != nil {
		t.Fatal(err)
	}
	updated, _ := s.Workspace.Info(root, root)
	if updated.WorkspaceID != newID || updated.CheckoutID != i.CheckoutID || updated.EntryCount != 1 {
		t.Fatal(updated)
	}
}
func TestCorruptTrustPreserved(t *testing.T) {
	s, f, root := fixture(t)
	for _, bad := range []string{`{}`, `{"schema_version":2,"records":[]}`, `{"schema_version":1,"records":[],"extra":true}`, `{"schema_version":1,"records":[]} {}`} {
		p := filepath.Join(s.Workspace.Locations.StateDir, "trust.json")
		put(t, p, bad)
		_, err := s.Open(root, Request{Selector: root}, func(Plan) bool { return true })
		wantCode(t, err, "trust_state")
		b, _ := os.ReadFile(p)
		if string(b) != bad || len(f.actions) > 0 {
			t.Fatal("changed corrupt state")
		}
	}
}
func TestDigestChangesWithEffectiveTarget(t *testing.T) {
	s, _, root := fixture(t)
	recipe(t, root)
	p, _ := s.Prepare(root, Request{Selector: root})
	path := filepath.Join(root, "workspace.toml")
	b, _ := os.ReadFile(path)
	put(t, path, string(b)+"\n# formatting only\n")
	q, _ := s.Prepare(root, Request{Selector: root})
	if p.Digest != q.Digest {
		t.Fatal("formatting invalidates trust")
	}
	put(t, path, strings.ReplaceAll(string(b), "q=one", "q=two"))
	q, _ = s.Prepare(root, Request{Selector: root})
	if p.Digest == q.Digest {
		t.Fatal("target did not invalidate trust")
	}
	encoded, _ := json.Marshal(q)
	if !json.Valid(encoded) {
		t.Fatal("invalid output")
	}
}

func TestConcurrentEntriesAndMRU(t *testing.T) {
	s, _, root := fixture(t)
	other := filepath.Join(filepath.Dir(root), "z-last")
	if err := os.Mkdir(other, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Workspace.Init(other, "unentered"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Open(root, Request{Selector: root}, func(Plan) bool { return true }); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := &Service{workspace.New(s.Workspace.Locations), &fakePlatform{}}
			_, err := local.Open(root, Request{Selector: root}, nil)
			failures <- err
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.Workspace.List()
	if err != nil || len(list) != 2 || list[0].RootPath != root || list[0].EntryCount != 5 {
		t.Fatalf("MRU/concurrency: %+v %v", list, err)
	}
}

func TestAllFailedEntryDoesNotRegister(t *testing.T) {
	s, f, root := fixture(t)
	f.fail = "all"
	_, err := s.Open(root, Request{Selector: root}, func(Plan) bool { return true })
	wantCode(t, err, "launch_failure")
	i, err := s.Workspace.Info(root, root)
	if err != nil || i.Registered || i.EntryCount != 0 {
		t.Fatalf("failed entry registered: %+v %v", i, err)
	}
}

func TestConfiguredApplicationAndUnavailablePersonalData(t *testing.T) {
	s, f, root := fixture(t)
	put(t, filepath.Join(root, "workspace.toml"), "schema_version=1\n[workspace]\n[[resource]]\nid='viewer'\ntype='app'\nprofile='viewer'\n")
	config := fmt.Sprintf("schema_version=1\n[apps.viewer]\nexecutable=%q\nargs=['literal','& ; $()']\n", filepath.Join(s.Workspace.Locations.Home, "viewer.exe"))
	put(t, s.Workspace.Locations.ConfigFile, config)
	if _, err := s.Open(root, Request{Selector: root}, func(Plan) bool { return true }); err != nil {
		t.Fatal(err)
	}
	if len(f.actions) != 1 || f.actions[0].Directory != root || !reflect.DeepEqual(f.actions[0].Args, []string{"literal", "& ; $()"}) {
		t.Fatal(f.actions)
	}
	put(t, s.Workspace.Locations.ConfigFile, strings.Replace(config, "schema_version=1", fmt.Sprintf("schema_version=1\npersonal_data_dir=%q", filepath.Join(s.Workspace.Locations.Home, "unavailable")), 1))
	_, err := s.Prepare(root, Request{Selector: root})
	wantCode(t, err, "personal_data_unavailable")
}
