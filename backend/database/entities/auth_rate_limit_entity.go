package entities

import "time"

type AuthRateLimit struct {
	Scope        string    `gorm:"type:text;primaryKey"`
	Subject      string    `gorm:"type:text;primaryKey"`
	WindowEndsAt time.Time `gorm:"type:timestamptz;not null;index:auth_rate_limits_expiry_idx"`
	Attempts     int       `gorm:"type:integer;not null"`
}
