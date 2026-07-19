package proxy

import (
	"kiro-go/config"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBuildStreamingHeaderValuesAlignsWithKiroIDEFormat(t *testing.T) {
	account := &config.Account{MachineId: "machine-123"}
	values := buildStreamingHeaderValues(account, "q.us-east-1.amazonaws.com")

	if values.Host != "q.us-east-1.amazonaws.com" {
		t.Fatalf("expected host to be preserved, got %q", values.Host)
	}
	if !strings.Contains(values.UserAgent, "aws-sdk-js/1.0.34") {
		t.Fatalf("expected streaming sdk version in user agent, got %q", values.UserAgent)
	}
	if !strings.Contains(values.UserAgent, "api/codewhispererstreaming#1.0.34") {
		t.Fatalf("expected streaming API marker in user agent, got %q", values.UserAgent)
	}
	if !strings.Contains(values.UserAgent, "KiroIDE-0.11.107-machine-123") {
		t.Fatalf("expected kiro version and machine id in user agent, got %q", values.UserAgent)
	}
	if !strings.Contains(values.AmzUserAgent, "aws-sdk-js/1.0.34 KiroIDE-0.11.107-machine-123") {
		t.Fatalf("expected x-amz-user-agent to include version and machine id, got %q", values.AmzUserAgent)
	}
}

func TestBuildRuntimeHeaderValuesUsesRuntimeAPIFormat(t *testing.T) {
	account := &config.Account{MachineId: "machine-456"}
	values := buildRuntimeHeaderValues(account, "codewhisperer.us-east-1.amazonaws.com")

	if !strings.Contains(values.UserAgent, "aws-sdk-js/1.0.0") {
		t.Fatalf("expected runtime sdk version in user agent, got %q", values.UserAgent)
	}
	if !strings.Contains(values.UserAgent, "api/codewhispererruntime#1.0.0") {
		t.Fatalf("expected runtime API marker in user agent, got %q", values.UserAgent)
	}
	if !strings.Contains(values.UserAgent, "m/N,E") {
		t.Fatalf("expected runtime mode marker in user agent, got %q", values.UserAgent)
	}
}

func TestApplyKiroBaseHeadersUsesKiroAPIKeyTokenType(t *testing.T) {
	req := httptest.NewRequest("POST", "https://q.us-east-1.amazonaws.com/", nil)
	account := &config.Account{
		AuthMethod: "api_key",
		KiroAPIKey: "ksk_test_key",
	}

	applyKiroBaseHeaders(req, account, kiroHeaderValues{
		UserAgent:    "ua",
		AmzUserAgent: "amz",
		Host:         "q.us-east-1.amazonaws.com",
	})

	if got := req.Header.Get("Authorization"); got != "Bearer ksk_test_key" {
		t.Fatalf("expected API key bearer token, got %q", got)
	}
	if got := req.Header.Get("TokenType"); got != "API_KEY" {
		t.Fatalf("expected API_KEY token type, got %q", got)
	}
	if req.Host != "q.us-east-1.amazonaws.com" {
		t.Fatalf("expected req.Host override, got %q", req.Host)
	}
}
