package finance

import (
	"errors"
	"math/big"
	"time"
)

const (
	dateLayout   = "2006-01-02"
	secondsInDay = 86400
	safe         = "AMAN"
	watch        = "WASPADA"
	atRisk       = "BERISIKO"
	insufficient = "BELUM_CUKUP_DATA"
)

// Snapshot contains totals in rupiah for one financial period.
type Snapshot struct {
	StartDate         string
	NextIncomeDate    string
	AsOfDate          string
	OpeningBalance    int64
	IncomeReceived    int64
	Expenses          int64
	UnpaidObligations int64
	Reserve           int64
}

type Period struct {
	StartDate      string `json:"start_date"`
	NextIncomeDate string `json:"next_income_date"`
	AsOfDate       string `json:"as_of_date"`
	RemainingDays  int64  `json:"remaining_days"`
	ElapsedDays    int64  `json:"elapsed_days"`
}

type Summary struct {
	Currency             string   `json:"currency"`
	Period               Period   `json:"period"`
	PeriodFunds          int64    `json:"period_funds"`
	Expenses             int64    `json:"expenses"`
	UnpaidObligations    int64    `json:"unpaid_obligations"`
	Reserve              int64    `json:"reserve"`
	AvailableFunds       int64    `json:"available_funds"`
	SafeToSpendDaily     int64    `json:"safe_to_spend_daily"`
	AverageDailySpending int64    `json:"average_daily_spending"`
	RunwayDays           *float64 `json:"runway_days"`
	ProjectedBalance     int64    `json:"projected_balance"`
	Coverage             *float64 `json:"coverage"`
	RiskStatus           string   `json:"risk_status"`
}

func Calculate(snapshot Snapshot) (Summary, error) {
	for _, amount := range []int64{snapshot.OpeningBalance, snapshot.IncomeReceived,
		snapshot.Expenses, snapshot.UnpaidObligations, snapshot.Reserve} {
		if amount < 0 {
			return Summary{}, errors.New("nominal input harus nonnegatif")
		}
	}

	start, startErr := time.Parse(dateLayout, snapshot.StartDate)
	next, nextErr := time.Parse(dateLayout, snapshot.NextIncomeDate)
	today, todayErr := time.Parse(dateLayout, snapshot.AsOfDate)
	if startErr != nil || nextErr != nil || todayErr != nil || !next.After(start) || today.Before(start) {
		return Summary{}, errors.New("tanggal periode tidak valid")
	}
	remaining := max((next.Unix()-today.Unix())/secondsInDay, 1)
	elapsed := (min(today.Unix(), next.Unix()-secondsInDay)-start.Unix())/secondsInDay + 1

	funds := new(big.Int).Add(big.NewInt(snapshot.OpeningBalance), big.NewInt(snapshot.IncomeReceived))
	available := new(big.Int).Set(funds)
	available.Sub(available, big.NewInt(snapshot.Expenses))
	available.Sub(available, big.NewInt(snapshot.UnpaidObligations))
	available.Sub(available, big.NewInt(snapshot.Reserve))

	projectedExpenses := new(big.Int).Mul(big.NewInt(snapshot.Expenses), big.NewInt(remaining))
	projectedExpenses.Quo(projectedExpenses, big.NewInt(elapsed))
	projectedBalance := new(big.Int).Sub(available, projectedExpenses)
	if !funds.IsInt64() || !available.IsInt64() || !projectedBalance.IsInt64() {
		return Summary{}, errors.New("hasil perhitungan melebihi rentang rupiah yang didukung")
	}

	summary := Summary{
		Currency: "IDR",
		Period: Period{
			StartDate: snapshot.StartDate, NextIncomeDate: snapshot.NextIncomeDate,
			AsOfDate: snapshot.AsOfDate, RemainingDays: remaining, ElapsedDays: elapsed,
		},
		PeriodFunds: funds.Int64(), Expenses: snapshot.Expenses,
		UnpaidObligations: snapshot.UnpaidObligations, Reserve: snapshot.Reserve,
		AvailableFunds: available.Int64(), SafeToSpendDaily: max(available.Int64(), 0) / remaining,
		AverageDailySpending: snapshot.Expenses / elapsed, ProjectedBalance: projectedBalance.Int64(),
		RiskStatus: insufficient,
	}

	if snapshot.Expenses > 0 {
		availableDays := new(big.Int).Mul(available, big.NewInt(elapsed))
		expenseDays := new(big.Int).Mul(big.NewInt(snapshot.Expenses), big.NewInt(remaining))
		runway, _ := new(big.Rat).SetFrac(availableDays, big.NewInt(snapshot.Expenses)).Float64()
		coverage, _ := new(big.Rat).SetFrac(availableDays, expenseDays).Float64()
		summary.RunwayDays, summary.Coverage = &runway, &coverage
		summary.RiskStatus = coverageStatus(availableDays, expenseDays)
	}
	if available.Sign() <= 0 {
		summary.RiskStatus = atRisk
	}
	return summary, nil
}

func coverageStatus(availableDays, expenseDays *big.Int) string {
	if availableDays.Cmp(expenseDays) >= 0 {
		return safe
	}
	// Compare against 4/5 exactly, before converting coverage for display.
	left := new(big.Int).Mul(availableDays, big.NewInt(5))
	right := new(big.Int).Mul(expenseDays, big.NewInt(4))
	if left.Cmp(right) >= 0 {
		return watch
	}
	return atRisk
}
