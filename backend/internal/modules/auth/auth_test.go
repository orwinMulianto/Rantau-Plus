package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rantauplus/backend/database"
	"rantauplus/backend/database/postgres"
)

func TestPasswordPolicyAndHash(t *testing.T) {
	p, err := newPasswords("")
	if err != nil {
		t.Fatal(err)
	}
	const password = "Lampu baca di kafe "
	for _, value := range []string{"pendek", "passwordpassword", strings.Repeat(" ", 16), strings.Repeat("a", 129)} {
		if err := p.validate(value, "student@example.test", "Student"); !errors.Is(err, errPasswordPolicy) {
			t.Errorf("weak password passed: %v", err)
		}
	}
	if err := p.validate(password, "", ""); err != nil {
		t.Fatal(err)
	}
	a, err := p.hash(t.Context(), password)
	if err != nil {
		t.Fatal(err)
	}
	b, err := p.hash(t.Context(), "Lampu baca di kafe\u0301 ")
	if err != nil {
		t.Fatal(err)
	}
	matched, err := p.matches(t.Context(), password, a)
	if err != nil || !matched {
		t.Fatalf("correct password: matched=%t err=%v", matched, err)
	}
	matched, err = p.matches(t.Context(), "Lampu baca di kaf\u00e9 ", b)
	if err != nil || !matched {
		t.Fatalf("normalized password: matched=%t err=%v", matched, err)
	}
	matched, err = p.matches(t.Context(), strings.TrimSpace(password), a)
	if err != nil || matched {
		t.Fatalf("whitespace was not retained: matched=%t err=%v", matched, err)
	}
	matched, err = p.matches(t.Context(), password, "unrecognized-hash")
	if err != nil || matched {
		t.Fatalf("unrecognized hash: matched=%t err=%v", matched, err)
	}
}

func TestTokens(t *testing.T) {
	tokens, err := issueTokens()
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken == tokens.RefreshToken || tokens.TokenType != "Bearer" {
		t.Fatal("invalid token pair")
	}
	hash, err := tokenDigest(tokens.AccessToken)
	if err != nil || hash == tokens.AccessToken || len(hash) != 64 {
		t.Fatal("invalid token digest")
	}
	if _, err := tokenDigest("malformed-token"); err == nil {
		t.Fatal("malformed token passed")
	}
	if tokens.RefreshExpiresAt.Sub(tokens.AccessExpiresAt) != sessionLifetime-accessLifetime {
		t.Fatal("token lifetimes do not match configuration")
	}
}

func TestProtectedRouteRejectsMissingSession(t *testing.T) {
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), &Service{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected private response with no-store, got HTTP %d", rec.Code)
	}
}

type mailCapture struct {
	to      string
	subject string
	body    string
}

func (m *mailCapture) Send(_ context.Context, to, subject, body string) error {
	m.to, m.subject, m.body = to, subject, body
	return nil
}

