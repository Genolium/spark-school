package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
)

func formatPriceRub(amount int) string {
	s := strconv.Itoa(amount)
	if len(s) > 3 {
		return s[:len(s)-3] + " " + s[len(s)-3:]
	}
	return s
}

// Verified Permanent Telegram File IDs
const (
	PhotoGoldenGate  = "AgACAgIAAxkDAAOWasIxYseSAb_vnw85duiZzZ0AATAQAAIMIGsbt0QRShR4PG9JDiVBAQADAgADdwADPQQ"
	PhotoProgramInfo = "AgACAgIAAxkDAAOYasIxa-IKgi8-xStMQEfjbXFfeAEAAg4gaxu3RBFKEmLHSujsPDMBAAMCAAN3AAM9BA"
	PhotoEssayAudit  = "AgACAgIAAxkDAAOTasIxW9iIlsbAlYmg1cYlnc21TQkAAgogaxu3RBFKrmPgaWGgkNcBAAMCAAN5AAM9BA"
	PhotoInterview   = "AgACAgIAAxkDAAOXasIxZiE9Oldv2KfBsG2Q_z97ZVsAAg0gaxu3RBFKSi8nrfCubP8BAAMCAAN5AAM9BA"
)

// MainMenuKeyboard returns the primary navigation keyboard with blog link.
func MainMenuKeyboard(frontendURL string, options ...any) *telego.InlineKeyboardMarkup {
	canUpgrade := false
	p := models.DefaultCoursePricing

	for _, opt := range options {
		switch v := opt.(type) {
		case bool:
			canUpgrade = v
		case models.CoursePricing:
			if v.AcceleratorPrice > 0 {
				p = v
			}
		}
	}

	minPrice := p.BasicPrice
	if minPrice <= 0 {
		minPrice = p.AcceleratorPrice
	}
	buyText := fmt.Sprintf("💳 Купить курс (от %s ₽)", formatPriceRub(int(minPrice)))

	rows := [][]telego.InlineKeyboardButton{
		{
			tu.InlineKeyboardButton("🎯 Рассчитать шансы на грант $20,000").WithCallbackData("calc_start"),
		},
	}

	if canUpgrade {
		rows = append(rows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("🚀 Апгрейд тарифа").WithCallbackData("action_upgrade"),
			tu.InlineKeyboardButton("🎓 Мой статус").WithCallbackData("action_status"),
		})
	}

	rows = append(rows,
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("📋 План подготовки").WithCallbackData("action_plan"),
			tu.InlineKeyboardButton("🎙 Тренажёр интервью").WithCallbackData("action_interview"),
		},
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton(buyText).WithCallbackData("action_payment"),
			tu.InlineKeyboardButton("🎟 Промокод -5%").WithCallbackData("promo_enter"),
		},
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("🇺🇸 Блог про мою поездку в США").WithURL("https://t.me/so_called_spark"),
		},
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("🌐 О проекте").WithURL(frontendURL),
		},
		[]telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("💬 Консультация с так называемым Илем").WithURL("https://t.me/ilyan_vas"),
		},
	)

	return tu.InlineKeyboard(rows...)
}

// Legacy Tribute external payment links (kept for test verification and reference):
// https://web.tribute.tg/p/GIB
// https://web.tribute.tg/p/GID

