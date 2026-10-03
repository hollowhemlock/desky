//go:build linux || darwin

package fileio

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPrivatePublicationAndLockPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "device")
	release, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	p := filepath.Join(dir, "state.json")
	if err := Write(p, []byte("private"), false); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, []byte("replacement"), true); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{dir, p, filepath.Join(dir, "registry.lock")} {
		st, err := os.Stat(path)
		if err != nil || st.Mode().Perm()&0077 != 0 {
			t.Fatalf("group/other access to %s: %v %v", path, st, err)
		}
	}
}
