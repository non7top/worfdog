package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"worfdog/config"
)

// getProjectRoot returns the project root directory
func getProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	// If we're in the project root, return it
	if _, err := os.Stat("go.mod"); err == nil {
		return cwd
	}
	// Otherwise try parent
	return filepath.Dir(cwd)
}

// TestBuildBinary tests that the binary builds successfully
func TestBuildBinary(t *testing.T) {
	projectRoot := getProjectRoot()
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", "worfdog-test", ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build binary: %v\n%s", err, string(output))
	}

	// Clean up
	defer os.Remove(filepath.Join(projectRoot, "worfdog-test"))

	// Verify binary exists and is executable
	info, err := os.Stat(filepath.Join(projectRoot, "worfdog-test"))
	if err != nil {
		t.Fatalf("Binary not found: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("Binary is not executable")
	}
}

// TestVersionFlag tests that the version flag works
func TestVersionFlag(t *testing.T) {
	projectRoot := getProjectRoot()
	binary := filepath.Join(projectRoot, "worfdog-test")

	// Build first
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}
	defer os.Remove(binary)

	cmd = exec.Command(binary, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run version command: %v\n%s", err, string(output))
	}

	if !strings.Contains(string(output), "worfdog") {
		t.Errorf("Expected output to contain 'worfdog', got %q", string(output))
	}
}

// TestHelpFlag tests that the help flag works
func TestHelpFlag(t *testing.T) {
	projectRoot := getProjectRoot()
	binary := filepath.Join(projectRoot, "worfdog-test")

	// Build first
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}
	defer os.Remove(binary)

	cmd = exec.Command(binary, "-h")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run help command: %v\n%s", err, string(output))
	}

	helpOutput := string(output)
	expectedFlags := []string{"-config", "-interval", "-status", "-version", "-dry_run"}
	for _, flag := range expectedFlags {
		if !strings.Contains(helpOutput, flag) {
			t.Errorf("Help output missing expected flag: %s", flag)
		}
	}
}

// TestDryRunFlag tests that the dry_run flag is recognized
func TestDryRunFlag(t *testing.T) {
	projectRoot := getProjectRoot()
	binary := filepath.Join(projectRoot, "worfdog-test")

	// Build first
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", binary, ".")
	cmd.Dir = projectRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build: %v\n%s", err, string(out))
	}
	defer os.Remove(binary)

	// Create a minimal test config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "worfdog.ini")
	configContent := `
[reboot]
enabled = false

[nginx]
type = systemd
unit = nginx
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Run with dry_run and short timeout
	cmd = exec.Command(binary, "-config", configPath, "-dry_run", "-interval", "1s")
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Start the command
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command: %v", err)
	}

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// Kill it (it should be running)
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("failed to kill process: %v", err)
	}
}

// TestLoadDefaultConfig tests loading the example configuration
func TestLoadDefaultConfig(t *testing.T) {
	projectRoot := getProjectRoot()
	configPath := filepath.Join(projectRoot, "worfdog.ini.example")

	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load example config: %v", err)
	}

	if cfg == nil {
		t.Fatal("Config is nil")
	}

	// Verify reboot config
	if !cfg.Reboot.Enabled {
		t.Error("Expected reboot to be enabled in example config")
	}
	if cfg.Reboot.MaxReboots != 3 {
		t.Errorf("Expected max_reboots=3, got %d", cfg.Reboot.MaxReboots)
	}

	// Verify services are loaded
	if len(cfg.Services) == 0 {
		t.Error("Expected services to be loaded from example config")
	}

	// Check for expected services
	serviceNames := make(map[string]bool)
	for _, svc := range cfg.Services {
		serviceNames[svc.Name] = true
	}

	expectedServices := []string{"nginx", "apache", "webapp", "api"}
	for _, expected := range expectedServices {
		if !serviceNames[expected] {
			t.Errorf("Expected service %q not found in config", expected)
		}
	}
}

// TestNewWatchdog tests watchdog creation
func TestNewWatchdog(t *testing.T) {
	cfg := &config.Config{
		Reboot: config.RebootConfig{
			Enabled:     false,
			MaxRestarts: 3,
			MaxReboots:  3,
			WindowHours: 24,
		},
		Services: []config.ServiceConfig{
			{Name: "test", Type: "systemd", Unit: "nginx"},
		},
	}

	watchdog := NewWatchdog(cfg, 30*time.Second, true)

	if watchdog == nil {
		t.Fatal("NewWatchdog returned nil")
	}
	if !watchdog.dryRun {
		t.Error("Expected dryRun to be true")
	}
	if len(watchdog.plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(watchdog.plugins))
	}
}
