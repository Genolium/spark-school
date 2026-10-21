package bot

import "testing"

func TestReceiptPromoCode(t *testing.T) {
	tests := []struct {
		name         string
		sessionPromo string
		discountPct  int
		userPromo    string
		want         string
	}{
		{name: "session code wins", sessionPromo: "spark5", discountPct: 5, userPromo: "OTHER", want: "SPARK5"},
		{name: "legacy session falls back to profile", sessionPromo: "", discountPct: 5, userPromo: "spark5", want: "SPARK5"},
		{name: "no discount means no promo", sessionPromo: "", discountPct: 0, userPromo: "SPARK5", want: ""},
		{name: "no promo anywhere", sessionPromo: "", discountPct: 5, userPromo: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := receiptPromoCode(tt.sessionPromo, tt.discountPct, tt.userPromo); got != tt.want {
				t.Fatalf("receiptPromoCode() = %q, want %q", got, tt.want)
			}
		})
	}
}
