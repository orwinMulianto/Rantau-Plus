package finance

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func getDemoSummary(c *gin.Context) {
	summary, err := demoSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code": "SUMMARY_ERROR", "message": "Ringkasan demo gagal diproses.",
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary, "meta": gin.H{"demo": true}})
}
