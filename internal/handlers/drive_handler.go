package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"upload-drive-script/internal/media"
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

	const uploadDir = "upload"

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao preparar diretório de upload"})
		return
	}

	fileNameOnDisk, err := sanitizeFilename(file.Filename)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fileNameOnDisk = ensureUniqueFilename(uploadDir, fileNameOnDisk)
	filePath := filepath.Join(uploadDir, fileNameOnDisk)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	folderID := c.PostForm("folder_id")

	fileName := c.PostForm("file_name")

	mimeType, err := media.DetectMimeType(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	isVideo := media.IsVideoMime(mimeType)

	fileID, err := services.UploadFile(filePath, folderID, fileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"file_id":  fileID,
		"file_url": buildPublicFileURL(c, fileNameOnDisk),
	}

	if isVideo {
		audioPath, err := media.ExtractAudio(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(audioPath)

		audioFileName := media.BuildAudioFileName(fileName, filePath)
		audioFileID, err := services.UploadFile(audioPath, folderID, audioFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response["audio_file_id"] = audioFileID
	}

	c.JSON(http.StatusOK, response)
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

	mimeType, err := media.DetectMimeType(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	isVideo := media.IsVideoMime(mimeType)

	fileID, err := services.UploadFile(tempPath, folderID, fileName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{"file_id": fileID}

	if isVideo {
		audioPath, err := media.ExtractAudio(tempPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer os.Remove(audioPath)

		audioFileName := media.BuildAudioFileName(fileName, filepath.Base(parsedURL.Path))
		audioFileID, err := services.UploadFile(audioPath, folderID, audioFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response["audio_file_id"] = audioFileID
	}

	c.JSON(http.StatusOK, response)
}

func GetUploadedFile(c *gin.Context) {
	fileNameParam := c.Param("filename")

	fileName, err := sanitizeFilename(fileNameParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nome de arquivo inválido"})
		return
	}

	const uploadDir = "upload"

	filePath := filepath.Join(uploadDir, fileName)

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Arquivo não encontrado"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao acessar arquivo"})
		return
	}

	c.File(filePath)
}

func sanitizeFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("Nome de arquivo inválido")
	}

	cleanName := filepath.Base(name)
	if cleanName == "." || cleanName == ".." || cleanName == "" {
		return "", errors.New("Nome de arquivo inválido")
	}

	if strings.ContainsAny(cleanName, "/\\") {
		return "", errors.New("Nome de arquivo inválido")
	}

	return cleanName, nil
}

func ensureUniqueFilename(dir, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := name
	counter := 1

	for {
		if _, err := os.Stat(filepath.Join(dir, candidate)); errors.Is(err, os.ErrNotExist) {
			return candidate
		}

		candidate = fmt.Sprintf("%s-%d%s", base, counter, ext)
		counter++
	}
}

func buildPublicFileURL(c *gin.Context, filename string) string {
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := c.Request.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}

	if host == "" {
		return "/uploads/" + url.PathEscape(filename)
	}

	return scheme + "://" + host + "/uploads/" + url.PathEscape(filename)
}
