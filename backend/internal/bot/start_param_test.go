package bot

import "testing"

func TestParsePayStartParam(t *testing.T) {
	cases := []struct {
		in                    string
		tier, promo, ref, src string
		attr                  string
		lower                 bool
	}{
		{in: "pay_accelerator", tier: "accelerator"},
		{in: "pay_vip_DEADLINE", tier: "vip", promo: "DEADLINE"},
		{in: "pay_accelerator_DEADLINE_a_apy123abc", tier: "accelerator", promo: "DEADLINE", attr: "apy123abc"},
		{in: "pay_accelerator_a_apy123abc", tier: "accelerator", attr: "apy123abc"},
		{in: "pay_accelerator_a_apy123abc_ref_alex", tier: "accelerator", attr: "apy123abc", ref: "alex"},
		{in: "pay_accelerator_pe_lower_a_apy123abc", tier: "accelerator", attr: "apy123abc", lower: true},
		{in: "pay_accelerator_DEADLINE_a_apy123abc_ref_alex", tier: "accelerator", promo: "DEADLINE", attr: "apy123abc", ref: "alex"},
		{in: "pay_accelerator_src_youtube", tier: "accelerator", src: "youtube"},
		{in: "pay_vip_DEADLINE_src_youtube_a_apy123abc_ref_alex", tier: "vip", promo: "DEADLINE", attr: "apy123abc", ref: "alex", src: "youtube"},
	}
	for _, tc := range cases {
		got := parsePayStartParam(tc.in)
		if got.Tier != tc.tier || got.Promo != tc.promo || got.Ref != tc.ref || got.Source != tc.src || got.AttrToken != tc.attr || got.PriceLower != tc.lower {
			t.Errorf("parsePayStartParam(%q) = %+v, want tier=%q promo=%q ref=%q src=%q attr=%q lower=%v",
				tc.in, got, tc.tier, tc.promo, tc.ref, tc.src, tc.attr, tc.lower)
		}
	}
}

func TestDiscountedPriceMatchesBackendRounding(t *testing.T) {
	cases := []struct{ base, pct, want int }{
		{10990, 14, 9451},
		{19990, 14, 17191},
		{10990, 5, 10441},
		{3333, 14, 2866},
		{10990, 0, 10990},
	}
	for _, tc := range cases {
		if got := discountedPrice(tc.base, tc.pct); got != tc.want {
			t.Errorf("discountedPrice(%d, %d) = %d, want %d", tc.base, tc.pct, got, tc.want)
		}
	}
}
