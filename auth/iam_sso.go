package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type IamSsoSession struct {
	ID              string
	ClientID        string
	ClientSecret    string
	DeviceCode      string
	UserCode        string
	VerificationUri string
	Interval        int
	Region          string
	AuthRegion      string
	StartUrl        string
	ExpiresAt       time.Time
}

var (
	sessions   = make(map[string]*IamSsoSession)
	sessionsMu sync.RWMutex
)

var scopes = []string{
	"codewhisperer:completions",
	"codewhisperer:analysis",
	"codewhisperer:conversations",
	"codewhisperer:transformations",
	"codewhisperer:taskassist",
}

var (
	portalOIDCEndpointPattern = regexp.MustCompile(`"oidcApiEndpoint"\s*:\s*"https://oidc\.([a-z0-9-]+)\.amazonaws\.com"`)
	portalRegionPattern       = regexp.MustCompile(`"region"\s*:\s*"([a-z0-9-]+)"`)
	appAWSPortalHostPattern   = regexp.MustCompile(`(^|\.)portal\.([a-z0-9-]+)\.app\.aws$`)
)

// StartIamSsoLogin starts an IAM Identity Center login using the device authorization
// flow, matching the AWS SSO OIDC flow used by KiroRS and the AWS CLI.
func StartIamSsoLogin(startUrl, region string) (*IamSsoSession, error) {
	profileRegion := strings.ToLower(strings.TrimSpace(region))
	if profileRegion == "" {
		profileRegion = "us-east-1"
	}
	normalizedStartURL, err := normalizeIamSsoStartURL(startUrl)
	if err != nil {
		return nil, err
	}
	startUrl = normalizedStartURL

	authRegion := profileRegion
	if detectedRegion, err := discoverIamSsoRegion(startUrl); err == nil && detectedRegion != "" {
		authRegion = strings.ToLower(strings.TrimSpace(detectedRegion))
	}

	oidcBase := fmt.Sprintf("https://oidc.%s.amazonaws.com", authRegion)

	clientID, clientSecret, err := registerDeviceClient(oidcBase, startUrl)
	if err != nil {
		return nil, fmt.Errorf("注册客户端失败: %w", err)
	}

	device, err := startIamDeviceAuth(oidcBase, clientID, clientSecret, startUrl)
	if err != nil {
		return nil, fmt.Errorf("设备授权失败: %w", err)
	}

	if device.Interval == 0 {
		device.Interval = 5
	}
	if device.ExpiresIn == 0 {
		device.ExpiresIn = 600
	}

	verificationUri := device.VerificationUriComplete
	if verificationUri == "" {
		verificationUri = device.VerificationUri
	}

	session := &IamSsoSession{
		ID:              uuid.New().String(),
		ClientID:        clientID,
		ClientSecret:    clientSecret,
		DeviceCode:      device.DeviceCode,
		UserCode:        device.UserCode,
		VerificationUri: verificationUri,
		Interval:        device.Interval,
		Region:          profileRegion,
		AuthRegion:      authRegion,
		StartUrl:        startUrl,
		ExpiresAt:       time.Now().Add(time.Duration(device.ExpiresIn) * time.Second),
	}

	sessionsMu.Lock()
	sessions[session.ID] = session
	sessionsMu.Unlock()

	go cleanupExpiredSessions()

	return session, nil
}

