package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func GetDriveClient(tokenString string) (*http.Client, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token de acesso é obrigatório")
	}

	token := &oauth2.Token{
		AccessToken: tokenString,
	}
	return oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token)), nil
}

func GetDriveService(tokenString string) (*drive.Service, error) {
	client, err := GetDriveClient(tokenString)
	if err != nil {
		return nil, err
	}
	return drive.NewService(context.Background(), option.WithHTTPClient(client))
}

func UploadFile(tokenString string, filePath string, folderID string, fileName string) (string, error) {
	srv, err := GetDriveService(tokenString)
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

func UploadFileStream(tokenString string, content io.Reader, folderID string, fileName string) (string, error) {
	srv, err := GetDriveService(tokenString)
	if err != nil {
		return "", err
	}

	file := &drive.File{
		Name: fileName,
	}
	if folderID != "" {
		file.Parents = []string{folderID}
	}

	res, err := srv.Files.Create(file).
		Media(content).
		Do()
	if err != nil {
		return "", err
	}

	return res.Id, nil
}
