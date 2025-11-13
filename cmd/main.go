package main

import (
	"net/http"

	"upload-drive-script/internal/config"
	"upload-drive-script/internal/handlers"
	"upload-drive-script/pkg/logger"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.MaxMultipartMemory = 500 << 20
	r.Use(allowAllCORS())

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

func allowAllCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", "*")
		headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Origin,Accept")
		headers.Set("Access-Control-Expose-Headers", "Content-Disposition")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
