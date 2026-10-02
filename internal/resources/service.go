package resources

import (
	"strings"
	"time"

	"github.com/hollowhemlock/desky/internal/fileio"
	"github.com/hollowhemlock/desky/internal/workspace"
)

type Service struct {
	Workspace *workspace.Service
	write     func(string, []byte, bool) error
}

func New(w *workspace.Service) *Service { return &Service{w, fileio.Write} }

func (s *Service) ListWorkspaces() ([]workspace.ListedCheckout, error) { return s.Workspace.List() }

func (s *Service) ListResources(cwd, id string, all bool) ([]Resource, error) {
	i, err := s.Workspace.ResourceTarget(cwd, id)
	if err != nil {
		return nil, err
	}
	dir, err := s.Workspace.PersonalDirectory(i.RootPath)
	if err != nil {
		return nil, err
	}
	items, err := Load(dir, i.WorkspaceID)
	if err != nil {
		return nil, err
	}
	out := []Resource{}
	for _, r := range i.Resources {
		if r.Type == "url" {
			title := *r.URL
			if r.Name != nil {
				title = *r.Name
			}
			out = append(out, Resource{ID: r.ID, Origin: "shared", Type: "url", URL: *r.URL, Title: title, Status: "shared", Heads: []Revision{}})
		}
	}
	for _, r := range items {
		if all || r.Status != "archived" {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Service) SaveURL(cwd, id, raw, title string, pin bool) (Resource, error) {
	key, err := workspace.NormalizeURL(raw)
	if err != nil {
		return Resource{}, err
	}
	if title == "" {
		title = raw
	}
	if strings.TrimSpace(title) == "" {
		return Resource{}, workspace.Failure(5, "invalid_title", "title must not be empty")
	}
	var out Resource
	err = s.Workspace.MutateResources(cwd, id, true, func(i workspace.Info) error {
		items, err := Load(i.PersonalDataDir, i.WorkspaceID)
		if err != nil {
			return err
		}
		for _, r := range items {
			for _, head := range r.Heads {
				normal, _ := workspace.NormalizeURL(head.URL)
				if normal == key {
					out = r
					return nil
				}
			}
		}
		rid, err := workspace.NewID()
		if err != nil {
			return err
		}
		rev, err := workspace.NewID()
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		status := "saved"
		if pin {
			status = "pinned"
		}
		r := Revision{1, i.WorkspaceID, rid, rev, []string{}, "url", raw, title, status, now, now}
		if err := publish(i.PersonalDataDir, r, s.write); err != nil {
			return err
		}
		out = Resource{rid, "personal", "url", raw, title, status, []Revision{r}, ""}
		return nil
	})
	return out, err
}

func (s *Service) SetURLStatus(cwd, id, rid, status string, resolve bool) (Resource, error) {
	if !validStatus(status) {
		return Resource{}, workspace.Failure(2, "usage", "invalid URL status")
	}
	if !workspace.ValidID(rid) {
		return Resource{}, workspace.Failure(5, "read_only_resource", "personal commands require a resource UUID; edit shared URLs in workspace.toml")
	}
	var out Resource
	err := s.Workspace.MutateResources(cwd, id, false, func(i workspace.Info) error {
		items, err := Load(i.PersonalDataDir, i.WorkspaceID)
		if err != nil {
			return err
		}
		for _, item := range items {
			if item.ID != rid {
				continue
			}
			if len(item.Heads) == 0 || (item.Problem != "" && !resolve) {
				return failure("resource " + rid + ": " + item.Problem)
			}
			r := item.Heads[0]
			parents := []string{}
			for _, head := range item.Heads {
				if head.URL != r.URL || head.Title != r.Title {
					return failure("resource " + rid + ": URL/title differ; preserve and restore valid files from backup/provider history")
				}
				parents = append(parents, head.RevisionID)
			}
			if item.Problem == "" && r.Status == status {
				out = item
				return nil
			}
			r.RevisionID, err = workspace.NewID()
			if err != nil {
				return err
			}
			r.Parents, r.Status, r.UpdatedAt = parents, status, time.Now().UTC().Format(time.RFC3339Nano)
			if err := publish(i.PersonalDataDir, r, s.write); err != nil {
				return err
			}
			out = Resource{rid, "personal", "url", r.URL, r.Title, status, []Revision{r}, ""}
			return nil
		}
		return workspace.Failure(3, "resource_not_found", "no personal URL with that resource ID")
	})
	return out, err
}
