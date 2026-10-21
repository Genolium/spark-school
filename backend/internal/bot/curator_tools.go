package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/outbox"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *BotService) send(m *telego.SendMessageParams) error {
	if s.bot == nil || m == nil {
		return nil
	}
	_, err := s.bot.SendMessage(m)
	return err
}

// SendPreparationPlan renders the 6-step preparation plan and current progress.
func (s *BotService) SendPreparationPlan(chatID telego.ChatID, from *telego.User) {
	if from == nil {
		return
	}

	var user models.User
	hasAccess := false
	if s.db != nil {
		if err := s.db.Where("telegram_id = ?", from.ID).First(&user).Error; err == nil {
			hasAccess = user.HasAccess
		}
	}

	var progress models.StudentProgress
	if s.db != nil {
		err := s.db.Where("telegram_id = ?", from.ID).First(&progress).Error
		if err != nil {
			progress = models.StudentProgress{
				TelegramID: from.ID,
			}
			s.db.Create(&progress)
		}
	}

	steps := []struct {
		num  int
		name string
		done bool
		desc string
	}{
		{1, "Распаковка бэкграунда и Narrative Spine", progress.Step1Done, "Поиск центральной идеи, исключение ловушки туриста и выбор фокуса civic engagement."},
		{2, "Одностраничное резюме США (Action Verbs)", progress.Step2Done, "Строгий 1-страничный формат без фото/возраста с сильными глаголами действия и метриками."},
		{3, "Эссе на 500 слов (Тема года)", progress.Step3Done, "Раскрытие темы столкновения мнений в академической среде, Action Hook в первых 5 секундах."},
		{4, "Видеопитч (80–100 секунд)", progress.Step4Done, "Драматургия первых кадров, живая речь без суфлера, съемка и свет."},
		{5, "Академические выписки и TAL-безопасность", progress.Step5Done, "Перевод оценок без нотариуса, проверка на запрещенные технологии двойного назначения."},
		{6, "20-мин Zoom-интервью и экзамен DET", progress.Step6Done, "Личная репетиция с финалистом SPARK по методике STAR и сдача Duolingo Test."},
	}

	completedCount := 0
	for _, st := range steps {
		if st.done {
			completedCount++
		}
	}

	sb := strings.Builder{}
	sb.WriteString("📋 <b>Персональный план подготовки к SPARK 2027</b>\n\n")

	if hasAccess {
		tierTitle := "Акселератор"
		if strings.ToLower(strings.TrimSpace(user.Tier)) == "vip" {
			tierTitle = "VIP"
		} else if strings.ToLower(strings.TrimSpace(user.Tier)) == "basic" {
			tierTitle = "Базовый"
		}
		sb.WriteString(fmt.Sprintf("🎓 Ваш тариф: <b>%s</b> • Прогресс: <b>%d из 6 этапов (%d%%)</b>\n\n", tierTitle, completedCount, (completedCount*100)/6))
	} else {
		sb.WriteString("💡 <i>Ознакомьтесь с пошаговым маршрутом. Выберите тариф для сопровождения куратором:</i>\n\n")
	}

	for _, st := range steps {
		icon := "⏳"
		statusTxt := "в процессе"
		if st.done {
			icon = "✅"
			statusTxt = "готово"
		}
		sb.WriteString(fmt.Sprintf("%s <b>Шаг %d: %s</b> <i>[%s]</i>\n%s\n\n", icon, st.num, st.name, statusTxt, st.desc))
	}

	sb.WriteString("<i>Чтобы отметить шаг завершенным, нажмите кнопку ниже или отправьте команду:</i> <code>/done_step N</code>")

	rows := [][]telego.InlineKeyboardButton{
		{
			tu.InlineKeyboardButton(toggleLabel(1, progress.Step1Done)).WithCallbackData("plan_toggle_1"),
			tu.InlineKeyboardButton(toggleLabel(2, progress.Step2Done)).WithCallbackData("plan_toggle_2"),
			tu.InlineKeyboardButton(toggleLabel(3, progress.Step3Done)).WithCallbackData("plan_toggle_3"),
		},
		{
			tu.InlineKeyboardButton(toggleLabel(4, progress.Step4Done)).WithCallbackData("plan_toggle_4"),
			tu.InlineKeyboardButton(toggleLabel(5, progress.Step5Done)).WithCallbackData("plan_toggle_5"),
			tu.InlineKeyboardButton(toggleLabel(6, progress.Step6Done)).WithCallbackData("plan_toggle_6"),
		},
		// [Временно приостановлено: Сценарий 2 (Сдача работ на проверку куратору)]
		// {
		// 	tu.InlineKeyboardButton("📝 Сдать работу на проверку (/submit_work)").WithCallbackData("action_submit_work_info"),
		// },
		{
			tu.InlineKeyboardButton("⬅️ Главное меню").WithCallbackData("action_menu"),
		},
	}

	m := tu.Message(chatID, sb.String()).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(rows...))
	s.send(m)
}

