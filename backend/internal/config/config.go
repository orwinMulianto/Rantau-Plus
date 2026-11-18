package config

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address      string
	DatabaseURL  string
	AIServiceURL string
	DemoEnabled  bool
	Auth         *AuthConfig
}

type AuthConfig struct {
	EmailTokenKey     []byte
	PasswordBlocklist string
	SMTPAddr          string
	SMTPFrom          string
	SMTPUsername      string
	SMTPPassword      string
	SMTPPlaintext     bool
	VerifyURL         *url.URL
	ResetURL          *url.URL
	TrustedProxies    []string
}

func Load() (Config, error) {
	cfg := Config{
		Address:      os.Getenv("APP_ADDR"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		AIServiceURL: os.Getenv("AI_SERVICE_URL"),
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:8080"
	}
	if value, present := os.LookupEnv("RANTAU_DEMO"); present {
		enabled, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("RANTAU_DEMO harus berupa boolean: %w", err)
		}
		cfg.DemoEnabled = enabled
	}
	if smtpAddr := os.Getenv("AUTH_SMTP_ADDR"); smtpAddr != "" {
		auth, err := loadAuth(cfg.DatabaseURL, smtpAddr)
		if err != nil {
			return Config{}, err
		}
		cfg.Auth = auth
	}
	return cfg, nil
}

func loadAuth(databaseURL, smtpAddr string) (*AuthConfig, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL harus diisi untuk mengaktifkan autentikasi")
	}
	key, err := base64.StdEncoding.Strict().DecodeString(os.Getenv("AUTH_EMAIL_TOKEN_KEY"))
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("AUTH_EMAIL_TOKEN_KEY harus berisi 32 byte acak dalam format Base64")
	}
	plaintext := false
	if value := os.Getenv("AUTH_SMTP_LOCAL_PLAINTEXT"); value != "" {
		plaintext, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("AUTH_SMTP_LOCAL_PLAINTEXT harus berupa boolean")
		}
	}
	verifyURL, err := emailLink("AUTH_VERIFY_URL")
	if err != nil {
		return nil, err
	}
	resetURL, err := emailLink("AUTH_RESET_URL")
	if err != nil {
		return nil, err
	}
	return &AuthConfig{
		EmailTokenKey: key, PasswordBlocklist: os.Getenv("AUTH_PASSWORD_BLOCKLIST_FILE"),
		SMTPAddr: smtpAddr, SMTPFrom: os.Getenv("AUTH_SMTP_FROM"),
		SMTPUsername: os.Getenv("AUTH_SMTP_USERNAME"), SMTPPassword: os.Getenv("AUTH_SMTP_PASSWORD"),
		SMTPPlaintext: plaintext, VerifyURL: verifyURL, ResetURL: resetURL,
		TrustedProxies: strings.Fields(os.Getenv("AUTH_TRUSTED_PROXIES")),
	}, nil
}

func emailLink(key string) (*url.URL, error) {
	value := os.Getenv(key)
	if value == "" {
		return nil, nil
	}
	link, err := url.Parse(value)
	if err != nil || link.Scheme != "https" || link.Hostname() == "" || link.User != nil || link.Fragment != "" {
		return nil, fmt.Errorf("%s harus berupa URL HTTPS tanpa fragment atau kredensial", key)
	}
	return link, nil
}
