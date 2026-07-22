package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"kiro-go/config"
	"kiro-go/logger"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	kiroSocialDefaultAuthEndpoint = "https://prod.us-east-1.auth.desktop.kiro.dev"
	kiroSocialCallbackPortsEnv    = "KIRO_SOCIAL_CALLBACK_PORTS"
	kiroSocialCallbackBindEnv     = "KIRO_SOCIAL_CALLBACK_BIND"
	kiroSocialLoginTimeout        = 10 * time.Minute
)

// KiroSocialLoginOptions starts the Kiro hosted Google/GitHub flow. Email is
// the expected account; Kiro's hosted page still decides the actual account
// from the browser session, so completion verifies the returned token against it
// when the access token exposes an email claim.
type KiroSocialLoginOptions struct {
	Email        string
	ProxyURL     string
	AuthEndpoint string
}

// KiroSocialSession is an isolated Google/GitHub login session. It deliberately
// does not share state with the Enterprise/Microsoft SSO state machine.
type KiroSocialSession struct {
	ID           string
	State        string
	CodeVerifier string
	RedirectURI  string
	CallbackPort string
	Email        string
	ProxyURL     string
	AuthEndpoint string
	ExpiresAt    time.Time

	srv       *http.Server
	resultCh  chan kiroSocialCallback
	once      sync.Once
	closeOnce sync.Once
	timer     *time.Timer
}

type KiroSocialManualCallback struct {
	Code        string
	State       string
	LoginOption string
	Path        string
}

type KiroSocialResult struct {
	AccessToken  string
	RefreshToken string
	AuthMethod   string
	Provider     string
	ProfileArn   string
	Region       string
	ExpiresIn    int
	ExpiresAt    int64
	Email        string
}

type kiroSocialCallback struct {
	code        string
	state       string
	loginOption string
	path        string
	err         error
}

var (
	kiroSocialSessions   = make(map[string]*KiroSocialSession)
	kiroSocialSessionsMu sync.RWMutex
)

func StartKiroSocialLogin(opts KiroSocialLoginOptions) (*KiroSocialSession, string, error) {
	authEndpoint := strings.TrimRight(strings.TrimSpace(opts.AuthEndpoint), "/")
	if authEndpoint == "" {
		authEndpoint = kiroSocialDefaultAuthEndpoint
	}
	if err := validateKiroSocialAuthEndpoint(authEndpoint); err != nil {
		return nil, "", err
	}

	proxyURL := strings.TrimSpace(opts.ProxyURL)
	if proxyURL == "" {
		proxyURL = config.GetProxyURL()
	}
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)
	state := uuid.New().String()

	session := &KiroSocialSession{
		ID:           uuid.New().String(),
		State:        state,
		CodeVerifier: codeVerifier,
		Email:        strings.TrimSpace(opts.Email),
		ProxyURL:     proxyURL,
		AuthEndpoint: authEndpoint,
		ExpiresAt:    time.Now().Add(kiroSocialLoginTimeout),
		resultCh:     make(chan kiroSocialCallback, 1),
	}
	if err := session.startListener(kiroSocialCallbackPorts()); err != nil {
		return nil, "", err
	}
	session.RedirectURI = "http://" + net.JoinHostPort("127.0.0.1", session.redirectPort())
	portalURL := buildKiroSocialPortalURL(session.State, codeChallenge, session.RedirectURI)

	kiroSocialSessionsMu.Lock()
	kiroSocialSessions[session.ID] = session
	kiroSocialSessionsMu.Unlock()

	session.timer = time.AfterFunc(kiroSocialLoginTimeout, func() {
		session.close()
		removeKiroSocialSession(session.ID)
	})
	return session, portalURL, nil
}

func buildKiroSocialPortalURL(state, challenge, redirectURI string) string {
	params := url.Values{}
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("redirect_uri", strings.TrimSpace(redirectURI))
	params.Set("redirect_from", kiroRedirectFrom)
	return kiroSignInBaseURL + "?" + params.Encode()
}

