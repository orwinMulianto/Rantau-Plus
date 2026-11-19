package assistant

import (
	"context"
	"encoding/json"
	"testing"

	"rantauplus/backend/internal/domain/finance"
)

type scriptedPlanner struct {
	ChatClient
	plan func(PlanRequest) Plan
}

func (p scriptedPlanner) Plan(_ context.Context, in PlanRequest) (Plan, error) {
	return p.plan(in), nil
}

func TestCompanionBudgetUsesSessionAndUserAmounts(t *testing.T) {
	svc := NewService(scriptedPlanner{plan: func(in PlanRequest) Plan {
		switch in.Message {
		case "Aku mau siapkan Rp50 ribu untuk kado":
			return Plan{Answer: "Kadonya untuk siapa?"}
		case "Kalau aku jajan 25k juga masih cukup?":
			if len(in.History) != 2 {
				t.Fatalf("history = %+v", in.History)
			}
			return Plan{Tool: "simulate_purchase", Arguments: json.RawMessage(`{"amounts":[50000,25000]}`)}
		default:
			if len(in.History) != 0 {
				t.Fatal("cross-session chat history")
			}
			return Plan{Tool: "simulate_purchase", Arguments: json.RawMessage(`{"amounts":[50000]}`)}
		}
	}})
	reader := readFinance(func(_ context.Context, id, tz string) (*finance.Summary, error) {
		if id != "A" || tz != "Asia/Jakarta" {
			t.Fatalf("unexpected identity %q %q", id, tz)
		}
		return &finance.Summary{Period: finance.Period{RemainingDays: 10}, AvailableFunds: 1000000, SafeToSpendDaily: 100000, ProjectedBalance: 1000000}, nil
	})
	ctx := context.Background()
	if _, err := svc.converse(ctx, reader, "A", "first", "Asia/Jakarta", "Aku mau siapkan Rp50 ribu untuk kado", nil); err != nil {
		t.Fatal(err)
	}
	reply, err := svc.converse(ctx, reader, "A", "first", "Asia/Jakarta", "Kalau aku jajan 25k juga masih cukup?", nil)
	if err != nil || reply.Simulation == nil || reply.Simulation.Amount != 75000 || reply.Simulation.DailyLimitAfter != 92500 {
		t.Fatalf("reply = %+v, err = %v", reply, err)
	}
	other, err := svc.converse(ctx, nil, "B", "second", "UTC", "Kalau aku beli itu?", nil)
	if err != nil || other.Mode != "needs_amount" {
		t.Fatalf("unexpected invented amount: %+v %v", other, err)
	}
}

func TestCompanionRejectsUnknownToolAndInventedLocation(t *testing.T) {
	svc := NewService(nil)
	in := PlanRequest{Message: "Cari tempat makan dekat sini."}
	_, err := svc.execute(context.Background(), nil, "A", "Asia/Jakarta", in, nil, Plan{Tool: "other"})
	if err != errModel {
		t.Fatalf("unknown tool err = %v", err)
	}
	reply, err := svc.execute(context.Background(), nil, "A", "Asia/Jakarta", in, nil, Plan{Tool: "find_food", Arguments: json.RawMessage(`{"area":"Surabaya"}`)})
	if err != nil || reply.Mode != "needs_location" {
		t.Fatalf("location = %+v %v", reply, err)
	}
}
