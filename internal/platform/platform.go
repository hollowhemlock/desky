// Package platform owns native launch mechanics, not workspace or trust policy.
package platform

import "github.com/hollowhemlock/desky/internal/workspace"

type Action struct {
	Type       string   `json:"type"`
	Target     string   `json:"target"`
	Executable string   `json:"executable,omitempty"`
	Args       []string `json:"args,omitempty"`
	Directory  string   `json:"directory"`
	Wait       bool     `json:"wait_for_dispatch,omitempty"`
}

// Adapter is the OS seam used by launch planning and dispatch tests.
type Adapter interface {
	Resolve(kind string, profile workspace.Profile, root string) (workspace.Profile, bool, error)
	Dispatch(Action) error
}

type Native struct{}
