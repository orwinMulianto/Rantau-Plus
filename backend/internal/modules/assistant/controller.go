package assistant

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Service) postDemoChat(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	var input Request
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code": "INVALID_JSON", "message": "Format JSON tidak valid.",
		}})
		return
	}
	answer, err := s.chat(c.Request.Context(), input)
	switch {
	case errors.Is(err, errInvalidMessage):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code": "INVALID_MESSAGE", "message": "Isi pesan dengan 1-1000 karakter.",
		}})
	case errors.Is(err, ErrUnknownSkill):
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
			"code": "UNKNOWN_SKILL", "message": "Skill belum tersedia.",
		}})
	case err != nil:
		c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{
			"code": "AI_SERVICE_UNAVAILABLE", "message": "AI Service sedang tidak tersedia.",
		}})
	default:
		c.JSON(http.StatusOK, gin.H{"data": answer, "meta": gin.H{"demo": true}})
	}
}
