package assistant

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/modules/auth"
)

func RegisterDemoRoutes(routes *gin.RouterGroup, service *Service) {
	routes.POST("/assistant/chat", service.postDemoChat)
}

func RegisterPersonalRoutes(api *gin.RouterGroup, sessions *auth.Service, guide *Service, reader FinanceReader) {
	routes := api.Group("/assistant", func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		c.Next()
	}, sessions.RequireSession)
	routes.POST("/chat", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		var input struct {
			Message  string    `json:"message"`
			Location *Location `json:"location"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code": "INVALID_JSON", "message": "Format JSON tidak valid.",
			}})
			return
		}
		identity, _ := auth.SessionIdentity(c)
		if !input.Location.valid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "INVALID_LOCATION", "message": "Koordinat lokasi tidak valid."}})
			return
		}
		answer, err := guide.converse(c.Request.Context(), reader, identity.Account.ID, identity.SessionID, identity.Account.Timezone, input.Message, input.Location)
		switch {
		case errors.Is(err, errInvalidMessage):
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"code": "INVALID_MESSAGE", "message": "Isi pesan dengan 1-1000 karakter.",
			}})
		case errors.Is(err, errGuideUnavailable):
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{
				"code": "GUIDE_UNAVAILABLE", "message": "Panduan sedang tidak tersedia. Coba lagi nanti.",
			}})
		case errors.Is(err, errModel):
			c.JSON(http.StatusBadGateway, gin.H{"error": gin.H{"code": "COMPANION_UNAVAILABLE", "message": "AI Companion sedang tidak tersedia. Coba lagi nanti."}})
		case err != nil:
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"code": "SUMMARY_ERROR", "message": "Ringkasan keuangan gagal dimuat. Coba lagi.",
			}})
		default:
			c.JSON(http.StatusOK, gin.H{"data": answer})
		}
	})
}
