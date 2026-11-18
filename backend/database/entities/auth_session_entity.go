package entities

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type AuthSession struct {
	ID              pgtype.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid();uniqueIndex:auth_sessions_id_user_id_key,priority:1"`
	UserID          pgtype.UUID `gorm:"type:uuid;not null;uniqueIndex:auth_sessions_id_user_id_key,priority:2;index:auth_sessions_user_idx,priority:1"`
	AccessTokenHash string      `gorm:"type:text;not null;uniqueIndex:auth_sessions_access_token_hash_key" json:"-"`
	AccessExpiresAt time.Time   `gorm:"type:timestamptz;not null;check:auth_sessions_check,access_expires_at <= expires_at"`
	CreatedAt       time.Time   `gorm:"type:timestamptz;not null;default:now();index:auth_sessions_user_idx,priority:2,sort:desc"`
	RefreshedAt     time.Time   `gorm:"type:timestamptz;not null;default:now()"`
	ExpiresAt       time.Time   `gorm:"type:timestamptz;not null;index:auth_sessions_expiry_idx"`
	RevokedAt       *time.Time  `gorm:"type:timestamptz"`
}
