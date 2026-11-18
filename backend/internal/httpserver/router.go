package httpserver

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/modules/assistant"
	"rantauplus/backend/internal/modules/finance"
)

func New(demoEnabled bool, assistantService *assistant.Service, checkDatabase func(context.Context) error) http.Handler {
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ok"}})
	})
	router.GET("/readyz", readiness(checkDatabase))
	if demoEnabled {
		demoRoutes := router.Group("/api/v1/demo")
		finance.RegisterDemoRoutes(demoRoutes)
		if assistantService != nil {
			assistant.RegisterDemoRoutes(demoRoutes, assistantService)
		}
	}
	return router
}
