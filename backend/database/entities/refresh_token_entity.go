package entities

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type RefreshToken struct {
	ID        pgtype.UUID `db:"id"`
	UserID    pgtype.UUID `db:"user_id"`
	TokenHash string      `db:"token_hash" json:"-"`
	CreatedAt time.Time   `db:"created_at"`
	ExpiresAt time.Time   `db:"expires_at"`
	RevokedAt *time.Time  `db:"revoked_at"`
}
