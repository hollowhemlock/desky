package platform

import (
	"strings"
	"testing"
)

func TestRequireLocalDesktop(t *testing.T) {
	keys := []string{"SSH_CONNECTION", "SSH_TTY", "WSL_DISTRO_NAME", "WSL_INTEROP"}
	for _, key := range keys {
		t.Setenv(key, "")
	}
	if err := requireDesktop(func() bool { return true }); err != nil {
		t.Fatal(err)
	}
	if err := requireDesktop(func() bool { return false }); err == nil || !strings.Contains(err.Error(), "local desktop") {
		t.Fatalf("headless preflight: %v", err)
	}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, "fixture")
			if err := requireDesktop(func() bool { return true }); err == nil {
				t.Fatal("accepted remote session")
			}
		})
	}
}
