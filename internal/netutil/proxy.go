// Package netutil holds small networking helpers shared by the API client and
// the AI service. Keeping proxy resolution in one place means the "Network &
// proxy" settings apply identically to API requests and AI provider requests.
package netutil

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"Postwoman/internal/model"
)

// supportedProtocols are the proxy schemes the transports understand.
var supportedProtocols = map[string]bool{
	"http":    true,
	"https":   true,
	"socks5":  true,
	"socks5h": true,
}

// NormalizeProxy trims a proxy configuration and validates it. A nil or
// host-less proxy resolves to (nil, nil) so callers can treat "no proxy" as a
// valid outcome rather than an error.
func NormalizeProxy(p *model.Proxy) (*model.Proxy, error) {
	if p == nil {
		return nil, nil
	}
	out := *p
	out.Host = strings.TrimSpace(out.Host)
	out.Protocol = strings.ToLower(strings.TrimSpace(out.Protocol))
	if out.Protocol == "" {
		out.Protocol = "http"
	}
	if out.Host == "" {
		return nil, nil
	}
	if !supportedProtocols[out.Protocol] {
		return nil, fmt.Errorf("unsupported proxy protocol %q (use http, https or socks5)", out.Protocol)
	}
	if out.Port <= 0 || out.Port > 65535 {
		return nil, fmt.Errorf("proxy port must be between 1 and 65535")
	}
	out.Username = strings.TrimSpace(out.Username)
	bypass := make([]string, 0, len(out.Bypass))
	for _, pattern := range out.Bypass {
		if trimmed := strings.TrimSpace(pattern); trimmed != "" {
			bypass = append(bypass, trimmed)
		}
	}
	out.Bypass = bypass
	return &out, nil
}

// ProxyURL builds the *url.URL a transport dials for a proxy-hop. It returns
// (nil, nil) when the configuration does not describe a proxy.
func ProxyURL(p *model.Proxy) (*url.URL, error) {
	normalized, err := NormalizeProxy(p)
	if err != nil {
		return nil, err
	}
	if normalized == nil {
		return nil, nil
	}
	proxyURL := &url.URL{
		Scheme: normalized.Protocol,
		Host:   net.JoinHostPort(normalized.Host, fmt.Sprintf("%d", normalized.Port)),
	}
	if normalized.Username != "" {
		proxyURL.User = url.UserPassword(normalized.Username, normalized.Password)
	}
	return proxyURL, nil
}

// ProxyFunc returns an http.Transport.Proxy function that routes requests
// through the configured proxy while honouring the bypass list. A nil proxy
// falls back to the environment (HTTP_PROXY/HTTPS_PROXY/NO_PROXY).
func ProxyFunc(p *model.Proxy) (func(*http.Request) (*url.URL, error), error) {
	proxyURL, err := ProxyURL(p)
	if err != nil {
		return nil, err
	}
	if proxyURL == nil {
		return http.ProxyFromEnvironment, nil
	}
	bypass := p.Bypass
	return func(req *http.Request) (*url.URL, error) {
		for _, pattern := range bypass {
			if HostMatches(req.URL.Hostname(), pattern) {
				return nil, nil
			}
		}
		return proxyURL, nil
	}, nil
}

// HostMatches reports whether host matches a bypass pattern. Patterns use "*"
// as a wildcard and otherwise match the whole host (or any subdomain when the
// pattern has no scheme), mirroring Postman's proxy bypass behaviour.
func HostMatches(host, pattern string) bool {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return false
	}
	host = strings.ToLower(strings.TrimSpace(host))
	pattern = strings.ToLower(pattern)
	// Treat "example.com" as "example.com or *.example.com".
	target := "^(?:" + wildcardExpr(pattern) + ")$"
	if !strings.Contains(pattern, "*") {
		target = "^(?:" + wildcardExpr(pattern) + "|" + wildcardExpr("*."+pattern) + ")$"
	}
	matched, _ := regexp.MatchString(target, host)
	return matched
}

func wildcardExpr(pattern string) string {
	pattern = strings.ReplaceAll(pattern, ".", `\.`)
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	return pattern
}

// Probe dials the proxy host and port to confirm it accepts TCP connections.
// It is used by the "Test proxy" button: a successful dial proves the desktop
// app can reach the proxy without leaking a real request to a third party.
func Probe(p *model.Proxy, timeout time.Duration) error {
	normalized, err := NormalizeProxy(p)
	if err != nil {
		return err
	}
	if normalized == nil {
		return fmt.Errorf("set a proxy host and port first")
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	address := net.JoinHostPort(normalized.Host, fmt.Sprintf("%d", normalized.Port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("cannot reach proxy %s: %w", address, err)
	}
	return conn.Close()
}
