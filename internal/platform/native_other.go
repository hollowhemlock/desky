//go:build !windows && !linux && !darwin

package platform

import "github.com/hollowhemlock/desky/internal/workspace"

func (Native) Resolve(_ string, p workspace.Profile, _ string) (workspace.Profile, bool, error) {
	return p, false, workspace.Failure(7, "unsupported_action", "native launching requires Windows, macOS or Linux desktop")
}
func (Native) Dispatch(Action) error {
	return workspace.Failure(7, "unsupported_action", "native launching requires Windows, macOS or Linux desktop")
}
