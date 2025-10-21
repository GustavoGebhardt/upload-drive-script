package config

import "os"

const (
	defaultCredentialsFile = "credentials.json"
	defaultTokenFile       = "token.json"
	defaultOAuthState      = "state-token"
	defaultBaseURL         = "localhost"
	defaultServerPort      = ":3000"
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

func envOrDefault(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return defaultValue
}