func validateKiroSocialAuthEndpoint(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid Kiro social auth endpoint: %w", err)
	}
	if u.Scheme != "https" || u.Hostname() == "" {
		return fmt.Errorf("Kiro social auth endpoint must be an https URL")
	}
	host := strings.ToLower(u.Hostname())
	if host == "prod.us-east-1.auth.desktop.kiro.dev" || strings.HasSuffix(host, ".auth.desktop.kiro.dev") {
		return nil
	}
	return fmt.Errorf("Kiro social auth endpoint host %q is not supported", host)
}

func kiroSocialCallbackPorts() []string {
	raw := strings.TrimSpace(os.Getenv(kiroSocialCallbackPortsEnv))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(kiroCallbackPortsEnv))
	}
	if raw == "" {
		return []string{kiroRedirectPort}
	}
	seen := map[string]bool{}
	ports := []string{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n'
	}) {
		port := strings.TrimSpace(part)
		if port == "" || seen[port] {
			continue
		}
		if _, err := net.LookupPort("tcp", port); err != nil {
			logger.Debugf("[KiroSocial] callback port %q ignored: %v", port, err)
			continue
		}
		seen[port] = true
		ports = append(ports, port)
	}
	if len(ports) == 0 {
		return []string{kiroRedirectPort}
	}
	return ports
}

func kiroSocialCallbackBindAddrs(port string) []string {
	if strings.TrimSpace(port) == "" {
		port = kiroRedirectPort
	}
	if bind := strings.TrimSpace(os.Getenv(kiroSocialCallbackBindEnv)); bind != "" {
		return []string{net.JoinHostPort(bind, port)}
	}
	if bind := strings.TrimSpace(os.Getenv("KIRO_SSO_CALLBACK_BIND")); bind != "" {
		return []string{net.JoinHostPort(bind, port)}
	}
	return []string{"127.0.0.1:" + port, "[::1]:" + port}
}

func (s *KiroSocialSession) startListener(candidatePorts []string) error {
	var lastErr error
	for _, port := range candidatePorts {
		addrs := kiroSocialCallbackBindAddrs(port)
		ln, err := net.Listen("tcp", addrs[0])
		if err != nil {
			lastErr = fmt.Errorf("cannot bind %s for the Social callback (is the port already in use?): %w", addrs[0], err)
			continue
		}
		s.CallbackPort = port
		mux := http.NewServeMux()
		mux.HandleFunc("/", s.handleCallback)
		s.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}

		serve := func(l net.Listener) {
			go func() {
				if errServe := s.srv.Serve(l); errServe != nil && errServe != http.ErrServerClosed {
					logger.Debugf("[KiroSocial] callback listener (%s) stopped: %v", l.Addr(), errServe)
				}
			}()
		}
		serve(ln)
		for _, addr := range addrs[1:] {
			if extra, errExtra := net.Listen("tcp", addr); errExtra == nil {
				serve(extra)
			} else {
				logger.Debugf("[KiroSocial] secondary callback bind %s skipped: %v", addr, errExtra)
			}
		}
		return nil
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no Social callback ports configured")
}

func (s *KiroSocialSession) redirectPort() string {
	if v := strings.TrimSpace(s.CallbackPort); v != "" {
		return v
	}
	if strings.TrimSpace(s.RedirectURI) != "" {
		if u, err := url.Parse(s.RedirectURI); err == nil {
			if port := strings.TrimSpace(u.Port()); port != "" {
				return port
			}
		}
	}
	return kiroRedirectPort
}

func (s *KiroSocialSession) close() {
	s.closeOnce.Do(func() {
		if s.timer != nil {
			s.timer.Stop()
		}
		if s.srv != nil {
			_ = s.srv.Close()
		}
	})
}

func (s *KiroSocialSession) deliver(capture kiroSocialCallback) {
	s.once.Do(func() { s.resultCh <- capture })
}

