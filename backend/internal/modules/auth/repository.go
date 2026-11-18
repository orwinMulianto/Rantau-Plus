package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gorm.io/gorm"

	"rantauplus/backend/database/entities"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

type credentials struct {
	id           string
	name         string
	email        string
	passwordHash string
}

type challengePurpose string

const (
	verifyEmail     challengePurpose = "VERIFY_EMAIL"
	resetPassword   challengePurpose = "RESET_PASSWORD"
	passwordChanged challengePurpose = "PASSWORD_CHANGED"
)

func (r *Repository) credentials(ctx context.Context, email string) (credentials, error) {
	var user credentials
	err := r.db.WithContext(ctx).Model(&entities.User{}).
		Select("id::text, name, lower(email), password_hash").
		Where("lower(email) = ? AND email_verified_at IS NOT NULL", email).
		Row().Scan(&user.id, &user.name, &user.email, &user.passwordHash)
	return user, err
}

func (r *Repository) allow(ctx context.Context, scope, subject string, seconds, limit int) error {
	var allowed int
	err := r.db.WithContext(ctx).Raw(`
		INSERT INTO auth_rate_limits AS rl (scope, subject, window_ends_at, attempts)
		VALUES (?, ?, now() + ?::integer * interval '1 second', 1)
		ON CONFLICT (scope, subject) DO UPDATE SET
			window_ends_at = CASE WHEN rl.window_ends_at <= now()
				THEN EXCLUDED.window_ends_at ELSE rl.window_ends_at END,
			attempts = CASE WHEN rl.window_ends_at <= now() THEN 1 ELSE rl.attempts + 1 END
		WHERE rl.window_ends_at <= now() OR rl.attempts < ?
		RETURNING attempts`, scope, digest(subject), seconds, limit).Row().Scan(&allowed)
	if errors.Is(err, sql.ErrNoRows) {
		return errRateLimited
	}
	return err
}

