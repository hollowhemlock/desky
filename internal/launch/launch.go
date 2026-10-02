// Package launch owns entry planning, consent and dispatch results.
package launch

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/hollowhemlock/desky/internal/fileio"
	"github.com/hollowhemlock/desky/internal/platform"
	"github.com/hollowhemlock/desky/internal/workspace"
)

type Request struct {
	Selector, WorkspaceID, Checkout, Trust string
	DryRun, AcceptIdentityChange           bool
}
type Item struct {
	ID string `json:"id"`
	platform.Action
}
type Plan struct {
	Workspace workspace.Info `json:"workspace"`
	Items     []Item         `json:"items"`
	Digest    string         `json:"digest"`
	Trusted   bool           `json:"trusted"`
}
type ItemResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
type Result struct {
	Plan         Plan         `json:"plan"`
	Items        []ItemResult `json:"items"`
	StateWarning string       `json:"state_warning,omitempty"`
}
type Service struct {
	Workspace *workspace.Service
	Platform  platform.Adapter
}

func New(w *workspace.Service) *Service { return &Service{w, platform.Native{}} }

// InspectTrust leaves metadata available when launch preflight cannot complete.
func (s *Service) InspectTrust(i workspace.Info) workspace.Info {
	p, err := s.build(i)
	if err != nil {
		i.Trust = "not_evaluated: " + err.Error()
		return i
	}
	i.Trust = p.Workspace.Trust
	return i
}

func (s *Service) Prepare(cwd string, r Request) (Plan, error) {
	i, err := s.Workspace.ResolveEntry(cwd, r.Selector, r.WorkspaceID, r.Checkout, r.AcceptIdentityChange)
	if err != nil {
		return Plan{}, err
	}
	return s.build(i)
}

func (s *Service) build(i workspace.Info) (Plan, error) {
	p := Plan{Workspace: i, Items: []Item{}}
	d, err := workspace.LoadDevice(s.Workspace.Locations)
	if err != nil {
		return p, err
	}
	if filepath.Clean(d.PersonalDataDir) != filepath.Clean(s.Workspace.Locations.PersonalDir) {
		if st, e := os.Stat(d.PersonalDataDir); e != nil || !st.IsDir() {
			return p, workspace.Failure(9, "personal_data_unavailable", "configured personal directory must be available before entry")
		}
	}
	// Personal resources are not implemented yet. Never silently ignore an existing
	// personal workspace store created by another version of the application.
	if i.WorkspaceID != "" {
		if _, e := os.Stat(filepath.Join(d.PersonalDataDir, "workspaces", i.WorkspaceID)); e == nil {
			return p, workspace.Failure(9, "unsupported_personal_data", "personal workspace resources require increment 3")
		} else if !os.IsNotExist(e) {
			return p, e
		}
	}
	seenURLs := map[string]bool{}
	for _, r := range i.Resources {
		a := platform.Action{Type: r.Type, Directory: i.RootPath}
		var profile workspace.Profile
		switch r.Type {
		case "url":
			a.Target = *r.URL
			a.Directory = s.Workspace.Locations.Home
			u, _ := url.Parse(a.Target)
			u.Scheme = strings.ToLower(u.Scheme)
			u.Host = strings.ToLower(u.Host)
			if (u.Scheme == "https" && u.Port() == "443") || (u.Scheme == "http" && u.Port() == "80") {
				u.Host = u.Hostname()
				if strings.Contains(u.Host, ":") {
					u.Host = "[" + u.Host + "]"
				}
			}
			if seenURLs[u.String()] {
				continue
			}
			seenURLs[u.String()] = true
		case "editor", "terminal":
			a.Target, err = filepath.EvalSymlinks(filepath.Join(i.RootPath, *r.Path))
			if err != nil {
				return p, err
			}
			a.Directory = a.Target
			if st, e := os.Stat(a.Target); e == nil && !st.IsDir() {
				a.Directory = filepath.Dir(a.Target)
			}
			profile = d.Launchers[r.Type]
		case "app":
			var ok bool
			profile, ok = d.Apps[*r.Profile]
			if !ok {
				return p, workspace.Failure(7, "missing_launcher", "configure app profile "+*r.Profile)
			}
			a.Target = *r.Profile
		}
		profile, a.Wait, err = s.Platform.Resolve(r.Type, profile, i.RootPath)
		if err != nil {
			return p, err
		}
		a.Executable = profile.Executable
		a.Args = append([]string(nil), profile.Args...)
		for j, arg := range a.Args {
			if arg == "{path}" {
				a.Args[j] = a.Target
			}
		}
		p.Items = append(p.Items, Item{r.ID, a})
	}
	if len(p.Items) == 0 {
		return p, workspace.Failure(5, "nothing_to_open", "workspace has no entry resources")
	}
	// The trust record key binds the checkout UUID; the digest remains reproducible
	// across a read-only dry run and first registration of a directory workspace.
	id := ""
	if i.IdentitySource == "explicit" {
		id = i.WorkspaceID
	}
	data, _ := json.Marshal(struct {
		Root, WorkspaceID string
		Items             []Item
	}{i.RootPath, id, p.Items})
	h := sha256.Sum256(data)
	p.Digest = hex.EncodeToString(h[:])
	t, err := readTrust(s.Workspace.Locations.StateDir)
	if err != nil {
		return p, err
	}
	for _, r := range t.Records {
		if r.CheckoutID == i.CheckoutID && i.PreviousWorkspaceID == "" && r.Digest == p.Digest {
			p.Trusted = true
		}
	}
	p.Workspace.Trust = "approval_required"
	if p.Trusted {
		p.Workspace.Trust = "approved"
	}
	return p, nil
}

