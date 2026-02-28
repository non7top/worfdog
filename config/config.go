package config

import (
	"fmt"
	"os"

	"gopkg.in/ini.v1"
)

// ServiceConfig holds configuration for a monitored service
type ServiceConfig struct {
	Name              string
	Type              string // "systemd" or "https"
	Unit              string // systemd unit name (for systemd type)
	URL               string // URL to check (for https type)
	Timeout           int    // timeout in seconds
	RestartCmd        string // optional custom restart command
	MaxRestarts       int    // max restart attempts before reboot (0 = use global default)
	InsecureSkipVerify bool   // skip TLS certificate verification
	TLSHostnames      string // comma-separated list of acceptable TLS hostnames
	MaxRetries        int    // max retries for health check before marking as failed (0 = no retries)
}

// RebootConfig holds reboot-related configuration
type RebootConfig struct {
	Enabled       bool
	MaxRestarts   int    // maximum service restart attempts before reboot
	MaxReboots    int    // maximum number of reboots allowed
	WindowHours   int    // time window for counting reboots
	SudoPassword  string // optional sudo password
}

// Config holds the entire configuration
type Config struct {
	Reboot  RebootConfig
	Services []ServiceConfig
}

// Load reads and parses the INI configuration file
func Load(path string) (*Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg := &Config{}

	// Parse reboot section
	rebootSec := f.Section("reboot")
	cfg.Reboot.Enabled = rebootSec.Key("enabled").MustBool(false)
	cfg.Reboot.MaxRestarts = rebootSec.Key("max_restarts").MustInt(3)
	cfg.Reboot.MaxReboots = rebootSec.Key("max_reboots").MustInt(3)
	cfg.Reboot.WindowHours = rebootSec.Key("window_hours").MustInt(24)
	cfg.Reboot.SudoPassword = rebootSec.Key("sudo_password").String()

	// Parse services
	for _, section := range f.Sections() {
		if section.Name() == "DEFAULT" || section.Name() == "reboot" {
			continue
		}

		svc := ServiceConfig{
			Name:       section.Name(),
			Type:       section.Key("type").String(),
			Unit:       section.Key("unit").String(),
			URL:        section.Key("url").String(),
			Timeout:    section.Key("timeout").MustInt(10),
			RestartCmd: section.Key("restart_cmd").String(),
			MaxRestarts: section.Key("max_restarts").MustInt(0),
			InsecureSkipVerify: section.Key("insecure_skip_verify").MustBool(false),
			TLSHostnames: section.Key("tls_hostnames").String(),
			MaxRetries: section.Key("max_retries").MustInt(0),
		}

		// Set defaults based on type
		if svc.Type == "systemd" && svc.Unit == "" {
			svc.Unit = svc.Name
		}

		cfg.Services = append(cfg.Services, svc)
	}

	return cfg, nil
}

// LoadDefault loads configuration from standard paths
func LoadDefault() (*Config, error) {
	paths := []string{
		"worfdog.ini",
		"/etc/worfdog/worfdog.ini",
		"/etc/worfdog.ini",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		}
	}

	return nil, fmt.Errorf("no configuration file found")
}
