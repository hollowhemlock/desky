// Package cli owns argument parsing, consent prompts and human/JSON presentation.
package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/hollowhemlock/desky/internal/launch"
	"github.com/hollowhemlock/desky/internal/resources"
	"github.com/hollowhemlock/desky/internal/workspace"
)

const help = `Desky Windows workspace launcher

Usage:
  desk
  desk <selector> [--dry-run] [--trust <digest>] [--json]
  desk open <selector> [--accept-identity-change] [--dry-run] [--trust <digest>] [--json]
  desk open --workspace <uuid> [--checkout <path>] [--dry-run] [--trust <digest>] [--json]
  desk init [--name <name>] [--json]
  desk info [<selector>] [--json]
  desk list [--json]
  desk url add <url> [--title <text>] [--pin] [--workspace <uuid>] [--json]
  desk url list [--all] [--workspace <uuid>] [--json]
  desk url pin|unpin|archive|restore <id> [--resolve] [--workspace <uuid>] [--json]
  desk config path [--json]
  desk config get <personal_data_dir|launchers|apps> [--json]
  desk --help
  desk --version

Selectors: directory, exact workspace UUID or exact name. Use -- before a
selector beginning with '-'. Interactive ambiguity uses a numbered, filterable picker.
Windows entry opens shared resources and healthy personal pins; conflicting URLs are skipped.
`

type envelope struct {
	SchemaVersion int              `json:"schema_version"`
	OK            bool             `json:"ok"`
	Data          any              `json:"data,omitempty"`
	Error         *workspace.Error `json:"error,omitempty"`
}
type options struct {
	entry               launch.Request
	json, help, version bool
	name                *string
	words               []string
	title               *string
	pin, all, resolve   bool
}

func parse(args []string) (options, error) {
	var p options
	seen := map[string]bool{}
	literal := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if literal {
			p.words = append(p.words, a)
			continue
		}
		if a == "--" {
			literal = true
			continue
		}
		if !strings.HasPrefix(a, "-") || a == "-" {
			p.words = append(p.words, a)
			continue
		}
		key, value, eq := strings.Cut(a, "=")
		if seen[key] {
			return p, workspace.Failure(2, "usage", "duplicate option "+key)
		}
		seen[key] = true
		switch key {
		case "--json", "--help", "--version", "--dry-run", "--accept-identity-change", "--pin", "--all", "--resolve":
			if eq {
				return p, workspace.Failure(2, "usage", key+" does not take a value")
			}
			switch key {
			case "--pin":
				p.pin = true
			case "--all":
				p.all = true
			case "--resolve":
				p.resolve = true
			case "--dry-run":
				p.entry.DryRun = true
			case "--accept-identity-change":
				p.entry.AcceptIdentityChange = true
			case "--json":
				p.json = true
			case "--help":
				p.help = true
			case "--version":
				p.version = true
			}
		case "--name", "--trust", "--workspace", "--checkout", "--title":
			if !eq {
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "--") {
					return p, workspace.Failure(2, "usage", key+" requires a value")
				}
				value = args[i]
			}
			if strings.TrimSpace(value) == "" {
				return p, workspace.Failure(2, "usage", key+" must not be empty")
			}
			switch key {
			case "--title":
				p.title = &value
			case "--name":
				p.name = &value
			case "--trust":
				p.entry.Trust = value
			case "--workspace":
				p.entry.WorkspaceID = value
			case "--checkout":
				p.entry.Checkout = value
			}
		default:
			return p, workspace.Failure(2, "usage", "unknown option "+key)
		}
	}
	return p, nil
}

func wantsJSON(args []string) bool {
	for _, a := range args {
		if a == "--" {
			break
		}
		if a == "--json" {
			return true
		}
	}
	return false
}

// Run resolves OS locations only for operations that use workspace state.
func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr, os.Getwd, workspace.DefaultLocations)
}

func run(args []string, stdout, stderr io.Writer, cwd func() (string, error), locations func() (workspace.Locations, error)) int {
	return runInteractive(args, stdout, stderr, cwd, locations, os.Stdin, consoleInput())
}