// Open re-preflights after consent and persists approval before any dispatch.
func (s *Service) Open(cwd string, r Request, approve func(Plan) bool) (Result, error) {
	p, err := s.Prepare(cwd, r)
	result := Result{Plan: p, Items: []ItemResult{}}
	if err != nil {
		return result, err
	}
	if r.DryRun {
		return result, nil
	}
	if r.Trust != "" && r.Trust != p.Digest {
		return result, workspace.Failure(6, "trust_mismatch", "approval digest does not match; inspect a new dry run")
	}
	if !p.Trusted && r.Trust == "" {
		if approve == nil {
			return result, workspace.Failure(6, "trust_required", "review open --dry-run, then pass its digest with --trust")
		}
		if !approve(p) {
			return result, workspace.Failure(130, "cancelled", "entry cancelled; nothing launched")
		}
	}
	var launchErr error
	stateErr := s.Workspace.VisitEntry(p.Workspace, func(actual workspace.Info) bool {
		fresh, e := s.build(actual)
		if e != nil {
			launchErr = e
			return false
		}
		if fresh.Digest != p.Digest {
			launchErr = workspace.Failure(6, "plan_changed", "launch recipe changed during approval; review a new dry run")
			return false
		}
		if e := saveTrust(s.Workspace.Locations.StateDir, actual.CheckoutID, p.Digest); e != nil {
			launchErr = e
			return false
		}
		result.Plan = fresh
		result.Plan.Trusted = true
		result.Plan.Workspace.Trust = "approved"
		anySuccess := false
		for _, item := range fresh.Items {
			entry := ItemResult{ID: item.ID, Status: "dispatched"}
			if e := s.Platform.Dispatch(item.Action); e != nil {
				entry.Status = "failed"
				entry.Error = e.Error()
				launchErr = workspace.Failure(8, "launch_failure", "one or more resources failed; inspect per-resource results before retrying")
			} else {
				anySuccess = true
			}
			result.Items = append(result.Items, entry)
		}
		return anySuccess
	})
	if stateErr != nil {
		result.StateWarning = stateErr.Error()
		if len(result.Items) > 0 {
			result.StateWarning = "applications were dispatched but entry state could not be saved; inspect results before retrying: " + stateErr.Error()
		}
		if launchErr == nil {
			launchErr = stateErr
		}
	}
	return result, launchErr
}

type trustRecord struct {
	CheckoutID string    `json:"checkout_id"`
	Digest     string    `json:"digest"`
	ApprovedAt time.Time `json:"approved_at"`
}
type trustFile struct {
	SchemaVersion int           `json:"schema_version"`
	Records       []trustRecord `json:"records"`
}

var trustID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func readTrust(dir string) (trustFile, error) {
	t := trustFile{1, []trustRecord{}}
	b, err := fileio.ReadRegular(filepath.Join(dir, "trust.json"))
	if os.IsNotExist(err) {
		return t, nil
	}
	if err != nil {
		return t, workspace.Failure(9, "trust_state", err.Error())
	}
	t = trustFile{}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&t); err != nil || decoder.Decode(new(any)) != io.EOF || t.SchemaVersion != 1 || t.Records == nil {
		return t, workspace.Failure(9, "trust_state", "invalid trust.json; preserve it and restore a known-good backup")
	}
	seen := map[string]bool{}
	for _, r := range t.Records {
		_, digestErr := hex.DecodeString(r.Digest)
		if !trustID.MatchString(r.CheckoutID) || seen[r.CheckoutID] || digestErr != nil || len(r.Digest) != 64 || r.ApprovedAt.IsZero() {
			return t, workspace.Failure(9, "trust_state", "invalid trust record")
		}
		seen[r.CheckoutID] = true
	}
	return t, nil
}
func saveTrust(dir, id, digest string) error {
	t, err := readTrust(dir)
	if err != nil {
		return err
	}
	for j, r := range t.Records {
		if r.CheckoutID == id {
			if r.Digest == digest {
				return nil
			}
			t.Records = append(t.Records[:j], t.Records[j+1:]...)
			break
		}
	}
	t.Records = append(t.Records, trustRecord{id, digest, time.Now().UTC()})
	b, _ := json.MarshalIndent(t, "", "  ")
	path := filepath.Join(dir, "trust.json")
	if old, e := fileio.ReadRegular(path); e == nil {
		if e = fileio.Write(path+".bak", old, true); e != nil {
			return e
		}
	}
	if err = fileio.Write(path, append(b, '\n'), true); err != nil {
		return workspace.Failure(9, "trust_state", fmt.Sprintf("cannot persist approval: %v", err))
	}
	return nil
}
