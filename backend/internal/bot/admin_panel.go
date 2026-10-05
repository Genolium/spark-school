package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/handlers"
	"github.com/spark-school/backend/internal/models"
)

// IsAdmin checks if Telegram user ID matches configured AdminTelegramID.
func (s *BotService) IsAdmin(userID int64) bool {
	if s.cfg == nil || s.cfg.AdminTelegramID == 0 {
		return false
	}
	return userID == s.cfg.AdminTelegramID
}

// SendAdminDashboard displays the main curator in-line cockpit.
func (s *BotService) SendAdminDashboard(chatID telego.ChatID) {
	if s.bot == nil {
		return
	}

	caption := `🎛 <b>Панель куратора проекта «так называемый SPARK»</b>

Здравствуйте, <b>так называемый Иль</b>!

Здесь доступно моментальное управление проектом прямо из Telegram:
• Живая статистика выручки и мест
• Модерация чеков об оплате
• Создание промокодов на лету:
  <code>/promo create FRIEND10 10%</code>
• Рассылка анонсов всем пользователям бота`

	m := tu.Message(chatID, caption).WithParseMode(telego.ModeHTML).WithReplyMarkup(AdminPanelKeyboard())
	_, _ = s.bot.SendMessage(m)
}

// SendAdminLiveStats fetches real-time stats from Postgres and sends formatted message.
func (s *BotService) SendAdminLiveStats(chatID telego.ChatID) {
	if s.bot == nil || s.db == nil {
		return
	}

	var totalStudents int64
	var activeStudents int64
	s.db.Model(&models.User{}).Count(&totalStudents)
	s.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Count(&activeStudents)

	totalCapacity, spotsLeft := handlers.CalculateCapacityAndSpots(activeStudents)

	var revenue float64
	s.db.Model(&models.PaymentReceipt{}).Where("status = ?", "approved").Select("COALESCE(SUM(amount), 0)").Scan(&revenue)

	var pendingReceipts int64
	s.db.Model(&models.PaymentReceipt{}).Where("status = ?", "pending").Count(&pendingReceipts)

	var activePromos int64
	s.db.Model(&models.PromoCode{}).Where("is_active = true").Count(&activePromos)

	statsText := fmt.Sprintf(`📊 <b>Живая статистика проекта «так называемый SPARK»</b>
<i>Обновлено: %s</i>

💰 <b>Касса проекта:</b> %.0f ₽
👥 <b>Всего пользователей в боте:</b> %d
🎓 <b>Студентов с доступом:</b> %d
🔥 <b>Вместимость потока:</b> %d мест (свободно: <b>%d</b>)

⏳ <b>Чеков на проверке:</b> %d
🎟 <b>Активных промокодов:</b> %d`,
		time.Now().Format("15:04:05 02.01.2006"),
		revenue, totalStudents, activeStudents, totalCapacity, spotsLeft,
		pendingReceipts, activePromos,
	)

	btnRefresh := tu.InlineKeyboardButton("🔄 Обновить").WithCallbackData("admin_stats")
	btnBack := tu.InlineKeyboardButton("⬅️ Назад в админку").WithCallbackData("admin_menu")
	kb := tu.InlineKeyboard(tu.InlineKeyboardRow(btnRefresh), tu.InlineKeyboardRow(btnBack))

	m := tu.Message(chatID, statsText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	_, _ = s.bot.SendMessage(m)
}

// HandleAdminPromoCommand processes /promo create [CODE] [DISCOUNT%]
func (s *BotService) HandleAdminPromoCommand(chatID telego.ChatID, from *telego.User, text string) {
	if !s.IsAdmin(from.ID) {
		return
	}

	parts := strings.Fields(text)
	// Expecting: /promo create CODE DISCOUNT%
	if len(parts) < 3 {
		helpText := `ℹ️ <b>Формат создания промокода:</b>

<code>/promo create FRIEND10 10%</code>
или
<code>/promo create SPARK5 5</code>

<i>Код будет сразу активен для студентов и применится при оплате!</i>`
		m := tu.Message(chatID, helpText).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return
	}

	action := strings.ToLower(parts[1])
	if action != "create" && action != "add" {
		helpText := "Неизвестное действие. Используйте: <code>/promo create КОД ПРОЦЕНТ</code>"
		m := tu.Message(chatID, helpText).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(parts[2]))
	discountPct := 5
	if len(parts) >= 4 {
		pctStr := strings.TrimSuffix(parts[3], "%")
		if val, err := strconv.Atoi(pctStr); err == nil && val > 0 && val <= 100 {
			discountPct = val
		}
	}

	if s.db == nil {
		return
	}

	// Check existing
	var existing models.PromoCode
	if err := s.db.Where("UPPER(code) = ?", code).First(&existing).Error; err == nil {
		reply := fmt.Sprintf("⚠️ Промокод <code>%s</code> уже существует (Скидка: %d%%, Использований: %d).", code, existing.DiscountPercent, existing.UsesCount)
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return
	}

	promo := models.PromoCode{
		Code:            code,
		DiscountPercent: discountPct,
		OwnerTelegramID: from.ID,
		OwnerUsername:   from.Username,
		RewardAmount:    1000.00,
		IsActive:        true,
	}

	if err := s.db.Create(&promo).Error; err != nil {
		reply := "❌ Ошибка при сохранении промокода: " + err.Error()
		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(m)
		return
	}

	successMsg := fmt.Sprintf(`✅ <b>Промокод успешно создан и активен!</b>

🎟 Код: <code>%s</code>
🏷 Скидка: <b>%d%%</b>
👤 Создатель: так называемый Иль (@%s)

Студенты могут активировать его прямо в боте или на сайте!`, code, discountPct, from.Username)

	m := tu.Message(chatID, successMsg).WithParseMode(telego.ModeHTML)
	_, _ = s.bot.SendMessage(m)
}

// ExecuteBroadcast sends message to all users in database.
func (s *BotService) ExecuteBroadcast(adminChatID telego.ChatID, broadcastText string) {
	if s.db == nil || s.bot == nil {
		return
	}

	var users []models.User
	if err := s.db.Where("telegram_id != 0").Find(&users).Error; err != nil {
		m := tu.Message(adminChatID, "❌ Ошибка загрузки списка пользователей: "+err.Error())
		_, _ = s.bot.SendMessage(m)
		return
	}

	startMsg := fmt.Sprintf("🚀 Запуск рассылки на <b>%d</b> получателей...", len(users))
	m := tu.Message(adminChatID, startMsg).WithParseMode(telego.ModeHTML)
	_, _ = s.bot.SendMessage(m)

	sentCount := 0
	failCount := 0

	go func(recipientList []models.User, text string) {
		for _, u := range recipientList {
			// Don't send duplicate to admin
			if u.TelegramID == s.cfg.AdminTelegramID {
				continue
			}

			userChatID := tu.ID(u.TelegramID)
			broadcastMsg := tu.Message(userChatID, text).
				WithParseMode(telego.ModeHTML).
				WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(
					tu.InlineKeyboardButton("Открыть меню").WithCallbackData("action_menu"),
				)))

			if _, err := s.bot.SendMessage(broadcastMsg); err != nil {
				failCount++
				log.Printf("[Broadcast Fail] telegram_id=%d: %v", u.TelegramID, err)
			} else {
				sentCount++
			}
			// Rate limiting to respect Telegram's 30 msgs/sec
			time.Sleep(40 * time.Millisecond)
		}

		reportMsg := fmt.Sprintf(`✅ <b>Рассылка завершена!</b>

📬 Успешно доставлено: <b>%d</b>
⚠️ Ошибок доставки (блокировка бота): <b>%d</b>`, sentCount, failCount)

		report := tu.Message(adminChatID, reportMsg).WithParseMode(telego.ModeHTML)
		_, _ = s.bot.SendMessage(report)
	}(users, broadcastText)
}
