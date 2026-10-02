// Package cli owns argument parsing and human/JSON presentation. It never launches apps.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/hollowhemlock/desky/internal/workspace"
)

const help = `Desky workspace metadata (increment 1)

Usage:
  desk init [--name <name>] [--json]
  desk info [<selector>] [--json]
  desk list [--json]
  desk config path [--json]
  desk config get <personal_data_dir|launchers|apps> [--json]
  desk --help
  desk --version

Selectors: directory, exact workspace UUID or exact name. Use -- before a
selector beginning with '-'. Ambiguous/fuzzy selections fail explicitly.
Opening applications, URL commands and the interactive picker are not implemented.
`

type envelope struct {
	SchemaVersion int              `json:"schema_version"`
	OK            bool             `json:"ok"`
	Data          any              `json:"data,omitempty"`
	Error         *workspace.Error `json:"error,omitempty"`
}
type options struct {
	json, help, version bool
	name                *string
	words               []string
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
		case "--json", "--help", "--version":
			if eq {
				return p, workspace.Failure(2, "usage", key+" does not take a value")
			}
			switch key {
			case "--json":
				p.json = true
			case "--help":
				p.help = true
			case "--version":
				p.version = true
			}
		case "--name":
			if !eq {
				i++
				if i >= len(args) || strings.HasPrefix(args[i], "--") {
					return p, workspace.Failure(2, "usage", "--name requires a value")
				}
				value = args[i]
			}
			if strings.TrimSpace(value) == "" {
				return p, workspace.Failure(2, "usage", "--name must not be empty")
			}
			p.name = &value
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
	p, err := parse(args)
	jsonMode := wantsJSON(args)
	if err != nil {
		return render(nil, err, jsonMode, stdout, stderr)
	}
	if p.help {
		return render(help, nil, jsonMode, stdout, stderr)
	}
	if p.version {
		return render("desk 0.1.0-dev (workspace metadata)", nil, jsonMode, stdout, stderr)
	}
	if len(p.words) == 0 {
		return render(nil, workspace.Failure(2, "not_implemented", "workspace picker is not implemented; use desk init, info or list"), jsonMode, stdout, stderr)
	}
	command := p.words[0]
	if command != "init" && command != "info" && command != "list" && command != "config" {
		return render(nil, workspace.Failure(2, "not_implemented", "opening workspaces, URL operations and other commands are not implemented; use --help"), jsonMode, stdout, stderr)
	}
	if p.name != nil && command != "init" {
		return render(nil, workspace.Failure(2, "usage", "--name is only valid for init"), jsonMode, stdout, stderr)
	}
	l, err := locations()
	if err != nil {
		return render(nil, err, jsonMode, stdout, stderr)
	}
	s := workspace.New(l)
	var data any
	usage := func() error { return workspace.Failure(2, "usage", "invalid arguments; use desk --help") }
	switch command {
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
			data, err = s.Init(dir, name)
		} else {
			selector := ""
			if len(p.words) == 2 {
				selector = p.words[1]
			}
			data, err = s.Info(dir, selector)
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
		if result.Error.Details != nil {
			b, _ := json.Marshal(result.Error.Details)
			fmt.Fprintln(stderr, safe(string(b)))
		}
		return code
	}
	var output string
	switch d := data.(type) {
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
		output = b.String()
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
		output = string(b) + "\n"
	}
	if _, err := io.WriteString(stdout, output); err != nil {
		fmt.Fprintln(stderr, "could not write output")
		return 9
	}
	return code
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
