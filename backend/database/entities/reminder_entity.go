package entities

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type ReminderStatus string

const (
	ReminderStatusPending   ReminderStatus = "PENDING"
	ReminderStatusCompleted ReminderStatus = "COMPLETED"
)

type Reminder struct {
	ID     pgtype.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID pgtype.UUID    `gorm:"type:uuid;not null;index:reminders_user_due_idx,priority:1"`
	Title  string         `gorm:"type:text;not null"`
	DueAt  time.Time      `gorm:"type:timestamptz;not null;index:reminders_user_due_idx,priority:2"`
	Status ReminderStatus `gorm:"type:text;not null;default:PENDING;check:reminders_status_check,status IN ('PENDING', 'COMPLETED')"`
	Timestamps
}
