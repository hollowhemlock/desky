package platform

import (
	"os"

	"github.com/hollowhemlock/desky/internal/workspace"
)

func requireDesktop(available func() bool) error {
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" ||
		os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" || !available() {
		return workspace.Failure(7, "unsupported_action", "launching requires a local desktop session; SSH, WSL and headless entry are unsupported")
	}
	return nil
}
