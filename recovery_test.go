package main

import (
	"errors"
	"io"
	"log"
	"path/filepath"
	"testing"

	"worfdog/config"
	"worfdog/plugins"
	"worfdog/reboot"
)

type fakePlugin struct {
	restarts int
}

func (p *fakePlugin) Name() string { return "svc" }

func (p *fakePlugin) Check() plugins.CheckResult {
	return plugins.CheckResult{Status: plugins.StatusCritical, Service: "svc"}
}

func (p *fakePlugin) Restart() error {
	p.restarts++
	return errors.New("restart failed")
}

func (p *fakePlugin) GetConfig() config.ServiceConfig {
	return config.ServiceConfig{Name: "svc", RestartCmd: "true"}
}

func TestRecoveryKeepsRestartingWhenRebootBlocked(t *testing.T) {
	p := &fakePlugin{}
	cfg := &config.Config{Reboot: config.RebootConfig{Enabled: true, MaxRestarts: 1}}
	w := &Watchdog{
		cfg:           cfg,
		plugins:       []plugins.Plugin{p},
		restartCounts: map[string]int{},
		failureCounts: map[string]int{},
		// maxReboots=0 means every reboot is blocked
		rebootTracker: reboot.NewTracker(0, 24, "", filepath.Join(t.TempDir(), "state.json")),
		logger:        log.New(io.Discard, "", 0),
	}

	for range 4 {
		w.attemptRecovery("svc")
	}

	if p.restarts != 4 {
		t.Fatalf("expected 4 restart attempts despite blocked reboots, got %d", p.restarts)
	}
}