func TestAuthPostgresLifecycle(t *testing.T) {
	dsn := os.Getenv("AUTH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set AUTH_TEST_DATABASE_URL to run against PostgreSQL")
	}
	ctx := t.Context()
	pool := testPool(t, dsn)
	service, err := NewService(NewRepository(pool), bytes.Repeat([]byte{1}, 32), "")
	if err != nil {
		t.Fatal(err)
	}
	const (
		email     = "owen@example.test"
		password  = "Kertas catatan pagi"
		password2 = "Sepeda biru depan rumah"
		password3 = "Novel lama di perpustakaan"
	)
	router := gin.New()
	RegisterRoutes(router.Group("/api/v1"), service)
	call := func(method, path, access string, input any, want int) []byte {
		t.Helper()
		body, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, "/api/v1/auth"+path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if access != "" {
			req.Header.Set("Authorization", "Bearer "+access)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("%s %s: status %d, wanted %d", method, path, rec.Code, want)
		}
		return rec.Body.Bytes()
	}
	deliver := func() mailCapture {
		t.Helper()
		var captured mailCapture
		if err := service.deliverEmail(ctx, &captured, EmailLinks{}); err != nil {
			t.Fatal(err)
		}
		if captured.to != email {
			t.Fatal("mail was not delivered to the requested user")
		}
		return captured
	}
	readToken := func(message mailCapture) string {
		t.Helper()
		for _, line := range strings.Split(message.body, "\n") {
			if _, err := tokenDigest(line); err == nil {
				return line
			}
		}
		t.Fatal("expected an email challenge token")
		return ""
	}
	decodeTokens := func(raw []byte) Tokens {
		t.Helper()
		var result struct{ Data Tokens }
		if err := json.Unmarshal(raw, &result); err != nil || result.Data.AccessToken == "" || result.Data.RefreshToken == "" {
			t.Fatal("invalid token response")
		}
		return result.Data
	}
	login := func(pass string) Tokens {
		t.Helper()
		return decodeTokens(call("POST", "/login", "", loginRequest{Email: email, Password: pass}, 200))
	}

	call("POST", "/register", "", emailRequest{Email: "  OWEN@EXAMPLE.TEST  "}, 202)
	call("POST", "/login", "", loginRequest{Email: email, Password: password}, 401)
	verification := readToken(deliver())
	verify := verifyRequest{Token: verification, Name: "Owen", Password: password}
	call("POST", "/verify-email", "", verify, 204)
	call("POST", "/verify-email", "", verify, 400)
	first := login(password)
	profile := call("GET", "/me", first.AccessToken, nil, 200)
	if !bytes.Contains(profile, []byte(email)) || bytes.Contains(profile, []byte("password")) {
		t.Fatal("unexpected profile fields")
	}
	call("GET", "/sessions", first.AccessToken, nil, 200)
	rotated := decodeTokens(call("POST", "/refresh", "", refreshRequest{RefreshToken: first.RefreshToken}, 200))
	call("GET", "/me", first.AccessToken, nil, 401)
	call("GET", "/me", rotated.AccessToken, nil, 200)
	call("POST", "/refresh", "", refreshRequest{RefreshToken: first.RefreshToken}, 401)
	call("GET", "/me", rotated.AccessToken, nil, 401)
	call("POST", "/refresh", "", refreshRequest{RefreshToken: rotated.RefreshToken}, 401)

	third := login(password)
	fourth := login(password)
	missing := call("POST", "/forgot-password", "", emailRequest{Email: "missing@example.test"}, 202)
	present := call("POST", "/forgot-password", "", emailRequest{Email: email}, 202)
	if !bytes.Equal(missing, present) {
		t.Fatal("forgot-password responses revealed account existence")
	}
	expired := readToken(deliver())
	if err := pool.WithContext(ctx).Exec("UPDATE auth_email_jobs SET expires_at = now() WHERE purpose = 'RESET_PASSWORD'").Error; err != nil {
		t.Fatal(err)
	}
	call("POST", "/reset-password", "", resetRequest{Token: expired, Password: password2}, 400)
	call("POST", "/forgot-password", "", emailRequest{Email: email}, 202)
	reset := resetRequest{Token: readToken(deliver()), Password: password2}
	call("POST", "/reset-password", "", reset, 204)
	call("POST", "/reset-password", "", reset, 400)
	call("GET", "/me", third.AccessToken, nil, 401)
	call("POST", "/refresh", "", refreshRequest{RefreshToken: fourth.RefreshToken}, 401)
	call("POST", "/login", "", loginRequest{Email: email, Password: password}, 401)
	if notice := deliver(); !strings.Contains(notice.subject, "berubah") {
		t.Fatal("missing password change notice")
	}

	current := login(password2)
	call("POST", "/change-password", current.AccessToken, changePasswordRequest{CurrentPassword: password2, NewPassword: password3}, 204)
	call("GET", "/me", current.AccessToken, nil, 401)
	last := login(password3)
	call("POST", "/logout", last.AccessToken, nil, 204)
	call("POST", "/refresh", "", refreshRequest{RefreshToken: last.RefreshToken}, 401)

	deviceA := login(password3)
	deviceB := login(password3)
	var sessions struct{ Data []Session }
	if err := json.Unmarshal(call("GET", "/sessions", deviceA.AccessToken, nil, 200), &sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions.Data) != 2 {
		t.Fatalf("expected two active sessions, got %d", len(sessions.Data))
	}
	for _, session := range sessions.Data {
		if !session.Current {
			call("DELETE", "/sessions/"+session.ID, deviceA.AccessToken, nil, 204)
		}
	}
	call("GET", "/me", deviceB.AccessToken, nil, 401)
	call("POST", "/logout-all", deviceA.AccessToken, nil, 204)
	call("GET", "/me", deviceA.AccessToken, nil, 401)

	// Failed and successful login attempts share the same per-account limit.
	call("POST", "/login", "", loginRequest{Email: email, Password: "wrong password"}, 401)
	call("POST", "/login", "", loginRequest{Email: email, Password: "wrong password"}, 429)
}

func testPool(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	admin, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminPool.Close() })
	token, err := newToken()
	if err != nil {
		t.Fatal(err)
	}
	schema := "rantau_auth_" + digest(token)[:16]
	// Only the fixed prefix and hexadecimal digest enter this identifier.
	schemaSQL := "\"" + schema + "\""
	if err := admin.WithContext(ctx).Exec("CREATE SCHEMA " + schemaSQL).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := admin.WithContext(cleanup).Exec("DROP SCHEMA " + schemaSQL + " CASCADE").Error; err != nil {
			t.Error(err)
		}
	})
	schemaDSN := dsn + " search_path=" + schema
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		settings, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		query := settings.Query()
		query.Set("search_path", schema)
		settings.RawQuery = query.Encode()
		schemaDSN = settings.String()
	}
	db, err := postgres.Open(ctx, schemaDSN)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if _, err := database.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if applied, err := database.Apply(ctx, db, false); err != nil || len(applied) != 0 {
		t.Fatalf("migration rerun applied %v, error %v", applied, err)
	}
	if err := database.CheckReady(ctx, db); err != nil {
		t.Fatal(err)
	}
	return db
}