// PaymentKeyboard returns the checkout keyboard with direct SBP transfer.
func PaymentKeyboard(pricing ...models.CoursePricing) *telego.InlineKeyboardMarkup {
	basicPrice := 3990
	accPrice := 6900
	vipPrice := 14900
	if len(pricing) > 0 {
		if pricing[0].BasicPrice > 0 {
			basicPrice = int(pricing[0].BasicPrice)
		}
		if pricing[0].AcceleratorPrice > 0 {
			accPrice = int(pricing[0].AcceleratorPrice)
		}
		if pricing[0].VIPPrice > 0 {
			vipPrice = int(pricing[0].VIPPrice)
		}
	}
	basicText := fmt.Sprintf("1️⃣ Базовый (%s ₽) — без куратора", formatPriceRub(basicPrice))
	accText := fmt.Sprintf("2️⃣ Акселератор (%s ₽) — СБП", formatPriceRub(accPrice))
	vipText := fmt.Sprintf("3️⃣ VIP (%s ₽) — СБП", formatPriceRub(vipPrice))

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(basicText).WithCallbackData("pay_sbp_basic"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(accText).WithCallbackData("pay_sbp_accelerator"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(vipText).WithCallbackData("pay_sbp_vip"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🎟 Применить промокод -5%").WithCallbackData("promo_enter"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Поддержка куратора в Telegram").WithURL("https://t.me/spark_prep_bot"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// PromoAppliedKeyboard returns keyboard after successful promo code redemption.
func PromoAppliedKeyboard(pricing ...models.CoursePricing) *telego.InlineKeyboardMarkup {
	basicPrice := 3790
	accPrice := 6555
	vipPrice := 14155
	if len(pricing) > 0 {
		if pricing[0].BasicPrice > 0 {
			base := int(pricing[0].BasicPrice)
			basicPrice = base - (base*5)/100
		}
		if pricing[0].AcceleratorPrice > 0 {
			base := int(pricing[0].AcceleratorPrice)
			accPrice = base - (base*5)/100
		}
		if pricing[0].VIPPrice > 0 {
			base := int(pricing[0].VIPPrice)
			vipPrice = base - (base*5)/100
		}
	}
	basicText := fmt.Sprintf("1️⃣ Базовый (%s ₽) — со скидкой 5%%", formatPriceRub(basicPrice))
	accText := fmt.Sprintf("2️⃣ Акселератор (%s ₽) — СБП", formatPriceRub(accPrice))
	vipText := fmt.Sprintf("3️⃣ VIP (%s ₽) — СБП", formatPriceRub(vipPrice))

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(basicText).WithCallbackData("pay_sbp_basic"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(accText).WithCallbackData("pay_sbp_accelerator"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(vipText).WithCallbackData("pay_sbp_vip"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Поддержка куратора в Telegram").WithURL("https://t.me/spark_prep_bot"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// YooKassaSelectTierKeyboard routes to direct SBP checkout.
func YooKassaSelectTierKeyboard(hasPromo bool, pricing ...models.CoursePricing) *telego.InlineKeyboardMarkup {
	basicPrice := 3990
	accPrice := 6900
	vipPrice := 14900
	if len(pricing) > 0 {
		if pricing[0].BasicPrice > 0 {
			basicPrice = int(pricing[0].BasicPrice)
		}
		if pricing[0].AcceleratorPrice > 0 {
			accPrice = int(pricing[0].AcceleratorPrice)
		}
		if pricing[0].VIPPrice > 0 {
			vipPrice = int(pricing[0].VIPPrice)
		}
	}
	if hasPromo {
		basicPrice = basicPrice - (basicPrice*5)/100
		accPrice = accPrice - (accPrice*5)/100
		vipPrice = vipPrice - (vipPrice*5)/100
	}
	basicText := fmt.Sprintf("1️⃣ Оплатить Базовый (%s ₽) — СБП", formatPriceRub(basicPrice))
	accText := fmt.Sprintf("2️⃣ Оплатить Акселератор (%s ₽) — СБП", formatPriceRub(accPrice))
	vipText := fmt.Sprintf("3️⃣ Оплатить VIP (%s ₽) — СБП", formatPriceRub(vipPrice))
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(basicText).WithCallbackData("pay_sbp_basic"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(accText).WithCallbackData("pay_sbp_accelerator"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(vipText).WithCallbackData("pay_sbp_vip"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ Назад ко всем способам").WithCallbackData("action_payment"),
		),
	)
}

// PromoRequestKeyboard returns keyboard when asking user to type promo code.
func PromoRequestKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💳 Оплатить без промокода").WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// Calculator Step 1 Keyboard: Age & Citizenship
func CalcQ1Keyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("✅ Да, мне 18–21 и я гражданин РФ").WithCallbackData("calc_q1_yes"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("❌ Нет, мне меньше 18 или старше 21").WithCallbackData("calc_q1_no"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// Calculator Step 2 Keyboard: University Year
func CalcQ2Keyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("1–2 курс бакалавриата / специалитета").WithCallbackData("calc_q2_12"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("3 курс (ИЛИ остаётся ещё год учёбы)").WithCallbackData("calc_q2_3"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Выпускной курс (заканчиваю летом)").WithCallbackData("calc_q2_senior"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// Calculator Step 3 Keyboard: English Level
func CalcQ3Keyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🥉 B1 (Intermediate)").WithCallbackData("calc_q3_b1"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🥈 B2 (Upper-Intermediate)").WithCallbackData("calc_q3_b2"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🥇 C1 (Advanced)").WithCallbackData("calc_q3_c1"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// Calculator Result Keyboard
func CalcResultKeyboard(pricing ...models.CoursePricing) *telego.InlineKeyboardMarkup {
	p := models.DefaultCoursePricing
	if len(pricing) > 0 && pricing[0].AcceleratorPrice > 0 {
		p = pricing[0]
	}
	buyText := fmt.Sprintf("💳 Купить курс (от %s ₽)", formatPriceRub(int(p.AcceleratorPrice)))

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(buyText).WithCallbackData("action_payment"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🎟 Применить промокод -5%").WithCallbackData("promo_enter"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🇺🇸 Блог про мою поездку в США").WithURL("https://t.me/so_called_spark"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// UpgradeOptionsKeyboard returns upgrade choices based on student's current tier.
func UpgradeOptionsKeyboard(currentTier string, diffAmount ...float64) *telego.InlineKeyboardMarkup {
	currentTier = strings.ToLower(strings.TrimSpace(currentTier))
	if currentTier == "vip" {
		return tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
			),
		)
	}

	if currentTier == "basic" {
		diffAcc := 2910  // 6900 - 3990
		diffVIP := 10910 // 14900 - 3990
		if len(diffAmount) > 0 && diffAmount[0] > 0 {
			diffAcc = int(diffAmount[0])
		}
		if len(diffAmount) > 1 && diffAmount[1] > 0 {
			diffVIP = int(diffAmount[1])
		}
		return tu.InlineKeyboard(
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf("🚀 До Акселератора (доплата %s ₽)", formatPriceRub(diffAcc))).WithCallbackData("upgrade_to_accelerator"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton(fmt.Sprintf("👑 До VIP (доплата %s ₽)", formatPriceRub(diffVIP))).WithCallbackData("upgrade_to_vip"),
			),
			tu.InlineKeyboardRow(
				tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
			),
		)
	}

	diff := 8000
	if len(diffAmount) > 0 && diffAmount[0] > 0 {
		diff = int(diffAmount[0])
	}
	btnText := fmt.Sprintf("👑 До VIP (доплата %s ₽)", formatPriceRub(diff))

	// For accelerator (or any active non-VIP student): upgrade to VIP
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton(btnText).WithCallbackData("upgrade_to_vip"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// AdminPanelKeyboard returns control buttons for curator in-line panel.
func AdminPanelKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📊 Статистика онлайн").WithCallbackData("admin_stats"),
			tu.InlineKeyboardButton("📰 Дайджест за сегодня").WithCallbackData("admin_digest"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🧾 Чеки на модерации").WithCallbackData("admin_receipts"),
			tu.InlineKeyboardButton("🎟 Список промокодов").WithCallbackData("admin_promos"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📢 Рассылка по базе").WithCallbackData("admin_broadcast_prompt"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ Закрыть панель").WithCallbackData("action_menu"),
		),
	)
}

// BasicUpsellAuditKeyboard returns buttons for the +3 days upsell message (Order Bump: Текстовый аудит эссе за 1 900 ₽).
func BasicUpsellAuditKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("✍️ Оформить аудит эссе (1 900 ₽)").WithCallbackData("pay_upsell_audit"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Задать вопрос так называемому Илю").WithURL("https://t.me/ilyan_vas"),
		),
	)
}
