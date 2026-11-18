package finance

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"rantauplus/backend/internal/modules/auth"
)

func (s *SetupService) getSetup(c *gin.Context) {
	identity, _ := auth.SessionIdentity(c)
	result, err := s.load(c.Request.Context(), identity.Account.ID, userToday(time.Now(), identity.Account.Timezone))
	setupResponse(c, result, err)
}

func (s *SetupService) getSummary(c *gin.Context) {
	identity, _ := auth.SessionIdentity(c)
	result, err := s.summary(c.Request.Context(), identity.Account.ID, userToday(time.Now(), identity.Account.Timezone))
	if err == nil && result == nil {
		c.Status(http.StatusNoContent)
		return
	}
	setupResponse(c, result, err)
}

func (s *SetupService) previewSetup(c *gin.Context) {
	input, ok := bindSetup(c)
	if !ok {
		return
	}
	identity, _ := auth.SessionIdentity(c)
	result, err := s.preview(c.Request.Context(), identity.Account.ID, userToday(time.Now(), identity.Account.Timezone), input)
	setupResponse(c, result, err)
}

func (s *SetupService) saveSetup(c *gin.Context) {
	input, ok := bindSetup(c)
	if !ok {
		return
	}
	identity, _ := auth.SessionIdentity(c)
	result, err := s.save(c.Request.Context(), identity.Account.ID, userToday(time.Now(), identity.Account.Timezone), input)
	setupResponse(c, result, err)
}

func bindSetup(c *gin.Context) (setupInput, bool) {
	body := http.MaxBytesReader(c.Writer, c.Request.Body, 32768)
	defer body.Close()
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	var input *setupInput
	if err := decoder.Decode(&input); err != nil || input == nil || decoder.Decode(new(any)) != io.EOF {
		setupResponse(c, nil, setupValidation("Format pengaturan tidak valid."))
		return setupInput{}, false
	}
	return *input, true
}

func setupResponse(c *gin.Context, result any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"data": result})
		return
	}
	status, code, message := http.StatusInternalServerError, "FINANCIAL_SETUP_ERROR", "Data keuangan gagal diproses."
	var invalid setupValidation
	switch {
	case errors.As(err, &invalid):
		status, code, message = http.StatusBadRequest, "INVALID_SETUP", invalid.Error()
	case errors.Is(err, errSetupChanged):
		status, code, message = http.StatusConflict, "SETUP_CHANGED", "Daftar kewajiban sudah berubah. Buka kembali pengaturan untuk memuat data terbaru."
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
