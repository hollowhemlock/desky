//go:build !windows && !linux && !darwin

package cli

func consoleInput() bool { return false }
