//go:build !windows

package platform

import "github.com/hollowhemlock/desky/internal/workspace"

func (Native) Resolve(_ string, p workspace.Profile, _ string) (workspace.Profile, bool, error) {
	return p, false, workspace.Failure(7, "unsupported_action", "native launching is implemented on Windows only; other OS adapters are increment 4")
}
func (Native) Dispatch(Action) error {
	return workspace.Failure(7, "unsupported_action", "native launching is implemented on Windows only")
}
