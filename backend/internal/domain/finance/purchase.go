package finance

import (
	"errors"
	"math"
)

type PurchaseImpact struct {
	Amount            int64 `json:"amount"`
	AvailableAfter    int64 `json:"available_after"`
	DailyLimitAfter   int64 `json:"daily_limit_after"`
	ProjectedAfter    int64 `json:"projected_after"`
	EnoughFunds       bool  `json:"enough_funds"`
	ExceedsDailyLimit bool  `json:"exceeds_daily_limit"`
}

// SimulatePurchase applies a one-time expense while keeping projected routine spending fixed.
func SimulatePurchase(summary Summary, amount int64) (PurchaseImpact, error) {
	if amount <= 0 || summary.Period.RemainingDays <= 0 ||
		summary.AvailableFunds < math.MinInt64+amount || summary.ProjectedBalance < math.MinInt64+amount {
		return PurchaseImpact{}, errors.New("nominal simulasi di luar rentang yang didukung")
	}
	remaining := summary.AvailableFunds - amount
	return PurchaseImpact{
		Amount: amount, AvailableAfter: remaining,
		DailyLimitAfter: max(remaining, 0) / summary.Period.RemainingDays,
		ProjectedAfter:  summary.ProjectedBalance - amount,
		EnoughFunds:     remaining >= 0, ExceedsDailyLimit: amount > summary.SafeToSpendDaily,
	}, nil
}