func (r *Repository) enqueue(ctx context.Context, email string, purpose challengePurpose, hash string, nonce []byte, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO auth_email_jobs AS challenge (email, purpose, token_hash, delivery_nonce, expires_at)
		SELECT ?, ?, ?, ?, ?
		WHERE (?::text = 'RESET_PASSWORD') = EXISTS (
			SELECT FROM users WHERE lower(email) = ? AND email_verified_at IS NOT NULL
		)
		ON CONFLICT (email, purpose) DO UPDATE SET
			token_hash = EXCLUDED.token_hash,
			delivery_nonce = EXCLUDED.delivery_nonce,
			expires_at = EXCLUDED.expires_at,
			consumed_at = NULL,
			delivery_after = now(),
			delivery_attempts = 0,
			delivered_at = NULL
		WHERE challenge.expires_at <= now() OR challenge.consumed_at IS NOT NULL
			OR challenge.delivered_at IS NOT NULL OR challenge.delivery_attempts >= 5`,
		email, purpose, hash, nonce, expiresAt, purpose, email).Error
}

func (r *Repository) pendingChallengeEmail(ctx context.Context, hash string, purpose challengePurpose) (string, error) {
	var email string
	err := r.db.WithContext(ctx).Model(&entities.AuthEmailJob{}).Select("email").
		Where("token_hash = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > now()", hash, purpose).
		Row().Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errInvalidChallenge
	}
	return email, err
}

func consumeChallenge(tx *gorm.DB, hash string, purpose challengePurpose) (string, error) {
	var email string
	err := tx.Raw(`
		UPDATE auth_email_jobs SET consumed_at = now()
		WHERE token_hash = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > now()
		RETURNING email`, hash, purpose).Row().Scan(&email)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errInvalidChallenge
	}
	return email, err
}

func (r *Repository) finishRegistration(ctx context.Context, hash, name, passwordHash string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		email, err := consumeChallenge(tx, hash, verifyEmail)
		if err != nil {
			return err
		}
		var userID string
		err = tx.Raw(`
			INSERT INTO users (email, name, password_hash, email_verified_at)
			VALUES (?, ?, ?, now())
			ON CONFLICT (lower(email)) DO UPDATE SET
				name = EXCLUDED.name, password_hash = EXCLUDED.password_hash,
				email_verified_at = now(), updated_at = now()
			WHERE users.email_verified_at IS NULL
			RETURNING id::text`, email, name, passwordHash).Row().Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			return errInvalidChallenge
		}
		if err != nil {
			return err
		}
		return revokeAll(tx, userID)
	})
}

func (r *Repository) finishReset(ctx context.Context, hash, passwordHash string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		email, err := consumeChallenge(tx, hash, resetPassword)
		if err != nil {
			return err
		}
		var userID string
		err = tx.Raw(`
			UPDATE users SET password_hash = ?, updated_at = now()
			WHERE lower(email) = ? AND email_verified_at IS NOT NULL
			RETURNING id::text`, passwordHash, email).Row().Scan(&userID)
		if errors.Is(err, sql.ErrNoRows) {
			return errInvalidChallenge
		}
		if err != nil {
			return err
		}
		return invalidateSessionsAndNotify(tx, userID, email)
	})
}

func (r *Repository) changePassword(ctx context.Context, userID, email, expectedHash, passwordHash string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Take locks in the same order as finishReset: challenge, user, sessions.
		if err := tx.Exec(`
			UPDATE auth_email_jobs SET consumed_at = now()
			WHERE email = ? AND purpose = 'RESET_PASSWORD' AND consumed_at IS NULL`, email).Error; err != nil {
			return err
		}
		result := tx.Exec(`
			UPDATE users SET password_hash = ?, updated_at = now()
			WHERE id = ?::text::uuid AND password_hash = ? AND email_verified_at IS NOT NULL`,
			passwordHash, userID, expectedHash)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errCredentials
		}
		return invalidateSessionsAndNotify(tx, userID, email)
	})
}

func invalidateSessionsAndNotify(tx *gorm.DB, userID, email string) error {
	if err := revokeAll(tx, userID); err != nil {
		return err
	}
	return tx.Exec(`
		INSERT INTO auth_email_jobs (email, purpose, token_hash, delivery_nonce, expires_at)
		VALUES (?, 'PASSWORD_CHANGED', gen_random_uuid()::text, ''::bytea, now() + interval '1 day')
		ON CONFLICT (email, purpose) DO UPDATE SET token_hash = EXCLUDED.token_hash,
			expires_at = EXCLUDED.expires_at, delivery_after = now(), delivery_attempts = 0, delivered_at = NULL`, email).Error
}

type emailDelivery struct {
	email   string
	purpose challengePurpose
	hash    string
	nonce   []byte
}

func (r *Repository) claimDelivery(ctx context.Context) (emailDelivery, error) {
	var item emailDelivery
	err := r.db.WithContext(ctx).Raw(`
		WITH pending AS (
			SELECT token_hash FROM auth_email_jobs
			WHERE delivered_at IS NULL AND consumed_at IS NULL AND expires_at > now()
				AND delivery_after <= now() AND delivery_attempts < 5
			ORDER BY delivery_after LIMIT 1 FOR UPDATE SKIP LOCKED
		)
		UPDATE auth_email_jobs AS challenge
		SET delivery_after = now() + interval '1 minute', delivery_attempts = delivery_attempts + 1
		FROM pending WHERE challenge.token_hash = pending.token_hash
		RETURNING challenge.email, challenge.purpose, challenge.token_hash, challenge.delivery_nonce`).
		Row().Scan(&item.email, &item.purpose, &item.hash, &item.nonce)
	return item, err
}

func (r *Repository) markDelivered(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Model(&entities.AuthEmailJob{}).Where("token_hash = ?", hash).
		UpdateColumn("delivered_at", gorm.Expr("now()")).Error
}

func (r *Repository) cleanExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		WITH expired_challenges AS (
			DELETE FROM auth_email_jobs WHERE expires_at <= now()
		), expired_sessions AS (
			DELETE FROM auth_sessions WHERE expires_at <= now()
		)
		DELETE FROM auth_rate_limits WHERE window_ends_at <= now()`).Error
}
