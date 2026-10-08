package reboot

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Tracker tracks system reboots and enforces limits
type Tracker struct {
	mu          sync.Mutex
	maxReboots  int
	windowHours int
	rebootFile  string
	reboots     []time.Time
	stateErr    error // set when the state file exists but cannot be read; blocks reboots
	sudoPass    string
}

// NewTracker creates a new reboot tracker
func NewTracker(maxReboots, windowHours int, sudoPass, stateFile string) *Tracker {
	if stateFile == "" {
		stateFile = "/var/lib/worfdog/reboot_state.json"
	}

	t := &Tracker{
		maxReboots:  maxReboots,
		windowHours: windowHours,
		rebootFile:  stateFile,
		reboots:     []time.Time{},
		sudoPass:    sudoPass,
	}

	// Load existing state
	t.stateErr = t.load()

	return t
}

// CanReboot checks if we're allowed to reboot within the configured limits
func (t *Tracker) CanReboot() (bool, string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cleanOldReboots()

	if t.stateErr != nil {
		return false, fmt.Sprintf("reboot state %s is unreadable (%v); fix it or run -reset-reboots", t.rebootFile, t.stateErr)
	}

	if len(t.reboots) >= t.maxReboots {
		return false, fmt.Sprintf("Maximum reboots (%d) reached within %d hours", t.maxReboots, t.windowHours)
	}

	return true, ""
}

// RecordReboot records a reboot event
func (t *Tracker) RecordReboot() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.reboots = append(t.reboots, time.Now())
	return t.save()
}

// GetRebootCount returns the number of reboots in the current window
func (t *Tracker) GetRebootCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cleanOldReboots()
	return len(t.reboots)
}

// Reboot initiates a system reboot
func (t *Tracker) Reboot() error {
	// Check if reboot is allowed
	if allowed, reason := t.CanReboot(); !allowed {
		return fmt.Errorf("reboot not allowed: %s", reason)
	}

	// Record the reboot before executing
	if err := t.RecordReboot(); err != nil {
		return fmt.Errorf("failed to record reboot: %w", err)
	}

	// Execute reboot command
	var cmd *exec.Cmd
	if t.sudoPass != "" {
		// Use sudo with password
		cmd = exec.Command("sudo", "-S", "reboot")
		cmd.Stdin = strings.NewReader(t.sudoPass + "\n")
	} else {
		// Try reboot directly (may require sudo privileges)
		cmd = exec.Command("reboot")
	}

	if err := cmd.Run(); err != nil {
		// Reboot command may not return on success
		if err.Error() != "exit status 255" {
			return fmt.Errorf("reboot command failed: %w", err)
		}
	}

	return nil
}

// cleanOldReboots removes reboot records outside the time window
func (t *Tracker) cleanOldReboots() {
	cutoff := time.Now().Add(-time.Duration(t.windowHours) * time.Hour)
	cleaned := []time.Time{}

	for _, rt := range t.reboots {
		if rt.After(cutoff) {
			cleaned = append(cleaned, rt)
		}
	}

	t.reboots = cleaned
}

// save persists the reboot state to disk
func (t *Tracker) save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(t.rebootFile), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(t.reboots)
	if err != nil {
		return err
	}

	// Rename is atomic, so a crash never leaves a truncated state file.
	tmp := t.rebootFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, t.rebootFile)
}

// load restores the reboot state from disk
func (t *Tracker) load() error {
	data, err := os.ReadFile(t.rebootFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &t.reboots)
}

// Reset clears all reboot records
func (t *Tracker) Reset() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.reboots = []time.Time{}
	if err := t.save(); err != nil {
		return err
	}
	t.stateErr = nil
	return nil
}

// Status returns a human-readable status of the reboot tracker
func (t *Tracker) Status() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cleanOldReboots()

	status := fmt.Sprintf("Reboots in last %d hours: %d/%d",
		t.windowHours, len(t.reboots), t.maxReboots)
	if t.stateErr != nil {
		status += fmt.Sprintf(" (state unreadable: %v; reboots blocked)", t.stateErr)
	}
	return status
}
