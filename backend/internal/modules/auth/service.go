package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	errInput            = errors.New("invalid input")
	errPasswordPolicy   = errors.New("password does not meet policy")
	errCredentials      = errors.New("invalid credentials")
	errInvalidChallenge = errors.New("invalid email challenge")
	errSession          = errors.New("invalid session")
	errRateLimited      = errors.New("too many requests")
	errBusy             = errors.New("auth service busy")
)

const (
	accessLifetime  = 15 * time.Minute
	sessionLifetime = 30 * 24 * time.Hour
)

type Service struct {
	repo     *Repository
	password *passwords
	emailKey []byte
}

func NewService(repo *Repository, emailKey []byte, blocklistPath string) (*Service, error) {
	if len(emailKey) != tokenBytes {
		return nil, errors.New("AUTH_EMAIL_TOKEN_KEY harus berisi 32 byte")
	}
	password, err := newPasswords(blocklistPath)
	if err != nil {
		return nil, err
	}
	return &Service{repo: repo, password: password, emailKey: append([]byte(nil), emailKey...)}, nil
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if len(email) > 254 || strings.IndexFunc(email, func(c rune) bool {
		return c <= ' ' || c >= 127
	}) >= 0 {
		return "", errInput
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Name != "" || address.Address != email || strings.Count(email, "@") != 1 {
		return "", errInput
	}
	return email, nil
}

func (s *Service) requestChallenge(ctx context.Context, ip, address string, purpose challengePurpose) error {
	scope := "email:" + string(purpose)
	if err := s.repo.allow(ctx, scope+":ip", ip, 60*60, 20); err != nil {
		return err
	}
	email, err := normalizeEmail(address)
	if err != nil {
		return err
	}
	if err := s.repo.allow(ctx, scope+":email", email, 60*60, 3); err != nil {
		return err
	}
	lifetime := 24 * time.Hour
	if purpose == resetPassword {
		lifetime = 30 * time.Minute
	}
	nonce, err := randomBytes(tokenBytes)
	if err != nil {
		return err
	}
	token := emailToken(s.emailKey, nonce)
	return s.repo.enqueue(ctx, email, purpose, digest(token), nonce, time.Now().UTC().Add(lifetime))
}

func (s *Service) verify(ctx context.Context, ip string, input verifyRequest) error {
	if err := s.repo.allow(ctx, "verify:ip", ip, 15*60, 20); err != nil {
		return err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || utf8.RuneCountInString(name) > 100 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return errInput
	}
	hash, passwordHash, err := s.challengePassword(ctx, input.Token, input.Password, name, verifyEmail)
	if err != nil {
		return err
	}
	return s.repo.finishRegistration(ctx, hash, name, passwordHash)
}

func (s *Service) reset(ctx context.Context, ip string, input resetRequest) error {
	if err := s.repo.allow(ctx, "reset:ip", ip, 15*60, 20); err != nil {
		return err
	}
	hash, passwordHash, err := s.challengePassword(ctx, input.Token, input.Password, "", resetPassword)
	if err != nil {
		return err
	}
	return s.repo.finishReset(ctx, hash, passwordHash)
}

func (s *Service) challengePassword(ctx context.Context, token, value, name string, purpose challengePurpose) (string, string, error) {
	hash, err := tokenDigest(token)
	if err != nil {
		return "", "", errInvalidChallenge
	}
	email, err := s.repo.pendingChallengeEmail(ctx, hash, purpose)
	if err != nil {
		return "", "", err
	}
	if purpose == resetPassword {
		user, err := s.repo.credentials(ctx, email)
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", errInvalidChallenge
		}
		if err != nil {
			return "", "", err
		}
		name = user.name
	}
	if err := s.password.validate(value, email, name); err != nil {
		return "", "", err
	}
	passwordHash, err := s.password.hash(ctx, value)
	return hash, passwordHash, err
}

func (s *Service) login(ctx context.Context, ip string, input loginRequest) (Tokens, error) {
	if err := s.repo.allow(ctx, "login:ip", ip, 15*60, 60); err != nil {
		return Tokens{}, err
	}
	email, err := normalizeEmail(input.Email)
	if err != nil || input.Password == "" || len(input.Password) > 1024 {
		return Tokens{}, errCredentials
	}
	if err := s.repo.allow(ctx, "login:email", email, 15*60, 10); err != nil {
		return Tokens{}, err
	}
	user, err := s.repo.credentials(ctx, email)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Tokens{}, err
	}
	matched, err := s.password.matches(ctx, input.Password, user.passwordHash)
	if err != nil {
		return Tokens{}, err
	}
	if !found || !matched {
		return Tokens{}, errCredentials
	}
	tokens, err := issueTokens()
	if err != nil {
		return Tokens{}, err
	}
	if err := s.repo.createSession(ctx, user, tokens); err != nil {
		return Tokens{}, err
	}
	return tokens, nil
}

func issueTokens() (Tokens, error) {
	access, err := newToken()
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := newToken()
	if err != nil {
		return Tokens{}, err
	}
	now := time.Now().UTC()
	return Tokens{
		AccessToken: access, RefreshToken: refresh, TokenType: "Bearer",
		AccessExpiresAt: now.Add(accessLifetime), RefreshExpiresAt: now.Add(sessionLifetime),
	}, nil
}

func (s *Service) refresh(ctx context.Context, ip, token string) (Tokens, error) {
	if err := s.repo.allow(ctx, "refresh:ip", ip, 60, 120); err != nil {
		return Tokens{}, err
	}
	hash, err := tokenDigest(token)
	if err != nil {
		return Tokens{}, errSession
	}
	tokens, err := issueTokens()
	if err != nil {
		return Tokens{}, err
	}
	return s.repo.rotateSession(ctx, hash, tokens)
}

func (s *Service) authenticate(ctx context.Context, token string) (Identity, error) {
	hash, err := tokenDigest(token)
	if err != nil {
		return Identity{}, errSession
	}
	return s.repo.authenticate(ctx, hash)
}

func (s *Service) revoke(ctx context.Context, identity Identity, sessionID string) error {
	var id pgtype.UUID
	if len(sessionID) != 36 || id.Scan(sessionID) != nil || id.String() != strings.ToLower(sessionID) {
		return errInput
	}
	return revokeSession(s.repo.db.WithContext(ctx), identity.Account.ID, id.String())
}

func (s *Service) logoutAll(ctx context.Context, identity Identity) error {
	return revokeAll(s.repo.db.WithContext(ctx), identity.Account.ID)
}

func (s *Service) changePassword(ctx context.Context, identity Identity, input changePasswordRequest) error {
	account := identity.Account
	if err := s.repo.allow(ctx, "change-password:user", account.ID, 15*60, 10); err != nil {
		return err
	}
	if len(input.CurrentPassword) > 1024 || input.CurrentPassword == "" {
		return errCredentials
	}
	email := strings.ToLower(account.Email)
	user, err := s.repo.credentials(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return errCredentials
	}
	if err != nil {
		return err
	}
	matched, err := s.password.matches(ctx, input.CurrentPassword, user.passwordHash)
	if err != nil {
		return err
	}
	if !matched || user.id != account.ID {
		return errCredentials
	}
	if err := s.password.validate(input.NewPassword, email, account.Name); err != nil {
		return err
	}
	hash, err := s.password.hash(ctx, input.NewPassword)
	if err != nil {
		return err
	}
	return s.repo.changePassword(ctx, user.id, email, user.passwordHash, hash)
}
