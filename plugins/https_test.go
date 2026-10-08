package plugins

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"

	"worfdog/config"
)

func newTLSServer(t *testing.T) (*httptest.Server, *x509.CertPool) {
	t.Helper()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, srv.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
}

func TestTLSHostnamesStillVerifiesChain(t *testing.T) {
	srv, _ := newTLSServer(t)

	p := NewHTTPSPlugin(config.ServiceConfig{Name: "web", URL: srv.URL, Timeout: 5, TLSHostnames: "127.0.0.1"})
	if got := p.Check(); got.Status != StatusCritical {
		t.Fatalf("untrusted certificate with a matching name must fail, got %v: %s", got.Status, got.Message)
	}
}

func TestVerifyAgainstHostnames(t *testing.T) {
	srv, roots := newTLSServer(t)

	check := func(hostnames string) error {
		conn, err := tls.Dial("tcp", srv.Listener.Addr().String(), &tls.Config{InsecureSkipVerify: true})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		return verifyAgainstHostnames([]string{hostnames}, roots)(conn.ConnectionState())
	}

	if err := check("127.0.0.1"); err != nil {
		t.Errorf("trusted certificate with matching name should pass: %v", err)
	}
	if err := check("other.example"); err == nil {
		t.Error("certificate must be rejected for a name it is not valid for")
	}
}

func TestInsecureSkipVerifyStillAllowsUntrusted(t *testing.T) {
	srv, _ := newTLSServer(t)

	p := NewHTTPSPlugin(config.ServiceConfig{Name: "web", URL: srv.URL, Timeout: 5, InsecureSkipVerify: true})
	if got := p.Check(); got.Status != StatusOK {
		t.Fatalf("expected OK with insecure_skip_verify, got %v: %s", got.Status, got.Message)
	}
}
