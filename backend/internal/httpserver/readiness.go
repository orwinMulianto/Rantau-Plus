package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func readiness(checkDatabase func(context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if checkDatabase == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"data": gin.H{
				"status": "not_ready", "database": "not_configured",
			}})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		if err := checkDatabase(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"data": gin.H{
				"status": "not_ready", "database": "unavailable",
			}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ready", "database": "ready"}})
	}
}
