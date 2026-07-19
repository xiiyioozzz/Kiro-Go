package proxy

import (
	"fmt"
	"kiro-go/config"
	"net/http"
	"strings"
)

const (
	kiroStreamingSDKVersion = "1.0.34"
	kiroRuntimeSDKVersion   = "1.0.0"
)

type kiroHeaderValues struct {
	UserAgent    string
	AmzUserAgent string
	Host         string
}

func buildStreamingHeaderValues(account *config.Account, host string) kiroHeaderValues {
	return buildKiroHeaderValues(account, host, "codewhispererstreaming", kiroStreamingSDKVersion, "m/E")
}

func buildRuntimeHeaderValues(account *config.Account, host string) kiroHeaderValues {
	return buildKiroHeaderValues(account, host, "codewhispererruntime", kiroRuntimeSDKVersion, "m/N,E")
}

func buildCLIStreamingHeaderValues(account *config.Account, host string) kiroHeaderValues {
	clientCfg := config.GetKiroClientConfig()
	userAgent := fmt.Sprintf(
		"aws-sdk-rust/1.3.15 ua/2.1 api/codewhispererstreaming/0.1.14474 os/%s lang/rust/1.92.0 md/appVersion-%s app/AmazonQ-For-CLI",
		clientCfg.SystemVersion,
		clientCfg.KiroVersion,
	)
	amzUserAgent := fmt.Sprintf(
		"aws-sdk-rust/1.3.15 ua/2.1 api/codewhispererstreaming/0.1.14474 os/%s lang/rust/1.92.0 m/F app/AmazonQ-For-CLI",
		clientCfg.SystemVersion,
	)

	return kiroHeaderValues{
		UserAgent:    userAgent,
		AmzUserAgent: amzUserAgent,
		Host:         host,
	}
}

func buildKiroHeaderValues(account *config.Account, host, apiName, sdkVersion, mode string) kiroHeaderValues {
	clientCfg := config.GetKiroClientConfig()
	machineID := ""
	if account != nil {
		machineID = account.MachineId
	}

	userAgent := fmt.Sprintf(
		"aws-sdk-js/%s ua/2.1 os/%s lang/js md/nodejs#%s api/%s#%s %s KiroIDE-%s",
		sdkVersion,
		clientCfg.SystemVersion,
		clientCfg.NodeVersion,
		apiName,
		sdkVersion,
		mode,
		clientCfg.KiroVersion,
	)
	amzUserAgent := fmt.Sprintf("aws-sdk-js/%s KiroIDE-%s", sdkVersion, clientCfg.KiroVersion)
	if machineID != "" {
		userAgent += "-" + machineID
		amzUserAgent += "-" + machineID
	}

	return kiroHeaderValues{
		UserAgent:    userAgent,
		AmzUserAgent: amzUserAgent,
		Host:         host,
	}
}

func accountBearerToken(account *config.Account) string {
	if account == nil {
		return ""
	}
	if config.IsAPIKeyAccount(account) {
		if key := strings.TrimSpace(account.KiroAPIKey); key != "" {
			return key
		}
	}
	return strings.TrimSpace(account.AccessToken)
}

func accountTokenTypeHeader(account *config.Account) string {
	if config.IsAPIKeyAccount(account) {
		return "API_KEY"
	}
	if account != nil && strings.EqualFold(strings.TrimSpace(account.AuthMethod), "external_idp") {
		return "EXTERNAL_IDP"
	}
	return ""
}

func applyKiroBaseHeaders(req *http.Request, account *config.Account, values kiroHeaderValues) {
	if token := accountBearerToken(account); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("User-Agent", values.UserAgent)
	req.Header.Set("x-amz-user-agent", values.AmzUserAgent)
	req.Header.Set("x-amzn-codewhisperer-optout", "true")
	// External IdP (enterprise SSO, e.g. Azure AD) tokens MUST carry this header or
	// CodeWhisperer does not recognize the token type and silently returns an empty
	// profile list (and rejects data-plane calls). With it, a provisioned account
	// resolves its profile; an unprovisioned one gets a clear 403.
	if tokenType := accountTokenTypeHeader(account); tokenType != "" {
		req.Header.Set("TokenType", tokenType)
	}
	if values.Host != "" {
		req.Host = values.Host
	}
}
