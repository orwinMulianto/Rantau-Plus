package assistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Location struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

func (l *Location) valid() bool {
	return l == nil || (l.Latitude != nil && l.Longitude != nil && math.Abs(*l.Latitude) <= 90 && math.Abs(*l.Longitude) <= 180)
}

type Place struct {
	Name         string          `json:"name"`
	Address      string          `json:"address,omitempty"`
	MapsURL      string          `json:"maps_url"`
	OpenNow      *bool           `json:"open_now,omitempty"`
	PriceLevel   string          `json:"price_level,omitempty"`
	PriceRange   *PriceRange     `json:"price_range,omitempty"`
	Attributions json.RawMessage `json:"attributions,omitempty"`
}

type Money struct {
	Currency string `json:"currencyCode"`
	Units    int64  `json:"units,string,omitempty"`
	Nanos    int    `json:"nanos,omitempty"`
}

type PriceRange struct {
	Start *Money `json:"startPrice,omitempty"`
	End   *Money `json:"endPrice,omitempty"`
}

type FoodFinder interface {
	FindFood(context.Context, string, *Location) ([]Place, error)
}

type PlanRequest struct {
	Message     string    `json:"message"`
	History     []Message `json:"history,omitempty"`
	HasLocation bool      `json:"has_location"`
}

type Plan struct {
	Answer    string          `json:"answer"`
	Tool      string          `json:"tool"`
	Arguments json.RawMessage `json:"arguments"`
}

type Planner interface {
	Plan(context.Context, PlanRequest) (Plan, error)
}

var errModel = errors.New("model percakapan sedang tidak tersedia")
var affordability = regexp.MustCompile(`(?i)\b(cukup|masih aman|masih bisa jajan|masih bisa beli)\b`)

const askForTotal = "Sebutkan total belanja yang ingin dicek, termasuk rencana sebelumnya jika masih berlaku."

func (s *Service) converse(ctx context.Context, reader FinanceReader, account, session, timezone, message string, location *Location) (Answer, error) {
	message, err := validateMessage(message)
	if err != nil {
		return Answer{}, err
	}
	if s == nil {
		return personalChat(ctx, nil, reader, account, timezone, message)
	}
	planner, ok := s.client.(Planner)
	if !ok {
		return personalChat(ctx, s, reader, account, timezone, message)
	}
	input := PlanRequest{Message: message, History: s.history.read(session), HasLocation: location != nil}
	var plan Plan
	amount := purchaseAmount(strings.ToLower(message))
	if n := len(input.History); n > 0 && input.History[n-1].Content == askForTotal && amount > 0 {
		plan = simulationPlan(amount)
	} else {
		plan, err = planner.Plan(ctx, input)
	}
	if err != nil {
		if ctx.Err() != nil {
			return Answer{}, ctx.Err()
		}
		// The original finance routes remain usable when chat inference is offline.
		if len(input.History) > 0 && affordability.MatchString(message) {
			return Answer{}, errModel
		}
		answer, fallbackErr := personalChat(ctx, s, reader, account, timezone, message)
		if fallbackErr == nil && answer.Mode == "help" {
			return Answer{}, errModel
		}
		return answer, fallbackErr
	}
	// Small models sometimes answer an explicit budget question without calling a tool.
	lower := strings.ToLower(message)
	if plan.Tool == "" && affordability.MatchString(message) && purchaseIntent.MatchString(lower) && currencyHint.MatchString(lower) {
		if currencyHint.MatchString(strings.ToLower(userText(PlanRequest{History: input.History}))) {
			s.history.remember(session, message, askForTotal)
			return companionAnswer("needs_amount", askForTotal), nil
		}
		plan = simulationPlan(amount)
	} else if plan.Tool == "" && guideTopic(lower) == "Safe-to-Spend" && strings.Contains(lower, "berapa") && !guideIntent.MatchString(lower) {
		plan = Plan{Tool: "financial_summary", Arguments: json.RawMessage("{}")}
	}
	answer, err := s.execute(ctx, reader, account, timezone, input, location, plan)
	if err == nil {
		s.history.remember(session, message, answer.Answer)
	}
	return answer, err
}

func simulationPlan(amount int64) Plan {
	args, _ := json.Marshal(map[string][]int64{"amounts": {amount}})
	return Plan{Tool: "simulate_purchase", Arguments: args}
}