func (s *KiroSocialSession) handleCallback(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	path := req.URL.Path
	if path != kiroOAuthCallbackPath && path != "/signin/callback" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	q := req.URL.Query()
	code := strings.TrimSpace(q.Get("code"))
	errParam := strings.TrimSpace(q.Get("error"))
	state := strings.TrimSpace(q.Get("state"))
	if code == "" && errParam == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if s.State == "" || state != s.State {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if errParam != "" {
		desc := strings.TrimSpace(q.Get("error_description"))
		writeKiroCallbackPage(w, false)
		s.deliver(kiroSocialCallback{err: fmt.Errorf("Social authorization error: %s %s", errParam, desc)})
		return
	}
	writeKiroCallbackPage(w, true)
	s.deliver(kiroSocialCallback{
		code:        code,
		state:       state,
		loginOption: strings.TrimSpace(q.Get("login_option")),
		path:        path,
	})
}

func PollKiroSocialLogin(sessionID string) (*KiroSocialResult, string, error) {
	kiroSocialSessionsMu.RLock()
	session, ok := kiroSocialSessions[sessionID]
	kiroSocialSessionsMu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("session not found or expired")
	}

	select {
	case cb := <-session.resultCh:
		session.close()
		removeKiroSocialSession(sessionID)
		if cb.err != nil {
			return nil, "", cb.err
		}
		return session.exchange(cb)
	default:
		if time.Now().After(session.ExpiresAt) {
			session.close()
			removeKiroSocialSession(sessionID)
			return nil, "expired", nil
		}
		return nil, "pending", nil
	}
}

func CompleteKiroSocialLogin(sessionID string, cb KiroSocialManualCallback) (*KiroSocialResult, string, error) {
	kiroSocialSessionsMu.RLock()
	session, ok := kiroSocialSessions[sessionID]
	kiroSocialSessionsMu.RUnlock()
	if !ok {
		return nil, "", fmt.Errorf("session not found or expired")
	}
	if time.Now().After(session.ExpiresAt) {
		session.close()
		removeKiroSocialSession(sessionID)
		return nil, "expired", nil
	}
	code := strings.TrimSpace(cb.Code)
	if code == "" {
		return nil, "", fmt.Errorf("missing OAuth authorization code")
	}
	state := strings.TrimSpace(cb.State)
	if session.State == "" || state != session.State {
		return nil, "", fmt.Errorf("OAuth state mismatch, please restart login")
	}
	path := strings.TrimSpace(cb.Path)
	if path == "" {
		path = kiroOAuthCallbackPath
	}
	if path != kiroOAuthCallbackPath && path != "/signin/callback" {
		return nil, "", fmt.Errorf("unsupported Kiro callback path: %s", path)
	}

	session.close()
	removeKiroSocialSession(sessionID)
	return session.exchange(kiroSocialCallback{
		code:        code,
		state:       state,
		loginOption: strings.TrimSpace(cb.LoginOption),
		path:        path,
	})
}

func CancelKiroSocialLogin(sessionID string) {
	kiroSocialSessionsMu.RLock()
	session, ok := kiroSocialSessions[sessionID]
	kiroSocialSessionsMu.RUnlock()
	if !ok {
		return
	}
	session.close()
	removeKiroSocialSession(sessionID)
}

func (s *KiroSocialSession) exchange(cb kiroSocialCallback) (*KiroSocialResult, string, error) {
	redirectURI := s.fullRedirectURI(cb.path, cb.loginOption)
	token, err := exchangeKiroSocialCode(GetAuthClientForProxy(s.ProxyURL), s.AuthEndpoint, cb.code, s.CodeVerifier, redirectURI)
	if err != nil {
		return nil, "", fmt.Errorf("Social token exchange failed: %w", err)
	}
	email, err := resolveKiroSocialEmail(s.Email, ExtractEmailFromJWT(token.AccessToken))
	if err != nil {
		return nil, "", err
	}
	provider := kiroSocialProviderFromLoginOption(cb.loginOption)
	if provider == "" {
		provider = "Google/GitHub"
	}
	expiresAt := token.ExpiresAtUnix()
	if expiresAt == 0 && token.ExpiresIn > 0 {
		expiresAt = time.Now().Unix() + int64(token.ExpiresIn)
	}
	region := kiroSocialRegionFromProfileArn(token.ProfileArn)
	if region == "" {
		region = "us-east-1"
	}
	return &KiroSocialResult{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		AuthMethod:   "social",
		Provider:     provider,
		ProfileArn:   token.ProfileArn,
		Region:       region,
		ExpiresIn:    token.ExpiresIn,
		ExpiresAt:    expiresAt,
		Email:        email,
	}, "completed", nil
}

