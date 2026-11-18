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
	ID     pgtype.UUID    `db:"id"`
	UserID pgtype.UUID    `db:"user_id"`
	Title  string         `db:"title"`
	DueAt  time.Time      `db:"due_at"`
	Status ReminderStatus `db:"status"`
	Timestamps
}
