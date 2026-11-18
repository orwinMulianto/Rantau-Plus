package entities

import "github.com/jackc/pgx/v5/pgtype"

type FinancialPeriod struct {
	ID             pgtype.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid();uniqueIndex:financial_periods_user_id_id_key,priority:2"`
	UserID         pgtype.UUID `gorm:"type:uuid;not null;uniqueIndex:financial_periods_user_id_id_key,priority:1;uniqueIndex:financial_periods_user_id_start_date_key,priority:1"`
	StartDate      pgtype.Date `gorm:"type:date;not null;uniqueIndex:financial_periods_user_id_start_date_key,priority:2"`
	NextIncomeDate pgtype.Date `gorm:"type:date;not null;check:financial_periods_next_income_date_check,next_income_date > start_date"`
	OpeningBalance int64       `gorm:"type:bigint;not null;check:financial_periods_opening_balance_check,opening_balance >= 0"`
	ReserveAmount  int64       `gorm:"type:bigint;not null;default:0;check:financial_periods_reserve_amount_check,reserve_amount >= 0"`
	Timestamps
}
