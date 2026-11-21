package httpclient

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"time"

	"Postwoman/internal/model"
)

// recordOrigin remembers an origin we have communicated with so the cookie
// manager can enumerate its cookies later (cookiejar has no enumeration API).
func (c *Client) recordOrigin(u *url.URL) {
	if u == nil || u.Host == "" {
		return
	}
	origin := strings.ToLower(u.Scheme + "://" + u.Host)
	c.mu.Lock()
	if c.origins == nil {
		c.origins = map[string]struct{}{}
	}
	c.origins[origin] = struct{}{}
	c.mu.Unlock()
}

// ListCookies returns every cookie in the jar, grouped by origin.
func (c *Client) ListCookies() []model.StoredCookie {
	c.mu.RLock()
	origins := make([]string, 0, len(c.origins))
	for origin := range c.origins {
		origins = append(origins, origin)
	}
	c.mu.RUnlock()
	sort.Strings(origins)

	out := []model.StoredCookie{}
	seen := map[string]bool{}
	jar := c.jarHandle()
	for _, origin := range origins {
		u, err := url.Parse(origin)
		if err != nil {
			continue
		}
		for _, ck := range jar.Cookies(u) {
			key := strings.ToLower(ck.Name + "|" + ck.Domain + "|" + ck.Path)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, model.StoredCookie{URL: origin, Cookie: toCookie(ck)})
		}
	}
	return out
}

// SetCookie stores (or replaces) a cookie for the supplied URL.
func (c *Client) SetCookie(rawURL string, cookie model.Cookie) error {
	u, err := parseCookieURL(rawURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cookie.Name) == "" {
		return fmt.Errorf("cookie name is required")
	}
	ck := &http.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HTTPOnly,
	}
	if ck.Path == "" {
		ck.Path = "/"
	}
	if cookie.Expires > 0 {
		ck.Expires = time.UnixMilli(cookie.Expires)
	}
	if cookie.MaxAge != 0 {
		ck.MaxAge = cookie.MaxAge
	}
	switch strings.ToLower(cookie.SameSite) {
	case "strict":
		ck.SameSite = http.SameSiteStrictMode
	case "none":
		ck.SameSite = http.SameSiteNoneMode
	case "lax":
		ck.SameSite = http.SameSiteLaxMode
	}
	c.jarHandle().SetCookies(u, []*http.Cookie{ck})
	c.recordOrigin(u)
	return nil
}

// DeleteCookie removes a cookie by name for a URL. Existing cookies are
// expired individually so cookies with non-root paths are removed too.
func (c *Client) DeleteCookie(rawURL, name string) error {
	u, err := parseCookieURL(rawURL)
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("cookie name is required")
	}
	jar := c.jarHandle()
	paths := map[string]bool{}
	for _, existing := range jar.Cookies(u) {
		if existing.Name == name {
			paths[existing.Path] = true
		}
	}
	if len(paths) == 0 {
		paths["/"] = true
	}
	deletions := make([]*http.Cookie, 0, len(paths))
	for path := range paths {
		if path == "" {
			path = "/"
		}
		deletions = append(deletions, &http.Cookie{
			Name: name, Value: "", Path: path, MaxAge: -1, Expires: time.Unix(0, 0),
		})
	}
	c.jarHandle().SetCookies(u, deletions)
	return nil
}

// ClearCookies discards the whole jar and the remembered origins.
func (c *Client) ClearCookies() error {
	jar, err := newJar()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.jar = jar
	c.origins = map[string]struct{}{}
	c.mu.Unlock()
	return nil
}

// RestoreCookies loads persisted cookies (used at application start-up).
func (c *Client) RestoreCookies(cookies []model.StoredCookie) {
	for _, sc := range cookies {
		_ = c.SetCookie(sc.URL, sc.Cookie)
	}
}

func parseCookieURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("a URL is required for a cookie")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("cookies can only be stored for http/https URLs")
	}
	// Cookies are scoped to the origin; strip path/query so the jar uses the
	// correct host scope.
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

// newJar is a small seam so tests can force a cookiejar failure.
func newJar() (*cookiejar.Jar, error) {
	return cookiejar.New(nil)
}
