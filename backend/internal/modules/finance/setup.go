package finance

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgtype"
	"gorm.io/gorm"

	"rantauplus/backend/database/entities"
	domain "rantauplus/backend/internal/domain/finance"
)

const setupDateLayout = "2006-01-02"

var errSetupChanged = errors.New("kewajiban setup sudah berubah")

type setupInput struct {
	OpeningBalance int64             `json:"opening_balance"`
	NextIncomeDate string            `json:"next_income_date"`
	ReserveAmount  int64             `json:"reserve_amount"`
	Obligations    []setupObligation `json:"obligations"`
}

type setupObligation struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Amount  int64  `json:"amount"`
	DueDate string `json:"due_date,omitempty"`
}

type savedSetup struct {
	Today string      `json:"today"`
	Setup *setupInput `json:"setup"`
}

type setupValidation string

func (e setupValidation) Error() string { return string(e) }

type SetupService struct {
	db *gorm.DB
}

func NewSetupService(db *gorm.DB) *SetupService { return &SetupService{db: db} }

func userToday(now time.Time, timezone string) string {
	location, err := time.LoadLocation(timezone)
	if err != nil || timezone == "" {
		location, _ = time.LoadLocation("Asia/Jakarta")
	}
	return now.In(location).Format(setupDateLayout)
}

func normalizeSetup(input setupInput, today string) (setupInput, int64, error) {
	if input.OpeningBalance < 0 || input.ReserveAmount < 0 {
		return setupInput{}, 0, setupValidation("Saldo awal dan dana cadangan harus bernilai nol atau lebih.")
	}
	if date, err := time.Parse(setupDateLayout, input.NextIncomeDate); err != nil ||
		date.Format(setupDateLayout) != input.NextIncomeDate || input.NextIncomeDate <= today {
		return setupInput{}, 0, setupValidation("Tanggal pemasukan berikutnya harus setelah hari ini.")
	}
	if len(input.Obligations) > 50 {
		return setupInput{}, 0, setupValidation("Satu pengaturan dapat memuat hingga 50 kewajiban.")
	}
	input.Obligations = append([]setupObligation{}, input.Obligations...)
	ids := make(map[string]bool)
	var total int64
	for index, item := range input.Obligations {
		item.Name = strings.TrimSpace(item.Name)
		if item.Name == "" || utf8.RuneCountInString(item.Name) > 100 || strings.ContainsRune(item.Name, 0) || item.Amount <= 0 {
			return setupInput{}, 0, setupValidation(fmt.Sprintf("Isi nama (maksimal 100 karakter) dan nominal yang valid pada kewajiban %d.", index+1))
		}
		if item.DueDate == "" {
			item.DueDate = input.NextIncomeDate
		}
		if date, err := time.Parse(setupDateLayout, item.DueDate); err != nil || date.Year() < 1 ||
			date.Format(setupDateLayout) != item.DueDate || item.DueDate > input.NextIncomeDate {
			return setupInput{}, 0, setupValidation(fmt.Sprintf("Jatuh tempo kewajiban %d harus pada atau sebelum tanggal pemasukan berikutnya.", index+1))
		}
		if item.ID != "" {
			var id pgtype.UUID
			if err := id.Scan(item.ID); err != nil || !id.Valid || ids[id.String()] {
				return setupInput{}, 0, setupValidation("Daftar kewajiban memuat ID yang tidak valid atau berulang.")
			}
			item.ID = id.String()
			ids[item.ID] = true
		}
		if item.Amount > math.MaxInt64-total {
			return setupInput{}, 0, setupValidation("Jumlah kewajiban melebihi batas rupiah yang didukung.")
		}
		total += item.Amount
		input.Obligations[index] = item
	}
	return input, total, nil
}

func (s *SetupService) load(ctx context.Context, userID, today string) (savedSetup, error) {
	result := savedSetup{Today: today}
	db := s.db.WithContext(ctx)
	period, err := activePeriod(db, userID, today)
	if err != nil || period == nil {
		return result, err
	}
	items, err := setupItems(db, userID, period.ID.String())
	if err != nil {
		return result, err
	}
	result.Setup = &setupInput{
		OpeningBalance: period.OpeningBalance, NextIncomeDate: period.NextIncomeDate.Time.Format(setupDateLayout),
		ReserveAmount: period.ReserveAmount, Obligations: items,
	}
	return result, nil
}

func (s *SetupService) preview(ctx context.Context, userID, today string, input setupInput) (domain.Summary, error) {
	_, _, summary, err := prepareSetup(s.db.WithContext(ctx), userID, today, input)
	return summary, err
}

func (s *SetupService) save(ctx context.Context, userID, today string, input setupInput) (domain.Summary, error) {
	var result domain.Summary
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize setup writes for this account, including creation of its first period.
		var id string
		if err := tx.Raw("SELECT id::text FROM users WHERE id = ?::text::uuid FOR UPDATE", userID).Row().Scan(&id); err != nil {
			return err
		}
		period, draft, summary, err := prepareSetup(tx, userID, today, input)
		if err != nil {
			return err
		}
		periodID, err := writePeriod(tx, userID, today, period, draft)
		if err != nil {
			return err
		}
		if err := writeSetupItems(tx, userID, periodID, draft.Obligations); err != nil {
			return err
		}
		result = summary
		return nil
	})
	return result, err
}

func prepareSetup(db *gorm.DB, userID, today string, input setupInput) (*entities.FinancialPeriod, setupInput, domain.Summary, error) {
	draft, total, err := normalizeSetup(input, today)
	if err != nil {
		return nil, setupInput{}, domain.Summary{}, err
	}
	period, err := activePeriod(db, userID, today)
	if err != nil {
		return nil, setupInput{}, domain.Summary{}, err
	}
	start, periodID := today, ""
	if period != nil {
		start, periodID = period.StartDate.Time.Format(setupDateLayout), period.ID.String()
	}
	totals, err := readTotals(db, userID, periodID, today, draft.NextIncomeDate, true)
	if err != nil {
		return nil, setupInput{}, domain.Summary{}, err
	}
	if total > math.MaxInt64-totals.Unpaid {
		return nil, setupInput{}, domain.Summary{}, setupValidation("Jumlah kewajiban melebihi batas rupiah yang didukung.")
	}
	totals.Unpaid += total
	summary, err := calculateSetup(start, draft.NextIncomeDate, today, draft.OpeningBalance, draft.ReserveAmount, totals)
	return period, draft, summary, err
}

func (s *SetupService) summary(ctx context.Context, userID, today string) (*domain.Summary, error) {
	db := s.db.WithContext(ctx)
	period, err := activePeriod(db, userID, today)
	if err != nil || period == nil {
		return nil, err
	}
	next := period.NextIncomeDate.Time.Format(setupDateLayout)
	totals, err := readTotals(db, userID, period.ID.String(), today, next, false)
	if err != nil {
		return nil, err
	}
	summary, err := calculateSetup(period.StartDate.Time.Format(setupDateLayout), next, today,
		period.OpeningBalance, period.ReserveAmount, totals)
	return &summary, err
}

func calculateSetup(start, next, today string, opening, reserve int64, totals financeTotals) (domain.Summary, error) {
	result, err := domain.Calculate(domain.Snapshot{
		StartDate: start, NextIncomeDate: next, AsOfDate: today,
		OpeningBalance: opening, Reserve: reserve, IncomeReceived: totals.Income,
		Expenses: totals.Expenses, UnpaidObligations: totals.Unpaid,
	})
	if err != nil {
		return domain.Summary{}, setupValidation("Nominal atau periode berada di luar batas perhitungan.")
	}
	return result, nil
}
