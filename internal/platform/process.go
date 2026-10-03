package platform

import (
	"fmt"
	"os/exec"
	"time"
)

func dispatchProcess(cmd *exec.Cmd, wait bool, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap direct children while this process remains alive, including helpers
	// that finish after a timeout. Exiting the CLI never kills those children.
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	if !wait {
		return nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		return fmt.Errorf("dispatch helper timed out; outcome unknown, do not retry automatically")
	}
}
