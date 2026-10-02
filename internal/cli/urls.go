package cli

import (
	"github.com/hollowhemlock/desky/internal/resources"
	"github.com/hollowhemlock/desky/internal/workspace"
)

func resourceLabel(r resources.Resource) string {
	if r.Title == r.URL || r.Title == "" {
		return "(untitled URL)"
	}
	return safe(r.Title)
}

func urlCommand(s *resources.Service, cwd string, p options) (any, error) {
	usage := func() (any, error) {
		return nil, workspace.Failure(2, "usage", "invalid URL command arguments; use desk --help")
	}
	if len(p.words) < 2 {
		return usage()
	}
	id := p.entry.WorkspaceID
	switch p.words[1] {
	case "add":
		if len(p.words) != 3 || p.all || p.resolve {
			return usage()
		}
		title := ""
		if p.title != nil {
			title = *p.title
		}
		return s.SaveURL(cwd, id, p.words[2], title, p.pin)
	case "list":
		if len(p.words) != 2 || p.title != nil || p.pin || p.resolve {
			return usage()
		}
		return s.ListResources(cwd, id, p.all)
	case "pin", "unpin", "archive", "restore":
		if len(p.words) != 3 || p.title != nil || p.pin || p.all {
			return usage()
		}
		status := map[string]string{"pin": "pinned", "unpin": "saved", "archive": "archived", "restore": "saved"}[p.words[1]]
		return s.SetURLStatus(cwd, id, p.words[2], status, p.resolve)
	}
	return usage()
}
