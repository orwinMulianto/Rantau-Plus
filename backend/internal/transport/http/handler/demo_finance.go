package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/service"
)

func DemoFinanceSummary(c *gin.Context) {
	summary, err := service.DemoFinanceSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
			"code": "SUMMARY_ERROR", "message": "Ringkasan demo gagal diproses.",
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": summary, "meta": gin.H{"demo": true}})
}
