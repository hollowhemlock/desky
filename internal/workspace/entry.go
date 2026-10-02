package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hollowhemlock/desky/internal/fileio"
)

// ResolveEntry adds exact, non-fuzzy integration selectors to ordinary resolution.
func (s *Service) ResolveEntry(cwd, selector, id, checkout string, acceptChange bool) (Info, error) {
	if id == "" {
		if checkout != "" {
			return Info{}, Failure(2, "usage", "--checkout requires --workspace")
		}
		if selector == "" {
			return Info{}, Failure(2, "usage", "open requires a selector or --workspace")
		}
		if acceptChange {
			p := expandHome(selector, s.Locations.Home)
			if !filepath.IsAbs(p) {
				p = filepath.Join(cwd, p)
			}
			d, err := LoadDevice(s.Locations)
			if err != nil {
				return Info{}, err
			}
			r, err := s.store.read()
			if err != nil {
				return Info{}, err
			}
			return s.inspectPathMode(p, r, d, true)
		}
		return s.Info(cwd, selector)
	}
	if !uuidPattern.MatchString(id) || selector != "" || acceptChange {
		return Info{}, Failure(2, "usage", "--workspace requires an exact UUID and cannot combine with a selector or identity rebinding")
	}
	id = strings.ToLower(id)
	if checkout != "" {
		path := expandHome(checkout, s.Locations.Home)
		if !filepath.IsAbs(path) {
			path = filepath.Join(cwd, path)
		}
		i, err := s.Info(cwd, path)
		if err != nil {
			return Info{}, err
		}
		if i.WorkspaceID != id {
			return Info{}, Failure(3, "workspace_not_found", "checkout does not belong to the requested workspace")
		}
		return i, nil
	}
	r, err := s.store.read()
	if err != nil {
		return Info{}, err
	}
	var matches []Checkout
	for _, c := range r.Checkouts {
		if c.WorkspaceID == id {
			if st, e := os.Stat(c.RootPath); e == nil && st.IsDir() {
				matches = append(matches, c)
			}
		}
	}
	if len(matches) == 0 {
		return Info{}, Failure(3, "workspace_not_found", "workspace has no available local checkout")
	}
	if len(matches) > 1 {
		root, _, e := discover(cwd)
		if e == nil {
			for _, c := range matches {
				if sameDirectory(root, c.RootPath) {
					return s.Info(cwd, c.RootPath)
				}
			}
		}
		failure := Failure(4, "ambiguous_checkout", "workspace has multiple checkouts; use --checkout")
		failure.Details = matches
		return Info{}, failure
	}
	return s.Info(cwd, matches[0].RootPath)
}

// VisitEntry serializes checkout identity, consent persistence and recency updates.
// The visitor reports whether any OS dispatch succeeded. Its runtime results are
// owned by the caller; this method reports only preflight/state failures.
func (s *Service) VisitEntry(expected Info, visit func(Info) bool) error {
	unlock, err := fileio.Lock(s.store.dir, s.store.lockTimeout)
	if err != nil {
		return stateError(err)
	}
	defer unlock()
	if _, err := os.Lstat(filepath.Join(s.store.dir, "pending-init.json")); err == nil {
		return Failure(9, "pending_init", "finish pending init before opening a workspace")
	} else if !os.IsNotExist(err) {
		return stateError(err)
	}
	r, err := s.store.read()
	if err != nil {
		return err
	}
	d, err := LoadDevice(s.Locations)
	if err != nil {
		return err
	}
	actual, err := s.inspectPathMode(expected.RootPath, r, d, expected.PreviousWorkspaceID != "")
	if err != nil {
		return err
	}
	// Compare identities/resources, but allow concurrent recency increments.
	a, b := actual, expected
	a.EntryCount, b.EntryCount = 0, 0
	a.LastEnteredAt, b.LastEnteredAt = nil, nil
	a.Trust, b.Trust = "", ""
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		return Failure(6, "plan_changed", "workspace changed during approval; inspect a fresh dry run")
	}
	if actual.CheckoutID == "" {
		actual.CheckoutID, err = newID()
		if err != nil {
			return stateError(err)
		}
	}
	if actual.WorkspaceID == "" {
		actual.WorkspaceID, err = newID()
		if err != nil {
			return stateError(err)
		}
	}
	if !visit(actual) {
		return nil
	}
	now := time.Now().UTC()
	actual.LastEnteredAt = &now
	actual.EntryCount++
	i := checkoutIndex(r, actual.RootPath)
	if i < 0 {
		r.Checkouts = append(r.Checkouts, actual.Checkout)
	} else {
		r.Checkouts[i] = actual.Checkout
	}
	return s.store.save(r)
}
