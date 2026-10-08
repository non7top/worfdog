package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// restartTimeout bounds restart commands so a hung one cannot stall the watchdog.
const restartTimeout = 60 * time.Second

// executeCommand runs a shell command and kills it after timeout.
func executeCommand(cmd string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := exec.CommandContext(ctx, "sh", "-c", cmd).Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out after %v: %w", timeout, err)
		}
		return err
	}
	return nil
}
