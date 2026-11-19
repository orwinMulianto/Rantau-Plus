package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(api *gin.RouterGroup, service *Service) {
	routes := api.Group("/auth", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	})
	routes.POST("/register", service.emailHandler(verifyEmail))
	routes.POST("/verify-email", service.verifyHandler)
	routes.POST("/resend-verification", service.emailHandler(verifyEmail))
	routes.POST("/login", service.loginHandler)
	routes.POST("/refresh", service.refreshHandler)
	routes.POST("/forgot-password", service.emailHandler(resetPassword))
	routes.POST("/reset-password", service.resetHandler)

	private := routes.Group("", service.RequireSession)
	private.GET("/me", service.meHandler)
	private.GET("/sessions", service.sessionsHandler)
	private.DELETE("/sessions/:id", service.revokeHandler)
	private.POST("/logout", service.logoutHandler)
	private.POST("/logout-all", service.logoutAllHandler)
	private.POST("/change-password", service.passwordHandler)
}
