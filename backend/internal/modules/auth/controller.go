package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const identityKey = "rantauplus.auth.identity"

func (s *Service) RequireSession(c *gin.Context) {
	scheme, token, ok := strings.Cut(c.GetHeader("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		fail(c, errSession)
		c.Abort()
		return
	}
	identity, err := s.authenticate(c.Request.Context(), token)
	if err != nil {
		fail(c, err)
		c.Abort()
		return
	}
	c.Set(identityKey, identity)
	c.Next()
}

// SessionIdentity supplies the user ID authenticated by RequireSession.
func SessionIdentity(c *gin.Context) (Identity, bool) {
	value, _ := c.Get(identityKey)
	identity, ok := value.(Identity)
	return identity, ok
}

func bind(c *gin.Context, input any) bool {
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": gin.H{
			"code": "UNSUPPORTED_MEDIA_TYPE", "message": "Gunakan format application/json.",
		}})
		return false
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil || decoder.Decode(new(any)) != io.EOF {
		fail(c, errInput)
		return false
	}
	return true
}

func fail(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code, message := "AUTH_ERROR", "Permintaan autentikasi gagal diproses."
	switch {
	case errors.Is(err, errInput):
		status, code, message = http.StatusBadRequest, "INVALID_INPUT", "Periksa kembali data yang dikirim."
	case errors.Is(err, errPasswordPolicy):
		status, code, message = http.StatusBadRequest, "PASSWORD_POLICY", "Pilih password 15–128 karakter yang sulit ditebak."
	case errors.Is(err, errCredentials):
		status, code, message = http.StatusUnauthorized, "INVALID_CREDENTIALS", "Kredensial tidak valid."
	case errors.Is(err, errSession):
		status, code, message = http.StatusUnauthorized, "INVALID_SESSION", "Sesi tidak valid atau sudah berakhir."
		c.Header("WWW-Authenticate", "Bearer")
	case errors.Is(err, errInvalidChallenge):
		status, code, message = http.StatusBadRequest, "INVALID_TOKEN", "Token tidak valid atau sudah berakhir."
	case errors.Is(err, errRateLimited):
		status, code, message = http.StatusTooManyRequests, "RATE_LIMITED", "Terlalu banyak percobaan. Coba lagi nanti."
	case errors.Is(err, errBusy):
		status, code, message = http.StatusServiceUnavailable, "AUTH_BUSY", "Layanan sedang sibuk. Coba lagi sebentar."
	default:
		log.Printf("Rantau+ auth: %s %s failed (%T)", c.Request.Method, c.FullPath(), err)
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

func (s *Service) emailHandler(purpose challengePurpose) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input emailRequest
		if !bind(c, &input) {
			return
		}
		if err := s.requestChallenge(c.Request.Context(), c.ClientIP(), input.Email, purpose); err != nil {
			fail(c, err)
			return
		}
		c.JSON(http.StatusAccepted, gin.H{"data": gin.H{
			"message": "Jika email memenuhi syarat, instruksi akan dikirim.",
		}})
	}
}

func (s *Service) verifyHandler(c *gin.Context) {
	var input verifyRequest
	if bind(c, &input) {
		finish(c, s.verify(c.Request.Context(), c.ClientIP(), input))
	}
}

func (s *Service) resetHandler(c *gin.Context) {
	var input resetRequest
	if bind(c, &input) {
		finish(c, s.reset(c.Request.Context(), c.ClientIP(), input))
	}
}

func (s *Service) loginHandler(c *gin.Context) {
	var input loginRequest
	if bind(c, &input) {
		tokens, err := s.login(c.Request.Context(), c.ClientIP(), input)
		respond(c, tokens, err)
	}
}

func (s *Service) refreshHandler(c *gin.Context) {
	var input refreshRequest
	if bind(c, &input) {
		tokens, err := s.refresh(c.Request.Context(), c.ClientIP(), input.RefreshToken)
		respond(c, tokens, err)
	}
}

func (s *Service) meHandler(c *gin.Context) {
	identity, _ := SessionIdentity(c)
	respond(c, identity.Account, nil)
}

func (s *Service) sessionsHandler(c *gin.Context) {
	identity, _ := SessionIdentity(c)
	sessions, err := s.repo.listSessions(c.Request.Context(), identity)
	respond(c, sessions, err)
}

func (s *Service) revokeHandler(c *gin.Context) {
	identity, _ := SessionIdentity(c)
	finish(c, s.revoke(c.Request.Context(), identity, c.Param("id")))
}

func (s *Service) logoutHandler(c *gin.Context) {
	identity, _ := SessionIdentity(c)
	finish(c, s.revoke(c.Request.Context(), identity, identity.SessionID))
}

func (s *Service) logoutAllHandler(c *gin.Context) {
	identity, _ := SessionIdentity(c)
	finish(c, s.logoutAll(c.Request.Context(), identity))
}

func (s *Service) passwordHandler(c *gin.Context) {
	var input changePasswordRequest
	if bind(c, &input) {
		identity, _ := SessionIdentity(c)
		finish(c, s.changePassword(c.Request.Context(), identity, input))
	}
}

func respond(c *gin.Context, data any, err error) {
	if err != nil {
		fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func finish(c *gin.Context, err error) {
	if err != nil {
		fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
