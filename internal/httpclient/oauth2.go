package httpclient

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"Postwoman/internal/model"
	"Postwoman/internal/variables"
)

// OAuth 2.0 helpers used by the auth editor's "Authorize" flow and by automatic
// token acquisition before a request is sent.
//
// Authorization Code and PKCE are interactive: the app builds the authorization
// URL, opens it in the system browser, and the user pastes the resulting code
// (or the full redirect URL) back into the app, which exchanges it here.

// pkcePair generates an RFC 7636 verifier/challenge pair (S256).
func pkcePair() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// StartOAuth2 builds the authorization URL for the authorization-code and PKCE
// flows. It returns an updated auth object containing the transient PKCE
// verifier and state so the caller can persist it on the request and complete
// the exchange later.
func (c *Client) StartOAuth2(oauth model.OAuth2Auth) (model.OAuth2Auth, string, error) {
	if oauth.GrantType == "client_credentials" || oauth.GrantType == "password" {
		return oauth, "", fmt.Errorf("the %s flow does not use an authorization URL", oauth.GrantType)
	}
	if strings.TrimSpace(oauth.AuthURL) == "" {
		return oauth, "", fmt.Errorf("an authorization URL is required")
	}
	authURL, err := url.Parse(oauth.AuthURL)
	if err != nil {
		return oauth, "", fmt.Errorf("authorization URL is invalid: %w", err)
	}
	if oauth.ClientID == "" {
		return oauth, "", fmt.Errorf("client ID is required")
	}
	redirect := oauth.RedirectURL
	if strings.TrimSpace(redirect) == "" {
		redirect = "http://localhost"
	}
	state, err := randomState()
	if err != nil {
		return oauth, "", err
	}
	oauth.State = state

	q := authURL.Query()
	q.Set("response_type", "code")
	q.Set("client_id", oauth.ClientID)
	q.Set("redirect_uri", redirect)
	q.Set("state", state)
	if oauth.Scope != "" {
		q.Set("scope", oauth.Scope)
	}

	usePKCE := oauth.PKCE || oauth.GrantType == "authorization_code_pkce"
	if usePKCE {
		verifier, challenge, err := pkcePair()
		if err != nil {
			return oauth, "", err
		}
		oauth.PKCE = true
		oauth.CodeVerifier = verifier
		q.Set("code_challenge", challenge)
		q.Set("code_challenge_method", "S256")
	}
	authURL.RawQuery = q.Encode()
	return oauth, authURL.String(), nil
}

// ExchangeOAuth2Code swaps an authorization code for tokens.
func (c *Client) ExchangeOAuth2Code(ctx context.Context, oauth model.OAuth2Auth, code string) (model.OAuth2Auth, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return oauth, fmt.Errorf("authorization code is required")
	}
	// Accept both a bare code and the full redirect URL pasted by the user.
	if strings.Contains(code, "://") || strings.Contains(code, "?code=") {
		if u, err := url.Parse(code); err == nil {
			if v := u.Query().Get("code"); v != "" {
				code = v
			}
			if u.Query().Get("error") != "" {
				return oauth, fmt.Errorf("authorization failed: %s %s", u.Query().Get("error"), u.Query().Get("error_description"))
			}
		}
	}
	if strings.TrimSpace(oauth.TokenURL) == "" {
		return oauth, fmt.Errorf("a token URL is required")
	}
	redirect := oauth.RedirectURL
	if strings.TrimSpace(redirect) == "" {
		redirect = "http://localhost"
	}
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", oauth.ClientID)
	form.Set("redirect_uri", redirect)
	if oauth.ClientSecret != "" {
		form.Set("client_secret", oauth.ClientSecret)
	}
	if oauth.CodeVerifier != "" {
		form.Set("code_verifier", oauth.CodeVerifier)
	}
	token, err := requestToken(ctx, c.tokenClient(), oauth.TokenURL, form)
	if err != nil {
		return oauth, err
	}
	oauth.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		oauth.RefreshToken = token.RefreshToken
	}
	oauth.TokenType = token.TokenType
	oauth.ExpiresAt = token.ExpiresAt
	oauth.CodeVerifier = ""
	oauth.State = ""
	return oauth, nil
}

