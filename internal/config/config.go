package config

import "os"

const (
	defaultCredentialsFile = "credentials.json"
	defaultTokenFile       = "token.json"
	defaultRedirectURL     = "http://localhost:3000/oauth2callback"
	defaultOAuthState      = "state-token"
	defaultServerAddr      = ":3000"
)

func CredentialsFile() string {
	return envOrDefault("GOOGLE_CREDENTIALS_FILE", defaultCredentialsFile)
}

func TokenFile() string {
	return envOrDefault("GOOGLE_TOKEN_FILE", defaultTokenFile)
}

func OAuthRedirectURL() string {
	return envOrDefault("GOOGLE_OAUTH_REDIRECT_URL", defaultRedirectURL)
}

func OAuthState() string {
	return envOrDefault("GOOGLE_OAUTH_STATE", defaultOAuthState)
}

func ServerAddr() string {
	return envOrDefault("HTTP_LISTEN_ADDR", defaultServerAddr)
}

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}
