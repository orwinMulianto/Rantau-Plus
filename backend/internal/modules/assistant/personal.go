package assistant

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"rantauplus/backend/internal/domain/finance"
)

var errGuideUnavailable = errors.New("panduan keuangan sedang tidak tersedia")

type FinanceReader interface {
	SummaryForAccount(ctx context.Context, accountID, timezone string) (*finance.Summary, error)
}

var (
	purchaseIntent        = regexp.MustCompile(`\b(simulasi|beli|membeli|belanja|bayar)\b`)
	hypotheticalIntent    = regexp.MustCompile(`\b(kalau|kalo|jika)\b`)
	purchaseWithoutAmount = regexp.MustCompile(`\b(simulasi|mau beli|ingin beli|(?:kalau|kalo|jika) (?:aku |saya )?(?:beli|belanja|bayar))\b`)
	guideIntent           = regexp.MustCompile(`\b(apa itu|apa arti|maksudnya|definisi|jelaskan|cara menghitung|bagaimana cara menghitung)\b`)
	personalIntent        = regexp.MustCompile(`\b(dana tersedia|uangku|saldo|batas harian|risiko|rekomendasi|saran|hemat|proyeksi|keuangan|uang saya|uang aku|gajian|pemasukan berikutnya)\b`)
)

func personalChat(ctx context.Context, guide *Service, reader FinanceReader, accountID, timezone, input string) (Answer, error) {
	message, err := validateMessage(input)
	if err != nil {
		return Answer{}, err
	}
	message = strings.ToLower(message)
	topic := guideTopic(message)
	if topic != "" && guideIntent.MatchString(message) {
		// Only fixed guide terms cross the AI Service boundary.
		if guide == nil {
			return Answer{}, errGuideUnavailable
		}
		answer, err := guide.chat(ctx, Request{Skill: "finance", Message: topic})
		if err != nil {
			return Answer{}, errGuideUnavailable
		}
		return answer, nil
	}

	simulate := purchaseWithoutAmount.MatchString(message) ||
		(purchaseIntent.MatchString(message) && amountPattern.MatchString(message)) ||
		(hypotheticalIntent.MatchString(message) && currencyHint.MatchString(message))
	if !simulate && topic == "" && !personalIntent.MatchString(message) {
		return companionAnswer("help", "Coba tanya Safe-to-Spend, kondisi keuanganmu, simulasi belanja, atau arti istilah di Dashboard."), nil
	}
	var amount int64
	if simulate {
		amount = purchaseAmount(message)
		if amount == 0 {
			return companionAnswer("needs_amount", "Tulis satu nominal rupiah seperti Rp300 ribu."), nil
		}
	}
	summary, err := reader.SummaryForAccount(ctx, accountID, timezone)
	if err != nil {
		return Answer{}, err
	}
	if summary == nil {
		return companionAnswer("setup_required", "Isi Financial Setup untuk periode ini supaya aku bisa membaca dana tersedia dan menghitung batas harianmu."), nil
	}
	if simulate {
		return purchaseAnswer(*summary, amount), nil
	}
	return summaryAnswer(*summary), nil
}

func guideTopic(message string) string {
	switch {
	case strings.Contains(message, "safe-to-spend"), strings.Contains(message, "safe to spend"), strings.Contains(message, "batas harian"):
		return "Safe-to-Spend"
	case strings.Contains(message, "runway"):
		return "Financial Runway"
	case strings.Contains(message, "risiko"), strings.Contains(message, "resiko"):
		return "Status risiko keuangan"
	case strings.Contains(message, "dana tersedia"), strings.Contains(message, "dana cadangan"), strings.Contains(message, "kewajiban"):
		return "Dana tersedia"
	default:
		return ""
	}
}

func companionAnswer(mode, message string, actions ...string) Answer {
	return Answer{Skill: "finance", Mode: mode, Answer: message, Sources: []Source{}, Actions: actions}
}

func summaryAnswer(summary finance.Summary) Answer {
	answer := companionAnswer("personal", "Batas harian dihitung dari dana yang bisa dipakai sampai pemasukan berikutnya, setelah memperhitungkan kewajiban dan dana cadangan.",
		"Gunakan Safe-to-Spend sebagai acuan pengeluaran harian.")
	switch {
	case summary.AvailableFunds <= 0 || summary.ProjectedBalance < 0:
		answer.Answer = "Dengan kondisi saat ini, ada risiko dana habis sebelum pemasukan berikutnya. Lihat dana tersedia dan proyeksi di bawah."
		answer.Actions = []string{"Tunda belanja opsional dan cek kewajiban yang jatuh tempo."}
	case summary.RunwayDays == nil:
		answer.Actions = []string{"Catat pengeluaran agar proyeksi bisa memperhitungkan pola belanjamu."}
	}
	answer.Financial = &FinancialContext{
		AvailableFunds: summary.AvailableFunds, SafeToSpendDaily: summary.SafeToSpendDaily,
		RemainingDays: summary.Period.RemainingDays, RunwayDays: summary.RunwayDays,
		ProjectedBalance: summary.ProjectedBalance,
	}
	return answer
}

func purchaseAnswer(summary finance.Summary, amount int64) Answer {
	impact, err := finance.SimulatePurchase(summary, amount)
	if err != nil {
		return companionAnswer("needs_amount", "Nominal ini di luar rentang simulasi. Coba masukkan nominal lain.")
	}
	var answer Answer
	switch {
	case !impact.EnoughFunds:
		answer = companionAnswer("simulation", "Pembelian ini melebihi dana yang bisa dipakai pada periode ini.", "Tunda pembelian atau coba nominal yang lebih kecil.")
	case impact.ProjectedAfter < 0:
		answer = companionAnswer("simulation", "Dana saat ini cukup untuk membeli, tetapi proyeksi saldo menjadi defisit jika pola pengeluaran berlanjut.", "Kurangi pengeluaran berikutnya atau tunda pembelian.")
	case impact.ExceedsDailyLimit:
		answer = companionAnswer("simulation", "Dana tersedia cukup. Nominal pembelian melewati batas harian saat ini dan mengurangi jatah hari berikutnya.", "Pakai batas harian setelah pembelian untuk menyesuaikan belanja berikutnya.")
	default:
		answer = companionAnswer("simulation", "Pembelian ini masih masuk dana tersedia dan batas harian saat ini.", "Cek sisa dana dan batas harian setelah pembelian.")
	}
	answer.Simulation = &impact
	return answer
}