func (s *Service) execute(ctx context.Context, reader FinanceReader, account, timezone string, input PlanRequest, location *Location, plan Plan) (Answer, error) {
	if plan.Tool == "" {
		answer := strings.TrimSpace(plan.Answer)
		if answer == "" || utf8.RuneCountInString(answer) > 6000 {
			return Answer{}, errModel
		}
		reply := companionAnswer("conversation", answer)
		reply.Skill = "companion"
		return reply, nil
	}
	switch plan.Tool {
	case "lookup_guide":
		var args struct {
			Topic string `json:"topic"`
		}
		if !decodeArgs(plan.Arguments, &args) || guideTopic(strings.ToLower(args.Topic)) != args.Topic || args.Topic == "" {
			return Answer{}, errModel
		}
		result, err := s.chat(ctx, Request{Skill: "finance", Message: args.Topic})
		if err != nil {
			return Answer{}, errGuideUnavailable
		}
		return result, nil
	case "find_food":
		var args struct {
			Area   string `json:"area"`
			Budget int64  `json:"budget"`
		}
		if !decodeArgs(plan.Arguments, &args) || utf8.RuneCountInString(args.Area) > 120 {
			return Answer{}, errModel
		}
		area := strings.TrimSpace(args.Area)
		text := strings.ToLower(userText(input))
		if area != "" && !strings.Contains(text, strings.ToLower(area)) {
			area = ""
		}
		if location == nil && area == "" {
			return companionAnswer("needs_location", "Kamu mau cari makan di area mana? Sebut nama kampus, kelurahan, atau bagikan lokasi."), nil
		}
		if s.places == nil {
			return companionAnswer("places_unavailable", "Pencarian tempat makan sedang tidak tersedia."), nil
		}
		places, err := s.places.FindFood(ctx, area, location)
		if err != nil {
			return companionAnswer("places_unavailable", "Pencarian tempat makan sedang tidak tersedia. Coba lagi nanti."), nil
		}
		if args.Budget > 0 && confirmedTotal(input, []int64{args.Budget}) > 0 {
			eligible := make([]Place, 0, len(places))
			var unknown []Place
			for _, place := range places {
				if place.PriceRange == nil || place.PriceRange.Start == nil || place.PriceRange.Start.Currency != "IDR" {
					unknown = append(unknown, place)
				} else if low := place.PriceRange.Start; low.Units < args.Budget || (low.Units == args.Budget && low.Nanos == 0) {
					eligible = append(eligible, place)
				}
			}
			places = append(eligible, unknown...)
		}
		if len(places) == 0 {
			return companionAnswer("places", "Belum menemukan tempat makan di area itu. Coba area lain."), nil
		}
		result := companionAnswer("places", "Ini pilihan tempat makan dari Google Maps. Cek menu untuk memastikan harga per porsi.")
		result.Sources = []Source{{ID: "google_maps", Title: "Google Maps"}}
		result.Places = places
		return result, nil
	case "financial_summary", "simulate_purchase":
		var amount int64
		if plan.Tool == "simulate_purchase" {
			var args struct {
				Amounts []int64 `json:"amounts"`
			}
			if decodeArgs(plan.Arguments, &args) {
				amount = confirmedTotal(input, args.Amounts)
			}
			if amount == 0 {
				return companionAnswer("needs_amount", "Berapa nominal yang mau kamu simulasikan? Tuliskan tiap rencana belanja dalam rupiah."), nil
			}
		} else if !decodeArgs(plan.Arguments, &struct{}{}) {
			return Answer{}, errModel
		}
		summary, err := reader.SummaryForAccount(ctx, account, timezone)
		if err != nil {
			return Answer{}, err
		}
		if summary == nil {
			return companionAnswer("setup_required", "Isi Financial Setup untuk menghitung anggaran periode ini."), nil
		}
		if amount > 0 {
			return purchaseAnswer(*summary, amount), nil
		}
		return summaryAnswer(*summary), nil
	default:
		return Answer{}, errModel
	}
}

func decodeArgs(raw json.RawMessage, target any) bool {
	if len(raw) == 0 || raw[0] != '{' {
		return false
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	return dec.Decode(target) == nil
}

func userText(input PlanRequest) string {
	var text strings.Builder
	for _, msg := range input.History {
		if msg.Role == "user" {
			text.WriteString(msg.Content)
			text.WriteByte('\n')
		}
	}
	text.WriteString(input.Message)
	return text.String()
}

// Only simulate positive amounts the user actually mentioned. Sum them in Go.
func confirmedTotal(input PlanRequest, amounts []int64) int64 {
	if len(amounts) == 0 || len(amounts) > 8 {
		return 0
	}
	mentioned := make(map[int64]int)
	for _, match := range amountPattern.FindAllString(strings.ToLower(userText(input)), -1) {
		mentioned[purchaseAmount(match)]++
	}
	var total int64
	for _, amount := range amounts {
		if amount <= 0 || mentioned[amount] == 0 || total > math.MaxInt64-amount {
			return 0
		}
		mentioned[amount]--
		total += amount
	}
	return total
}
