package plugins

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"worfdog/config"
)

// HTTPSPlugin monitors HTTPS endpoints
type HTTPSPlugin struct {
	cfg    config.ServiceConfig
	client *http.Client
}

// NewHTTPSPlugin creates a new HTTPS monitoring plugin
func NewHTTPSPlugin(cfg config.ServiceConfig) *HTTPSPlugin {
	return &HTTPSPlugin{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
		},
	}
}

func (p *HTTPSPlugin) Name() string {
	return p.cfg.Name
}

func (p *HTTPSPlugin) GetConfig() config.ServiceConfig {
	return p.cfg
}

func (p *HTTPSPlugin) Check() CheckResult {
	if p.cfg.URL == "" {
		return CheckResult{
			Status:  StatusUnknown,
			Message: "No URL configured",
			Service: p.cfg.Name,
		}
	}

	resp, err := p.client.Get(p.cfg.URL)
	if err != nil {
		return CheckResult{
			Status:  StatusCritical,
			Message: fmt.Sprintf("Connection failed: %v", err),
			Service: p.cfg.Name,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return CheckResult{
			Status:  StatusOK,
			Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
			Service: p.cfg.Name,
		}
	}

	return CheckResult{
		Status:  StatusCritical,
		Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
		Service: p.cfg.Name,
	}
}

func (p *HTTPSPlugin) Restart() error {
	if p.cfg.RestartCmd != "" {
		return executeCommand(p.cfg.RestartCmd)
	}
	return fmt.Errorf("no restart command configured for %s", p.cfg.Name)
}
