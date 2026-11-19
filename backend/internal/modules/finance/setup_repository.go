package finance

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"

	"rantauplus/backend/database/entities"
)

func activePeriod(db *gorm.DB, userID, today string) (*entities.FinancialPeriod, error) {
	var period entities.FinancialPeriod
	err := db.Where("user_id = ?::text::uuid AND start_date <= ?::text::date AND next_income_date > ?::text::date",
		userID, today, today).Order("start_date DESC").Take(&period).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &period, err
}

func pendingSetupItems(db *gorm.DB, userID, periodID string) *gorm.DB {
	return db.Model(&entities.Obligation{}).
		Where("user_id = ?::text::uuid AND setup_period_id = ?::text::uuid AND status = ?",
			userID, periodID, entities.ObligationStatusPending)
}

func setupItems(db *gorm.DB, userID, periodID string) ([]setupObligation, error) {
	items := []setupObligation{}
	err := pendingSetupItems(db, userID, periodID).
		Select("id::text AS id, name, amount, to_char(due_date, 'YYYY-MM-DD') AS due_date").
		Order("created_at, id").Scan(&items).Error
	return items, err
}

type financeTotals struct {
	Income   int64
	Expenses int64
	Unpaid   int64
}

func readTotals(db *gorm.DB, userID, periodID, today, next string, useDraft bool) (financeTotals, error) {
	var totals financeTotals
	err := db.Raw(`
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE type = 'INCOME'), 0)::bigint AS income,
			COALESCE(SUM(amount) FILTER (WHERE type = 'EXPENSE'), 0)::bigint AS expenses,
			(SELECT COALESCE(SUM(amount), 0)::bigint FROM obligations
			 WHERE user_id = ?::text::uuid AND status = 'PENDING' AND due_date <= ?::text::date
			   AND (NOT ? OR setup_period_id IS DISTINCT FROM NULLIF(?, '')::uuid)) AS unpaid
		FROM transactions WHERE user_id = ?::text::uuid AND period_id = NULLIF(?, '')::uuid
		  AND transaction_date <= ?::text::date`,
		userID, next, useDraft && periodID != "", periodID, userID, periodID, today).
		Row().Scan(&totals.Income, &totals.Expenses, &totals.Unpaid)
	return totals, err
}

func setupDate(value string) pgtype.Date {
	date, _ := time.Parse(setupDateLayout, value)
	return pgtype.Date{Time: date, Valid: true}
}

func writePeriod(db *gorm.DB, userID, today string, period *entities.FinancialPeriod, draft setupInput) (string, error) {
	next := setupDate(draft.NextIncomeDate)
	if period == nil {
		period = &entities.FinancialPeriod{
			StartDate: setupDate(today), NextIncomeDate: next,
			OpeningBalance: draft.OpeningBalance, ReserveAmount: draft.ReserveAmount,
		}
		if err := period.UserID.Scan(userID); err != nil {
			return "", err
		}
		if err := db.Create(period).Error; err != nil {
			return "", err
		}
		return period.ID.String(), nil
	}
	err := db.Model(&entities.FinancialPeriod{}).
		Where("id = ?::text::uuid AND user_id = ?::text::uuid", period.ID.String(), userID).
		Updates(map[string]any{"opening_balance": draft.OpeningBalance, "reserve_amount": draft.ReserveAmount, "next_income_date": next}).Error
	return period.ID.String(), err
}

func writeSetupItems(db *gorm.DB, userID, periodID string, items []setupObligation) error {
	previous, err := setupItems(db, userID, periodID)
	if err != nil {
		return err
	}
	claimed := make(map[string]bool)
	for _, item := range items {
		if item.ID != "" {
			claimed[item.ID] = true
		}
	}
	var keep []string
	for _, item := range items {
		if item.ID == "" {
			// An identical retry reuses the saved item when the first response was lost.
			for _, old := range previous {
				if !claimed[old.ID] && old.Name == item.Name && old.Amount == item.Amount && old.DueDate == item.DueDate {
					item.ID = old.ID
					claimed[old.ID] = true
					break
				}
			}
		}
		if item.ID != "" {
			update := pendingSetupItems(db, userID, periodID).Where("id = ?::text::uuid", item.ID).
				Updates(map[string]any{"name": item.Name, "amount": item.Amount, "due_date": setupDate(item.DueDate)})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected != 1 {
				return errSetupChanged
			}
		} else {
			row := entities.Obligation{Name: item.Name, Amount: item.Amount, DueDate: setupDate(item.DueDate)}
			if err := row.UserID.Scan(userID); err != nil {
				return err
			}
			if err := row.SetupPeriodID.Scan(periodID); err != nil {
				return err
			}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			item.ID = row.ID.String()
		}
		keep = append(keep, item.ID)
	}
	remove := pendingSetupItems(db, userID, periodID)
	if len(keep) > 0 {
		remove = remove.Where("id::text NOT IN ?", keep)
	}
	return remove.Delete(&entities.Obligation{}).Error
}
