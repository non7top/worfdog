package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

// ValidKeys defines the valid configuration keys for each section
var ValidKeys = map[string][]string{
	"worfdog": {"initial_delay", "interval", "dry_run"},
	"reboot":  {"enabled", "max_restarts", "max_reboots", "window_hours", "sudo_password"},
	"service": {"type", "unit", "url", "timeout", "restart_cmd", "max_restarts", "insecure_skip_verify", "tls_hostnames", "max_retries", "host", "port", "username", "password", "database"},
}

// ConfigWarning represents a configuration warning
type ConfigWarning struct {
	Section string
	Key     string
	Message string
}

// Config holds the entire configuration
type Config struct {
	Worfdog  WorfdogConfig
	Reboot   RebootConfig
	Services []ServiceConfig
	Warnings []ConfigWarning
}

// ServiceConfig holds configuration for a monitored service
type ServiceConfig struct {
	Name              string
	Type              string // "systemd", "https", or "mysql"
	Unit              string // systemd unit name (for systemd type)
	URL               string // URL to check (for https type)
	Host              string // host to connect to (for mysql type)
	Port              int    // port to connect to (for mysql type)
	Username          string // username (for mysql type)
	Password          string // password (for mysql type)
	Database          string // database name (for mysql type)
	Timeout           int    // timeout in seconds
	RestartCmd        string // optional custom restart command
	MaxRestarts       int    // max restart attempts before reboot (0 = use global default)
	InsecureSkipVerify bool   // skip TLS certificate verification
	TLSHostnames      string // comma-separated list of acceptable TLS hostnames
	MaxRetries        int    // max retries for health check before marking as failed
}

// RebootConfig holds reboot-related configuration
type RebootConfig struct {
	Enabled       bool
	MaxRestarts   int    // maximum service restart attempts before reboot
	MaxReboots    int    // maximum number of reboots allowed
	WindowHours   int    // time window for counting reboots
	SudoPassword  string // optional sudo password
}

// WorfdogConfig holds general worfdog configuration
type WorfdogConfig struct {
	InitialDelay int  // initial delay before first check in seconds
	Interval     int  // health check interval in seconds
	DryRun       bool // dry run mode (log actions without executing)
}

// Load reads and parses the INI configuration file
func Load(path string) (*Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg := &Config{
		Warnings: []ConfigWarning{},
	}

	// Validate worfdog section
	worfdogSec := f.Section("worfdog")
	cfg.Worfdog.InitialDelay = worfdogSec.Key("initial_delay").MustInt(30)
	cfg.Worfdog.Interval = worfdogSec.Key("interval").MustInt(30)
	cfg.Worfdog.DryRun = worfdogSec.Key("dry_run").MustBool(false)
	cfg.Warnings = append(cfg.Warnings, validateSection(worfdogSec, "worfdog")...)

	// Validate reboot section
	rebootSec := f.Section("reboot")
	cfg.Reboot.Enabled = rebootSec.Key("enabled").MustBool(false)
	cfg.Reboot.MaxRestarts = rebootSec.Key("max_restarts").MustInt(3)
	cfg.Reboot.MaxReboots = rebootSec.Key("max_reboots").MustInt(3)
	cfg.Reboot.WindowHours = rebootSec.Key("window_hours").MustInt(24)
	cfg.Reboot.SudoPassword = rebootSec.Key("sudo_password").String()
	cfg.Warnings = append(cfg.Warnings, validateSection(rebootSec, "reboot")...)

	// Parse and validate services
	for _, section := range f.Sections() {
		// Skip sections without a type field (not services)
		if section.Key("type").String() == "" {
			continue
		}

		svc := ServiceConfig{
			Name:       section.Name(),
			Type:       section.Key("type").String(),
			Unit:       section.Key("unit").String(),
			URL:        section.Key("url").String(),
			Host:       section.Key("host").String(),
			Port:       section.Key("port").MustInt(3306),
			Username:   section.Key("username").String(),
			Password:   section.Key("password").String(),
			Database:   section.Key("database").String(),
			Timeout:    section.Key("timeout").MustInt(10),
			RestartCmd: section.Key("restart_cmd").String(),
			MaxRestarts: section.Key("max_restarts").MustInt(0),
			InsecureSkipVerify: section.Key("insecure_skip_verify").MustBool(false),
			TLSHostnames: section.Key("tls_hostnames").String(),
			MaxRetries: section.Key("max_retries").MustInt(0),
		}

		// Validate service section
		cfg.Warnings = append(cfg.Warnings, validateSection(section, "service")...)

		// Set defaults based on type
		if svc.Type == "systemd" && svc.Unit == "" {
			svc.Unit = svc.Name
		}

		cfg.Services = append(cfg.Services, svc)
	}

	return cfg, nil
}

// validateSection checks for unknown keys in a config section
func validateSection(section *ini.Section, sectionType string) []ConfigWarning {
	var warnings []ConfigWarning
	validKeys := ValidKeys[sectionType]

	if validKeys == nil {
		return warnings
	}

	// Create a map of valid keys for quick lookup
	validKeyMap := make(map[string]bool)
	for _, key := range validKeys {
		validKeyMap[key] = true
	}

	// Check each key in the section
	for _, key := range section.KeyStrings() {
		if !validKeyMap[key] {
			warnings = append(warnings, ConfigWarning{
				Section: section.Name(),
				Key:     key,
				Message: fmt.Sprintf("unknown option '%s' in section [%s]", key, section.Name()),
			})
		}
	}

	return warnings
}

// GetWarnings returns formatted warning messages
func (c *Config) GetWarnings() []string {
	messages := make([]string, len(c.Warnings))
	for i, w := range c.Warnings {
		messages[i] = w.Message
	}
	sort.Strings(messages)
	return messages
}

// HasWarnings returns true if there are any configuration warnings
func (c *Config) HasWarnings() bool {
	return len(c.Warnings) > 0
}

// WarningString returns all warnings as a formatted string
func (c *Config) WarningString() string {
	if !c.HasWarnings() {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\nConfiguration warnings:\n")
	for _, msg := range c.GetWarnings() {
		sb.WriteString(fmt.Sprintf("  WARNING: %s\n", msg))
	}
	return sb.String()
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
