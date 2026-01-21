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

	"upload-drive-script/internal/config"
	"upload-drive-script/internal/media"
	"upload-drive-script/internal/services"
)

var errUnsupportedMediaType = errors.New("tipo de arquivo não suportado")

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
	// Extract token from header
	authHeader := c.GetHeader("Authorization")
	tokenString := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Usar MultipartReader para streaming
	reader, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Falha ao ler multipart request"})
		return
	}

	var folderID string
	var fileName string
	var driveFileID string
	var mimeType string
	var filePath string
	var fileNameOnDisk string
	const uploadDir = "upload"

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao preparar diretório de upload"})
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Erro ao ler parte do formulário"})
			return
		}

		switch part.FormName() {
		case "folder_id":
			buf := new(strings.Builder)
			if _, err := io.Copy(buf, part); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler folder_id"})
				return
			}
			folderID = buf.String()
		case "file_name":
			buf := new(strings.Builder)
			if _, err := io.Copy(buf, part); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao ler file_name"})
				return
			}
			fileName = buf.String()
		case "file":
			// Processo principal de upload
			if fileName == "" {
				fileName = part.FileName()
			}

			cleanName, err := sanitizeFilename(fileName)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			fileNameOnDisk = ensureUniqueFilename(uploadDir, cleanName)
			filePath = filepath.Join(uploadDir, fileNameOnDisk)

			// Criar arquivo local para backup/processamento
			out, err := os.Create(filePath)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao criar arquivo local"})
				return
			}
			defer out.Close() // Fecha o arquivo ao final da função, mas fecharemos explicitamente antes do processamento

			// TeeReader: Lê do part -> Escreve no out (disco) -> Retorna para o UploadFileStream
			tee := io.TeeReader(part, out)

			// Inicia Upload para o Drive usando o stream
			// O upload lê do 'tee', que lê do 'part' e escreve em 'out' simultaneamente.
			uploadedID, err := services.UploadFileStream(tokenString, tee, folderID, fileName)

			// Importante: Fechar o arquivo local explicitamente para garantir flush antes de usar
			out.Close()

			if err != nil {
				_ = os.Remove(filePath) // Limpa em caso de erro
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Erro no upload para o Drive: %v", err)})
				return
			}

			driveFileID = uploadedID

			// Detectar mime type do arquivo salvo localmente
			detectedMime, err := media.DetectMimeType(filePath)
			if err != nil {
				// Se falhar detecção, tenta pelo header (menos confiável, mas fallback)
				// Se não, assume erro.
				// Para robustez, vamos continuar ou retornar erro.
				// Mas se falhou detect, talvez o arquivo esteja corrompido.
				_ = os.Remove(filePath)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao detectar tipo de arquivo"})
				return
			}
			mimeType = detectedMime
		}
	}

	// Validar se houve processamento
	if driveFileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nenhum arquivo enviado ou processado"})
		return
	}

	// Validação de MimeType (estava em buildUploadResponse)
	isVideo := media.IsVideoMime(mimeType)
	isAudio := media.IsAudioMime(mimeType)

	if !isVideo && !isAudio {
		_ = os.Remove(filePath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Apenas arquivos de áudio ou vídeo são permitidos"})
		return
	}

	finalResponse := gin.H{
		"video_file_id":  nil,
		"audio_file_id":  nil,
		"video_file_url": nil,
		"audio_file_url": nil,
	}

	if isVideo {
		finalResponse["video_file_id"] = driveFileID
		finalResponse["video_file_url"] = buildPublicFileURL(c, fileNameOnDisk)

		// Extração de áudio
		audioTempPath, err := media.ExtractAudio(filePath)
		if err != nil {
			// Se falhar converter áudio, retornamos erro? Ou só o vídeo?
			// Código original retornava erro.
			_ = os.Remove(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		audioDriveName := media.BuildAudioFileName(fileName, filePath)
		audioFileNameOnDisk, audioFilePath, err := persistGeneratedFile(uploadDir, audioTempPath, audioDriveName)
		if err != nil {
			_ = os.Remove(audioTempPath)
			_ = os.Remove(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Upload do áudio (ainda usa arquivo local, tudo bem ser pequeno)
		audioFileID, err := services.UploadFile(tokenString, audioFilePath, folderID, audioDriveName)
		if err != nil {
			_ = os.Remove(filePath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		finalResponse["audio_file_id"] = audioFileID
		finalResponse["audio_file_url"] = buildPublicFileURL(c, audioFileNameOnDisk)
	} else {
		finalResponse["audio_file_id"] = driveFileID
		finalResponse["audio_file_url"] = buildPublicFileURL(c, fileNameOnDisk)
	}

	c.JSON(http.StatusOK, finalResponse)
}

func UploadURL(c *gin.Context) {
	// Extract token from header
	authHeader := c.GetHeader("Authorization")
	tokenString := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenString = strings.TrimPrefix(authHeader, "Bearer ")
	}

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

	const uploadDir = "upload"

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Falha ao preparar diretório de upload"})
		return
	}

	fileNameOnDisk, filePath, err := saveRemoteFile(resp.Body, uploadDir, parsedURL.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	folderID := c.PostForm("folder_id")

	fileName := c.PostForm("file_name")

	mimeType, err := media.DetectMimeType(filePath)
	if err != nil {
		_ = os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response, err := buildUploadResponse(c, tokenString, uploadDir, filePath, fileNameOnDisk, folderID, fileName, mimeType)
	if err != nil {
		_ = os.Remove(filePath)
		if errors.Is(err, errUnsupportedMediaType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Apenas arquivos de áudio ou vídeo são permitidos"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
	if baseURL, ok := config.PublicBaseURL(); ok {
		prefix := strings.TrimSuffix(baseURL.String(), "/")
		return prefix + "/uploads/" + url.PathEscape(filename)
	}

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

func buildUploadResponse(
	c *gin.Context,
	tokenString string,
	uploadDir string,
	filePath string,
	fileNameOnDisk string,
	folderID string,
	driveFileName string,
	mimeType string,
) (gin.H, error) {
	isVideo := media.IsVideoMime(mimeType)
	isAudio := media.IsAudioMime(mimeType)

	if !isVideo && !isAudio {
		return nil, errUnsupportedMediaType
	}

	response := gin.H{
		"video_file_id":  nil,
		"audio_file_id":  nil,
		"video_file_url": nil,
		"audio_file_url": nil,
	}

	if isVideo {
		videoFileID, err := services.UploadFile(tokenString, filePath, folderID, driveFileName)
		if err != nil {
			return nil, err
		}
		response["video_file_id"] = videoFileID
		response["video_file_url"] = buildPublicFileURL(c, fileNameOnDisk)

		audioTempPath, err := media.ExtractAudio(filePath)
		if err != nil {
			return nil, err
		}

		audioDriveName := media.BuildAudioFileName(driveFileName, filePath)
		audioFileNameOnDisk, audioFilePath, err := persistGeneratedFile(uploadDir, audioTempPath, audioDriveName)
		if err != nil {
			_ = os.Remove(audioTempPath)
			return nil, err
		}

		audioFileID, err := services.UploadFile(tokenString, audioFilePath, folderID, audioDriveName)
		if err != nil {
			return nil, err
		}
		response["audio_file_id"] = audioFileID
		response["audio_file_url"] = buildPublicFileURL(c, audioFileNameOnDisk)
		return response, nil
	}

	audioFileID, err := services.UploadFile(tokenString, filePath, folderID, driveFileName)
	if err != nil {
		return nil, err
	}
	response["audio_file_id"] = audioFileID
	response["audio_file_url"] = buildPublicFileURL(c, fileNameOnDisk)

	return response, nil
}

func persistGeneratedFile(uploadDir, tempPath, preferredName string) (string, string, error) {
	filename, err := sanitizeFilename(preferredName)
	if err != nil {
		return "", "", fmt.Errorf("nome de arquivo inválido para áudio: %w", err)
	}
	filename = ensureUniqueFilename(uploadDir, filename)
	destPath := filepath.Join(uploadDir, filename)

	if err := moveFile(tempPath, destPath); err != nil {
		return "", "", err
	}

	return filename, destPath, nil
}

func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		_ = os.Remove(dst)
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}
	return os.Remove(src)
}

func saveRemoteFile(body io.Reader, uploadDir, sourcePath string) (string, string, error) {
	filename, err := generateSafeFilename(filepath.Base(sourcePath))
	if err != nil {
		return "", "", err
	}

	filename = ensureUniqueFilename(uploadDir, filename)
	destPath := filepath.Join(uploadDir, filename)

	dest, err := os.Create(destPath)
	if err != nil {
		return "", "", fmt.Errorf("não foi possível criar arquivo de destino: %w", err)
	}

	if _, err := io.Copy(dest, body); err != nil {
		dest.Close()
		_ = os.Remove(destPath)
		return "", "", fmt.Errorf("erro ao salvar arquivo baixado: %w", err)
	}

	if err := dest.Close(); err != nil {
		_ = os.Remove(destPath)
		return "", "", fmt.Errorf("erro ao fechar arquivo baixado: %w", err)
	}

	return filename, destPath, nil
}

func generateSafeFilename(preferred string) (string, error) {
	if preferred != "" {
		if name, err := sanitizeFilename(preferred); err == nil {
			return name, nil
		}
	}
	fallback := fmt.Sprintf("download-%d.tmp", time.Now().Unix())
	return sanitizeFilename(fallback)
}
