package plugins

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"worfdog/config"
)

var logger = log.New(log.Writer(), "[worfdog] ", log.LstdFlags)

// HTTPSPlugin monitors HTTPS endpoints
type HTTPSPlugin struct {
	cfg    config.ServiceConfig
	client *http.Client
}

// NewHTTPSPlugin creates a new HTTPS monitoring plugin
func NewHTTPSPlugin(cfg config.ServiceConfig) *HTTPSPlugin {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	// If custom hostnames are specified, use custom verification
	if cfg.TLSHostnames != "" {
		hostnames := strings.Split(cfg.TLSHostnames, ",")
		for i, h := range hostnames {
			hostnames[i] = strings.TrimSpace(h)
		}
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = func(cs tls.ConnectionState) error {
			// Check if any of the configured hostnames match the certificate
			for _, certName := range cs.PeerCertificates[0].DNSNames {
				for _, allowedName := range hostnames {
					if certName == allowedName {
						return nil
					}
				}
			}
			// Also check IP addresses
			for _, certIP := range cs.PeerCertificates[0].IPAddresses {
				for _, allowedName := range hostnames {
					if ip := net.ParseIP(allowedName); ip != nil && ip.Equal(certIP) {
						return nil
					}
				}
			}
			return fmt.Errorf("certificate not valid for any configured hostname")
		}
	}

	return &HTTPSPlugin{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
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

	var lastStatusCode int
	maxRetries := p.cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			logger.Printf("[%s] Retry attempt %d/%d", p.cfg.Name, attempt, maxRetries)
		}
		resp, err := p.client.Get(p.cfg.URL)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(2 * time.Second)
				continue
			}
			return CheckResult{
				Status:  StatusCritical,
				Message: fmt.Sprintf("Connection failed: %v", err),
				Service: p.cfg.Name,
			}
		}

		lastStatusCode = resp.StatusCode
		resp.Body.Close()

		if lastStatusCode >= 200 && lastStatusCode < 400 {
			return CheckResult{
				Status:  StatusOK,
				Message: fmt.Sprintf("HTTP %d", lastStatusCode),
				Service: p.cfg.Name,
			}
		}

		if attempt < maxRetries {
			time.Sleep(2 * time.Second)
		}
	}

	return CheckResult{
		Status:  StatusCritical,
		Message: fmt.Sprintf("HTTP %d", lastStatusCode),
		Service: p.cfg.Name,
	}
}

func (p *HTTPSPlugin) Restart() error {
	if p.cfg.RestartCmd != "" {
		return executeCommand(p.cfg.RestartCmd)
	}
	return fmt.Errorf("no restart command configured for %s", p.cfg.Name)
}
