package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateSettingsPatchPreservesOmittedAPIKeyFields(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	if err := UpdateSettings("proxy-api-key", true, "admin-password"); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	if err := UpdateSettingsPatch(nil, nil, "new-admin-password"); err != nil {
		t.Fatalf("patch settings: %v", err)
	}

	if got := GetApiKey(); got != "proxy-api-key" {
		t.Fatalf("expected API key to be preserved, got %q", got)
	}
	if !IsApiKeyRequired() {
		t.Fatalf("expected requireApiKey to stay enabled")
	}
	if got := GetPassword(); got != "new-admin-password" {
		t.Fatalf("expected password to update, got %q", got)
	}
}

func TestAddAccountStampsCreatedAt(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}

	before := time.Now().Unix()
	if err := AddAccount(Account{
		ID:         "acct-created",
		Email:      "created@example.com",
		AuthMethod: "external_idp",
		Provider:   "AzureAD",
		Region:     "us-east-1",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("add account: %v", err)
	}
	after := time.Now().Unix()

	accounts := GetAccounts()
	if len(accounts) != 1 {
		t.Fatalf("expected one account, got %d", len(accounts))
	}
	if accounts[0].CreatedAt < before || accounts[0].CreatedAt > after {
		t.Fatalf("createdAt %d outside [%d, %d]", accounts[0].CreatedAt, before, after)
	}
}

func TestAddAccountNormalizesCreatedAtMillis(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if err := AddAccount(Account{
		ID:        "acct-created-ms",
		Email:     "created-ms@example.com",
		Region:    "us-east-1",
		CreatedAt: 1783246405555,
	}); err != nil {
		t.Fatalf("add account: %v", err)
	}

	accounts := GetAccounts()
	if got := accounts[0].CreatedAt; got != 1783246405 {
		t.Fatalf("expected millisecond timestamp to normalize to seconds, got %d", got)
	}
}

