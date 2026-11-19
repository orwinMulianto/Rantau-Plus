package assistant

import "rantauplus/backend/internal/domain/finance"

type Request struct {
	Skill   string `json:"skill,omitempty"`
	Message string `json:"message"`
}

type Source struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type Answer struct {
	Skill      string                  `json:"skill"`
	Mode       string                  `json:"mode"`
	Answer     string                  `json:"answer"`
	Sources    []Source                `json:"sources"`
	Financial  *FinancialContext       `json:"financial,omitempty"`
	Simulation *finance.PurchaseImpact `json:"simulation,omitempty"`
	Actions    []string                `json:"actions,omitempty"`
	Places     []Place                 `json:"places,omitempty"`
}

type FinancialContext struct {
	AvailableFunds   int64    `json:"available_funds"`
	SafeToSpendDaily int64    `json:"safe_to_spend_daily"`
	RemainingDays    int64    `json:"remaining_days"`
	RunwayDays       *float64 `json:"runway_days"`
	ProjectedBalance int64    `json:"projected_balance"`
}
