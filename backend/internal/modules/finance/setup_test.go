package finance

import (
	"math"
	"testing"
	"time"
)

func TestSetupUsesFinanceEngine(t *testing.T) {
	draft, sum, err := normalizeSetup(setupInput{
		OpeningBalance: 2_000_000, NextIncomeDate: "2026-12-01", ReserveAmount: 200_000,
		Obligations: []setupObligation{
			{Name: "  Uang kos  ", Amount: 500_000},
			{Name: "Internet", Amount: 100_000, DueDate: "2026-11-23"},
		},
	}, "2026-11-21")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Obligations[0].DueDate != "2026-12-01" || draft.Obligations[0].Name != "Uang kos" {
		t.Fatalf("unexpected normalized obligation: %+v", draft.Obligations[0])
	}

	result, err := calculateSetup("2026-11-21", draft.NextIncomeDate, "2026-11-21",
		draft.OpeningBalance, draft.ReserveAmount, financeTotals{Unpaid: sum})
	if err != nil {
		t.Fatal(err)
	}
	if result.AvailableFunds != 1_200_000 || result.SafeToSpendDaily != 120_000 || result.Period.RemainingDays != 10 {
		t.Fatalf("unexpected preview: %+v", result)
	}
	shortfall, err := calculateSetup("2026-11-21", draft.NextIncomeDate, "2026-11-21",
		100_000, draft.ReserveAmount, financeTotals{Unpaid: sum})
	if err != nil || shortfall.AvailableFunds != -700_000 || shortfall.SafeToSpendDaily != 0 {
		t.Fatalf("unexpected shortfall: %+v, %v", shortfall, err)
	}
}

func TestSetupRejectsInvalidDatesAndOverflow(t *testing.T) {
	for _, draft := range []setupInput{
		{NextIncomeDate: "2026-11-21"},
		{NextIncomeDate: "2026-11-30", Obligations: []setupObligation{{Name: "Kos", Amount: 1, DueDate: "2026-12-01"}}},
		{NextIncomeDate: "2026-11-30", Obligations: []setupObligation{{Name: "Kos", Amount: 0}}},
		{NextIncomeDate: "2026-11-30", Obligations: []setupObligation{{Name: "Kos", Amount: math.MaxInt64}, {Name: "Internet", Amount: 1}}},
	} {
		if _, _, err := normalizeSetup(draft, "2026-11-21"); err == nil {
			t.Errorf("expected invalid setup to be rejected: %+v", draft)
		}
	}
	if today := userToday(time.Date(2026, 11, 21, 18, 0, 0, 0, time.UTC), "Asia/Jakarta"); today != "2026-11-22" {
		t.Fatalf("setup used the wrong calendar day: %s", today)
	}
}
