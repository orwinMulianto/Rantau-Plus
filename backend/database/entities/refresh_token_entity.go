package entities

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type RefreshToken struct {
	ID        pgtype.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    pgtype.UUID `gorm:"type:uuid;not null;index:refresh_tokens_user_idx"`
	TokenHash string      `gorm:"type:text;not null;uniqueIndex:refresh_tokens_token_hash_key" json:"-"`
	CreatedAt time.Time   `gorm:"type:timestamptz;not null;default:now()"`
	ExpiresAt time.Time   `gorm:"type:timestamptz;not null;check:refresh_tokens_check,expires_at > created_at"`
	RevokedAt *time.Time  `gorm:"type:timestamptz"`
	SessionID pgtype.UUID `gorm:"type:uuid;-:migration"`
	UsedAt    *time.Time  `gorm:"type:timestamptz;-:migration"`
}
