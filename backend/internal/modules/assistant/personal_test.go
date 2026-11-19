package assistant

import (
	"context"
	"errors"
	"testing"

	"rantauplus/backend/internal/domain/finance"
)

type readFinance func(context.Context, string, string) (*finance.Summary, error)

func (read readFinance) SummaryForAccount(ctx context.Context, id, timezone string) (*finance.Summary, error) {
	return read(ctx, id, timezone)
}

type guideChat func(context.Context, Request) (Answer, error)

func (chat guideChat) Chat(ctx context.Context, input Request) (Answer, error) {
	return chat(ctx, input)
}

func TestPersonalChatUsesFinancialSummary(t *testing.T) {
	reader := readFinance(func(_ context.Context, id, timezone string) (*finance.Summary, error) {
		if id != "account-1" || timezone != "Asia/Jakarta" {
			t.Fatalf("unexpected account context: %q, %q", id, timezone)
		}
		return &finance.Summary{
			Period: finance.Period{RemainingDays: 5}, AvailableFunds: 500_000,
			SafeToSpendDaily: 100_000, ProjectedBalance: 250_000,
		}, nil
	})
	ctx := context.Background()
	answer, err := personalChat(ctx, nil, reader, "account-1", "Asia/Jakarta", "Berapa Safe-to-Spend saya?")
	if err != nil || answer.Financial == nil || answer.Financial.SafeToSpendDaily != 100_000 {
		t.Fatalf("unexpected personal summary: %+v, %v", answer, err)
	}
	answer, err = personalChat(ctx, nil, reader, "account-1", "Asia/Jakarta", "Kalau beli barang Rp300 ribu, uangku masih cukup?")
	if err != nil || answer.Simulation == nil {
		t.Fatalf("unexpected purchase answer: %+v, %v", answer, err)
	}
	impact := *answer.Simulation
	if impact.AvailableAfter != 200_000 || impact.DailyLimitAfter != 40_000 || impact.ProjectedAfter != -50_000 ||
		!impact.EnoughFunds || !impact.ExceedsDailyLimit || len(answer.Actions) == 0 {
		t.Fatalf("unexpected impact: %+v", impact)
	}
	answer, err = personalChat(ctx, nil, reader, "account-1", "Asia/Jakarta", "Kalau Rp600.000?")
	if err != nil || answer.Simulation == nil || answer.Simulation.EnoughFunds || answer.Simulation.DailyLimitAfter != 0 {
		t.Fatalf("unexpected answer for insufficient funds: %+v, %v", answer, err)
	}
}

func TestPersonalChatGuideKeepsFinancialDataLocal(t *testing.T) {
	guide := NewService(guideChat(func(_ context.Context, input Request) (Answer, error) {
		if input != (Request{Skill: "finance", Message: "Safe-to-Spend"}) {
			t.Fatalf("unexpected guide request: %+v", input)
		}
		return Answer{Mode: "retrieval", Sources: []Source{{ID: "finance.safe-to-spend", Title: "Safe-to-Spend"}}}, nil
	}))
	answer, err := personalChat(context.Background(), guide, nil, "account-1", "Asia/Jakarta", "Apa itu Safe-to-Spend? Saya punya Rp1 juta.")
	if err != nil || len(answer.Sources) != 1 || answer.Financial != nil {
		t.Fatalf("unexpected guide answer: %+v, %v", answer, err)
	}
	_, err = personalChat(context.Background(), nil, nil, "account-1", "Asia/Jakarta", "Apa itu Safe-to-Spend?")
	if !errors.Is(err, errGuideUnavailable) {
		t.Fatalf("missing guide returned %v", err)
	}
}

func TestPersonalChatRequiresFinancialSetup(t *testing.T) {
	reader := readFinance(func(context.Context, string, string) (*finance.Summary, error) { return nil, nil })
	answer, err := personalChat(context.Background(), nil, reader, "account-1", "Asia/Jakarta", "Berapa Safe-to-Spend saya?")
	if err != nil || answer.Mode != "setup_required" || answer.Financial != nil {
		t.Fatalf("unexpected answer without setup: %+v, %v", answer, err)
	}
}

func TestPurchaseAmount(t *testing.T) {
	for input, want := range map[string]int64{
		"rp300.000": 300_000, "rp300 ribu": 300_000, "300rb": 300_000, "1,5 juta": 1_500_000,
		"0": 0, "-rp300.000": 0, "rp300.00": 0, "300rb atau 500rb": 0, "rp9.223.372.036.854.775.808": 0,
	} {
		if got := purchaseAmount(input); got != want {
			t.Errorf("purchaseAmount(%q) = %d, want %d", input, got, want)
		}
	}
}
