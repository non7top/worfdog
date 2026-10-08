package config

import (
	"errors"
	"fmt"
	"math"
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
	Name               string `ini:"-"`
	Type               string `ini:"type"`                 // "systemd", "https", or "mysql"
	Unit               string `ini:"unit"`                 // systemd unit name (for systemd type)
	URL                string `ini:"url"`                  // URL to check (for https type)
	Host               string `ini:"host"`                 // host to connect to (for mysql type)
	Port               int    `ini:"port"`                 // port to connect to (for mysql type)
	Username           string `ini:"username"`             // username (for mysql type)
	Password           string `ini:"password"`             // password (for mysql type)
	Database           string `ini:"database"`             // database name (for mysql type)
	Timeout            int    `ini:"timeout"`              // timeout in seconds
	RestartCmd         string `ini:"restart_cmd"`          // optional custom restart command
	MaxRestarts        int    `ini:"max_restarts"`         // max restart attempts before reboot (0 = use global default)
	InsecureSkipVerify bool   `ini:"insecure_skip_verify"` // skip TLS certificate verification
	TLSHostnames       string `ini:"tls_hostnames"`        // comma-separated list of acceptable TLS hostnames
	MaxRetries         int    `ini:"max_retries"`          // max retries for health check before marking as failed
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

// loader collects every problem found while parsing so they are reported together.
type loader struct {
	errs []error
}

func (l *loader) fail(section, key, format string, args ...any) {
	l.errs = append(l.errs, fmt.Errorf("[%s] %s: %s", section, key, fmt.Sprintf(format, args...)))
}

// intKey reads an integer in [min, max]; an absent key yields def.
func (l *loader) intKey(sec *ini.Section, name string, def, min, max int) int {
	if !sec.HasKey(name) {
		return def
	}
	v, err := sec.Key(name).Int()
	if err != nil {
		l.fail(sec.Name(), name, "%q is not an integer", sec.Key(name).String())
		return def
	}
	if v < min || v > max {
		l.fail(sec.Name(), name, "%d is out of range (%d-%d)", v, min, max)
		return def
	}
	return v
}

// boolKey reads a boolean; an absent key yields def.
func (l *loader) boolKey(sec *ini.Section, name string, def bool) bool {
	if !sec.HasKey(name) {
		return def
	}
	v, err := sec.Key(name).Bool()
	if err != nil {
		l.fail(sec.Name(), name, "%q is not a boolean", sec.Key(name).String())
		return def
	}
	return v
}

// Load reads, parses and validates the INI configuration file
func Load(path string) (*Config, error) {
	f, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	cfg := &Config{
		Warnings: []ConfigWarning{},
	}
	l := &loader{}

	worfdogSec := f.Section("worfdog")
	cfg.Worfdog.InitialDelay = l.intKey(worfdogSec, "initial_delay", 30, 0, math.MaxInt32)
	cfg.Worfdog.Interval = l.intKey(worfdogSec, "interval", 30, 1, math.MaxInt32)
	cfg.Worfdog.DryRun = l.boolKey(worfdogSec, "dry_run", false)
	cfg.Warnings = append(cfg.Warnings, validateSection(worfdogSec, "worfdog")...)

	rebootSec := f.Section("reboot")
	cfg.Reboot.Enabled = l.boolKey(rebootSec, "enabled", false)
	cfg.Reboot.MaxRestarts = l.intKey(rebootSec, "max_restarts", 3, 0, math.MaxInt32)
	cfg.Reboot.MaxReboots = l.intKey(rebootSec, "max_reboots", 3, 0, math.MaxInt32)
	cfg.Reboot.WindowHours = l.intKey(rebootSec, "window_hours", 24, 1, math.MaxInt32)
	cfg.Reboot.SudoPassword = rebootSec.Key("sudo_password").String()
	cfg.Warnings = append(cfg.Warnings, validateSection(rebootSec, "reboot")...)

	// Sections without a type key are not services
	for _, section := range f.Sections() {
		if section.Key("type").String() == "" {
			continue
		}

		svc := ServiceConfig{
			Name:               section.Name(),
			Type:               section.Key("type").String(),
			Unit:               section.Key("unit").String(),
			URL:                section.Key("url").String(),
			Host:               section.Key("host").String(),
			Port:               l.intKey(section, "port", 3306, 1, 65535),
			Username:           section.Key("username").String(),
			Password:           section.Key("password").String(),
			Database:           section.Key("database").String(),
			Timeout:            l.intKey(section, "timeout", 10, 1, math.MaxInt32),
			RestartCmd:         section.Key("restart_cmd").String(),
			MaxRestarts:        l.intKey(section, "max_restarts", 0, 0, math.MaxInt32),
			InsecureSkipVerify: l.boolKey(section, "insecure_skip_verify", false),
			TLSHostnames:       section.Key("tls_hostnames").String(),
			MaxRetries:         l.intKey(section, "max_retries", 1, 0, math.MaxInt32),
		}
		// 0 has always meant "use the default"
		if svc.MaxRetries == 0 {
			svc.MaxRetries = 1
		}

		cfg.Warnings = append(cfg.Warnings, validateSection(section, "service")...)

		switch svc.Type {
		case "systemd":
			if svc.Unit == "" {
				svc.Unit = svc.Name
			}
		case "https", "http":
			if svc.URL == "" {
				l.fail(svc.Name, "url", "required for type %q", svc.Type)
			}
		case "mysql":
			if svc.Host == "" {
				l.fail(svc.Name, "host", "required for type mysql")
			}
			if svc.Username == "" {
				l.fail(svc.Name, "username", "required for type mysql")
			}
		default:
			l.fail(svc.Name, "type", "unknown type %q (expected systemd, https, http or mysql)", svc.Type)
		}

		cfg.Services = append(cfg.Services, svc)
	}

	if len(l.errs) > 0 {
		return nil, fmt.Errorf("invalid configuration:\n%w", errors.Join(l.errs...))
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
		fmt.Fprintf(&sb, "  WARNING: %s\n", msg)
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
