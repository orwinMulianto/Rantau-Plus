package assistant

import (
	"math/big"
	"regexp"
	"strings"
)

var (
	amountPattern = regexp.MustCompile(`(-[ \t]*)?\b(?:rp\.?[ \t]*)?([0-9]+(?:[.,][0-9]+)*)(?:[ \t]*(ribu|juta|rupiah|rb|jt|k))?\b`)
	currencyHint  = regexp.MustCompile(`\brp[. \t0-9]|[0-9][ \t]*(ribu|juta|rupiah|rb|jt|k)\b`)
	groupedAmount = regexp.MustCompile(`^[0-9]{1,3}(?:\.[0-9]{3})+$`)
)

func purchaseAmount(message string) int64 {
	matches := amountPattern.FindAllStringSubmatch(message, 2)
	if len(matches) != 1 || matches[0][1] != "" {
		return 0
	}
	number, unit := matches[0][2], matches[0][3]
	var multiplier int64 = 1
	switch unit {
	case "ribu", "rb", "k":
		multiplier = 1000
	case "juta", "jt":
		multiplier = 1000000
	}
	// Periods group rupiah digits. A comma is a decimal separator for a unit such as juta.
	whole, fractional, hasComma := strings.Cut(number, ",")
	if (strings.Contains(whole, ".") && !groupedAmount.MatchString(whole)) ||
		(hasComma && (multiplier == 1 || strings.ContainsAny(fractional, ".,"))) {
		return 0
	}
	number = strings.ReplaceAll(whole, ".", "")
	if hasComma {
		number += "." + fractional
	}
	value, ok := new(big.Rat).SetString(number)
	if !ok {
		return 0
	}
	value.Mul(value, big.NewRat(multiplier, 1))
	if !value.IsInt() || value.Sign() <= 0 || !value.Num().IsInt64() {
		return 0
	}
	return value.Num().Int64()
}