func runInteractive(args []string, stdout, stderr io.Writer, cwd func() (string, error), locations func() (workspace.Locations, error), input io.Reader, interactive bool) int {
	p, err := parse(args)
	jsonMode := wantsJSON(args)
	if err != nil {
		return render(nil, err, jsonMode, stdout, stderr)
	}
	if p.help {
		return render(help, nil, jsonMode, stdout, stderr)
	}
	if p.version {
		return render("desk 0.1.0-dev (Windows launcher)", nil, jsonMode, stdout, stderr)
	}
	bare := len(p.words) == 0
	if bare {
		p.words = []string{"pick"}
	}
	command := p.words[0]
	if command == "run" || command == "recent" || command == "find" || command == "resource" || command == "browser" || command == "session" || command == "-" {
		return render(nil, workspace.Failure(2, "not_implemented", "this command is not implemented; use --help"), jsonMode, stdout, stderr)
	}
	if command != "init" && command != "info" && command != "list" && command != "config" && command != "open" && command != "url" && !bare {
		p.words = append([]string{"open"}, p.words...)
		command = "open"
	}
	entryOptions := p.entry
	if command == "url" {
		entryOptions.WorkspaceID = ""
	}
	if command != "open" && command != "pick" && (entryOptions != launch.Request{}) {
		return render(nil, workspace.Failure(2, "usage", "entry options apply only to open"), jsonMode, stdout, stderr)
	}
	if p.name != nil && command != "init" {
		return render(nil, workspace.Failure(2, "usage", "--name is only valid for init"), jsonMode, stdout, stderr)
	}
	if command != "url" && (p.title != nil || p.pin || p.all || p.resolve) {
		return render(nil, workspace.Failure(2, "usage", "URL options apply only to url commands"), jsonMode, stdout, stderr)
	}
	l, err := locations()
	if err != nil {
		return render(nil, err, jsonMode, stdout, stderr)
	}
	s := workspace.New(l)
	reader := bufio.NewReader(input)
	interactive = interactive && !jsonMode
	var data any
	usage := func() error { return workspace.Failure(2, "usage", "invalid arguments; use desk --help") }
	switch command {
	case "open", "pick":
		if len(p.words) > 2 {
			err = usage()
			break
		}
		if len(p.words) == 2 {
			p.entry.Selector = p.words[1]
		}
		var dir string
		dir, err = cwd()
		if err != nil {
			break
		}
		if command == "pick" {
			if p.entry.WorkspaceID != "" || p.entry.Checkout != "" || p.entry.AcceptIdentityChange {
				err = usage()
				break
			}
			p.entry.Selector, err = pick(s, reader, stderr, interactive, nil)
		} else {
			_, err = s.ResolveEntry(dir, p.entry.Selector, p.entry.WorkspaceID, p.entry.Checkout, p.entry.AcceptIdentityChange)
			var e *workspace.Error
			if interactive && errors.As(err, &e) && e.Exit == 4 {
				p.entry.Selector, err = pick(s, reader, stderr, true, e.Details)
				p.entry.WorkspaceID, p.entry.Checkout = "", ""
			}
		}
		if err != nil {
			break
		}
		var consent func(launch.Plan) bool
		if interactive {
			consent = func(plan launch.Plan) bool {
				b, _ := json.MarshalIndent(plan, "", "  ")
				fmt.Fprintln(stderr, safeJSON(b))
				fmt.Fprint(stderr, "Approve this checkout and launch recipe? [y/N] ")
				line, e := reader.ReadString('\n')
				return e == nil && strings.EqualFold(strings.TrimSpace(line), "y")
			}
		}
		var result launch.Result
		result, err = launch.New(s).Open(dir, p.entry, consent)
		data = result
		if err != nil {
			var e *workspace.Error
			if errors.As(err, &e) && result.Plan.Workspace.RootPath != "" {
				e.Details = result
			}
		}
	case "url":
		var dir string
		dir, err = cwd()
		if err == nil {
			data, err = urlCommand(resources.New(s), dir, p)
		}
	case "init", "info":
		if (command == "init" && len(p.words) != 1) || (command == "info" && len(p.words) > 2) {
			err = usage()
			break
		}
		var dir string
		dir, err = cwd()
		if err != nil {
			break
		}
		if command == "init" {
			name := ""
			if p.name != nil {
				name = *p.name
			}
			var info workspace.Info
			info, err = s.Init(dir, name)
			if err == nil {
				info = launch.New(s).InspectTrust(info)
			}
			data = info
		} else {
			selector := ""
			if len(p.words) == 2 {
				selector = p.words[1]
			}
			var info workspace.Info
			info, err = s.Info(dir, selector)
			var e *workspace.Error
			if interactive && errors.As(err, &e) && e.Exit == 4 {
				selector, err = pick(s, reader, stderr, true, e.Details)
				if err == nil {
					info, err = s.Info(dir, selector)
				}
			}
			if err == nil {
				info = launch.New(s).InspectTrust(info)
			}
			data = info
		}
	case "list":
		if len(p.words) != 1 {
			err = usage()
			break
		}
		data, err = s.List()
	case "config":
		if len(p.words) == 2 && p.words[1] == "path" {
			data = l.ConfigFile
		} else if len(p.words) == 3 && p.words[1] == "get" {
			data, err = s.Config(p.words[2])
		} else {
			err = usage()
		}
	}
	return render(data, err, jsonMode, stdout, stderr)
}