func toggleLabel(num int, done bool) string {
	if done {
		return fmt.Sprintf("✅ Шаг %d", num)
	}
	return fmt.Sprintf("⬜️ Шаг %d", num)
}

// TogglePlanStep toggles milestone progress for a student.
func (s *BotService) TogglePlanStep(chatID telego.ChatID, from *telego.User, stepNum int) {
	if s.db == nil || from == nil || stepNum < 1 || stepNum > 6 {
		return
	}

	var progress models.StudentProgress
	err := s.db.Where("telegram_id = ?", from.ID).First(&progress).Error
	if err != nil {
		progress = models.StudentProgress{TelegramID: from.ID}
	}

	switch stepNum {
	case 1:
		progress.Step1Done = !progress.Step1Done
	case 2:
		progress.Step2Done = !progress.Step2Done
	case 3:
		progress.Step3Done = !progress.Step3Done
	case 4:
		progress.Step4Done = !progress.Step4Done
	case 5:
		progress.Step5Done = !progress.Step5Done
	case 6:
		progress.Step6Done = !progress.Step6Done
	}
	progress.UpdatedAt = time.Now()
	s.db.Save(&progress)

	s.SendPreparationPlan(chatID, from)
}

// GetSafeInviteLink returns a verified channel invite link (static or dynamically generated single-use link).
func (s *BotService) GetSafeInviteLink(ctx context.Context) string {
	if s.cfg.TelegramInviteLink != "" && strings.HasPrefix(s.cfg.TelegramInviteLink, "http") {
		return s.cfg.TelegramInviteLink
	}
	if s.cfg.TelegramChannelID != 0 && s.bot != nil {
		link, err := s.CreateOneTimeInviteLink(ctx, s.cfg.TelegramChannelID)
		if err == nil && link != "" && strings.HasPrefix(link, "http") {
			return link
		}
	}
	return ""
}

