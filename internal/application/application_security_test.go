// SPDX-License-Identifier: Apache-2.0

package application

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

// TestSecurityMiddleware_HSTSOnTLS verifies the default Strict-Transport-Security
// header is emitted on a request that arrived over TLS. HSTSMaxAge = 31536000
// (1 year) is the default.
func TestSecurityMiddleware_HSTSOnTLS(t *testing.T) {
	app := newAppWithMiddleware(t, DefaultConfig())

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	assertDefaultHSTS(t, rec.Header().Get(echo.HeaderStrictTransportSecurity))
}

// TestSecurityMiddleware_HSTSViaTrustedProxy covers TLS terminated at a load
// balancer: in proxy mode, X-Forwarded-Proto=https from a trusted proxy
// address (here a private-network peer, trusted by default) counts as HTTPS.
func TestSecurityMiddleware_HSTSViaTrustedProxy(t *testing.T) {
	app := newAppWithMiddleware(t, proxyModeConfig())

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "10.0.0.5:43210"
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	assertDefaultHSTS(t, rec.Header().Get(echo.HeaderStrictTransportSecurity))
}

// TestSecurityMiddleware_NoHSTSFromUntrustedForwardedProto guards against a
// client faking HTTPS (GO-2026-6879): in proxy mode, X-Forwarded-Proto from
// an address that isn't a trusted proxy is ignored.
func TestSecurityMiddleware_NoHSTSFromUntrustedForwardedProto(t *testing.T) {
	app := newAppWithMiddleware(t, proxyModeConfig())

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "192.0.2.1:43210" // public (TEST-NET-1), not a trusted proxy
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if hsts := rec.Header().Get(echo.HeaderStrictTransportSecurity); hsts != "" {
		t.Fatalf("did not expect HSTS for a forwarded scheme from an untrusted address, got %q", hsts)
	}
}

// TestSecurityMiddleware_DirectModeIgnoresForwardedProto verifies that in the
// default direct mode, forwarded scheme headers are ignored even from a
// loopback peer: only the connection itself decides whether it's HTTPS.
func TestSecurityMiddleware_DirectModeIgnoresForwardedProto(t *testing.T) {
	app := newAppWithMiddleware(t, DefaultConfig())

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "127.0.0.1:43210"
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if hsts := rec.Header().Get(echo.HeaderStrictTransportSecurity); hsts != "" {
		t.Fatalf("did not expect HSTS from a forwarded scheme in direct mode, got %q", hsts)
	}
}

// TestSecurityMiddleware_NoHSTSOnPlainHTTP confirms the middleware suppresses
// HSTS on non-TLS requests. Sending HSTS over an unencrypted channel gives a
// network attacker the chance to strip it; Echo's Secure middleware
// short-circuits the header set when the request isn't secure end-to-end.
func TestSecurityMiddleware_NoHSTSOnPlainHTTP(t *testing.T) {
	app := newAppWithMiddleware(t, DefaultConfig())

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if hsts := rec.Header().Get(echo.HeaderStrictTransportSecurity); hsts != "" {
		t.Fatalf("did not expect HSTS on plain HTTP, got %q", hsts)
	}
}

// TestSecurityMiddleware_HSTSDisabledByConfig verifies the env knob: setting
// HSTSMaxAge to 0 must suppress the header even on TLS, so consumers who
// terminate TLS upstream and don't want the header leaking can opt out.
func TestSecurityMiddleware_HSTSDisabledByConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Security.HSTSMaxAge = 0
	app := newAppWithMiddleware(t, cfg)

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if hsts := rec.Header().Get(echo.HeaderStrictTransportSecurity); hsts != "" {
		t.Fatalf("expected HSTS suppressed when MaxAge=0, got %q", hsts)
	}
}

// TestSecurityMiddleware_AlwaysOnHeaders pins the trio that should ride along
// regardless of TLS status: X-XSS-Protection, X-Content-Type-Options,
// X-Frame-Options. These come from Echo's DefaultSecureConfig and don't
// depend on the request being secure.
func TestSecurityMiddleware_AlwaysOnHeaders(t *testing.T) {
	app := newAppWithMiddleware(t, DefaultConfig())

	app.httpServer.GET("/probe", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rec := httptest.NewRecorder()
	app.httpServer.ServeHTTP(rec, req)

	if got := rec.Header().Get(echo.HeaderXContentTypeOptions); got != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff, got %q", got)
	}
	if got := rec.Header().Get(echo.HeaderXFrameOptions); got == "" {
		t.Fatalf("expected X-Frame-Options to be set, got empty")
	}
}

// newAppWithMiddleware builds an Application, wires the middleware stack from
// the supplied config, and configures the HTTP server and the proxy-dependent
// IP and scheme extractors. Mirrors the production boot order
// (NewApplication → Configure → Initialize) but skips the file/env
// resolution since the test passes cfg in directly.
func newAppWithMiddleware(t *testing.T, cfg *Config) *Application {
	t.Helper()
	app := NewApplication(cfg)
	app.buildMiddleware()
	if err := app.configureHttpServer(); err != nil {
		t.Fatalf("configureHttpServer: %v", err)
	}
	if err := app.configureProxySupport(); err != nil {
		t.Fatalf("configureProxySupport: %v", err)
	}
	return app
}

// proxyModeConfig returns the default config switched to X-Forwarded-For
// proxy mode with the default trust set (loopback, link-local, private nets).
func proxyModeConfig() *Config {
	cfg := DefaultConfig()
	cfg.Proxy.UseDirectIP = false
	cfg.Proxy.UseXFFHeader = true
	return cfg
}

// assertDefaultHSTS checks the header carries the default one-year max-age
// and the includeSubdomains directive.
func assertDefaultHSTS(t *testing.T, hsts string) {
	t.Helper()
	if hsts == "" {
		t.Fatalf("expected Strict-Transport-Security, got none")
	}
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Fatalf("expected max-age=31536000 in HSTS header, got %q", hsts)
	}
	if !strings.Contains(hsts, "includeSubdomains") {
		t.Fatalf("expected includeSubdomains in HSTS header, got %q", hsts)
	}
}
