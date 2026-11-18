package finance

import "github.com/gin-gonic/gin"

func RegisterDemoRoutes(routes *gin.RouterGroup) {
	routes.GET("/finance/summary", getDemoSummary)
}
