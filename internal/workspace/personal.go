package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/hollowhemlock/desky/internal/fileio"
)

// PersonalDirectory validates the selected storage location, without creating it.
func (s *Service) PersonalDirectory(root string) (string, error) {
	d, err := LoadDevice(s.Locations)
	if err != nil {
		return "", err
	}
	if err = checkScopes(s.Locations, d.PersonalDataDir, root); err != nil {
		return "", err
	}
	if filepath.Clean(d.PersonalDataDir) != filepath.Clean(s.Locations.PersonalDir) {
		if st, e := os.Stat(d.PersonalDataDir); e != nil || !st.IsDir() {
			return "", Failure(9, "personal_data_unavailable", "configured personal directory must already be available; no fallback was used")
		}
	}
	return d.PersonalDataDir, nil
}

// ResourceTarget supports exact integration identities even without a local
// checkout. Current-directory operations still enforce changed-identity checks.
func (s *Service) ResourceTarget(cwd, id string) (Info, error) {
	if id == "" {
		return s.Info(cwd, ".")
	}
	id = strings.ToLower(id)
	if !ValidID(id) {
		return Info{}, Failure(2, "usage", "--workspace requires a UUID v4")
	}
	dir, err := s.PersonalDirectory("")
	if err != nil {
		return Info{}, err
	}
	r, err := s.store.read()
	if err != nil {
		return Info{}, err
	}
	known := false
	for _, c := range r.Checkouts {
		if err := checkScopes(s.Locations, dir, c.RootPath); err != nil {
			return Info{}, err
		}
		if c.WorkspaceID == id {
			known = true
		}
	}
	st, e := os.Lstat(filepath.Join(dir, "workspaces", id))
	if e != nil && !os.IsNotExist(e) {
		return Info{}, e
	}
	if e == nil && !st.IsDir() {
		return Info{}, Failure(9, "resource_state", "personal workspace must be a directory")
	}
	if !known && os.IsNotExist(e) {
		return Info{}, Failure(3, "workspace_not_found", "unknown workspace UUID")
	}
	return Info{Checkout: Checkout{WorkspaceID: id}, PersonalDataDir: dir, Resources: []Resource{}}, nil
}

// MutateResources serializes local resource writers with registration. Persist
// identity first so a failed personal write/retry cannot orphan directory data.
// Registration does not change entry count, recency or trust.
func (s *Service) MutateResources(cwd, id string, register bool, visit func(Info) error) error {
	unlock, err := fileio.Lock(s.store.dir, s.store.lockTimeout)
	if err != nil {
		return stateError(err)
	}
	defer unlock()
	if _, err := os.Lstat(filepath.Join(s.store.dir, "pending-init.json")); err == nil {
		return Failure(9, "pending_init", "finish pending init before changing saved URLs")
	} else if !os.IsNotExist(err) {
		return stateError(err)
	}
	i, err := s.ResourceTarget(cwd, id)
	if err != nil {
		return err
	}
	if _, err = s.PersonalDirectory(i.RootPath); err != nil {
		return err
	}
	if register && id == "" && !i.Registered {
		r, err := s.store.read()
		if err != nil {
			return err
		}
		if i.WorkspaceID == "" {
			i.WorkspaceID, err = newID()
			if err != nil {
				return err
			}
		}
		i.CheckoutID, err = newID()
		if err != nil {
			return err
		}
		r.Checkouts = append(r.Checkouts, i.Checkout)
		if err = s.store.save(r); err != nil {
			return err
		}
	}
	return visit(i)
}
