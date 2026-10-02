package cli

import (
	"golang.org/x/sys/unix"
	"os"
)

func consoleInput() bool {
	_, in := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TIOCGETA)
	_, out := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TIOCGETA)
	return in == nil && out == nil
}
