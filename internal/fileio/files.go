// Package fileio contains the publication and locking primitives used by device state.
package fileio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ReadRegular refuses links and special files rather than following a state/config link.
func ReadRegular(path string) ([]byte, error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	return os.ReadFile(path)
}

// Write publishes a fully flushed file. replace=false never overwrites a destination.
func Write(path string, data []byte, replace bool) error {
	return write(path, data, replace, nil)
}

func write(path string, data []byte, replace bool, beforePublish func() error) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".desky-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if beforePublish != nil {
		if err = beforePublish(); err != nil {
			return err
		}
	}
	if replace {
		err = replaceFile(f.Name(), path)
	} else {
		// Linking a completed file publishes it without an overwrite race.
		err = os.Link(f.Name(), path)
	}
	if err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}

var ErrLockTimeout = errors.New("device state lock timed out")

// Lock uses an OS lock released even if the owning process exits unexpectedly.
func Lock(dir string, timeout time.Duration) (func(), error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	p := filepath.Join(dir, "registry.lock")
	if st, err := os.Lstat(p); err == nil && !st.Mode().IsRegular() {
		return nil, fmt.Errorf("invalid lock file: %s", p)
	} else if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		busy, err := tryLock(f)
		if err != nil {
			f.Close()
			return nil, err
		}
		if !busy {
			return func() { unlock(f); f.Close() }, nil
		}
		if !time.Now().Before(deadline) {
			f.Close()
			return nil, ErrLockTimeout
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func RemovePublished(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
