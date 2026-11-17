package handler

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/integration/assistant"
)

func DemoAssistantChat(client *assistant.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		var input assistant.Request
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code": "INVALID_JSON", "message": "Format JSON tidak valid.",
			}})
			return
		}
		input.Message = strings.TrimSpace(input.Message)
		if input.Message == "" || utf8.RuneCountInString(input.Message) > 1000 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code": "INVALID_MESSAGE", "message": "Isi pesan dengan 1–1000 karakter.",
			}})
			return
		}
		answer, err := client.Chat(c.Request.Context(), input)
		switch {
		case errors.Is(err, assistant.ErrUnknownSkill):
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
}
