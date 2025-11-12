package media

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const audioExtension = ".mp3"

// DetectMimeType infers the MIME type of a file by reading its header bytes.
func DetectMimeType(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("abrir arquivo para detectar MIME: %w", err)
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("ler arquivo para detectar MIME: %w", err)
	}
	return http.DetectContentType(buf[:n]), nil
}

func IsVideoMime(mime string) bool {
	return strings.HasPrefix(mime, "video/")
}

// ExtractAudio uses ffmpeg to extract an audio track from a video file.
// Returns the path to the generated audio file (caller must remove it).
func ExtractAudio(srcPath string) (string, error) {
	dst, err := os.CreateTemp("", "audio-*"+audioExtension)
	if err != nil {
		return "", fmt.Errorf("criar arquivo temporário para áudio: %w", err)
	}
	dstPath := dst.Name()
	dst.Close()

	var stderr bytes.Buffer
	cmd := exec.Command("ffmpeg", "-y", "-i", srcPath, "-vn", "-acodec", "libmp3lame", dstPath)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("falha ao extrair áudio (ffmpeg): %w - %s", err, stderr.String())
	}

	return dstPath, nil
}

func BuildAudioFileName(originalPreferredName, fallbackPath string) string {
	baseName := originalPreferredName
	if baseName == "" {
		baseName = filepath.Base(fallbackPath)
	}
	baseName = strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if baseName == "" {
		baseName = fmt.Sprintf("audio-%d", time.Now().Unix())
	}

	return baseName + "-audio" + audioExtension
}
