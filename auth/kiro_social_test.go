package auth

import (
	"net/url"
	"testing"
)

func TestBuildKiroSocialPortalURLMatchesKiroPortalShape(t *testing.T) {
	got := buildKiroSocialPortalURL("state-1", "challenge-1", "http://127.0.0.1:3128")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse portal URL: %v", err)
	}
	if u.Scheme != "https" || u.Host != "app.kiro.dev" || u.Path != "/signin" {
		t.Fatalf("unexpected portal URL: %s", got)
	}
	q := u.Query()
	assertQuery := func(key, want string) {
		t.Helper()
		if got := q.Get(key); got != want {
			t.Fatalf("query %s = %q, want %q", key, got, want)
		}
	}
	assertQuery("state", "state-1")
	assertQuery("code_challenge", "challenge-1")
	assertQuery("code_challenge_method", "S256")
	assertQuery("redirect_uri", "http://127.0.0.1:3128")
	assertQuery("redirect_from", "KiroIDE")
}

func TestKiroSocialFullRedirectURIIncludesCallbackPathAndLoginOption(t *testing.T) {
	session := &KiroSocialSession{RedirectURI: "http://127.0.0.1:3128"}
	got := session.fullRedirectURI("/oauth/callback", "google")
	want := "http://127.0.0.1:3128/oauth/callback?login_option=google"
	if got != want {
		t.Fatalf("redirect URI = %q, want %q", got, want)
	}
}

func TestKiroSocialRegionFromProfileArn(t *testing.T) {
	got := kiroSocialRegionFromProfileArn("arn:aws:codewhisperer:eu-central-1:123456789012:profile/ABC")
	if got != "eu-central-1" {
		t.Fatalf("region = %q, want eu-central-1", got)
	}
}
