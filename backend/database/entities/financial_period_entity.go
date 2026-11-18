package entities

import "github.com/jackc/pgx/v5/pgtype"

type FinancialPeriod struct {
	ID             pgtype.UUID `db:"id"`
	UserID         pgtype.UUID `db:"user_id"`
	StartDate      pgtype.Date `db:"start_date"`
	NextIncomeDate pgtype.Date `db:"next_income_date"`
	OpeningBalance int64       `db:"opening_balance"`
	ReserveAmount  int64       `db:"reserve_amount"`
	Timestamps
}
