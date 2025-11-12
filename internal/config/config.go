package config

import (
	"net"
	"net/url"
	"os"
	"strings"
)

type AuthMode string

const (
	AuthModeOAuth          AuthMode = "oauth"
	AuthModeServiceAccount AuthMode = "service_account"
)

const (
	defaultCredentialsFile = "credentials.json"
	defaultTokenFile       = "token.json"
	defaultOAuthState      = "state-token"
	defaultBaseURL         = "localhost"
	defaultServerPort      = ":3000"
	defaultAuthMode        = AuthModeOAuth
)

func CredentialsFile() string {
	return envOrDefault("GOOGLE_CREDENTIALS_FILE", defaultCredentialsFile)
}

func TokenFile() string {
	return envOrDefault("GOOGLE_TOKEN_FILE", defaultTokenFile)
}

func OAuthState() string {
	return envOrDefault("GOOGLE_OAUTH_STATE", defaultOAuthState)
}

func BaseURL() string { return envOrDefault("APP_BASE_URL", defaultBaseURL) }

func ServerPort() string {
	return envOrDefault("APP_SERVER_PORT", defaultServerPort)
}

func AuthenticationMode() AuthMode {
	value := strings.ToLower(envOrDefault("GOOGLE_AUTH_MODE", string(defaultAuthMode)))
	mode := AuthMode(value)

	switch mode {
	case AuthModeOAuth, AuthModeServiceAccount:
		return mode
	default:
		return defaultAuthMode
	}
}

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}

func PublicBaseURL() (*url.URL, bool) {
	raw, ok := lookupEnvNonEmpty("APP_BASE_URL")
	if !ok {
		return nil, false
	}

	hasScheme := strings.Contains(raw, "://")
	candidate := raw
	if !hasScheme {
		candidate = "http://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil || parsed.Host == "" {
		return nil, false
	}

	if !hasScheme {
		if looksLocal(parsed.Hostname()) {
			parsed.Scheme = "http"
		} else {
			parsed.Scheme = "https"
		}
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed, true
}

func lookupEnvNonEmpty(key string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed, true
		}
	}
	return "", false
}

func looksLocal(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "", "localhost":
		return true
	}

	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}

	return false
}
