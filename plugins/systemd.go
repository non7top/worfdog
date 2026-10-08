package plugins

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"worfdog/config"
)

// SystemdPlugin monitors systemd services
type SystemdPlugin struct {
	cfg config.ServiceConfig
}

// NewSystemdPlugin creates a new systemd monitoring plugin
func NewSystemdPlugin(cfg config.ServiceConfig) *SystemdPlugin {
	return &SystemdPlugin{
		cfg: cfg,
	}
}

func (p *SystemdPlugin) Name() string {
	return p.cfg.Name
}

func (p *SystemdPlugin) GetConfig() config.ServiceConfig {
	return p.cfg
}

func (p *SystemdPlugin) checkTimeout() time.Duration {
	if p.cfg.Timeout > 0 {
		return time.Duration(p.cfg.Timeout) * time.Second
	}
	return 10 * time.Second
}

func (p *SystemdPlugin) Check() CheckResult {
	if p.cfg.Unit == "" {
		return CheckResult{
			Status:  StatusUnknown,
			Message: "No systemd unit configured",
			Service: p.cfg.Name,
		}
	}

	// Check service status using systemctl is-active
	ctx, cancel := context.WithTimeout(context.Background(), p.checkTimeout())
	defer cancel()
	output, err := exec.CommandContext(ctx, "systemctl", "is-active", p.cfg.Unit).CombinedOutput()
	if err != nil {
		status := strings.TrimSpace(string(output))
		if status == "" {
			status = "unknown"
		}
		return CheckResult{
			Status:  StatusCritical,
			Message: fmt.Sprintf("Service inactive: %s", status),
			Service: p.cfg.Name,
		}
	}

	status := strings.TrimSpace(string(output))
	if status == "active" {
		return CheckResult{
			Status:  StatusOK,
			Message: "Service active",
			Service: p.cfg.Name,
		}
	}

	return CheckResult{
		Status:  StatusWarning,
		Message: fmt.Sprintf("Service status: %s", status),
		Service: p.cfg.Name,
	}
}

func (p *SystemdPlugin) Restart() error {
	if p.cfg.RestartCmd != "" {
		return executeCommand(p.cfg.RestartCmd, restartTimeout)
	}

	// Default: use systemctl restart
	ctx, cancel := context.WithTimeout(context.Background(), restartTimeout)
	defer cancel()
	if err := exec.CommandContext(ctx, "systemctl", "restart", p.cfg.Unit).Run(); err != nil {
		return fmt.Errorf("failed to restart %s: %w", p.cfg.Unit, err)
	}
	return nil
}
