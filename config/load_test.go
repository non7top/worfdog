package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "worfdog.ini")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"bad integer", "[worfdog]\ninterval = 3O\n", "[worfdog] interval"},
		{"zero interval", "[worfdog]\ninterval = 0\n", "out of range"},
		{"bad bool", "[reboot]\nenabled = maybe\n", "[reboot] enabled"},
		{"port range", "[db]\ntype = mysql\nhost = h\nusername = u\nport = 70000\n", "[db] port"},
		{"missing url", "[web]\ntype = https\n", "[web] url"},
		{"missing mysql host", "[db]\ntype = mysql\nusername = u\n", "[db] host"},
		{"missing mysql username", "[db]\ntype = mysql\nhost = h\n", "[db] username"},
		{"unknown type", "[x]\ntype = ftp\n", "unknown type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(writeConfig(t, tt.content))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestLoadReportsAllProblemsAtOnce(t *testing.T) {
	_, err := Load(writeConfig(t, "[worfdog]\ninterval = x\n[web]\ntype = https\n"))
	if err == nil || !strings.Contains(err.Error(), "interval") || !strings.Contains(err.Error(), "url") {
		t.Fatalf("expected both problems reported, got %v", err)
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(writeConfig(t, "[nginx]\ntype = systemd\n[web]\ntype = http\nurl = http://x\nmax_retries = 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Worfdog.Interval != 30 || cfg.Worfdog.InitialDelay != 30 || cfg.Reboot.WindowHours != 24 {
		t.Errorf("unexpected defaults: %+v %+v", cfg.Worfdog, cfg.Reboot)
	}
	if got := cfg.Services[0]; got.Unit != "nginx" || got.Timeout != 10 || got.MaxRetries != 1 {
		t.Errorf("unexpected systemd defaults: %+v", got)
	}
	if got := cfg.Services[1]; got.MaxRetries != 1 {
		t.Errorf("max_retries = 0 should mean the default of 1, got %d", got.MaxRetries)
	}
}

func TestExampleConfigLoads(t *testing.T) {
	cfg, err := Load("../worfdog.ini.example")
	if err != nil {
		t.Fatalf("the shipped example must be valid: %v", err)
	}
	if cfg.HasWarnings() {
		t.Fatalf("the shipped example must have no warnings: %v", cfg.GetWarnings())
	}
}
