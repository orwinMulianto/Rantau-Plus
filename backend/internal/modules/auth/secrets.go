package auth

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/text/unicode/norm"
)

const (
	tokenBytes      = 32
	saltBytes       = 16
	passwordProfile = "$argon2id$v=19$m=19456,t=2,p=1$"
)

type passwords struct {
	blocked map[string]struct{}
	slots   chan struct{}
	dummy   string
}

func newPasswords(blocklistPath string) (*passwords, error) {
	p := &passwords{blocked: make(map[string]struct{}), slots: make(chan struct{}, 4)}
	for _, password := range []string{
		"123456789012345", "1234567890123456", "passwordpassword",
		"password123456789", "qwertyuiopasdfgh", "qwerty123456789", "letmeinletmein123",
	} {
		p.blocked[password] = struct{}{}
	}
	if blocklistPath != "" {
		file, err := os.Open(blocklistPath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			p.blocked[passwordKey(scanner.Text())] = struct{}{}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	dummyPassword, err := randomBytes(tokenBytes)
	if err != nil {
		return nil, err
	}
	p.dummy, err = p.hash(context.Background(), base64.RawURLEncoding.EncodeToString(dummyPassword))
	return p, err
}

func passwordKey(value string) string {
	return strings.ToLower(norm.NFC.String(value))
}

func (p *passwords) validate(value, email, name string) error {
	if len(value) > 1024 || !utf8.ValidString(value) {
		return errPasswordPolicy
	}
	key := passwordKey(value)
	length := utf8.RuneCountInString(norm.NFC.String(value))
	if length < 15 || length > 128 || strings.TrimSpace(key) == "" {
		return errPasswordPolicy
	}
	localPart, _, _ := strings.Cut(email, "@")
	if _, blocked := p.blocked[key]; blocked || key == email || key == localPart || key == passwordKey(name) {
		return errPasswordPolicy
	}
	return nil
}

func (p *passwords) hash(ctx context.Context, value string) (string, error) {
	salt, err := randomBytes(saltBytes)
	if err != nil {
		return "", err
	}
	key, err := p.derive(ctx, value, salt)
	if err != nil {
		return "", err
	}
	return passwordProfile + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}

func (p *passwords) matches(ctx context.Context, value, encoded string) (bool, error) {
	salt, expected, valid := decodePassword(encoded)
	if !valid {
		salt, expected, _ = decodePassword(p.dummy)
	}
	actual, err := p.derive(ctx, value, salt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1 && valid, nil
}

func decodePassword(encoded string) (salt, key []byte, ok bool) {
	contents, hasProfile := strings.CutPrefix(encoded, passwordProfile)
	saltText, keyText, separated := strings.Cut(contents, "$")
	salt, saltErr := base64.RawStdEncoding.Strict().DecodeString(saltText)
	key, keyErr := base64.RawStdEncoding.Strict().DecodeString(keyText)
	return salt, key, hasProfile && separated && saltErr == nil && keyErr == nil && len(salt) == saltBytes && len(key) == tokenBytes
}

func (p *passwords) derive(ctx context.Context, value string, salt []byte) ([]byte, error) {
	select {
	case p.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, errBusy
	}
	defer func() { <-p.slots }()
	return argon2.IDKey(norm.NFC.Bytes([]byte(value)), salt, 2, 19*1024, 1, tokenBytes), nil
}

func randomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	_, err := rand.Read(bytes)
	return bytes, err
}

func newToken() (string, error) {
	value, err := randomBytes(tokenBytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func emailToken(key, nonce []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("rantauplus-email-token-v1:"))
	mac.Write(nonce)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func tokenDigest(value string) (string, error) {
	if len(value) != base64.RawURLEncoding.EncodedLen(tokenBytes) {
		return "", errors.New("invalid token length")
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(raw) != tokenBytes || base64.RawURLEncoding.EncodeToString(raw) != value {
		return "", errors.New("invalid token encoding")
	}
	return digest(value), nil
}

func digest(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
