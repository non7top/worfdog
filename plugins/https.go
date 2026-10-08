package plugins

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
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
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	// Custom hostnames replace the name check only; the chain and expiry are still verified.
	if cfg.TLSHostnames != "" && !cfg.InsecureSkipVerify {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyConnection = verifyAgainstHostnames(strings.Split(cfg.TLSHostnames, ","), nil)
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

// verifyAgainstHostnames verifies the peer chain against roots (system roots when nil)
// and accepts the certificate if it is valid for any of the given names.
func verifyAgainstHostnames(hostnames []string, roots *x509.CertPool) func(tls.ConnectionState) error {
	names := make([]string, 0, len(hostnames))
	for _, h := range hostnames {
		if h = strings.TrimSpace(h); h != "" {
			names = append(names, h)
		}
	}

	return func(cs tls.ConnectionState) error {
		if len(cs.PeerCertificates) == 0 {
			return fmt.Errorf("no peer certificate")
		}
		leaf := cs.PeerCertificates[0]

		opts := x509.VerifyOptions{Roots: roots, Intermediates: x509.NewCertPool()}
		for _, c := range cs.PeerCertificates[1:] {
			opts.Intermediates.AddCert(c)
		}
		if _, err := leaf.Verify(opts); err != nil {
			return err
		}

		for _, name := range names {
			if leaf.VerifyHostname(name) == nil {
				return nil
			}
		}
		return fmt.Errorf("certificate not valid for any configured hostname")
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
	defer func() { _ = resp.Body.Close() }()

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
