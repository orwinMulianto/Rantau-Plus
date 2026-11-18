package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"rantauplus/backend/database/entities"
)

func (r *Repository) createSession(ctx context.Context, user credentials, tokens Tokens) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lockedID string
		err := tx.Raw(`
			SELECT id::text FROM users
			WHERE id = ?::text::uuid AND password_hash = ? AND email_verified_at IS NOT NULL
			FOR UPDATE`, user.id, user.passwordHash).Row().Scan(&lockedID)
		if errors.Is(err, sql.ErrNoRows) {
			return errCredentials
		}
		if err != nil {
			return err
		}
		var sessionID string
		err = tx.Raw(`
			INSERT INTO auth_sessions (user_id, access_token_hash, access_expires_at, expires_at)
			VALUES (?::text::uuid, ?, ?, ?) RETURNING id::text`,
			lockedID, digest(tokens.AccessToken), tokens.AccessExpiresAt, tokens.RefreshExpiresAt).Row().Scan(&sessionID)
		if err != nil {
			return err
		}
		return insertRefreshToken(tx, lockedID, sessionID, tokens)
	})
}

func insertRefreshToken(tx *gorm.DB, userID, sessionID string, tokens Tokens) error {
	return tx.Exec(`
		INSERT INTO refresh_tokens (user_id, session_id, token_hash, expires_at)
		VALUES (?::text::uuid, ?::text::uuid, ?, ?)`, userID, sessionID, digest(tokens.RefreshToken), tokens.RefreshExpiresAt).Error
}

func (r *Repository) rotateSession(ctx context.Context, hash string, tokens Tokens) (Tokens, error) {
	reused := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var userID, sessionID string
		var expiry time.Time
		err := tx.Raw(`
			SELECT s.user_id::text, s.id::text, s.expires_at
			FROM refresh_tokens AS token JOIN auth_sessions AS s ON token.session_id = s.id
			WHERE token.token_hash = ? AND s.revoked_at IS NULL AND s.expires_at > clock_timestamp()
			FOR UPDATE OF s`, hash).Row().Scan(&userID, &sessionID, &expiry)
		if errors.Is(err, sql.ErrNoRows) {
			return errSession
		}
		if err != nil {
			return err
		}

		// The session lock serializes rotation. A separate statement sees changes
		// committed by the previous request even if this one waited on the lock.
		var used bool
		err = tx.Raw(`
			SELECT used_at IS NOT NULL FROM refresh_tokens
			WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > clock_timestamp()`, hash).Row().Scan(&used)
		if errors.Is(err, sql.ErrNoRows) {
			return errSession
		}
		if err != nil {
			return err
		}
		if used {
			reused = true
			return revokeSession(tx, userID, sessionID)
		}

		tokens.RefreshExpiresAt = expiry
		if tokens.AccessExpiresAt.After(expiry) {
			tokens.AccessExpiresAt = expiry
		}
		if err := tx.Model(&entities.RefreshToken{}).Where("token_hash = ?", hash).
			UpdateColumn("used_at", gorm.Expr("now()")).Error; err != nil {
			return err
		}
		if err := tx.Model(&entities.AuthSession{}).Where("id = ?::text::uuid", sessionID).
			UpdateColumns(map[string]any{
				"access_token_hash": digest(tokens.AccessToken),
				"access_expires_at": tokens.AccessExpiresAt,
				"refreshed_at":      gorm.Expr("now()"),
			}).Error; err != nil {
			return err
		}
		return insertRefreshToken(tx, userID, sessionID, tokens)
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err == nil && reused {
		err = errSession
	}
	if err != nil {
		return Tokens{}, err
	}
	return tokens, nil
}

func (r *Repository) authenticate(ctx context.Context, hash string) (Identity, error) {
	var identity Identity
	account := &identity.Account
	err := r.db.WithContext(ctx).Raw(`
		SELECT u.id::text, u.name, u.email, u.timezone, u.currency, u.email_verified_at, s.id::text
		FROM auth_sessions AS s JOIN users AS u ON s.user_id = u.id
		WHERE s.access_token_hash = ? AND s.revoked_at IS NULL AND s.access_expires_at > now()
			AND s.expires_at > now() AND u.email_verified_at IS NOT NULL`, hash).
		Row().Scan(&account.ID, &account.Name, &account.Email, &account.Timezone, &account.Currency, &account.VerifiedAt, &identity.SessionID)
	if errors.Is(err, sql.ErrNoRows) {
		return Identity{}, errSession
	}
	return identity, err
}

func (r *Repository) listSessions(ctx context.Context, identity Identity) ([]Session, error) {
	sessions := []Session{}
	err := r.db.WithContext(ctx).Raw(`
		SELECT id::text, created_at, refreshed_at, expires_at, id = ?::text::uuid AS "current"
		FROM auth_sessions WHERE user_id = ?::text::uuid AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC, id DESC`, identity.SessionID, identity.Account.ID).Scan(&sessions).Error
	return sessions, err
}

func revokeSession(db *gorm.DB, userID, sessionID string) error {
	return db.Model(&entities.AuthSession{}).
		Where("user_id = ?::text::uuid AND id = ?::text::uuid AND revoked_at IS NULL", userID, sessionID).
		UpdateColumn("revoked_at", gorm.Expr("now()")).Error
}

func revokeAll(db *gorm.DB, userID string) error {
	return db.Model(&entities.AuthSession{}).
		Where("user_id = ?::text::uuid AND revoked_at IS NULL AND expires_at > now()", userID).
		UpdateColumn("revoked_at", gorm.Expr("now()")).Error
}