func render(data any, err error, jsonMode bool, stdout, stderr io.Writer) int {
	result := envelope{SchemaVersion: 1, OK: err == nil, Data: data}
	code := 0
	if err != nil {
		var e *workspace.Error
		if !errors.As(err, &e) {
			e = workspace.Failure(9, "io_failure", err.Error())
		}
		result.Data, result.Error, code = nil, e, e.Exit
	}
	if jsonMode {
		if err := json.NewEncoder(stdout).Encode(result); err != nil {
			fmt.Fprintln(stderr, "could not write JSON output")
			return 9
		}
		return code
	}
	if result.Error != nil {
		fmt.Fprintf(stderr, "%s: %s\n", safe(result.Error.Code), safe(result.Error.Message))
		if entry, ok := result.Error.Details.(launch.Result); ok && len(entry.Items) > 0 {
			for _, item := range entry.Items {
				fmt.Fprintf(stderr, "%s: %s %s\n", safe(item.ID), safe(item.Status), safe(item.Error))
			}
			if entry.StateWarning != "" {
				fmt.Fprintln(stderr, safe(entry.StateWarning))
			}
		} else if result.Error.Details != nil {
			b, _ := json.Marshal(result.Error.Details)
			fmt.Fprintln(stderr, safe(string(b)))
		}
		return code
	}
	var output string
	switch d := data.(type) {
	case launch.Result:
		var b strings.Builder
		if len(d.Items) == 0 {
			plan, _ := json.MarshalIndent(d.Plan, "", "  ")
			b.WriteString(safeJSON(plan) + "\n")
		} else {
			fmt.Fprintf(&b, "Workspace: %s\n", safe(d.Plan.Workspace.RootPath))
			for _, item := range d.Items {
				fmt.Fprintf(&b, "%s: %s\n", safe(item.ID), item.Status)
			}
		}
		output = b.String()
	case string:
		if d == help {
			output = d
		} else {
			output = safe(d) + "\n"
		}
	case workspace.Info:
		id := d.WorkspaceID
		if id == "" {
			id = "unregistered (directory identity)"
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Workspace: %s\nID: %s\nCheckout: %s\nPath: %s\nConfig: %s\nRegistered: %t\nPersonal data: %s\nDevice state: %s\nTrust: %s\nResources:\n", safe(d.Name), safe(id), safe(d.CheckoutID), safe(d.RootPath), safe(d.ConfigPath), d.Registered, safe(d.PersonalDataDir), safe(d.DeviceStateDir), safe(d.Trust))
		for _, r := range d.Resources {
			label := r.ID
			if r.Name != nil {
				label = *r.Name
			}
			fmt.Fprintf(&b, "  %s (%s, %s)\n", safe(label), r.Type, r.Origin)
		}
		for _, warning := range d.ResourceWarnings {
			fmt.Fprintf(&b, "Warning: %s\n", safe(warning))
		}
		output = b.String()
	case []resources.Resource:
		var b strings.Builder
		for _, item := range d {
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", safe(item.ID), safe(item.Origin), safe(item.Status), resourceLabel(item))
			if item.Problem != "" {
				fmt.Fprintf(&b, "  %s\n", safe(item.Problem))
			}
		}
		if len(d) == 0 {
			b.WriteString("No saved URLs. Use desk url add <url>.\n")
		}
		output = b.String()
	case resources.Resource:
		output = fmt.Sprintf("%s: %s (%s)\n", safe(d.ID), resourceLabel(d), safe(d.Status))
	case []workspace.ListedCheckout:
		var b strings.Builder
		for _, item := range d {
			status := "available"
			if !item.Available {
				status = "unavailable"
			}
			fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", safe(item.Name), safe(item.RootPath), item.WorkspaceID, status)
		}
		if len(d) == 0 {
			b.WriteString("No registered workspaces. Run desk init in a project directory.\n")
		}
		output = b.String()
	default:
		b, _ := json.MarshalIndent(data, "", "  ")
		output = safeJSON(b) + "\n"
	}
	if _, err := io.WriteString(stdout, output); err != nil {
		fmt.Fprintln(stderr, "could not write output")
		return 9
	}
	return code
}

// Preserve JSON layout while escaping invisible formatting controls in labels.
func safeJSON(b []byte) string {
	lines := strings.Split(string(b), "\n")
	for i := range lines {
		lines[i] = safe(lines[i])
	}
	return strings.Join(lines, "\n")
}

func safe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.In(r, unicode.Cf) {
			q := strconv.QuoteRune(r)
			b.WriteString(q[1 : len(q)-1])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
