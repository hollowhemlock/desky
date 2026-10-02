// Package resources owns intentional URL snapshots and deterministic revision
// interpretation. It neither fetches URLs nor repairs/deletes synchronized files.
package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/hollowhemlock/desky/internal/fileio"
	"github.com/hollowhemlock/desky/internal/workspace"
)

type Revision struct {
	SchemaVersion int      `json:"schema_version"`
	WorkspaceID   string   `json:"workspace_id"`
	ResourceID    string   `json:"resource_id"`
	RevisionID    string   `json:"revision_id"`
	Parents       []string `json:"parents"`
	Type          string   `json:"type"`
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Status        string   `json:"status"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type Resource struct {
	ID      string     `json:"id"`
	Origin  string     `json:"origin"`
	Type    string     `json:"type"`
	URL     string     `json:"url,omitempty"`
	Title   string     `json:"title,omitempty"`
	Status  string     `json:"status"`
	Heads   []Revision `json:"heads"`
	Problem string     `json:"problem,omitempty"`
}

func failure(message string) error { return workspace.Failure(9, "resource_state", message) }
func validStatus(s string) bool    { return s == "saved" || s == "pinned" || s == "archived" }

// directory refuses linked internal storage components. The configured root may
// itself be a user-selected link, already checked against workspace/device scopes.
func directory(path string) error {
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return failure("personal storage component is not a regular directory")
	}
	return nil
}

func resourceRoot(dir, id string) (string, error) {
	if !workspace.ValidID(id) {
		return "", failure("invalid personal workspace ID")
	}
	p := dir
	for _, part := range []string{"workspaces", id, "resources"} {
		p = filepath.Join(p, part)
		if err := directory(p); err != nil {
			return p, err
		}
	}
	return p, nil
}

// Load isolates problems to their resource directory. Global enumeration errors
// still fail explicitly because the set of affected resources cannot be known.
func Load(dir, id string) ([]Resource, error) {
	items := []Resource{}
	if id == "" {
		return items, nil
	}
	root, err := resourceRoot(dir, id)
	if os.IsNotExist(err) {
		return items, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !workspace.ValidID(entry.Name()) {
			return nil, failure("unrecognized entry in personal resources directory; preserve it for recovery")
		}
	}
	// Independent immutable resource directories can be read concurrently. Keep
	// fixed slots in ReadDir order so scheduling never affects plans or conflicts.
	items = make([]Resource, len(entries))
	jobs := make(chan int)
	var workers sync.WaitGroup
	for j := 0; j < min(16, len(entries)); j++ {
		workers.Go(func() {
			for index := range jobs {
				entry := entries[index]
				items[index] = readResource(filepath.Join(root, entry.Name()), id, entry.Name())
			}
		})
	}
	for index := range entries {
		jobs <- index
	}
	close(jobs)
	workers.Wait()
	return items, nil
}

func decode(data []byte, wid, rid string) (Revision, error) {
	var r Revision
	if !utf8.Valid(data) {
		return r, failure("invalid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil || d.Decode(new(any)) != io.EOF {
		return r, failure("malformed or unknown revision fields")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return r, failure("malformed revision")
	}
	for _, key := range []string{"schema_version", "workspace_id", "resource_id", "revision_id", "parents", "type", "url", "status", "created_at", "updated_at"} {
		if v, ok := fields[key]; !ok || string(v) == "null" {
			return r, failure("missing required revision field")
		}
	}
	if _, ok := fields["title"]; !ok {
		r.Title = r.URL
	}
	// Duplicate object keys are ambiguous input even when encoding/json would
	// otherwise choose the last value. Revision objects have no nested objects.
	keys := json.NewDecoder(bytes.NewReader(data))
	if _, err := keys.Token(); err != nil {
		return r, failure("malformed revision")
	}
	seenKeys := map[string]bool{}
	for keys.More() {
		token, err := keys.Token()
		key, ok := token.(string)
		if err != nil || !ok || seenKeys[key] {
			return r, failure("duplicate revision field")
		}
		switch key {
		case "schema_version", "workspace_id", "resource_id", "revision_id", "parents", "type", "url", "title", "status", "created_at", "updated_at":
		default:
			return r, failure("unknown revision field")
		}
		seenKeys[key] = true
		if err := keys.Decode(new(json.RawMessage)); err != nil {
			return r, failure("malformed revision")
		}
	}
	if r.SchemaVersion != 1 || r.WorkspaceID != wid || r.ResourceID != rid || !workspace.ValidID(r.RevisionID) || r.Type != "url" || !validStatus(r.Status) || r.Parents == nil {
		return r, failure("unsupported schema or invalid revision identity/status")
	}
	if _, err := workspace.NormalizeURL(r.URL); err != nil {
		return r, failure("invalid saved URL")
	}
	if strings.TrimSpace(r.Title) == "" {
		return r, failure("empty revision title")
	}
	for _, stamp := range []string{r.CreatedAt, r.UpdatedAt} {
		if _, err := time.Parse(time.RFC3339Nano, stamp); err != nil || !strings.HasSuffix(stamp, "Z") {
			return r, failure("revision times must be UTC RFC3339")
		}
	}
	seen := map[string]bool{}
	for _, parent := range r.Parents {
		if !workspace.ValidID(parent) || seen[parent] {
			return r, failure("invalid revision parents")
		}
		seen[parent] = true
	}
	sort.Strings(r.Parents)
	return r, nil
}

func readResource(path, wid, rid string) Resource {
	item := Resource{ID: rid, Origin: "personal", Type: "url", Status: "conflict", Heads: []Revision{}}
	bad := func(reason string) Resource { item.Problem = reason; return item }
	if err := directory(path); err != nil {
		return bad("resource directory unavailable or linked; preserve files")
	}
	files, err := os.ReadDir(path)
	if err != nil {
		return bad("cannot read resource directory; preserve files")
	}
	revisions := map[string]Revision{}
	for _, file := range files {
		if !strings.EqualFold(filepath.Ext(file.Name()), ".json") {
			continue
		}
		data, err := fileio.ReadRegular(filepath.Join(path, file.Name()))
		if err != nil {
			return bad("unreadable or nonregular revision file: " + file.Name())
		}
		r, err := decode(data, wid, rid)
		if err != nil {
			return bad(err.Error() + ": " + file.Name())
		}
		stem := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		if workspace.ValidID(stem) && stem != r.RevisionID {
			return bad("revision ID does not match filename: " + file.Name())
		}
		if prior, ok := revisions[r.RevisionID]; ok && !reflect.DeepEqual(prior, r) {
			return bad("differing content for revision " + r.RevisionID)
		}
		revisions[r.RevisionID] = r
	}
	if len(revisions) == 0 {
		return bad("incomplete delivery: no published revisions")
	}
	parents := map[string]bool{}
	color := map[string]int{}
	var visit func(string) bool
	visit = func(id string) bool {
		if color[id] == 1 {
			return false
		}
		if color[id] == 2 {
			return true
		}
		color[id] = 1
		for _, parent := range revisions[id].Parents {
			if _, exists := revisions[parent]; !exists {
				return false
			}
			parents[parent] = true
			if !visit(parent) {
				return false
			}
		}
		color[id] = 2
		return true
	}
	for id := range revisions {
		if !visit(id) {
			return bad("incomplete delivery or cyclic revision history")
		}
	}
	for id, r := range revisions {
		if !parents[id] {
			item.Heads = append(item.Heads, r)
		}
	}
	sort.Slice(item.Heads, func(i, j int) bool { return item.Heads[i].RevisionID < item.Heads[j].RevisionID })
	if len(item.Heads) != 1 {
		return bad("concurrent revisions; use a status command with --resolve when URL/title agree")
	}
	head := item.Heads[0]
	item.URL, item.Title, item.Status = head.URL, head.Title, head.Status
	return item
}

func publish(dir string, r Revision, write func(string, []byte, bool) error) error {
	// Validate every ancestor before creating descendants, never traversing a
	// provider's linked resource directory to write elsewhere.
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	p := dir
	for _, part := range []string{"workspaces", r.WorkspaceID, "resources", r.ResourceID} {
		p = filepath.Join(p, part)
		if err := os.Mkdir(p, 0700); err != nil && !os.IsExist(err) {
			return err
		}
		if err := directory(p); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = write(filepath.Join(p, r.RevisionID+".json"), append(b, '\n'), false); err != nil {
		return failure(fmt.Sprintf("could not publish revision for resource %s; preserve files and inspect before retrying: %v", r.ResourceID, err))
	}
	return nil
}
