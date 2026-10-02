package cli

import "golang.org/x/sys/windows"

func consoleInput() bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Stdin, &mode) == nil && windows.GetConsoleMode(windows.Stdout, &mode) == nil
}
