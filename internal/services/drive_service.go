package services

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"

	"upload-drive-script/internal/config"
)

func GetDriveServiceConfig() *oauth2.Config {
	b, err := os.ReadFile(config.CredentialsFile)
	if err != nil {
		log.Fatalf("Erro ao ler credenciais: %v", err)
	}

	conf, err := google.ConfigFromJSON(b, drive.DriveFileScope)
	if err != nil {
		log.Fatalf("Erro ao criar config OAuth2: %v", err)
	}

	conf.RedirectURL = "http://localhost:3000/oauth2callback"
	return conf
}

func GetAuthURL() string {
	conf := GetDriveServiceConfig()
	return conf.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
}

func SaveToken(token *oauth2.Token) {
	f, err := os.Create(config.TokenFile)
	if err != nil {
		log.Fatalf("Erro ao criar token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func LoadToken() (*oauth2.Token, error) {
	f, err := os.Open(config.TokenFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func GetDriveClient() (*http.Client, error) {
	conf := GetDriveServiceConfig()
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
