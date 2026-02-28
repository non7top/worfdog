package plugins

import (
	"os/exec"
)

// executeCommand runs a shell command
func executeCommand(cmd string) error {
	return exec.Command("sh", "-c", cmd).Run()
}