// SendOrdersHistory displays student access state, payment history, and fiscal receipt link.
func (s *BotService) SendOrdersHistory(chatID telego.ChatID, from *telego.User) {
	if from == nil {
		return
	}

	var user models.User
	var receipts []models.PaymentReceipt
	var transactions []models.PaymentTransaction

	if s.db != nil {
		_ = s.db.Where("telegram_id = ?", from.ID).First(&user).Error
		s.db.Where("telegram_id = ?", from.ID).Order("created_at DESC").Find(&receipts)
		s.db.Where("telegram_id = ?", from.ID).Order("created_at DESC").Find(&transactions)
	}

	inviteLink := s.GetSafeInviteLink(context.Background())

	sb := strings.Builder{}
	sb.WriteString("🧾 <b>История заказов и статус доступа</b>\n\n")

	tierTitle := "Не выбран"
	if user.Tier != "" {
		switch strings.ToLower(user.Tier) {
		case "vip":
			tierTitle = "★ VIP"
		case "basic":
			tierTitle = "Базовый"
		default:
			tierTitle = "Акселератор"
		}
	}

	if user.HasAccess {
		sb.WriteString(fmt.Sprintf("🟢 <b>Доступ: АКТИВЕН</b>\n• Тариф: <b>%s</b>\n• Сумма оплаты: <b>%s ₽</b>\n", tierTitle, formatPriceRub(int(user.PaidAmount))))
		if user.ReceiptIssued && user.ReceiptURL != "" {
			sb.WriteString(fmt.Sprintf("• Чек самозанятого: <a href=\"%s\">Смотреть чек №%s</a>\n", user.ReceiptURL, user.ReceiptNumber))
		}
		if inviteLink != "" {
			sb.WriteString(fmt.Sprintf("\n🚀 <b>Закрытый канал потока:</b>\n%s\n\n", inviteLink))
		} else {
			sb.WriteString("\n🚀 <b>Закрытый канал потока:</b>\n<i>Ссылка формируется, обратитесь к куратору при задержке</i>\n\n")
		}
	} else if len(receipts) > 0 && receipts[0].Status == "pending" {
		sb.WriteString(fmt.Sprintf("⏳ <b>Доступ: НА ПРОВЕРКЕ</b>\n• Тариф: <b>%s</b>\n• Скриншот чека получен и проверяется куратором (до 15–30 минут).\n\n", tierTitle))
	} else {
		sb.WriteString("⚪️ <b>Доступ: НЕ АКТИВЕН</b>\nУ вас пока нет оплаченных тарифов.\n\n")
	}

	if len(receipts) > 0 || len(transactions) > 0 {
		sb.WriteString("<b>Операции:</b>\n")
		for _, r := range receipts {
			statusRu := "на проверке"
			if r.Status == "approved" {
				statusRu = "одобрен"
			} else if r.Status == "rejected" {
				statusRu = "отклонен"
			}
			sb.WriteString(fmt.Sprintf("• СБП перевод: %s ₽ (%s, %s)\n", formatPriceRub(int(r.Amount)), r.Tier, statusRu))
		}
		for _, t := range transactions {
			sb.WriteString(fmt.Sprintf("• Онлайн-оплата %s: %s ₽ (%s, %s)\n", t.PayMethod, formatPriceRub(int(t.Amount)), t.Tier, t.Status))
		}
	}

	rows := [][]telego.InlineKeyboardButton{}
	if user.HasAccess {
		if inviteLink != "" && strings.HasPrefix(inviteLink, "http") {
			rows = append(rows, []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton("Открыть закрытый канал 🚀").WithURL(inviteLink),
			})
		}
		rows = append(rows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("📋 Мой план подготовки (/plan)").WithCallbackData("action_plan"),
		})
	} else {
		rows = append(rows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("💳 Оформить тариф").WithCallbackData("action_payment"),
		})
	}
	rows = append(rows, []telego.InlineKeyboardButton{
		tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
	})

	m := tu.Message(chatID, sb.String()).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(rows...))
	s.send(m)
}

