package assistant

import "github.com/gin-gonic/gin"

func RegisterDemoRoutes(routes *gin.RouterGroup, service *Service) {
	routes.POST("/assistant/chat", service.postDemoChat)
}
