package money

import "testing"

func TestFromRublesAndBack(t *testing.T) {
	cases := []struct {
		rubles  float64
		kopecks Kopecks
	}{
		{6900, 690000},
		{6555.50, 655550},
		{0.1 + 0.2, 30},
		{3790.005, 379001},
	}
	for _, tc := range cases {
		if got := FromRubles(tc.rubles); got != tc.kopecks {
			t.Errorf("FromRubles(%v) = %d, want %d", tc.rubles, got, tc.kopecks)
		}
	}
	if FromRubles(14900).ToRubles() != 14900 {
		t.Fatal("round trip failed")
	}
}

func TestPercent(t *testing.T) {
	if got := Kopecks(690000).Percent(15); got != 103500 {
		t.Fatalf("15%% of 6900 = %d", got)
	}
	if got := Kopecks(399000).Percent(5); got != 19950 {
		t.Fatalf("5%% of 3990 = %d", got)
	}
}

func TestParse(t *testing.T) {
	for input, want := range map[string]Kopecks{
		"6900":    690000,
		"6900.50": 690050,
		"6900,50": 690050,
	} {
		got, err := Parse(input)
		if err != nil || got != want {
			t.Errorf("Parse(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
}