// SubmitWork processes a student's submission for curator feedback atomically.
// The user row lock serializes concurrent submissions by the same student, a
// conditional UPDATE protects the last audit credit, and a unique version
// index plus a short duplicate window make command retries safe.
func (s *BotService) SubmitWork(chatID telego.ChatID, from *telego.User, rawText string) error {
	if from == nil {
		return nil
	}
	if s.db == nil {
		_ = s.send(tu.Message(chatID, "❌ База данных временно недоступна. Пожалуйста, попробуйте позже."))
		return fmt.Errorf("database is unavailable")
	}

	// 1. Syntax verification happens before any state change.
	parts := strings.Fields(rawText)
	if len(parts) < 3 {
		help := `📝 <b>Отправка работы на проверку куратору</b>

Формат команды:
<code>/submit_work [тип] [ссылка на Google Docs или текст]</code>

<b>Доступные типы:</b>
• <code>essay</code> — эссе на 500 слов
• <code>resume</code> — американское резюме
• <code>video</code> — ссылка на видеовизитку
• <code>positioning</code> — таблица распаковки бэкграунда

<b>Пример:</b>
<code>/submit_work essay https://docs.google.com/document/d/...</code>

<i>Срок проверки куратором: до 48 часов с подробными комментариями прямо в документе!</i>`
		return s.send(tu.Message(chatID, help).WithParseMode(telego.ModeHTML))
	}

	workType := strings.ToLower(parts[1])
	validTypes := map[string]string{
		"essay":       "Эссе на 500 слов",
		"resume":      "Американское резюме",
		"video":       "Видеовизитка",
		"positioning": "Распаковка бэкграунда",
	}
	typeDisplay, isValid := validTypes[workType]
	if !isValid {
		return s.send(tu.Message(chatID, "❌ Неверный тип работы. Используйте: <code>essay</code>, <code>resume</code>, <code>video</code> или <code>positioning</code>.").WithParseMode(telego.ModeHTML))
	}

	rawContent := strings.Join(parts[2:], " ")
	if len(rawContent) > 2000 {
		rawContent = rawContent[:2000]
	}
	escapedContent := html.EscapeString(rawContent)

	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var user models.User
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("telegram_id = ?", from.ID).First(&user).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.send(tu.Message(chatID, "⛔️ Сначала зарегистрируйтесь в боте командой /start.").WithParseMode(telego.ModeHTML))
		}
		return err
	}

	// 2. Entitlements are re-checked under the row lock.
	if !user.HasAccess {
		tx.Rollback()
		text := `⛔️ <b>Проверка работ доступна только студентам программы</b>

Отправка эссе, резюме и видеовизиток на аудит куратору доступна после оформления тарифа.

Выберите подходящий формат подготовки:`
		kb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("💳 Выбрать тариф").WithCallbackData("action_payment")),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
		)
		return s.send(tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb))
	}

	userTier := strings.ToLower(strings.TrimSpace(user.Tier))
	if userTier == "basic" && (!user.HasAuditUpsell || user.AuditCredits <= 0) {
		tx.Rollback()
		text := `ℹ️ <b>В тариф «Базовый» не входит личная проверка куратором</b>

Вы обучаетесь на тарифе «Базовый» с самостоятельным изучением материалов.
Чтобы получить подробный построчный аудит вашего эссе с комментариями прямо в Google Docs, добавьте разовый аудит или сделайте апгрейд:`
		kb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("🔍 Добавить аудит эссе (1 900 ₽)").WithCallbackData("pay_upsell_audit")),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⚡️ Апгрейд до Акселератора").WithCallbackData("upgrade_to_accelerator")),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
		)
		return s.send(tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb))
	}
	if userTier == "basic" && workType != "essay" {
		tx.Rollback()
		text := `ℹ️ <b>Разовый аудит распространяется только на проверку эссе (500 слов)</b>

Для полной проверки резюме, видеовизитки и таблицы позиционирования сделайте апгрейд до тарифа «Акселератор»:`
		kb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⚡️ Апгрейд до Акселератора").WithCallbackData("upgrade_to_accelerator")),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
		)
		return s.send(tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb))
	}

	// 3. Retry guard: the same command committed a moment ago must not create
	// a second submission when Telegram redelivers the update.
	var duplicate models.StudentSubmission
	if err := tx.Where("telegram_id = ? AND work_type = ? AND content = ? AND submitted_at > ?",
		from.ID, workType, escapedContent, time.Now().Add(-10*time.Minute)).First(&duplicate).Error; err == nil {
		tx.Rollback()
		return nil
	}

	var pendingCount int64
	if err := tx.Model(&models.StudentSubmission{}).
		Where("telegram_id = ? AND status = ?", from.ID, "pending").Count(&pendingCount).Error; err != nil {
		tx.Rollback()
		return err
	}
	if pendingCount >= 3 {
		tx.Rollback()
		return s.send(tu.Message(chatID, "⏳ <b>Лимит очереди:</b> у вас уже есть 3 работы на проверке у куратора.\n\nПожалуйста, дождитесь комментариев по текущим работам перед отправкой новых.").WithParseMode(telego.ModeHTML))
	}

	var existingCount int64
	if err := tx.Model(&models.StudentSubmission{}).
		Where("telegram_id = ? AND work_type = ?", from.ID, workType).Count(&existingCount).Error; err != nil {
		tx.Rollback()
		return err
	}
	version := int(existingCount) + 1

	// 4. Conditional credit deduction for Basic. RowsAffected is the guard:
	// two parallel submissions cannot spend the same last credit twice.
	creditsNote := ""
	if userTier == "basic" {
		res := tx.Model(&models.User{}).
			Where("id = ? AND audit_credits > 0", user.ID).
			UpdateColumn("audit_credits", gorm.Expr("audit_credits - 1"))
		if res.Error != nil {
			tx.Rollback()
			return res.Error
		}
		if res.RowsAffected != 1 {
			tx.Rollback()
			return s.send(tu.Message(chatID, "⏳ <b>Оплаченный аудит уже использован.</b>\n\nДобавьте новый аудит эссе или сделайте апгрейд тарифа.").WithParseMode(telego.ModeHTML))
		}
		if err := tx.Model(&models.User{}).Where("id = ? AND audit_credits <= 0", user.ID).
			UpdateColumn("has_audit_upsell", false).Error; err != nil {
			tx.Rollback()
			return err
		}
		user.AuditCredits--
		creditsNote = fmt.Sprintf("\n• Остаток оплаченных аудитов: <b>%d</b>", user.AuditCredits)
	}

	submission := models.StudentSubmission{
		UserID:      user.ID,
		TelegramID:  from.ID,
		Username:    from.Username,
		WorkType:    workType,
		Content:     escapedContent,
		Version:     version,
		Status:      "pending",
		SubmittedAt: time.Now(),
	}
	if err := tx.Create(&submission).Error; err != nil {
		tx.Rollback()
		return err
	}

	successText := fmt.Sprintf(`✅ <b>Работа принята на проверку! (ID: #%d)</b>

• Тип: <b>%s</b> (версия %d)
• Материал: %s
• Статус: <b>В очереди у куратора</b>%s

Куратор разберёт материал в течение 48 часов и пришлёт разбор прямо сюда.`, submission.ID, typeDisplay, version, escapedContent, creditsNote)
	if err := outbox.EnqueueNotificationTx(tx, from.ID, successText); err != nil {
		tx.Rollback()
		return err
	}

	if s.cfg != nil && s.cfg.AdminTelegramID > 0 {
		alert := fmt.Sprintf(`📥 <b>Новая работа на проверку! (#%d)</b>
От: @%s (ID: %d)
Тип: <b>%s</b> (v%d)
Ссылка/текст: %s`, submission.ID, html.EscapeString(from.Username), from.ID, typeDisplay, version, escapedContent)
		if err := outbox.EnqueueNotificationTx(tx, s.cfg.AdminTelegramID, alert); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}
	return nil
}

