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
	if s.cfg == nil || s.cfg.AdminTelegramID <= 0 {
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

// SendAdminReceipts fetches pending receipts and presents them for moderation.
func (s *BotService) SendAdminReceipts(chatID telego.ChatID) {
	if s.bot == nil {
		return
	}

	var pendingReceipts []models.PaymentReceipt
	if s.db != nil {
		s.db.Where("status = ?", "pending").Order("created_at asc").Limit(10).Find(&pendingReceipts)
	}

	if len(pendingReceipts) == 0 {
		emptyText := `🧾 <b>Чеки на модерации</b>

🎉 <b>Все чеки проверены!</b>
На данный момент нет чеков, ожидающих подтверждения.

<i>Как только студент отправит чек в боте, вам моментально придёт карточка с кнопками подтверждения.</i>`

		btnRefresh := tu.InlineKeyboardButton("🔄 Обновить").WithCallbackData("admin_receipts")
		btnBack := tu.InlineKeyboardButton("⬅️ В админку").WithCallbackData("admin_menu")
		kb := tu.InlineKeyboard(tu.InlineKeyboardRow(btnRefresh), tu.InlineKeyboardRow(btnBack))

		m := tu.Message(chatID, emptyText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		_, _ = s.bot.SendMessage(m)
		return
	}

	headerText := fmt.Sprintf(`🧾 <b>Чеки на модерации (%d шт.)</b>

Найдено ожидающих проверки платежей: <b>%d</b>.
Ниже отправлены карточки каждого чека с кнопками подтверждения:`, len(pendingReceipts), len(pendingReceipts))

	m := tu.Message(chatID, headerText).WithParseMode(telego.ModeHTML)
	_, _ = s.bot.SendMessage(m)

	for _, receipt := range pendingReceipts {
		r := receipt
		_ = s.SendAdminReceiptCard(chatID.ID, &r)
		time.Sleep(35 * time.Millisecond)
	}

	controlText := "⬆️ Выберите действие по чекам выше или вернитесь в главное меню:"
	btnRefresh := tu.InlineKeyboardButton("🔄 Обновить список чеков").WithCallbackData("admin_receipts")
	btnBack := tu.InlineKeyboardButton("⬅️ Назад в админку").WithCallbackData("admin_menu")
	kb := tu.InlineKeyboard(tu.InlineKeyboardRow(btnRefresh), tu.InlineKeyboardRow(btnBack))

	ctrlMsg := tu.Message(chatID, controlText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	_, _ = s.bot.SendMessage(ctrlMsg)
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

// DailyDigestData encapsulates key metrics summarized in daily digest.
type DailyDigestData struct {
	Date                  string
	NewLeadsToday         int64
	ApprovedPaymentsToday int64
	RevenueToday          float64
	PendingReceipts       int64
	ActiveStudents        int64
	TotalCapacity         int64
	SpotsLeft             int64
	TotalUsers            int64
}

// SendCuratorDailyDigest computes today's summary metrics and sends a formatted digest to curator.
func (s *BotService) SendCuratorDailyDigest(targetChatID ...int64) (*DailyDigestData, error) {
	var adminID int64
	if len(targetChatID) > 0 && targetChatID[0] != 0 {
		adminID = targetChatID[0]
	} else if s.cfg != nil {
		adminID = s.cfg.AdminTelegramID
	}

	if adminID == 0 {
		log.Println("[Daily Digest] No AdminTelegramID configured.")
		return nil, nil
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	data := &DailyDigestData{
		Date: now.Format("02.01.2006"),
	}

	if s.db != nil {
		// 1. New leads registered today
		s.db.Model(&models.User{}).Where("created_at >= ? AND role != ?", todayStart, "admin").Count(&data.NewLeadsToday)

		// 2. Approved payments today (Receipts + Online Transactions)
		var receiptCount int64
		var receiptRevenue float64
		s.db.Model(&models.PaymentReceipt{}).Where("status = 'approved' AND (updated_at >= ? OR created_at >= ?)", todayStart, todayStart).Count(&receiptCount)
		s.db.Model(&models.PaymentReceipt{}).Where("status = 'approved' AND (updated_at >= ? OR created_at >= ?)", todayStart, todayStart).Select("COALESCE(SUM(amount), 0)").Scan(&receiptRevenue)

		var onlineCount int64
		var onlineRevenue float64
		s.db.Model(&models.PaymentTransaction{}).Where("status = 'SUCCESS' AND created_at >= ?", todayStart).Count(&onlineCount)
		s.db.Model(&models.PaymentTransaction{}).Where("status = 'SUCCESS' AND created_at >= ?", todayStart).Select("COALESCE(SUM(amount), 0)").Scan(&onlineRevenue)

		data.ApprovedPaymentsToday = receiptCount + onlineCount
		data.RevenueToday = receiptRevenue + onlineRevenue

		// 4. Pending receipts waiting for review
		s.db.Model(&models.PaymentReceipt{}).Where("status = 'pending'").Count(&data.PendingReceipts)

		// 5. Active students and remaining spots
		s.db.Model(&models.User{}).Where("has_access = true AND role != ?", "admin").Count(&data.ActiveStudents)
		s.db.Model(&models.User{}).Count(&data.TotalUsers)

		cap, left := handlers.CalculateCapacityAndSpots(data.ActiveStudents)
		data.TotalCapacity = int64(cap)
		data.SpotsLeft = int64(left)
	}

	msgText := fmt.Sprintf(`📰 <b>Ежедневный дайджест куратора SPARK</b>
📅 <i>Дата: %s (%s)</i>

👤 <b>Новых лидов сегодня:</b> +%d
💳 <b>Одобренных оплат сегодня:</b> %d
💰 <b>Выручка за сегодня:</b> %.0f ₽

⏳ <b>Чеков на модерации:</b> %d
🔥 <b>Свободных мест на потоке:</b> %d из %d (студентов: %d)

👥 <b>Всего пользователей в CRM:</b> %d

<i>Используйте /admin для быстрого управления проектом.</i>`,
		data.Date, now.Format("15:04"),
		data.NewLeadsToday,
		data.ApprovedPaymentsToday,
		data.RevenueToday,
		data.PendingReceipts,
		data.SpotsLeft, data.TotalCapacity, data.ActiveStudents,
		data.TotalUsers,
	)

	chatID := tu.ID(adminID)
	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🧾 Проверить чеки").WithCallbackData("admin_receipts"),
			tu.InlineKeyboardButton("📊 Живая статистика").WithCallbackData("admin_stats"),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("🎛 Панель управления").WithCallbackData("admin_menu"),
		),
	)

	if s.bot == nil {
		log.Printf("[Daily Digest Mock] Successfully computed digest for curator %d", adminID)
		return data, nil
	}

	m := tu.Message(chatID, msgText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	_, err := s.bot.SendMessage(m)
	if err != nil {
		log.Printf("[Daily Digest Error] SendMessage failed to %d: %v", adminID, err)
	} else {
		log.Printf("[Daily Digest] Successfully dispatched to curator %d", adminID)
	}
	return data, err
}
