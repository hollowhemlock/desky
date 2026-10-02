package workspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hollowhemlock/desky/internal/fileio"
)

func fixture(t *testing.T) (*Service, string) {
	t.Helper()
	base := t.TempDir()
	l := Locations{Home: base, ConfigFile: filepath.Join(base, "device-config", "config.toml"), StateDir: filepath.Join(base, "state"), PersonalDir: filepath.Join(base, "personal")}
	root := filepath.Join(base, "projects", "first")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return New(l), root
}
func put(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func code(t *testing.T, err error, want string) {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) || e.Code != want {
		t.Fatalf("error = %v, want %s", err, want)
	}
}
func read(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInitPersistenceAndNoOverwrite(t *testing.T) {
	s, root := fixture(t)
	i, err := s.Init(root, "project")
	if err != nil {
		t.Fatal(err)
	}
	if !i.Registered || !uuidPattern.MatchString(i.WorkspaceID) || i.IdentitySource != "explicit" {
		t.Fatalf("bad info: %+v", i)
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 1 || entries[0].Name() != "workspace.toml" {
		t.Fatalf("unexpected checkout artifacts: %v", entries)
	}
	config := read(t, i.ConfigPath)
	registryPath := filepath.Join(s.Locations.StateDir, "registry.json")
	state := read(t, registryPath)
	_, err = s.Init(root, "replacement")
	code(t, err, "configuration_exists")
	if !bytes.Equal(config, read(t, i.ConfigPath)) || !bytes.Equal(state, read(t, registryPath)) {
		t.Fatal("reinitialization changed files")
	}
	fresh, err := New(s.Locations).Info(root, ".")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.CheckoutID != i.CheckoutID || fresh.WorkspaceID != i.WorkspaceID {
		t.Fatal("identity did not persist")
	}
	if _, err := s.List(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(state, read(t, registryPath)) {
		t.Fatal("inspection changed state")
	}
}

func TestReadOnlyInspectionAndDiscovery(t *testing.T) {
	s, root := fixture(t)
	put(t, filepath.Join(root, ".git", "config"), "remote URL should never establish identity")
	child := filepath.Join(root, "src")
	os.MkdirAll(child, 0700)
	i, err := s.Info(child, ".")
	if err != nil {
		t.Fatal(err)
	}
	if i.RootPath != root || i.Registered || i.WorkspaceID != "" {
		t.Fatalf("unexpected unregistered identity: %+v", i)
	}
	if _, err := os.Stat(s.Locations.StateDir); !os.IsNotExist(err) {
		t.Fatal("read created state")
	}
	other := filepath.Join(filepath.Dir(root), "other")
	os.MkdirAll(other, 0700)
	j, err := s.Info(other, ".")
	if err != nil {
		t.Fatal(err)
	}
	if j.RootPath == i.RootPath {
		t.Fatal("directories merged")
	}
	if _, err := s.Init(root, "parent"); err != nil {
		t.Fatal(err)
	}
	nested, err := s.Init(child, "nested")
	if err != nil {
		t.Fatal(err)
	}
	i, err = s.Info(child, ".")
	if err != nil || i.WorkspaceID != nested.WorkspaceID {
		t.Fatalf("nested config not selected: %v", err)
	}
	put(t, filepath.Join(child, "workspace.toml"), "not valid TOML !")
	_, err = s.Info(child, ".")
	code(t, err, "invalid_configuration")
	// A Git boundary prevents using an outer config.
	boundary := filepath.Join(root, "separate")
	put(t, filepath.Join(boundary, ".git"), "gitdir: ignored")
	i, err = s.Info(boundary, ".")
	if err != nil {
		t.Fatal(err)
	}
	if i.ConfigPath != "" || i.RootPath != boundary {
		t.Fatal("crossed Git boundary")
	}
}

func TestWorktreesNamesAndChangedIdentity(t *testing.T) {
	s, root := fixture(t)
	i, err := s.Init(root, "duplicate")
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(filepath.Dir(root), "second")
	os.MkdirAll(other, 0700)
	j, err := s.Init(other, "duplicate")
	if err != nil {
		t.Fatal(err)
	}
	if i.WorkspaceID == j.WorkspaceID {
		t.Fatal("directory identities merged")
	}
	_, err = s.Info(s.Locations.Home, "duplicate")
	code(t, err, "ambiguous_workspace")
	_, err = s.Info(s.Locations.Home, "dupl")
	code(t, err, "ambiguous_workspace")
	_, err = s.Info(s.Locations.Home, "./duplicate")
	code(t, err, "workspace_not_found")
	_, err = s.Info(root, "workspace.toml")
	code(t, err, "invalid_target")
	// Two independently located, unregistered worktrees carrying a committed ID.
	third := filepath.Join(filepath.Dir(root), "worktree")
	os.MkdirAll(third, 0700)
	put(t, filepath.Join(third, "workspace.toml"), string(read(t, i.ConfigPath)))
	k, err := s.Info(third, ".")
	if err != nil {
		t.Fatal(err)
	}
	if k.WorkspaceID != i.WorkspaceID || k.RootPath == i.RootPath || k.Registered {
		t.Fatal("worktree identity wrong")
	}
	// A changed explicit ID is never silently adopted.
	put(t, i.ConfigPath, strings.ReplaceAll(string(read(t, i.ConfigPath)), i.WorkspaceID, j.WorkspaceID))
	_, err = s.Info(root, ".")
	code(t, err, "identity_changed")
}

func TestInitReusesDirectoryID(t *testing.T) {
	s, root := fixture(t)
	id, _ := newID()
	checkoutID, _ := newID()
	os.MkdirAll(s.Locations.StateDir, 0700)
	r := registry{1, []Checkout{{CheckoutID: checkoutID, WorkspaceID: id, RootPath: root, Name: "prior", IdentitySource: "directory"}}}
	if err := s.store.save(r); err != nil {
		t.Fatal(err)
	}
	i, err := s.Init(root, "new name")
	if err != nil {
		t.Fatal(err)
	}
	if i.WorkspaceID != id || i.CheckoutID != checkoutID {
		t.Fatal("directory ID was lost during init")
	}
}

func TestMissingPathsRemainListed(t *testing.T) {
	s, root := fixture(t)
	i, err := s.Init(root, "moving")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(root, root+"-moved"); err != nil {
		t.Fatal(err)
	}
	items, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Available || items[0].WorkspaceID != i.WorkspaceID {
		t.Fatal("missing checkout was removed or reassigned")
	}
	_, err = s.Info(s.Locations.Home, "moving")
	code(t, err, "workspace_not_found")
}

func TestCorruptRegistryPreserved(t *testing.T) {
	s, root := fixture(t)
	p := filepath.Join(s.Locations.StateDir, "registry.json")
	for _, bad := range []string{"{broken", `{"schema_version":2,"checkouts":[]}`, `{"schema_version":1,"checkouts":null}`, `{"checkouts":[]}`, `{"schema_version":1,"checkouts":[],"surprise":true}`} {
		put(t, p, bad)
		_, err := s.Init(root, "x")
		code(t, err, "state_failure")
		if string(read(t, p)) != bad {
			t.Fatal("corrupt registry was overwritten")
		}
		if _, err := os.Stat(filepath.Join(root, "workspace.toml")); !os.IsNotExist(err) {
			t.Fatal("config created despite corruption")
		}
	}
}

func TestInterruptedInitRecovery(t *testing.T) {
	s, root := fixture(t)
	s.store.write = func(path string, data []byte, replace bool) error {
		if filepath.Base(path) == "registry.json" {
			return os.ErrPermission
		}
		return fileio.Write(path, data, replace)
	}
	_, err := s.Init(root, "recover")
	code(t, err, "state_failure")
	config := read(t, filepath.Join(root, "workspace.toml"))
	if _, err := os.Stat(filepath.Join(s.Locations.StateDir, "pending-init.json")); err != nil {
		t.Fatal("missing recovery journal")
	}
	fresh := New(s.Locations)
	i, err := fresh.Init(root, "ignored-on-recovery")
	if err != nil {
		t.Fatal(err)
	}
	if !i.Registered || i.Name != "recover" || !bytes.Equal(config, read(t, i.ConfigPath)) {
		t.Fatal("recovery changed original intent")
	}
	if _, err := os.Stat(filepath.Join(s.Locations.StateDir, "pending-init.json")); !os.IsNotExist(err) {
		t.Fatal("journal not completed")
	}
}

func TestRecoveryDoesNotOverwriteChangedConfig(t *testing.T) {
	s, root := fixture(t)
	s.store.write = func(path string, data []byte, replace bool) error {
		if filepath.Base(path) == "registry.json" {
			return os.ErrPermission
		}
		return fileio.Write(path, data, replace)
	}
	s.Init(root, "prepared")
	path := filepath.Join(root, "workspace.toml")
	put(t, path, "user changes")
	_, err := New(s.Locations).Init(root, "other")
	code(t, err, "state_failure")
	if string(read(t, path)) != "user changes" {
		t.Fatal("overwrote user data during recovery")
	}
}

func TestConcurrentRegistrationsAndBackup(t *testing.T) {
	s, root := fixture(t)
	const n = 8
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(filepath.Dir(root), string(rune('a'+i)))
		os.MkdirAll(p, 0700)
		wg.Add(1)
		go func() { defer wg.Done(); _, err := New(s.Locations).Init(p, "same"); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	items, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != n {
		t.Fatalf("lost registrations: %d", len(items))
	}
	var backup registry
	if err := json.Unmarshal(read(t, filepath.Join(s.Locations.StateDir, "registry.json.bak")), &backup); err != nil {
		t.Fatal(err)
	}
	if len(backup.Checkouts) != n-1 {
		t.Fatal("backup is not previous complete state")
	}
}

func TestSymlinkAndCaseAliases(t *testing.T) {
	s, root := fixture(t)
	i, err := s.Init(root, "alias")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("case", func(t *testing.T) {
		upper := strings.ToUpper(root)
		if !sameDirectory(root, upper) {
			t.Skip("filesystem is case-sensitive")
		}
		j, err := s.Info(upper, ".")
		if err != nil {
			t.Fatal(err)
		}
		if j.CheckoutID != i.CheckoutID {
			t.Fatal("case alias created another identity")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		link := root + "-link"
		if err := os.Symlink(root, link); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		j, err := s.Info(link, ".")
		if err != nil {
			t.Fatal(err)
		}
		if j.CheckoutID != i.CheckoutID {
			t.Fatal("link created another identity")
		}
		other := root + "-other"
		os.Mkdir(other, 0700)
		if err := os.Symlink(i.ConfigPath, filepath.Join(other, "workspace.toml")); err != nil {
			t.Fatal(err)
		}
		_, err = s.Info(other, ".")
		code(t, err, "invalid_configuration")
	})
}
