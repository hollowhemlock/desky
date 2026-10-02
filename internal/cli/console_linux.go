package cli

import (
	"golang.org/x/sys/unix"
	"os"
)

func consoleInput() bool {
	_, in := unix.IoctlGetTermios(int(os.Stdin.Fd()), unix.TCGETS)
	_, out := unix.IoctlGetTermios(int(os.Stdout.Fd()), unix.TCGETS)
	return in == nil && out == nil
}
