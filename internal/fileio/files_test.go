package fileio

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestPublicationDoesNotLosePreviousFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state.json")
	if err := Write(p, []byte("old"), false); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, []byte("wrong"), false); err == nil {
		t.Fatal("overwrote existing file")
	}
	interrupted := errors.New("interrupted before publication")
	if err := write(p, []byte("new"), true, func() error { return interrupted }); !errors.Is(err, interrupted) {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "old" {
		t.Fatalf("old file lost: %q %v", b, err)
	}
	if err := Write(p, []byte("new"), true); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(p)
	if string(b) != "new" {
		t.Fatal("replacement failed")
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Fatal("temporary files leaked")
	}
}

func TestLockTimeoutAndRelease(t *testing.T) {
	dir := t.TempDir()
	release, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Lock(dir, 30*time.Millisecond)
	if !errors.Is(err, ErrLockTimeout) {
		release()
		t.Fatalf("expected timeout: %v", err)
	}
	release()
	release, err = Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestLockProcessExit(t *testing.T) {
	if dir := os.Getenv("DESKY_TEST_LOCK_DIR"); dir != "" {
		if _, err := Lock(dir, time.Second); err != nil {
			os.Exit(2)
		}
		// No deferred unlock: exercise kernel cleanup when a process disappears.
		os.Exit(0)
	}
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestLockProcessExit$")
	cmd.Env = append(os.Environ(), "DESKY_TEST_LOCK_DIR="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
	release, err := Lock(dir, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	release()
}
