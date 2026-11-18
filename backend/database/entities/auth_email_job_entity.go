package entities

import "time"

type AuthEmailJob struct {
	Email            string     `gorm:"type:text;primaryKey"`
	Purpose          string     `gorm:"type:text;primaryKey;check:auth_email_jobs_purpose_check,purpose IN ('VERIFY_EMAIL', 'RESET_PASSWORD', 'PASSWORD_CHANGED')"`
	TokenHash        string     `gorm:"type:text;not null;uniqueIndex:auth_email_jobs_token_hash_key" json:"-"`
	DeliveryNonce    []byte     `gorm:"type:bytea;not null;check:auth_email_jobs_check,octet_length(delivery_nonce) = CASE WHEN purpose = 'PASSWORD_CHANGED' THEN 0 ELSE 32 END" json:"-"`
	ExpiresAt        time.Time  `gorm:"type:timestamptz;not null;index:auth_email_expiry_idx"`
	ConsumedAt       *time.Time `gorm:"type:timestamptz"`
	DeliveryAfter    time.Time  `gorm:"type:timestamptz;not null;default:now();index:auth_email_delivery_idx,where:delivered_at IS NULL AND consumed_at IS NULL"`
	DeliveryAttempts int        `gorm:"type:integer;not null;default:0"`
	DeliveredAt      *time.Time `gorm:"type:timestamptz"`
}
