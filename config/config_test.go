package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWorfdogSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	// Test with valid keys
	validConfig := `
[worfdog]
initial_delay = 30
interval = 30
dry_run = false
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.HasWarnings() {
		t.Errorf("Expected no warnings for valid config, got: %v", cfg.GetWarnings())
	}

	// Test with invalid key
	invalidConfig := `
[worfdog]
initial_delay = 30
unknown_option = true
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err = Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.HasWarnings() {
		t.Error("Expected warnings for unknown option, got none")
	}

	found := false
	for _, w := range cfg.Warnings {
		if w.Key == "unknown_option" && w.Section == "worfdog" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning for 'unknown_option' in [worfdog], got: %v", cfg.GetWarnings())
	}
}

func TestValidateRebootSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	// Test with valid keys
	validConfig := `
[reboot]
enabled = true
max_restarts = 3
max_reboots = 3
window_hours = 24
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.HasWarnings() {
		t.Errorf("Expected no warnings for valid config, got: %v", cfg.GetWarnings())
	}

	// Test with invalid key
	invalidConfig := `
[reboot]
enabled = true
invalid_key = 123
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err = Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	found := false
	for _, w := range cfg.Warnings {
		if w.Key == "invalid_key" && w.Section == "reboot" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning for 'invalid_key' in [reboot], got: %v", cfg.GetWarnings())
	}
}

func TestValidateServiceSection(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	// Test with valid keys
	validConfig := `
[nginx]
type = systemd
unit = nginx
max_restarts = 5
`
	if err := os.WriteFile(configPath, []byte(validConfig), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.HasWarnings() {
		t.Errorf("Expected no warnings for valid config, got: %v", cfg.GetWarnings())
	}

	// Test with invalid key
	invalidConfig := `
[webapp]
type = https
url = https://localhost/health
bad_option = true
`
	if err := os.WriteFile(configPath, []byte(invalidConfig), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err = Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	found := false
	for _, w := range cfg.Warnings {
		if w.Key == "bad_option" && w.Section == "webapp" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected warning for 'bad_option' in [webapp], got: %v", cfg.GetWarnings())
	}
}

func TestValidateMultipleSections(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	// Test with multiple invalid keys across sections
	config := `
[worfdog]
interval = 30
worfdog_bad = 1

[reboot]
enabled = true
reboot_bad = 2

[nginx]
type = systemd
unit = nginx
service_bad = 3
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !cfg.HasWarnings() {
		t.Fatal("Expected warnings for unknown options")
	}

	expectedWarnings := []string{"worfdog_bad", "reboot_bad", "service_bad"}
	foundWarnings := make(map[string]bool)
	for _, w := range cfg.Warnings {
		foundWarnings[w.Key] = true
	}

	for _, expected := range expectedWarnings {
		if !foundWarnings[expected] {
			t.Errorf("Expected warning for '%s', got: %v", expected, cfg.GetWarnings())
		}
	}
}

func TestGetWarnings(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	config := `
[worfdog]
zeta_option = 1
alpha_option = 2
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	warnings := cfg.GetWarnings()
	if len(warnings) != 2 {
		t.Fatalf("Expected 2 warnings, got %d", len(warnings))
	}

	// Check that warnings are sorted
	if warnings[0] > warnings[1] {
		t.Error("Expected warnings to be sorted alphabetically")
	}
}

func TestWarningString(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	config := `
[worfdog]
bad_option = 1
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	warningStr := cfg.WarningString()
	if warningStr == "" {
		t.Error("Expected non-empty warning string")
	}
	if !contains(warningStr, "Configuration warnings:") {
		t.Error("Expected 'Configuration warnings:' in warning string")
	}
	if !contains(warningStr, "bad_option") {
		t.Error("Expected 'bad_option' in warning string")
	}
}

func TestNoWarningsForValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.ini")

	config := `
[worfdog]
initial_delay = 60
interval = 15
dry_run = false

[reboot]
enabled = true
max_restarts = 5
max_reboots = 3
window_hours = 24
sudo_password = secret

[nginx]
type = systemd
unit = nginx
max_restarts = 3

[webapp]
type = https
url = https://localhost/health
timeout = 10
max_retries = 3
tls_hostnames = localhost,example.com
insecure_skip_verify = false
restart_cmd = systemctl restart webapp
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.HasWarnings() {
		t.Errorf("Expected no warnings for fully valid config, got: %v", cfg.GetWarnings())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
