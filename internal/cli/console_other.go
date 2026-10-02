//go:build !windows

package cli

func consoleInput() bool { return false }
