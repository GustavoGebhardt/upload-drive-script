package main

import (
	"upload-drive-script/internal/config"
	"upload-drive-script/internal/handlers"
	"upload-drive-script/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.MaxMultipartMemory = 500 << 20

	if config.AuthenticationMode() == config.AuthModeOAuth {
		r.GET("/auth", handlers.Auth)
		r.GET("/oauth2callback", handlers.OAuth2Callback)
	}
	r.POST("/upload", handlers.Upload)
	r.POST("/upload-url", handlers.UploadURL)
	r.GET("/uploads/:filename", handlers.GetUploadedFile)

	if err := r.Run(config.ServerPort()); err != nil {
		logger.Error("erro ao iniciar servidor: " + err.Error())
	}
}
