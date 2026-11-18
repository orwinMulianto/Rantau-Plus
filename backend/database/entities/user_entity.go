package entities

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID              pgtype.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name            string      `gorm:"type:text;not null"`
	Email           string      `gorm:"type:text;not null;uniqueIndex:users_email_idx,expression:lower(email)"`
	PasswordHash    string      `gorm:"type:text;not null" json:"-"`
	Timezone        string      `gorm:"type:text;not null;default:Asia/Jakarta"`
	Currency        string      `gorm:"type:text;not null;default:IDR;check:users_currency_check,currency = 'IDR'"`
	EmailVerifiedAt *time.Time  `gorm:"type:timestamptz;-:migration"`
	Timestamps
}