// SendCuratorQueue renders the curator SLA queue: oldest first, VIP priority,
// with age and overdue markers. Database remains the source of truth.
func (s *BotService) SendCuratorQueue(chatID telego.ChatID) {
	if s.db == nil {
		return
	}

	var pending []models.StudentSubmission
	s.db.Where("status IN ?", []string{"pending", "reviewing"}).
		Order("submitted_at ASC").Limit(50).Find(&pending)

	if len(pending) == 0 {
		m := tu.Message(chatID, "🎉 <b>Очередь работ пуста!</b> Все эссе, резюме и видео проверены.").WithParseMode(telego.ModeHTML)
		s.send(m)
		return
	}

	tiers := make(map[int64]string, len(pending))
	var userIDs []uint
	for _, sub := range pending {
		userIDs = append(userIDs, sub.UserID)
	}
	var users []models.User
	if len(userIDs) > 0 {
		s.db.Select("id, telegram_id, tier").Where("id IN ?", userIDs).Find(&users)
		for _, u := range users {
			tiers[u.TelegramID] = strings.ToLower(u.Tier)
		}
	}
	sort.SliceStable(pending, func(i, j int) bool {
		iVIP := tiers[pending[i].TelegramID] == "vip"
		jVIP := tiers[pending[j].TelegramID] == "vip"
		if iVIP != jVIP {
			return iVIP
		}
		return pending[i].SubmittedAt.Before(pending[j].SubmittedAt)
	})
	if len(pending) > 15 {
		pending = pending[:15]
	}

	now := time.Now()
	overdue := 0
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("📬 <b>Очередь на проверку (%d работ):</b>\n\n", len(pending)))

	var rows [][]telego.InlineKeyboardButton
	for _, sub := range pending {
		age := now.Sub(sub.SubmittedAt)
		deadline := sub.SubmittedAt.Add(48 * time.Hour)
		deadlineLabel := "до " + deadline.Format("02.01 15:04")
		if now.After(deadline) {
			overdue++
			deadlineLabel = "⚠️ ПРОСРОЧЕНО"
		}
		priority := ""
		if tiers[sub.TelegramID] == "vip" {
			priority = " · ⭐ VIP"
		}
		sb.WriteString(fmt.Sprintf("<b>#%d [%s]%s</b> @%s (v%d)\nСсылка: %s\nВозраст: %s · SLA: %s\n\n",
			sub.ID, html.EscapeString(sub.WorkType), priority, html.EscapeString(sub.Username), sub.Version,
			sub.Content, humanAge(age), deadlineLabel))

		rows = append(rows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton(fmt.Sprintf("✅ Принять #%d", sub.ID)).WithCallbackData(fmt.Sprintf("sub_review_%d", sub.ID)),
			tu.InlineKeyboardButton(fmt.Sprintf("🔄 Доработка #%d", sub.ID)).WithCallbackData(fmt.Sprintf("sub_changes_%d", sub.ID)),
		})
	}

	sb.WriteString(fmt.Sprintf("<b>Просрочено:</b> %d\n\n", overdue))
	sb.WriteString("<b>Команды куратора:</b>\n")
	sb.WriteString("• <code>/reviewed [ID] [комментарий]</code> — одобрить и поставить уведомление в очередь\n")
	sb.WriteString("• <code>/changes [ID] [замечания]</code> — запросить доработку")

	m := tu.Message(chatID, sb.String()).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(rows...))
	s.send(m)
}

