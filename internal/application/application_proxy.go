// SPDX-License-Identifier: Apache-2.0

package application

import (
	"net"
	"strings"

	"github.com/PandasWhoCode/pr-mirror-app/internal/logging"
	"github.com/labstack/echo/v5"
)

// configureProxySupport wires the IP and scheme extractors for both servers
// based on the proxy config. Mutually-exclusive-flag validation is enforced
// earlier at Configure time via ProxyConfig.Validate, so by the time we reach
// here at most one of the three flags is true.
//
// The scheme extractor decides whether a request counts as HTTPS, which
// drives the HSTS header and the HTTPS redirect (HTTPSRedirectWithConfig
// reads c.Scheme()). Forwarded scheme headers (X-Forwarded-Proto and
// friends) are honored only in proxy mode and only from the same addresses
// trusted for the client IP; in direct mode only the connection itself
// counts. Trusting those headers from any client let a caller fake HTTPS
// (GO-2026-6879).
func (app *Application) configureProxySupport() error {
	pc := app.config.Proxy
	if pc == nil {
		return nil
	}

	var (
		ipExtractor     echo.IPExtractor
		schemeExtractor echo.SchemeExtractor
	)

	switch {
	case pc.UseXFFHeader:
		trust := app.resolveProxyTrustOptions()
		ipExtractor = echo.ExtractIPFromXFFHeader(trust...)
		schemeExtractor = echo.ExtractSchemeFromHeaders(trust...)
	case pc.UseXRealIPHeader:
		trust := app.resolveProxyTrustOptions()
		ipExtractor = echo.ExtractIPFromRealIPHeader(trust...)
		schemeExtractor = echo.ExtractSchemeFromHeaders(trust...)
	default:
		ipExtractor = echo.ExtractIPDirect()
		schemeExtractor = echo.ExtractSchemeDirect()
	}

	app.httpServer.IPExtractor = ipExtractor
	app.tlsServer.IPExtractor = ipExtractor

	app.httpServer.SchemeExtractor = schemeExtractor
	// Every request on the TLS listener has req.TLS set, and every
	// extractor reports "https" for those, so this assignment doesn't
	// change behaviour today. It keeps both servers on the same policy if
	// that ever stops being true.
	app.tlsServer.SchemeExtractor = schemeExtractor
	return nil
}

// resolveProxyTrustOptions builds the slice of echo.TrustOption values
// passed to ExtractIPFromRealIPHeader / ExtractIPFromXFFHeader. Configured
// CIDRs are validated; invalid entries are dropped with a warn-level log
// rather than refused outright (we already validate at Configure time, so
// anything bad here is a runtime regression worth surfacing). Loopback,
// private, and link-local nets are always trusted.
func (app *Application) resolveProxyTrustOptions() []echo.TrustOption {
	pc := app.config.Proxy
	if pc == nil {
		return []echo.TrustOption{}
	}

	var invalidRanges = make([]string, 0)
	var opts = make([]echo.TrustOption, 0)
	for _, trustedRange := range pc.TrustedIPRanges {
		trustedRange = strings.TrimSpace(trustedRange)
		if trustedRange == "" {
			continue
		}

		if _, network, err := net.ParseCIDR(trustedRange); err == nil && network != nil {
			opts = append(opts, echo.TrustIPRange(network))
		} else {
			invalidRanges = append(invalidRanges, trustedRange)
		}
	}

	if len(invalidRanges) > 0 {
		logging.Daemon.
			Warn().
			Strs("cidrAddresses", invalidRanges).
			Msg("proxy config contains invalid CIDR addresses")
	}

	opts = append(opts,
		echo.TrustPrivateNet(true),
		echo.TrustLoopback(true),
		echo.TrustLinkLocal(true))

	return opts
}
