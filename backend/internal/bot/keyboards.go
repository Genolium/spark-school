package bot

import (
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/mymmrac/telego"
)

// Verified Permanent Telegram File IDs
const (
	PhotoGoldenGate = "AgACAgIAAxkDAAOWasIxYseSAb_vnw85duiZzZ0AATAQAAIMIGsbt0QRShR4PG9JDiVBAQADAgADdwADPQQ"
	PhotoProgramInfo = "AgACAgIAAxkDAAOYasIxa-IKgi8-xStMQEfjbXFfeAEAAg4gaxu3RBFKEmLHSujsPDMBAAMCAAN3AAM9BA"
	PhotoEssayAudit = "AgACAgIAAxkDAAOTasIxW9iIlsbAlYmg1cYlnc21TQkAAgogaxu3RBFKrmPgaWGgkNcBAAMCAAN5AAM9BA"
	PhotoInterview   = "AgACAgIAAxkDAAOXasIxZiE9Oldv2KfBsG2Q_z97ZVsAAg0gaxu3RBFKSi8nrfCubP8BAAMCAAN5AAM9BA"
)

// MainMenuKeyboard returns the primary navigation keyboard with blog link.
func MainMenuKeyboard(frontendURL string) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🎯 Рассчитать шансы на грант $20,000").WithCallbackData("calc_start"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💳 Купить курс (от 6 900 ₽)").WithCallbackData("action_payment"),
			tu.InlineKeyboardButton("🎟 Промокод -5%").WithCallbackData("promo_enter"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🇺🇸 Блог про мою поездку в США").WithURL("https://t.me/so_called_spark"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🌐 Регламент и структура проекта").WithURL(frontendURL),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Консультация с так называемым Илем").WithURL("https://t.me/ilyan_vas"),
		),
	)
}

// PaymentKeyboard returns the checkout keyboard with direct links and back button.
func PaymentKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🎟 Применить промокод -5%").WithCallbackData("promo_enter"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Написать так называемому Илю").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
		),
	)
}

// PromoAppliedKeyboard returns keyboard after successful promo code redemption.
func PromoAppliedKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💬 Написать так называемому Илю в Telegram").WithURL("https://t.me/ilyan_vas"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("⬅️ В меню").WithCallbackData("action_menu"),
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
func CalcResultKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("💳 Купить курс (от 6 900 ₽)").WithCallbackData("action_payment"),
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

// AdminPanelKeyboard returns control buttons for curator in-line panel.
func AdminPanelKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("📊 Статистика онлайн").WithCallbackData("admin_stats"),
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

