package plugins

import (
	"strings"
	"testing"
	"time"
)

func TestExecuteCommandTimesOut(t *testing.T) {
	start := time.Now()
	err := executeCommand("sleep 10", 200*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("command was not killed promptly: %v", elapsed)
	}
}

func TestExecuteCommandSuccessAndFailure(t *testing.T) {
	if err := executeCommand("true", time.Second); err != nil {
		t.Fatal(err)
	}
	if err := executeCommand("false", time.Second); err == nil {
		t.Fatal("expected an error from a failing command")
	}
}
