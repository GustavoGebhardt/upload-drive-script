package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"upload-drive-script/internal/config"
)

func GetDriveServiceConfig() (*oauth2.Config, error) {
	b, err := os.ReadFile(config.CredentialsFile())
	if err != nil {
		return nil, fmt.Errorf("ler credenciais: %w", err)
	}

	conf, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		return nil, fmt.Errorf("criar config OAuth2: %w", err)
	}

	conf.RedirectURL = fmt.Sprintf("http://%s%d/oauth2callback", config.BaseURL())
	return conf, nil
}

func GetAuthURL() (string, error) {
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

func GetDriveService() (*drive.Service, error) {
	client, err := GetDriveClient()
	if err != nil {
		return nil, err
	}
	return drive.NewService(context.Background(), option.WithHTTPClient(client))
}

func UploadFile(filePath string, folderID string) (string, error) {
	srv, err := GetDriveService()
	if err != nil {
		return "", err
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	file := &drive.File{
		Name: filepath.Base(filePath),
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
