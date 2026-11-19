package httpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"rantauplus/backend/database/entities"
	"rantauplus/backend/database/postgres"
	assistantclient "rantauplus/backend/internal/integration/assistant"
	"rantauplus/backend/internal/modules/assistant"
	"rantauplus/backend/internal/modules/auth"
	"rantauplus/backend/internal/modules/finance"
)

// Opt-in local end-to-end test. All account and financial fixtures are rolled back.
func TestLiveCompanionPurchase(t *testing.T) {
	dsn, aiURL := os.Getenv("COMPANION_TEST_DATABASE_URL"), os.Getenv("COMPANION_TEST_AI_URL")
	if dsn == "" || aiURL == "" {
		t.Skip("set COMPANION_TEST_DATABASE_URL and COMPANION_TEST_AI_URL for the live test")
	}
	db, err := postgres.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal("test database unavailable")
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	now := time.Now()
	user := entities.User{Name: "Companion test", Email: rand.Text() + "@example.test", PasswordHash: "unused", EmailVerifiedAt: &now}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	raw := make([]byte, 32)
	rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	if err := tx.Create(&entities.AuthSession{UserID: user.ID, AccessTokenHash: hex.EncodeToString(hash[:]), AccessExpiresAt: now.Add(time.Hour), ExpiresAt: now.Add(time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`INSERT INTO financial_periods(user_id,start_date,next_income_date,opening_balance,reserve_amount)
		VALUES (?,(now() at time zone 'Asia/Jakarta')::date,(now() at time zone 'Asia/Jakarta')::date+10,500000,0)`, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	client, err := assistantclient.NewClient(aiURL)
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := auth.NewService(auth.NewRepository(tx), raw, "")
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	router := New(false, assistant.NewService(client), nil, sessions, finance.NewSetupService(tx))
	chat := func(message string) assistant.Answer {
		t.Helper()
		body, _ := json.Marshal(map[string]string{"message": message})
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/assistant/chat", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var result struct{ Data assistant.Answer }
		if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &result) != nil {
			t.Fatalf("chat HTTP %d: %s", rec.Code, rec.Body.String())
		}
		return result.Data
	}
	if reply := chat("Aku cuma mau cerita. Aku lagi kangen rumah."); reply.Mode != "conversation" || reply.Answer == "" {
		t.Fatalf("mode curhat = %q", reply.Mode)
	}
	reply := chat("Kalau beli buku Rp300 ribu, uangku cukup sampai gajian?")
	if reply.Simulation == nil || reply.Simulation.AvailableAfter != 200000 || reply.Simulation.DailyLimitAfter != 20000 {
		t.Fatalf("unexpected simulation: %+v", reply)
	}
}
