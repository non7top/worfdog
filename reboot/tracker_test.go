package reboot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestTracker(t *testing.T, maxReboots int) (*Tracker, string) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "state", "reboots.json")
	return NewTracker(maxReboots, 24, "", file), file
}

func TestCanRebootEnforcesLimit(t *testing.T) {
	tr, _ := newTestTracker(t, 2)

	for i := 0; i < 2; i++ {
		if ok, reason := tr.CanReboot(); !ok {
			t.Fatalf("reboot %d should be allowed: %s", i, reason)
		}
		if err := tr.RecordReboot(); err != nil {
			t.Fatal(err)
		}
	}

	if ok, _ := tr.CanReboot(); ok {
		t.Fatal("third reboot within the window must be blocked")
	}
	if got := tr.GetRebootCount(); got != 2 {
		t.Fatalf("expected 2 reboots counted, got %d", got)
	}
}

func TestOldRebootsFallOutOfWindow(t *testing.T) {
	tr, _ := newTestTracker(t, 1)
	tr.reboots = []time.Time{time.Now().Add(-25 * time.Hour)}

	if ok, reason := tr.CanReboot(); !ok {
		t.Fatalf("reboot older than the window must not count: %s", reason)
	}
	if got := tr.GetRebootCount(); got != 0 {
		t.Fatalf("expected 0 reboots in window, got %d", got)
	}
}

func TestStatePersistsAndCreatesConfiguredDirectory(t *testing.T) {
	tr, file := newTestTracker(t, 3)
	if err := tr.RecordReboot(); err != nil {
		t.Fatalf("saving into a missing directory failed: %v", err)
	}

	reloaded := NewTracker(3, 24, "", file)
	if got := reloaded.GetRebootCount(); got != 1 {
		t.Fatalf("expected persisted reboot to be reloaded, got %d", got)
	}
}

func TestResetClearsReboots(t *testing.T) {
	tr, file := newTestTracker(t, 3)
	if err := tr.RecordReboot(); err != nil {
		t.Fatal(err)
	}
	if err := tr.Reset(); err != nil {
		t.Fatal(err)
	}

	if got := NewTracker(3, 24, "", file).GetRebootCount(); got != 0 {
		t.Fatalf("expected reset to persist, got %d reboots", got)
	}
}

func TestCorruptStateBlocksRebootsUntilReset(t *testing.T) {
	_, file := newTestTracker(t, 3)
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(`[{"truncated`), 0o644); err != nil {
		t.Fatal(err)
	}

	tr := NewTracker(3, 24, "", file)
	if ok, reason := tr.CanReboot(); ok || !strings.Contains(reason, "unreadable") {
		t.Fatalf("corrupt state must block reboots with a clear reason, got ok=%v reason=%q", ok, reason)
	}
	if !strings.Contains(tr.Status(), "reboots blocked") {
		t.Fatalf("status should mention the problem: %s", tr.Status())
	}

	if err := tr.Reset(); err != nil {
		t.Fatal(err)
	}
	if ok, reason := tr.CanReboot(); !ok {
		t.Fatalf("reset should clear the block: %s", reason)
	}
}

func TestSaveLeavesNoTempFile(t *testing.T) {
	tr, file := newTestTracker(t, 3)
	if err := tr.RecordReboot(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(file + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temp file should be renamed away, stat err = %v", err)
	}
}