func discoverIamSsoRegion(startUrl string) (string, error) {
	normalizedStartURL, err := normalizeIamSsoStartURL(startUrl)
	if err != nil {
		return "", err
	}
	startUrl = normalizedStartURL

	if region, ok := regionFromIamSsoPortalURL(startUrl); ok {
		return region, nil
	}

	if !strings.Contains(startUrl, ".awsapps.com/start") {
		return "", fmt.Errorf("not an AWS access portal URL")
	}

	req, err := http.NewRequest("GET", startUrl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("portal returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	html := string(body)

	if match := portalOIDCEndpointPattern.FindStringSubmatch(html); len(match) == 2 {
		return match[1], nil
	}
	if match := portalRegionPattern.FindStringSubmatch(html); len(match) == 2 {
		return match[1], nil
	}
	return "", fmt.Errorf("portal region not found")
}

func normalizeIamSsoStartURL(raw string) (string, error) {
	startURL := strings.TrimSpace(raw)
	if startURL == "" {
		return "", fmt.Errorf("startUrl is required")
	}

	if strings.HasPrefix(startURL, "//http://") || strings.HasPrefix(startURL, "//https://") {
		startURL = strings.TrimLeft(startURL, "/")
	}
	if !strings.HasPrefix(startURL, "http://") && !strings.HasPrefix(startURL, "https://") {
		startURL = "https://" + strings.TrimLeft(startURL, "/")
	}

	u, err := url.Parse(startURL)
	if err != nil {
		return "", fmt.Errorf("invalid startUrl: %w", err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("startUrl must use https")
	}
	if strings.TrimSpace(u.Hostname()) == "" {
		return "", fmt.Errorf("startUrl host is required")
	}

	u.RawQuery = ""
	u.Fragment = ""
	u.Host = strings.ToLower(strings.TrimSpace(u.Host))
	host := strings.ToLower(u.Hostname())

	if strings.HasSuffix(host, ".awsapps.com") {
		path := "/" + strings.Trim(strings.TrimSpace(u.Path), "/")
		if path == "/" {
			path = "/start"
		}
		u.Path = path
	} else if _, ok := regionFromAppAWSPortalHost(host); ok {
		u.Path = "/"
	} else {
		u.Path = "/" + strings.Trim(strings.TrimSpace(u.Path), "/")
		if u.Path == "/" {
			u.Path = ""
		}
	}

	return u.String(), nil
}

func regionFromIamSsoPortalURL(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", false
	}
	return regionFromAppAWSPortalHost(strings.ToLower(u.Hostname()))
}

func regionFromAppAWSPortalHost(host string) (string, bool) {
	match := appAWSPortalHostPattern.FindStringSubmatch(strings.ToLower(strings.TrimSpace(host)))
	if len(match) != 3 {
		return "", false
	}
	return match[2], true
}

// PollIamSsoAuth polls IAM Identity Center for the device authorization result.
func PollIamSsoAuth(sessionID string) (accessToken, refreshToken, clientID, clientSecret, region, authRegion, startUrl string, expiresIn int, status string, err error) {
	sessionsMu.RLock()
	session, ok := sessions[sessionID]
	sessionsMu.RUnlock()

	if !ok {
		return "", "", "", "", "", "", "", 0, "", fmt.Errorf("session not found or expired")
	}

	if time.Now().After(session.ExpiresAt) {
		sessionsMu.Lock()
		delete(sessions, sessionID)
		sessionsMu.Unlock()
		return "", "", "", "", "", "", "", 0, "", fmt.Errorf("authorization expired")
	}

	tokenRegion := session.AuthRegion
	if tokenRegion == "" {
		tokenRegion = session.Region
	}
	oidcBase := fmt.Sprintf("https://oidc.%s.amazonaws.com", tokenRegion)
	payload := map[string]string{
		"clientId":     session.ClientID,
		"clientSecret": session.ClientSecret,
		"grantType":    "urn:ietf:params:oauth:grant-type:device_code",
		"deviceCode":   session.DeviceCode,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", oidcBase+"/token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return "", "", "", "", "", "", "", 0, "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var result struct {
			AccessToken  string `json:"accessToken"`
			RefreshToken string `json:"refreshToken"`
			ExpiresIn    int    `json:"expiresIn"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return "", "", "", "", "", "", "", 0, "", fmt.Errorf("parse token response failed: %w", err)
		}

		sessionsMu.Lock()
		delete(sessions, sessionID)
		sessionsMu.Unlock()

		return result.AccessToken, result.RefreshToken, session.ClientID, session.ClientSecret, session.Region, tokenRegion, session.StartUrl, result.ExpiresIn, "completed", nil
	}

	if resp.StatusCode == 400 {
		var errResult struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResult)

		switch errResult.Error {
		case "authorization_pending":
			return "", "", "", "", "", "", "", 0, "pending", nil
		case "slow_down":
			sessionsMu.Lock()
			if s, ok := sessions[sessionID]; ok {
				s.Interval += 5
			}
			sessionsMu.Unlock()
			return "", "", "", "", "", "", "", 0, "slow_down", nil
		case "expired_token":
			sessionsMu.Lock()
			delete(sessions, sessionID)
			sessionsMu.Unlock()
			return "", "", "", "", "", "", "", 0, "", fmt.Errorf("device code expired")
		case "access_denied":
			sessionsMu.Lock()
			delete(sessions, sessionID)
			sessionsMu.Unlock()
			return "", "", "", "", "", "", "", 0, "", fmt.Errorf("user denied authorization")
		default:
			return "", "", "", "", "", "", "", 0, "", fmt.Errorf("authorization error: %s", errResult.Error)
		}
	}

	respBody, _ := io.ReadAll(resp.Body)
	return "", "", "", "", "", "", "", 0, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
}

func GetIamSsoSession(sessionID string) *IamSsoSession {
	sessionsMu.RLock()
	defer sessionsMu.RUnlock()
	return sessions[sessionID]
}

type iamDeviceAuthResult struct {
	DeviceCode              string `json:"deviceCode"`
	UserCode                string `json:"userCode"`
	VerificationUri         string `json:"verificationUri"`
	VerificationUriComplete string `json:"verificationUriComplete"`
	Interval                int    `json:"interval"`
	ExpiresIn               int    `json:"expiresIn"`
}

func startIamDeviceAuth(oidcBase, clientID, clientSecret, startUrl string) (*iamDeviceAuthResult, error) {
	payload := map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
		"startUrl":     startUrl,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", oidcBase+"/device_authorization", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result iamDeviceAuthResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

func cleanupExpiredSessions() {
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	now := time.Now()
	for id, s := range sessions {
		if now.After(s.ExpiresAt) {
			delete(sessions, id)
		}
	}
}
