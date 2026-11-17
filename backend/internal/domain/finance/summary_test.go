package finance

import "testing"

func TestCalculateFinancialSummary(t *testing.T) {
	summary, err := Calculate(Snapshot{
		StartDate: "2026-11-02", NextIncomeDate: "2026-12-01", AsOfDate: "2026-11-11",
		OpeningBalance: 2_300_000, IncomeReceived: 200_000, Expenses: 800_000,
		UnpaidObligations: 250_000, Reserve: 150_000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.PeriodFunds != 2_500_000 || summary.AvailableFunds != 1_300_000 ||
		summary.SafeToSpendDaily != 65_000 || summary.AverageDailySpending != 80_000 ||
		summary.ProjectedBalance != -300_000 || summary.RiskStatus != watch {
		t.Fatalf("ringkasan tidak sesuai: %+v", summary)
	}
	if summary.Period.RemainingDays != 20 || summary.Period.ElapsedDays != 10 ||
		summary.RunwayDays == nil || *summary.RunwayDays != 16.25 {
		t.Fatalf("periode atau runway tidak sesuai: %+v", summary)
	}
}

func TestCalculateRiskBoundaries(t *testing.T) {
	cases := []struct {
		name     string
		funds    int64
		expenses int64
		status   string
	}{
		{"aman pada coverage 1", 200, 100, safe},
		{"waspada pada coverage 0.8", 180, 100, watch},
		{"berisiko di bawah coverage 0.8", 179, 100, atRisk},
		{"data kosong", 100, 0, insufficient},
		{"saldo nol", 0, 0, atRisk},
		{"defisit", 0, 1, atRisk},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			summary, err := Calculate(Snapshot{
				StartDate: "2026-11-01", NextIncomeDate: "2026-11-02", AsOfDate: "2026-11-01",
				OpeningBalance: tc.funds, Expenses: tc.expenses,
			})
			if err != nil {
				t.Fatal(err)
			}
			if summary.RiskStatus != tc.status || summary.AvailableFunds != tc.funds-tc.expenses {
				t.Fatalf("status atau dana tersedia tidak sesuai: %+v", summary)
			}
			if tc.expenses == 0 && (summary.RunwayDays != nil || summary.Coverage != nil) {
				t.Fatal("runway dan coverage seharusnya tidak tersedia")
			}
			if summary.AvailableFunds <= 0 && summary.SafeToSpendDaily != 0 {
				t.Fatal("Safe-to-Spend untuk dana nol atau defisit harus nol")
			}
		})
	}
}

func TestCalculatePastIncomeDate(t *testing.T) {
	input := Snapshot{StartDate: "2026-11-01", NextIncomeDate: "2026-11-11", AsOfDate: "2026-11-13"}
	summary, err := Calculate(input)
	if err != nil || summary.Period.RemainingDays != 1 || summary.Period.ElapsedDays != 10 {
		t.Fatalf("periode yang telah lewat tidak sesuai: %+v, %v", summary.Period, err)
	}
	input.NextIncomeDate = input.StartDate
	if _, err := Calculate(input); err == nil {
		t.Fatal("tanggal pemasukan yang sama dengan tanggal awal harus ditolak")
	}
	input.NextIncomeDate, input.Expenses = "2026-11-11", -1
	if _, err := Calculate(input); err == nil {
		t.Fatal("nominal pengeluaran negatif harus ditolak")
	}
}
