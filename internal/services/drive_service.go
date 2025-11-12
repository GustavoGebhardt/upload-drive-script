package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"upload-drive-script/internal/config"
)

func GetDriveServiceConfig() (*oauth2.Config, error) {
	if config.AuthenticationMode() == config.AuthModeServiceAccount {
		return nil, fmt.Errorf("configuração OAuth indisponível no modo service_account")
	}

	b, err := os.ReadFile(config.CredentialsFile())
	if err != nil {
		return nil, fmt.Errorf("ler credenciais: %w", err)
	}

	conf, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("criar config OAuth2: %w", err)
	}

	conf.RedirectURL = buildOAuthRedirectURL()
	return conf, nil
}

func GetAuthURL() (string, error) {
	if config.AuthenticationMode() == config.AuthModeServiceAccount {
		return "", fmt.Errorf("rota de auth indisponível no modo service_account")
	}

	conf, err := GetDriveServiceConfig()
	if err != nil {
		return "", err
	}
	return conf.AuthCodeURL(config.OAuthState(), oauth2.AccessTypeOffline), nil
}

func SaveToken(token *oauth2.Token) error {
	f, err := os.Create(config.TokenFile())
	if err != nil {
		return fmt.Errorf("criar arquivo de token: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(token); err != nil {
		return fmt.Errorf("persistir token: %w", err)
	}
	return nil
}

func LoadToken() (*oauth2.Token, error) {
	f, err := os.Open(config.TokenFile())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	if err := json.NewDecoder(f).Decode(tok); err != nil {
		return nil, fmt.Errorf("ler token salvo: %w", err)
	}
	return tok, nil
}

func GetDriveClient() (*http.Client, error) {
	if config.AuthenticationMode() == config.AuthModeServiceAccount {
		return getServiceAccountClient()
	}

	return getOAuthClient()
}

func getOAuthClient() (*http.Client, error) {
	conf, err := GetDriveServiceConfig()
	if err != nil {
		return nil, err
	}
	tok, err := LoadToken()
	if err != nil {
		return nil, err
	}
	return conf.Client(context.Background(), tok), nil
}

func getServiceAccountClient() (*http.Client, error) {
	credentialsJSON, err := os.ReadFile(config.CredentialsFile())
	if err != nil {
		return nil, fmt.Errorf("ler credenciais: %w", err)
	}

	creds, err := google.CredentialsFromJSON(context.Background(), credentialsJSON, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("criar credenciais de service account: %w", err)
	}

	return oauth2.NewClient(context.Background(), creds.TokenSource), nil
}

func GetDriveService() (*drive.Service, error) {
	client, err := GetDriveClient()
	if err != nil {
		return nil, err
	}
	return drive.NewService(context.Background(), option.WithHTTPClient(client))
}

func UploadFile(filePath string, folderID string, fileName string) (string, error) {
	srv, err := GetDriveService()
	if err != nil {
		return "", err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if fileName == "" {
		fileName = filepath.Base(filePath)
	}

	file := &drive.File{
		Name: fileName,
	}
	if folderID != "" {
		file.Parents = []string{folderID}
	}

	res, err := srv.Files.Create(file).
		Media(f).
		Do()
	if err != nil {
		return "", err
	}

	return res.Id, nil
}

func buildOAuthRedirectURL() string {
	if baseURL, ok := config.PublicBaseURL(); ok {
		return strings.TrimSuffix(baseURL.String(), "/") + "/oauth2callback"
	}

	host := sanitizeHost(config.BaseURL())
	port := normalizePort(config.ServerPort())
	return fmt.Sprintf("http://%s%s/oauth2callback", host, port)
}

func sanitizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimSuffix(host, "/")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if host == "" {
		return "localhost"
	}
	return host
}

func normalizePort(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ":3000"
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}
