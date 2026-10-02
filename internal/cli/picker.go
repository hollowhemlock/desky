package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/hollowhemlock/desky/internal/workspace"
)

func matches(query, text string) bool {
	q := []rune(strings.ToLower(query))
	if len(q) == 0 {
		return true
	}
	j := 0
	for _, r := range strings.ToLower(text) {
		if r == q[j] {
			j++
			if j == len(q) {
				return true
			}
		}
	}
	return false
}

func pick(s *workspace.Service, input *bufio.Reader, output io.Writer, interactive bool, scope any) (string, error) {
	if !interactive {
		return "", workspace.Failure(2, "interaction_required", "recent selection requires a terminal; use desk list or an explicit desk open selector")
	}
	items, err := s.List()
	if err != nil {
		return "", err
	}
	allowed := map[string]bool{}
	if scoped, ok := scope.([]workspace.Checkout); ok {
		for _, c := range scoped {
			allowed[c.RootPath] = true
		}
	}
	base := []workspace.ListedCheckout{}
	for _, c := range items {
		if c.Available && (scope == nil || allowed[c.RootPath]) {
			base = append(base, c)
		}
	}
	if len(base) == 0 {
		return "", workspace.Failure(3, "workspace_not_found", "no available workspaces; run desk . or desk init in a project")
	}
	query := ""
	for {
		shown := []workspace.ListedCheckout{}
		fmt.Fprintln(output, "Recent workspaces:")
		lastName := ""
		for _, c := range base {
			if matches(query, c.Name) || matches(query, c.RootPath) {
				shown = append(shown, c)
				if c.Name != lastName {
					fmt.Fprintf(output, "  %s\n", safe(c.Name))
					lastName = c.Name
				}
				fmt.Fprintf(output, "    %d. %s\n", len(shown), safe(c.RootPath))
			}
		}
		if len(shown) == 0 {
			fmt.Fprintln(output, "  No matches. Type another query or / to clear.")
		}
		fmt.Fprint(output, "Number or query (Enter: first, /: clear, :q: cancel): ")
		line, err := input.ReadString('\n')
		if err != nil {
			return "", workspace.Failure(130, "cancelled", "selection cancelled")
		}
		line = strings.TrimSpace(line)
		if line == ":q" || line == "\x1b" {
			return "", workspace.Failure(130, "cancelled", "selection cancelled")
		}
		if line == "" && len(shown) > 0 {
			return shown[0].RootPath, nil
		}
		if n, e := strconv.Atoi(line); e == nil {
			if n > 0 && n <= len(shown) {
				return shown[n-1].RootPath, nil
			}
			fmt.Fprintln(output, "Choose a displayed number.")
			continue
		}
		if line == "/" {
			query = ""
		} else {
			query = line
		}
	}
}