func TestAddAccountRejectsDuplicateProfile(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	profileArn := "arn:aws:codewhisperer:us-east-1:123456789012:profile/ABC"
	if err := AddAccount(Account{ID: "acct-1", Email: "first@example.com", ProfileArn: profileArn, Region: "us-east-1"}); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	err := AddAccount(Account{ID: "acct-2", Email: "second@example.com", ProfileArn: profileArn, Region: "us-east-1"})
	if err == nil {
		t.Fatalf("expected duplicate profile to be rejected")
	}
	if !IsDuplicateAccountError(err) {
		t.Fatalf("expected DuplicateAccountError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "first@example.com") {
		t.Fatalf("expected error to identify existing account safely, got %q", err.Error())
	}
	if got := len(GetAccounts()); got != 1 {
		t.Fatalf("expected duplicate not to be appended, got %d accounts", got)
	}
}

func TestAddAccountRejectsDuplicateKiroAPIKey(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	key := "ksk_test_duplicate_secret"
	first := Account{ID: "acct-1", Email: "first@example.com", AuthMethod: "api_key", Provider: "APIKey", KiroAPIKey: key, Region: "us-east-1"}
	second := Account{ID: "acct-2", Email: "second@example.com", AuthMethod: "api_key", Provider: "APIKey", KiroAPIKey: key, Region: "eu-central-1"}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	err := AddAccount(second)
	if err == nil {
		t.Fatalf("expected duplicate Kiro API Key to be rejected")
	}
	if !IsDuplicateAccountError(err) {
		t.Fatalf("expected DuplicateAccountError, got %T: %v", err, err)
	}
	if strings.Contains(err.Error(), key) {
		t.Fatalf("duplicate error leaked full API key: %q", err.Error())
	}
	if got := len(GetAccounts()); got != 1 {
		t.Fatalf("expected duplicate not to be appended, got %d accounts", got)
	}
}

func TestAddAccountAllowsSocialSharedProfileArnWithDifferentEmail(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	sharedProfileArn := "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
	first := Account{
		ID:           "acct-1",
		Email:        "first@example.com",
		AuthMethod:   "social",
		Provider:     "Google",
		ProfileArn:   sharedProfileArn,
		RefreshToken: "refresh-first",
		Region:       "us-east-1",
	}
	second := Account{
		ID:           "acct-2",
		Email:        "second@example.com",
		AuthMethod:   "social",
		Provider:     "Google",
		ProfileArn:   sharedProfileArn,
		RefreshToken: "refresh-second",
		Region:       "us-east-1",
	}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	if err := AddAccount(second); err != nil {
		t.Fatalf("expected different social accounts to allow shared profileArn, got %v", err)
	}
	if got := len(GetAccounts()); got != 2 {
		t.Fatalf("expected two social accounts, got %d", got)
	}
}

func TestAddAccountRejectsDuplicateRefreshToken(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	sharedProfileArn := "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
	first := Account{
		ID:           "acct-1",
		Email:        "first@example.com",
		AuthMethod:   "social",
		Provider:     "Google",
		ProfileArn:   sharedProfileArn,
		RefreshToken: "same-refresh-token",
		Region:       "us-east-1",
	}
	second := Account{
		ID:           "acct-2",
		Email:        "second@example.com",
		AuthMethod:   "social",
		Provider:     "Google",
		ProfileArn:   sharedProfileArn,
		RefreshToken: "same-refresh-token",
		Region:       "us-east-1",
	}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	if err := AddAccount(second); err == nil {
		t.Fatalf("expected duplicate refresh token to be rejected")
	} else if !IsDuplicateAccountError(err) {
		t.Fatalf("expected DuplicateAccountError, got %T: %v", err, err)
	}
	if got := len(GetAccounts()); got != 1 {
		t.Fatalf("expected duplicate not to be appended, got %d accounts", got)
	}
}

func TestAddAccountAllowsDifferentKiroAPIKeysWithSameLabel(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	first := Account{ID: "acct-1", Email: "Kiro API Key", AuthMethod: "api_key", Provider: "APIKey", KiroAPIKey: "ksk_first", Region: "us-east-1"}
	second := Account{ID: "acct-2", Email: "Kiro API Key", AuthMethod: "api_key", Provider: "APIKey", KiroAPIKey: "ksk_second", Region: "us-east-1"}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	if err := AddAccount(second); err != nil {
		t.Fatalf("expected different API keys to be allowed even with the same display label, got %v", err)
	}
	if got := len(GetAccounts()); got != 2 {
		t.Fatalf("expected two API key accounts, got %d", got)
	}
}

func TestIsAPIKeyAuthMethod(t *testing.T) {
	for _, method := range []string{"api_key", "API_KEY", "apikey", "ApiKey"} {
		if !IsAPIKeyAuthMethod(method) {
			t.Fatalf("expected %q to be treated as API key auth", method)
		}
	}
	if IsAPIKeyAuthMethod("external_idp") {
		t.Fatalf("external_idp should not be treated as API key auth")
	}
}

func TestAddAccountRejectsSameEmailAuthAndRegion(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	first := Account{ID: "acct-1", Email: "Same@example.com", AuthMethod: "external_idp", Provider: "AzureAD", Region: "us-east-1"}
	second := Account{ID: "acct-2", Email: "same@example.com", AuthMethod: "external_idp", Provider: "AzureAD", Region: "US-EAST-1"}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	if err := AddAccount(second); err == nil {
		t.Fatalf("expected same email/auth/region to be rejected")
	} else if !IsDuplicateAccountError(err) {
		t.Fatalf("expected DuplicateAccountError, got %T: %v", err, err)
	}
}

func TestAddAccountRejectsSameExternalEmailRegionWhenProviderWasMissing(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	first := Account{ID: "acct-1", Email: "azure@example.com", AuthMethod: "external_idp", Region: "us-east-1"}
	second := Account{ID: "acct-2", Email: "azure@example.com", AuthMethod: "external_idp", Provider: "AzureAD", Region: "us-east-1"}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	if err := AddAccount(second); err == nil {
		t.Fatalf("expected duplicate external_idp email/region to be rejected even when provider differs")
	} else if !IsDuplicateAccountError(err) {
		t.Fatalf("expected DuplicateAccountError, got %T: %v", err, err)
	}
}

func TestAddAccountAllowsSameEmailDifferentRegion(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	first := Account{ID: "acct-1", Email: "same@example.com", AuthMethod: "external_idp", Provider: "AzureAD", Region: "us-east-1"}
	second := Account{ID: "acct-2", Email: "same@example.com", AuthMethod: "external_idp", Provider: "AzureAD", Region: "eu-central-1"}
	if err := AddAccount(first); err != nil {
		t.Fatalf("add first account: %v", err)
	}
	if err := AddAccount(second); err != nil {
		t.Fatalf("expected different region to be allowed, got %v", err)
	}
	if got := len(GetAccounts()); got != 2 {
		t.Fatalf("expected two accounts, got %d", got)
	}
}

func TestUpdateSettingsPatchCanExplicitlyDisableAPIKey(t *testing.T) {
	if err := Init(filepath.Join(t.TempDir(), "config.json")); err != nil {
		t.Fatalf("init config: %v", err)
	}
	if err := UpdateSettings("proxy-api-key", true, "admin-password"); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	emptyKey := ""
	requireAPIKey := false
	if err := UpdateSettingsPatch(&emptyKey, &requireAPIKey, ""); err != nil {
		t.Fatalf("patch settings: %v", err)
	}

	if got := GetApiKey(); got != "" {
		t.Fatalf("expected API key to be cleared, got %q", got)
	}
	if IsApiKeyRequired() {
		t.Fatalf("expected requireApiKey to be disabled")
	}
	if got := GetPassword(); got != "admin-password" {
		t.Fatalf("expected password to be preserved, got %q", got)
	}
}

// TestAccountAllowOverageMigration verifies that a config.json from before the
// upstream-Overages-switch refactor (which carried `allowOverage: true` per
// account) is migrated into OverageStatus="ENABLED" on first load, and that
// the legacy field is cleared so future saves don't re-emit it.
func TestAccountAllowOverageMigration(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.json")

	seed := map[string]interface{}{
		"password":      "p",
		"port":          8080,
		"host":          "0.0.0.0",
		"requireApiKey": false,
		"accounts": []map[string]interface{}{
			{"id": "acc-allow", "enabled": true, "allowOverage": true},
			{"id": "acc-deny", "enabled": true, "allowOverage": false},
			{"id": "acc-already-set", "enabled": true, "allowOverage": true, "overageStatus": "DISABLED"},
		},
	}
	raw, err := json.MarshalIndent(seed, "", "  ")
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	if err := os.WriteFile(cfgFile, raw, 0600); err != nil {
		t.Fatalf("write seed: %v", err)
	}

	if err := Init(cfgFile); err != nil {
		t.Fatalf("init: %v", err)
	}

	accounts := GetAccounts()
	byID := map[string]Account{}
	for _, a := range accounts {
		byID[a.ID] = a
	}

	if got := byID["acc-allow"].OverageStatus; got != "ENABLED" {
		t.Fatalf("expected acc-allow to migrate to OverageStatus=ENABLED, got %q", got)
	}
	if byID["acc-allow"].LegacyAllowOverage {
		t.Fatalf("expected legacy allowOverage to be cleared after migration")
	}
	if got := byID["acc-deny"].OverageStatus; got != "" {
		t.Fatalf("expected acc-deny to keep empty OverageStatus, got %q", got)
	}
	// Pre-set OverageStatus must win over the legacy field.
	if got := byID["acc-already-set"].OverageStatus; got != "DISABLED" {
		t.Fatalf("expected acc-already-set OverageStatus to be preserved, got %q", got)
	}
	if byID["acc-already-set"].LegacyAllowOverage {
		t.Fatalf("expected legacy field to still be cleared on acc-already-set")
	}

	// Re-read the file and confirm legacy field is gone (so it doesn't drift
	// back in on later saves).
	on_disk, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var reloaded struct {
		Accounts []map[string]interface{} `json:"accounts"`
	}
	if err := json.Unmarshal(on_disk, &reloaded); err != nil {
		t.Fatalf("decode reload: %v", err)
	}
	for _, a := range reloaded.Accounts {
		if _, ok := a["allowOverage"]; ok {
			t.Fatalf("expected allowOverage to be omitted from persisted file, got %+v", a)
		}
	}
}
