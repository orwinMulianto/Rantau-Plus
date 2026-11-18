package finance

import "rantauplus/backend/internal/domain/finance"

// demoSummary uses a fixed example so a clean checkout can run without personal data.
func demoSummary() (finance.Summary, error) {
	return finance.Calculate(finance.Snapshot{
		StartDate: "2026-11-02", NextIncomeDate: "2026-12-01", AsOfDate: "2026-11-11",
		OpeningBalance: 2_300_000, IncomeReceived: 200_000, Expenses: 800_000,
		UnpaidObligations: 250_000, Reserve: 150_000,
	})
}