// FetchOAuth2Token acquires a token for the client-credentials, password or
// refresh-token grants.
func (c *Client) FetchOAuth2Token(ctx context.Context, oauth model.OAuth2Auth) (model.OAuth2Auth, error) {
	if strings.TrimSpace(oauth.TokenURL) == "" {
		return oauth, fmt.Errorf("a token URL is required")
	}
	form := url.Values{}
	form.Set("client_id", oauth.ClientID)
	if oauth.ClientSecret != "" {
		form.Set("client_secret", oauth.ClientSecret)
	}
	switch oauth.GrantType {
	case "client_credentials":
		form.Set("grant_type", "client_credentials")
		if oauth.Scope != "" {
			form.Set("scope", oauth.Scope)
		}
	case "password":
		form.Set("grant_type", "password")
		form.Set("username", oauth.Username)
		form.Set("password", oauth.Password)
		if oauth.Scope != "" {
			form.Set("scope", oauth.Scope)
		}
	case "refresh_token":
		if strings.TrimSpace(oauth.RefreshToken) == "" {
			return oauth, fmt.Errorf("a refresh token is required")
		}
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", oauth.RefreshToken)
	default:
		return oauth, fmt.Errorf("the %s grant requires the interactive authorization flow", oauth.GrantType)
	}
	token, err := requestToken(ctx, c.tokenClient(), oauth.TokenURL, form)
	if err != nil {
		return oauth, err
	}
	oauth.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		oauth.RefreshToken = token.RefreshToken
	}
	oauth.TokenType = token.TokenType
	oauth.ExpiresAt = token.ExpiresAt
	return oauth, nil
}

type tokenResponse struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresAt    int64
}

func requestToken(ctx context.Context, client *http.Client, tokenURL string, form url.Values) (*tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request could not be built: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("token endpoint returned invalid JSON: %w", err)
	}
	if errValue, ok := payload["error"].(string); ok && errValue != "" {
		return nil, fmt.Errorf("token endpoint error: %s %s", errValue, stringOr(payload["error_description"]))
	}
	access := stringOr(payload["access_token"])
	if access == "" {
		return nil, fmt.Errorf("token endpoint did not return an access_token")
	}
	out := &tokenResponse{
		AccessToken:  access,
		RefreshToken: stringOr(payload["refresh_token"]),
		TokenType:    stringOr(payload["token_type"]),
	}
	if out.TokenType == "" {
		out.TokenType = "Bearer"
	}
	if v, ok := payload["expires_in"]; ok {
		var seconds float64
		switch n := v.(type) {
		case float64:
			seconds = n
		case string:
			seconds, _ = strconv.ParseFloat(n, 64)
		}
		if seconds > 0 {
			out.ExpiresAt = time.Now().Add(time.Duration(seconds) * time.Second).UnixMilli()
		}
	}
	return out, nil
}

func stringOr(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// tokenClient is a small HTTP client used for token endpoint calls.
func (c *Client) tokenClient() *http.Client {
	return &http.Client{Jar: c.jarHandle(), Timeout: 30 * time.Second}
}

// resolveOAuth2 returns a copy of the auth config with every field interpolated
// through the request's variable resolver.
func resolveOAuth2(oauth model.OAuth2Auth, resolver *variables.Resolver) model.OAuth2Auth {
	r := resolver
	if r == nil {
		r = variables.New()
	}
	out := oauth
	out.ClientID = r.Resolve(out.ClientID)
	out.ClientSecret = r.Resolve(out.ClientSecret)
	out.AuthURL = r.Resolve(out.AuthURL)
	out.TokenURL = r.Resolve(out.TokenURL)
	out.Scope = r.Resolve(out.Scope)
	out.RedirectURL = r.Resolve(out.RedirectURL)
	out.Username = r.Resolve(out.Username)
	out.Password = r.Resolve(out.Password)
	out.AccessToken = r.Resolve(out.AccessToken)
	out.RefreshToken = r.Resolve(out.RefreshToken)
	return out
}

// ResolveOAuth2 exposes field interpolation to the application layer so the
// auth editor can preview/start flows with environment variables applied.
func ResolveOAuth2(oauth model.OAuth2Auth, resolver *variables.Resolver) model.OAuth2Auth {
	return resolveOAuth2(oauth, resolver)
}
