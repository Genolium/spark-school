// Package money contains integer-kopeck arithmetic for financial operations.
// Ruble floats are only used at the database/JSON boundary; every calculation
// inside the application is performed on int64 kopecks.
package money

import (
	"math"
	"strconv"
	"strings"
)

// Kopecks is an integer amount in Russian kopecks (1 RUB = 100 kopecks).
type Kopecks int64

// FromRubles converts a ruble amount to kopecks with half-up rounding.
func FromRubles(rubles float64) Kopecks {
	if math.IsNaN(rubles) || math.IsInf(rubles, 0) {
		return 0
	}
	return Kopecks(math.Round(rubles * 100))
}

// ToRubles converts kopecks to rubles for legacy numeric columns and JSON.
func (k Kopecks) ToRubles() float64 {
	return float64(k) / 100
}

func (k Kopecks) String() string {
	negative := k < 0
	value := k
	if negative {
		value = -value
	}
	rubles := int64(value) / 100
	cents := int64(value) % 100
	out := strconv.FormatInt(rubles, 10) + "."
	if cents < 10 {
		out += "0"
	}
	out += strconv.FormatInt(cents, 10)
	if negative {
		out = "-" + out
	}
	return out
}

// Percent returns a percentage of the amount, rounded half-up to kopecks.
func (k Kopecks) Percent(percent int) Kopecks {
	if percent <= 0 {
		return 0
	}
	return Kopecks((int64(k)*int64(percent) + 50) / 100)
}

// Add returns a new amount; callers should prefer explicit sums to avoid
// accidental float conversion.
func (k Kopecks) Add(other Kopecks) Kopecks { return k + other }

// Sub returns a new amount.
func (k Kopecks) Sub(other Kopecks) Kopecks { return k - other }

// Parse reads "1234.56" or "1234,56".
func Parse(raw string) (Kopecks, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, err
	}
	return FromRubles(value), nil
}