func humanAge(age time.Duration) string {
	if age < time.Hour {
		return fmt.Sprintf("%d мин", int(age.Minutes()))
	}
	if age < 48*time.Hour {
		return fmt.Sprintf("%d ч", int(age.Hours()))
	}
	return fmt.Sprintf("%d дн", int(age.Hours()/24))
}

// ProcessSubmissionReview records a curator decision and enqueues the student
// notification through the outbox. The status transition is conditional, so a
// repeated callback cannot review an already completed submission twice.
func (s *BotService) ProcessSubmissionReview(chatID telego.ChatID, from *telego.User, subID uint, notes string, isApproved bool) error {
	if s.db == nil || from == nil {
		return fmt.Errorf("database is unavailable")
	}
	if !s.IsAdmin(from.ID) {
		return s.send(tu.Message(chatID, "⛔️ Доступ ограничен. Действие доступно только куратору проекта."))
	}

	notes = strings.TrimSpace(notes)
	if len(notes) > 2000 {
		notes = notes[:2000]
	}
	escapedNotes := html.EscapeString(notes)
	if escapedNotes == "" {
		escapedNotes = "Комментарий не указан."
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var sub models.StudentSubmission
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", subID).First(&sub).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.send(tu.Message(chatID, fmt.Sprintf("❌ Работа #%d не найдена в базе данных.", subID)))
		}
		return err
	}

	// Terminal states are final. A retry is acknowledged without a second
	// update or a second student notification.
	if sub.Status == "reviewed" || sub.Status == "changes_requested" {
		tx.Rollback()
		return s.send(tu.Message(chatID, fmt.Sprintf("ℹ️ Работа #%d уже завершена (статус: %s). Повторная обработка не требуется.", sub.ID, sub.Status)))
	}
	if sub.Status != "pending" && sub.Status != "reviewing" {
		tx.Rollback()
		return s.send(tu.Message(chatID, fmt.Sprintf("❌ Работа #%d находится в недопустимом статусе: %s.", sub.ID, sub.Status)))
	}

	newStatus := "changes_requested"
	if isApproved {
		newStatus = "reviewed"
	}
	now := time.Now()
	res := tx.Model(&models.StudentSubmission{}).
		Where("id = ? AND status IN ?", sub.ID, []string{"pending", "reviewing"}).
		Updates(map[string]interface{}{
			"status":        newStatus,
			"curator_notes": notes,
			"reviewed_at":   &now,
		})
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if res.RowsAffected != 1 {
		tx.Rollback()
		return s.send(tu.Message(chatID, fmt.Sprintf("❌ Статус работы #%d уже изменился. Обновите очередь.", sub.ID)))
	}

	var studentMsg string
	if isApproved {
		studentMsg = fmt.Sprintf(`🎉 <b>Ваша работа проверена куратором! (#%d)</b>

• Тип: <b>%s</b> (версия %d)
• Статус: <b>Принято ✅</b>
• Комментарий куратора:
<i>%s</i>

Продолжайте подготовку по плану: <code>/plan</code>`, sub.ID, html.EscapeString(sub.WorkType), sub.Version, escapedNotes)
	} else {
		studentMsg = fmt.Sprintf(`📝 <b>Куратор оставил замечания к вашей работе (#%d)</b>

• Тип: <b>%s</b> (версия %d)
• Статус: <b>Требуются правки 🔄</b>
• Комментарий куратора:
<i>%s</i>

Внесите правки и отправьте обновлённую версию:
<code>/submit_work %s [ссылка на обновлённый документ]</code>`, sub.ID, html.EscapeString(sub.WorkType), sub.Version, escapedNotes, html.EscapeString(sub.WorkType))
	}
	if err := outbox.EnqueueNotificationTx(tx, sub.TelegramID, studentMsg); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}

	actionName := "одобрена"
	if !isApproved {
		actionName = "отправлена на доработку"
	}
	confirmAdmin := fmt.Sprintf("✅ Работа #%d (@%s, %s) %s. Уведомление студенту поставлено в очередь на доставку.", sub.ID, html.EscapeString(sub.Username), html.EscapeString(sub.WorkType), actionName)
	return s.send(tu.Message(chatID, confirmAdmin))
}

