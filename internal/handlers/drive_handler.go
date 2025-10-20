package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"upload-drive-script/internal/services"
)

func Auth(c *gin.Context) {
	url := services.GetAuthURL()
	c.Redirect(http.StatusFound, url)
}

func OAuth2Callback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum código recebido"})
		return
	}

	conf := services.GetDriveServiceConfig()
	tok, err := conf.Exchange(context.Background(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	services.SaveToken(tok)
	c.JSON(http.StatusOK, gin.H{"message": "Autenticado com sucesso!"})
}

func Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum arquivo enviado"})
		return
	}

	tempPath := "./" + filepath.Base(file.Filename)
	if err := c.SaveUploadedFile(file, tempPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.Remove(tempPath)

	folderID := c.PostForm("folder_id")

	fileID, err := services.UploadFile(tempPath, folderID)
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

	fileName := filepath.Base(fileURL)
	tempPath := "./" + fileName

	resp, err := http.Get(fileURL)
	if err != nil || resp.StatusCode != 200 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível baixar o arquivo"})
		return
	}
	defer resp.Body.Close()

	out, err := os.Create(tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível criar arquivo temporário"})
		return
	}
	defer out.Close()
	defer os.Remove(tempPath)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar arquivo temporário"})
		return
	}

	folderID := c.PostForm("folder_id")

	fileID, err := services.UploadFile(tempPath, folderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_id": fileID})
}
