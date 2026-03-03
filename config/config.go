package config

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

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

// getValidKeys extracts valid keys from a struct type using json tags
func getValidKeys(structType reflect.Type) []string {
	keys := make([]string, 0, structType.NumField())
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if tag := field.Tag.Get("ini"); tag != "" && tag != "-" {
			// Extract key name from ini tag (e.g., "initial_delay" from "initial_delay")
			key := strings.Split(tag, ",")[0]
			if key != "" {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

// ValidKeys defines the valid configuration keys for each section
var ValidKeys = map[string][]string{
	"worfdog": getValidKeys(reflect.TypeOf(WorfdogConfig{})),
	"reboot":  getValidKeys(reflect.TypeOf(RebootConfig{})),
	"service": getValidKeys(reflect.TypeOf(ServiceConfig{})),
}

// ServiceConfig holds configuration for a monitored service
type ServiceConfig struct {
	Name              string `ini:"-"`
	Type              string `ini:"type"`               // "systemd", "https", or "mysql"
	Unit              string `ini:"unit"`               // systemd unit name (for systemd type)
	URL               string `ini:"url"`                // URL to check (for https type)
	Host              string `ini:"host"`               // host to connect to (for mysql type)
	Port              int    `ini:"port"`               // port to connect to (for mysql type)
	Username          string `ini:"username"`           // username (for mysql type)
	Password          string `ini:"password"`           // password (for mysql type)
	Database          string `ini:"database"`           // database name (for mysql type)
	Timeout           int    `ini:"timeout"`            // timeout in seconds
	RestartCmd        string `ini:"restart_cmd"`        // optional custom restart command
	MaxRestarts       int    `ini:"max_restarts"`       // max restart attempts before reboot (0 = use global default)
	InsecureSkipVerify bool   `ini:"insecure_skip_verify"` // skip TLS certificate verification
	TLSHostnames      string `ini:"tls_hostnames"`      // comma-separated list of acceptable TLS hostnames
	MaxRetries        int    `ini:"max_retries"`        // max retries for health check before marking as failed
}

// RebootConfig holds reboot-related configuration
type RebootConfig struct {
	Enabled      bool   `ini:"enabled"`       // enable/disable reboot
	MaxRestarts  int    `ini:"max_restarts"`  // maximum service restart attempts before reboot
	MaxReboots   int    `ini:"max_reboots"`   // maximum number of reboots allowed
	WindowHours  int    `ini:"window_hours"`  // time window for counting reboots
	SudoPassword string `ini:"sudo_password"` // optional sudo password
}

// WorfdogConfig holds general worfdog configuration
type WorfdogConfig struct {
	InitialDelay int  `ini:"initial_delay"` // initial delay before first check in seconds
	Interval     int  `ini:"interval"`      // health check interval in seconds
	DryRun       bool `ini:"dry_run"`       // dry run mode (log actions without executing)
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
