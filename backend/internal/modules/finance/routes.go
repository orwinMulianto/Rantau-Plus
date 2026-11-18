package finance

import (
	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/modules/auth"
)

func RegisterDemoRoutes(routes *gin.RouterGroup) {
	routes.GET("/finance/summary", getDemoSummary)
}

func RegisterSetupRoutes(api *gin.RouterGroup, sessions *auth.Service, setup *SetupService) {
	routes := api.Group("/finance", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}, sessions.RequireSession)
	routes.GET("/setup", setup.getSetup)
	routes.POST("/setup/preview", setup.previewSetup)
	routes.PUT("/setup", setup.saveSetup)
	routes.GET("/summary", setup.getSummary)
}
