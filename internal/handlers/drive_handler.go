package handlers

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"upload-drive-script/internal/services"
)

func Auth(c *gin.Context) {
	url, err := services.GetAuthURL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Redirect(http.StatusFound, url)
}

func OAuth2Callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum código recebido"})
		return
	}

	conf, err := services.GetDriveServiceConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tok, err := conf.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := services.SaveToken(tok); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Autenticado com sucesso!"})
}

func Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum arquivo enviado"})
		return
	}

	tempFile, err := os.CreateTemp("", "upload-*"+filepath.Ext(file.Filename))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao preparar arquivo temporário"})
		return
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempPath)

	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	folderID := c.PostForm("folder_id")

	fileName := c.PostForm("file_name")

	fileID, err := services.UploadFile(tempPath, folderID, fileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_id": fileID})
}

func UploadURL(c *gin.Context) {
	fileURL := c.PostForm("url")
	if fileURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhuma URL fornecida"})
		return
	}

	parsedURL, err := url.Parse(fileURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL inválida"})
		return
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Apenas URLs HTTP/HTTPS são permitidas"})
		return
	}

	host := parsedURL.Host
	if strings.Contains(host, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL com credenciais embutidas não é permitida"})
		return
	}

	if h, _, splitErr := net.SplitHostPort(host); splitErr == nil {
		host = h
	}
	if strings.EqualFold(host, "localhost") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL não permitida"})
		return
	}
	trimmedHost := strings.Trim(host, "[]")
	if ip := net.ParseIP(trimmedHost); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "URL não permitida"})
			return
		}
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(parsedURL.String())
	if err != nil || resp.StatusCode != 200 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível baixar o arquivo"})
		return
	}
	defer resp.Body.Close()

	ext := filepath.Ext(parsedURL.Path)
	tempFile, err := os.CreateTemp("", "upload-*"+ext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível criar arquivo temporário"})
		return
	}
	defer tempFile.Close()

	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar arquivo temporário"})
		return
	}

	folderID := c.PostForm("folder_id")

	fileName := c.PostForm("file_name")

	fileID, err := services.UploadFile(tempPath, folderID, fileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_id": fileID})
}
