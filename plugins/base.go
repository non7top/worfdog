package plugins

import (
	"fmt"

	"worfdog/config"
)

// base holds what every plugin shares: its config and the configured restart command.
type base struct {
	cfg config.ServiceConfig
}

func (b base) Name() string {
	return b.cfg.Name
}

func (b base) GetConfig() config.ServiceConfig {
	return b.cfg
}

// Restart runs the configured restart_cmd; plugins without a default restart use it as-is.
func (b base) Restart() error {
	if b.cfg.RestartCmd != "" {
		return executeCommand(b.cfg.RestartCmd, restartTimeout)
	}
	return fmt.Errorf("no restart command configured for %s", b.cfg.Name)
}
