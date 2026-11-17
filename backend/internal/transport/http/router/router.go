package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/integration/assistant"
	"rantauplus/backend/internal/transport/http/handler"
)

func New(demoEnabled bool, aiClient *assistant.Client) http.Handler {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})
	if demoEnabled {
		router.GET("/api/v1/demo/finance/summary", handler.DemoFinanceSummary)
		if aiClient != nil {
			router.POST("/api/v1/demo/assistant/chat", handler.DemoAssistantChat(aiClient))
		}
	}
	return router
}
