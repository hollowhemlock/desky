package workspace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/hollowhemlock/desky/internal/fileio"
)

type Service struct {
	Locations Locations
	store     store
}

func New(l Locations) *Service { return &Service{Locations: l, store: newStore(l.StateDir)} }

func canonicalDirectory(path string) (string, error) {
	p, err := filepath.Abs(path)
	if err == nil {
		p, err = filepath.EvalSymlinks(p)
	}
	if err != nil {
		return "", Failure(3, "workspace_not_found", fmt.Sprintf("directory unavailable: %s", path))
	}
	st, err := os.Stat(p)
	if err != nil || !st.IsDir() {
		return "", Failure(5, "invalid_target", fmt.Sprintf("not a directory: %s", path))
	}
	return p, nil
}

func discover(path string) (root, config string, err error) {
	start, err := canonicalDirectory(path)
	if err != nil {
		return "", "", err
	}
	for p := start; ; p = filepath.Dir(p) {
		cfg := filepath.Join(p, "workspace.toml")
		if st, e := os.Lstat(cfg); e == nil {
			if !st.Mode().IsRegular() {
				return "", "", configError(cfg, "file", "must be regular, not a link")
			}
			return p, cfg, nil
		} else if !os.IsNotExist(e) {
			return "", "", configError(cfg, "file", "cannot be read")
		}
		if st, e := os.Lstat(filepath.Join(p, ".git")); e == nil {
			if st.IsDir() || st.Mode().IsRegular() {
				return p, "", nil
			}
		} else if !os.IsNotExist(e) {
			return "", "", Failure(5, "invalid_target", "cannot inspect Git boundary")
		}
		if filepath.Dir(p) == p {
			return start, "", nil
		}
	}
}

func (s *Service) inspectPath(path string, r registry, device DeviceConfig) (Info, error) {
	return s.inspectPathMode(path, r, device, false)
}

func (s *Service) inspectPathMode(path string, r registry, device DeviceConfig, acceptChange bool) (Info, error) {
	root, config, err := discover(path)
	if err != nil {
		return Info{}, err
	}
	if err := checkScopes(s.Locations, device.PersonalDataDir, root); err != nil {
		return Info{}, err
	}
	info := Info{Checkout: Checkout{RootPath: root, Name: filepath.Base(root), IdentitySource: "directory"}, ConfigPath: config, PersonalDataDir: device.PersonalDataDir, DeviceStateDir: s.Locations.StateDir, Available: true, Trust: "not_evaluated"}
	p := "."
	info.Resources = []Resource{{ID: "editor", Type: "editor", Path: &p, Origin: "default"}}
	if config != "" {
		d, err := loadDefinition(config, root)
		if err != nil {
			return Info{}, err
		}
		if d.Workspace.ID != nil {
			info.WorkspaceID = *d.Workspace.ID
			info.IdentitySource = "explicit"
		}
		if d.Workspace.Name != nil {
			info.Name = *d.Workspace.Name
		}
		info.Resources = d.Resources
	}
	if i := checkoutIndex(r, root); i >= 0 {
		old := r.Checkouts[i]
		if (info.WorkspaceID != "" && info.WorkspaceID != old.WorkspaceID) || (old.IdentitySource == "explicit" && info.IdentitySource != "explicit") {
			if acceptChange && info.WorkspaceID != "" {
				info.PreviousWorkspaceID = old.WorkspaceID
				info.CheckoutID = old.CheckoutID
				info.Registered = true
				return info, nil
			}
			e := Failure(5, "identity_changed", "workspace identity changed; inspect an explicit path with --accept-identity-change or restore the original configuration")
			e.Details = map[string]string{"old_id": old.WorkspaceID, "new_id": info.WorkspaceID}
			return Info{}, e
		}
		info.CheckoutID, info.WorkspaceID = old.CheckoutID, old.WorkspaceID
		info.LastEnteredAt, info.EntryCount = old.LastEnteredAt, old.EntryCount
		info.Registered = true
	}
	// Empty IDs are intentional: read-only inspection must not invent a durable ID.
	return info, nil
}

func (s *Service) Info(cwd, selector string) (Info, error) {
	d, err := LoadDevice(s.Locations)
	if err != nil {
		return Info{}, err
	}
	r, err := s.store.read()
	if err != nil {
		return Info{}, err
	}
	if selector == "" {
		selector = "."
	}
	expanded := expandHome(selector, s.Locations.Home)
	p := expanded
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	if _, err := os.Stat(p); err == nil {
		return s.inspectPath(p, r, d)
	} else if !os.IsNotExist(err) {
		return Info{}, Failure(3, "workspace_not_found", "selected path cannot be inspected")
	}
	if pathSelector(expanded) {
		return Info{}, Failure(3, "workspace_not_found", "selected directory does not exist")
	}
	var matches []Checkout
	for _, c := range r.Checkouts {
		if strings.EqualFold(c.WorkspaceID, selector) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		for _, c := range r.Checkouts {
			if strings.EqualFold(c.Name, selector) {
				matches = append(matches, c)
			}
		}
	}
	if len(matches) == 0 {
		for _, c := range r.Checkouts {
			if fuzzy(selector, c.Name) || fuzzy(selector, c.RootPath) {
				matches = append(matches, c)
			}
		}
		if len(matches) > 0 {
			e := Failure(4, "ambiguous_workspace", "fuzzy matches require terminal selection; use an exact path, UUID or name in scripts")
			e.Details = matches
			return Info{}, e
		}
		return Info{}, Failure(3, "workspace_not_found", "no known workspace matches the selector")
	}
	if len(matches) > 1 {
		// Only collapse multiple checkouts of one logical workspace when the
		// invoking directory resolves to one of them. Duplicate names stay ambiguous.
		oneID := true
		for _, c := range matches {
			oneID = oneID && c.WorkspaceID == matches[0].WorkspaceID
		}
		if oneID {
			currentRoot, _, e := discover(cwd)
			if e == nil {
				for _, c := range matches {
					if sameDirectory(currentRoot, c.RootPath) {
						return s.inspectPath(c.RootPath, r, d)
					}
				}
			}
		}
		e := Failure(4, "ambiguous_workspace", "multiple workspaces or checkouts match; select an exact directory")
		e.Details = matches
		return Info{}, e
	}
	return s.inspectPath(matches[0].RootPath, r, d)
}