func resolveKiroSocialEmail(expectedEmail, actualEmail string) (string, error) {
	expected := strings.TrimSpace(expectedEmail)
	actual := strings.TrimSpace(actualEmail)
	if expected != "" && actual != "" && !strings.EqualFold(expected, actual) {
		return "", fmt.Errorf("Kiro Social account mismatch: expected %s, but Kiro returned %s. Sign out of app.kiro.dev in the browser profile used for this login, or open the sign-in link in a fresh guest/incognito profile, then retry", expected, actual)
	}
	if actual != "" {
		return actual, nil
	}
	return expected, nil
}

func (s *KiroSocialSession) fullRedirectURI(path, loginOption string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		cleanPath = kiroOAuthCallbackPath
	}
	full := strings.TrimRight(s.RedirectURI, "/") + cleanPath
	if option := strings.TrimSpace(loginOption); option != "" {
		full += "?login_option=" + url.QueryEscape(option)
	}
	return full
}

type kiroSocialTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    string `json:"expiresAt"`
	ExpiresIn    int    `json:"expiresIn"`
	ProfileArn   string `json:"profileArn"`
}

func (r kiroSocialTokenResponse) ExpiresAtUnix() int64 {
	if strings.TrimSpace(r.ExpiresAt) == "" {
		return 0
	}
	if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(r.ExpiresAt)); err == nil {
		return ts.Unix()
	}
	return 0
}

func exchangeKiroSocialCode(client *http.Client, authEndpoint, code, codeVerifier, redirectURI string) (kiroSocialTokenResponse, error) {
	payload := map[string]interface{}{
		"code":            strings.TrimSpace(code),
		"code_verifier":   codeVerifier,
		"redirect_uri":    redirectURI,
		"invitation_code": nil,
	}
	body, _ := json.Marshal(payload)
	tokenURL := strings.TrimRight(authEndpoint, "/") + "/oauth/token"
	req, err := http.NewRequest(http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return kiroSocialTokenResponse{}, fmt.Errorf("failed to build Social token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "KiroIDE-"+config.GetKiroClientConfig().KiroVersion)
	if u, err := url.Parse(authEndpoint); err == nil && u.Host != "" {
		req.Host = u.Host
		req.Header.Set("host", u.Host)
	}

	resp, err := client.Do(req)
	if err != nil {
		return kiroSocialTokenResponse{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	var out kiroSocialTokenResponse
	_ = json.Unmarshal(respBody, &out)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || strings.TrimSpace(out.AccessToken) == "" {
		return kiroSocialTokenResponse{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
	if strings.TrimSpace(out.RefreshToken) == "" {
		return kiroSocialTokenResponse{}, fmt.Errorf("token response did not include refreshToken")
	}
	return out, nil
}

func kiroSocialRegionFromProfileArn(profileArn string) string {
	parts := strings.Split(strings.TrimSpace(profileArn), ":")
	if len(parts) >= 4 && strings.TrimSpace(parts[3]) != "" {
		return strings.TrimSpace(parts[3])
	}
	return ""
}

func removeKiroSocialSession(sessionID string) {
	kiroSocialSessionsMu.Lock()
	delete(kiroSocialSessions, sessionID)
	kiroSocialSessionsMu.Unlock()
}