// HandleReviewedCommand parses and executes /reviewed [ID] [notes]
func (s *BotService) HandleReviewedCommand(chatID telego.ChatID, from *telego.User, rawText string) error {
	if from == nil || !s.IsAdmin(from.ID) {
		m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
		return s.send(m)
	}

	parts := strings.Fields(rawText)
	if len(parts) < 2 {
		m := tu.Message(chatID, "ℹ️ Формат команды: <code>/reviewed [ID] [комментарий студенту]</code>").WithParseMode(telego.ModeHTML)
		return s.send(m)
	}

	idNum, err := strconv.Atoi(parts[1])
	if err != nil || idNum <= 0 {
		m := tu.Message(chatID, "❌ Неверный ID работы. Пример: <code>/reviewed 1 Отличный хук!</code>").WithParseMode(telego.ModeHTML)
		return s.send(m)
	}

	notes := "Работа проверена и одобрена куратором."
	if len(parts) > 2 {
		notes = strings.Join(parts[2:], " ")
	}

	return s.ProcessSubmissionReview(chatID, from, uint(idNum), notes, true)
}

// HandleChangesCommand parses and executes /changes [ID] [notes]
func (s *BotService) HandleChangesCommand(chatID telego.ChatID, from *telego.User, rawText string) error {
	if from == nil || !s.IsAdmin(from.ID) {
		m := tu.Message(chatID, "⛔️ Доступ ограничен. Команда доступна только куратору проекта.")
		return s.send(m)
	}

	parts := strings.Fields(rawText)
	if len(parts) < 2 {
		m := tu.Message(chatID, "ℹ️ Формат команды: <code>/changes [ID] [замечания студенту]</code>").WithParseMode(telego.ModeHTML)
		return s.send(m)
	}

	idNum, err := strconv.Atoi(parts[1])
	if err != nil || idNum <= 0 {
		m := tu.Message(chatID, "❌ Неверный ID работы. Пример: <code>/changes 1 Усилить вводный абзац</code>").WithParseMode(telego.ModeHTML)
		return s.send(m)
	}

	notes := "Куратор рекомендует доработать материал перед финальной подачей."
	if len(parts) > 2 {
		notes = strings.Join(parts[2:], " ")
	}

	return s.ProcessSubmissionReview(chatID, from, uint(idNum), notes, false)
}
