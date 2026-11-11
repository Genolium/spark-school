package bot

import (
	"strings"
	"testing"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/models"
)

func buttonTexts(kb *telego.InlineKeyboardMarkup) []string {
	var out []string
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			out = append(out, b.Text)
		}
	}
	return out
}

// requireButton checks that the button with the given callback shows exactly
// wantAmount rubles ("(N ₽", "+N ₽" or "доплата N ₽").
func requireButton(t *testing.T, name string, kb *telego.InlineKeyboardMarkup, callback, wantAmount string) {
	t.Helper()
	for _, row := range kb.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != callback {
				continue
			}
			for _, prefix := range []string{"(", "+", "доплата "} {
				if strings.Contains(b.Text, prefix+wantAmount+" ₽") {
					return
				}
			}
			t.Errorf("%s: button %q = %q, want amount %s ₽", name, callback, b.Text, wantAmount)
			return
		}
	}
	t.Errorf("%s: no button with callback %q in %v", name, callback, buttonTexts(kb))
}

func rub(v float64) string { return formatPriceRub(int(v)) }

// TestKeyboardFallbacksMatchDefaultCoursePricing fails if any bot keyboard
// fallback price diverges from models.DefaultCoursePricing, the single backend
// source of default prices.
func TestKeyboardFallbacksMatchDefaultCoursePricing(t *testing.T) {
	d := models.DefaultCoursePricing

	kb := PaymentKeyboard()
	requireButton(t, "PaymentKeyboard", kb, "pay_sbp_basic", rub(d.BasicPrice))
	requireButton(t, "PaymentKeyboard", kb, "pay_sbp_accelerator", rub(d.AcceleratorPrice))
	requireButton(t, "PaymentKeyboard", kb, "pay_sbp_vip", rub(d.VIPPrice))

	// Zero values in a pricing record also fall back to the defaults.
	kb = PaymentKeyboard(models.CoursePricing{})
	requireButton(t, "PaymentKeyboard(zero)", kb, "pay_sbp_basic", rub(d.BasicPrice))
	requireButton(t, "PaymentKeyboard(zero)", kb, "pay_sbp_accelerator", rub(d.AcceleratorPrice))
	requireButton(t, "PaymentKeyboard(zero)", kb, "pay_sbp_vip", rub(d.VIPPrice))

	kb = YooKassaSelectTierKeyboard(false)
	requireButton(t, "YooKassaSelectTierKeyboard", kb, "pay_sbp_basic", rub(d.BasicPrice))
	kb = YooKassaSelectTierKeyboard(true)
	base := int(d.BasicPrice)
	requireButton(t, "YooKassaSelectTierKeyboard(promo)", kb, "pay_sbp_basic", formatPriceRub(base-(base*standardPromoPercent)/100))

	kb = PromoAppliedKeyboard(nil, nil)
	requireButton(t, "PromoAppliedKeyboard", kb, "pay_sbp_basic", formatPriceRub(discountedPrice(int(d.BasicPrice), standardPromoPercent)))
	requireButton(t, "PromoAppliedKeyboard", kb, "pay_sbp_accelerator", formatPriceRub(discountedPrice(int(d.AcceleratorPrice), standardPromoPercent)))
	requireButton(t, "PromoAppliedKeyboard", kb, "pay_sbp_vip", formatPriceRub(discountedPrice(int(d.VIPPrice), standardPromoPercent)))

	kb = UpgradeOptionsKeyboard("basic")
	requireButton(t, "UpgradeOptionsKeyboard(basic)", kb, "upgrade_to_accelerator", rub(d.AcceleratorPrice-d.BasicPrice))
	requireButton(t, "UpgradeOptionsKeyboard(basic)", kb, "upgrade_to_vip", rub(d.VIPPrice-d.BasicPrice))
	kb = UpgradeOptionsKeyboard("accelerator")
	requireButton(t, "UpgradeOptionsKeyboard(accelerator)", kb, "upgrade_to_vip", rub(d.VIPPrice-d.AcceleratorPrice))

	kb = BasicUpsellAuditKeyboard()
	requireButton(t, "BasicUpsellAuditKeyboard", kb, "upgrade_to_accelerator", rub(d.AcceleratorPrice-d.BasicPrice))
}

// TestKeyboardFallbacksCurrentValues pins the rendered fallback amounts for the
// current default prices (3 333 / 10 990 / 19 990 ₽), so stale literals such as
// 3 990 / 10 900 / 14 900 ₽ cannot come back unnoticed.
func TestKeyboardFallbacksCurrentValues(t *testing.T) {
	kb := PaymentKeyboard()
	requireButton(t, "PaymentKeyboard", kb, "pay_sbp_basic", "3 333")
	requireButton(t, "PaymentKeyboard", kb, "pay_sbp_accelerator", "10 990")
	requireButton(t, "PaymentKeyboard", kb, "pay_sbp_vip", "19 990")

	kb = UpgradeOptionsKeyboard("basic")
	requireButton(t, "UpgradeOptionsKeyboard(basic)", kb, "upgrade_to_accelerator", "7 657")
	requireButton(t, "UpgradeOptionsKeyboard(basic)", kb, "upgrade_to_vip", "16 657")
}