func pathSelector(s string) bool {
	return s == "." || s == ".." || strings.HasPrefix(s, "~") || filepath.IsAbs(s) || strings.ContainsAny(s, `/\`) || filepath.VolumeName(s) != ""
}
func fuzzy(query, text string) bool {
	want := []rune(strings.ToLower(query))
	if len(want) == 0 {
		return false
	}
	i := 0
	for _, ch := range strings.ToLower(text) {
		if ch == want[i] {
			i++
			if i == len(want) {
				return true
			}
		}
	}
	return false
}

func (s *Service) List() ([]ListedCheckout, error) {
	r, err := s.store.read()
	if err != nil {
		return nil, err
	}
	items := make([]ListedCheckout, 0, len(r.Checkouts))
	for _, c := range r.Checkouts {
		st, err := os.Stat(c.RootPath)
		items = append(items, ListedCheckout{c, err == nil && st.IsDir()})
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.LastEnteredAt == nil && b.LastEnteredAt != nil {
			return false
		}
		if a.LastEnteredAt != nil && b.LastEnteredAt == nil {
			return true
		}
		if a.LastEnteredAt != nil && !a.LastEnteredAt.Equal(*b.LastEnteredAt) {
			return a.LastEnteredAt.After(*b.LastEnteredAt)
		}
		if a.EntryCount != b.EntryCount {
			return a.EntryCount > b.EntryCount
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].RootPath < items[j].RootPath
	})
	return items, nil
}

func initialConfig(id, name string) ([]byte, error) {
	p := "."
	d := Definition{SchemaVersion: 1, Resources: []Resource{{ID: "editor", Type: "editor", Path: &p}}}
	d.Workspace = &struct {
		ID   *string `toml:"id,omitempty"`
		Name *string `toml:"name,omitempty"`
	}{&id, &name}
	var b bytes.Buffer
	if err := toml.NewEncoder(&b).Encode(d); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (s *Service) Init(cwd, name string) (Info, error) {
	root, err := canonicalDirectory(cwd)
	if err != nil {
		return Info{}, err
	}
	if name == "" {
		name = filepath.Base(root)
	}
	if !validName(name) {
		return Info{}, Failure(5, "invalid_configuration", "workspace name must not be empty")
	}
	d, err := LoadDevice(s.Locations)
	if err != nil {
		return Info{}, err
	}
	if err := checkScopes(s.Locations, d.PersonalDataDir, root); err != nil {
		return Info{}, err
	}
	unlock, err := fileio.Lock(s.store.dir, s.store.lockTimeout)
	if err != nil {
		return Info{}, stateError(err)
	}
	defer unlock()
	r, err := s.store.read()
	if err != nil {
		return Info{}, err
	}
	recovered, err := s.store.recover(&r)
	if err != nil {
		return Info{}, err
	}
	if recovered != nil && sameDirectory(recovered.RootPath, root) {
		return s.inspectPath(root, r, d)
	}
	path := filepath.Join(root, "workspace.toml")
	if _, err := os.Lstat(path); err == nil {
		return Info{}, Failure(5, "configuration_exists", "workspace.toml already exists; it was not overwritten")
	} else if !os.IsNotExist(err) {
		return Info{}, configError(path, "file", "cannot be inspected")
	}
	c := Checkout{RootPath: root, Name: name, IdentitySource: "explicit"}
	if i := checkoutIndex(r, root); i >= 0 {
		c = r.Checkouts[i]
		if c.IdentitySource == "explicit" {
			return Info{}, Failure(5, "identity_changed", "registered explicit configuration is missing; restore workspace.toml")
		}
		c.Name, c.IdentitySource = name, "explicit"
	} else {
		c.CheckoutID, err = newID()
		if err != nil {
			return Info{}, stateError(err)
		}
		c.WorkspaceID, err = newID()
		if err != nil {
			return Info{}, stateError(err)
		}
	}
	config, err := initialConfig(c.WorkspaceID, c.Name)
	if err != nil {
		return Info{}, stateError(err)
	}
	journal, err := json.Marshal(pendingInit{1, c, config})
	if err != nil {
		return Info{}, stateError(err)
	}
	if err := s.store.write(filepath.Join(s.store.dir, "pending-init.json"), journal, false); err != nil {
		return Info{}, stateError(err)
	}
	if _, err := s.store.recover(&r); err != nil {
		return Info{}, err
	}
	return s.inspectPath(root, r, d)
}

func (s *Service) Config(key string) (any, error) {
	d, err := LoadDevice(s.Locations)
	if err != nil {
		return nil, err
	}
	if err := checkScopes(s.Locations, d.PersonalDataDir, ""); err != nil {
		return nil, err
	}
	switch key {
	case "personal_data_dir":
		return d.PersonalDataDir, nil
	case "launchers":
		return d.Launchers, nil
	case "apps":
		return d.Apps, nil
	default:
		return nil, Failure(2, "usage", fmt.Sprintf("unknown config key %q; use personal_data_dir, launchers or apps", key))
	}
}
