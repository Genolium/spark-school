package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"math/rand"
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

var interviewQuestions = []struct {
	Question string
	Context  string
	Tip      string
}{
	{
		Question: "Tell me about a time you had a serious disagreement with a peer or teammate. How did you handle it?",
		Context:  "Вопрос проверяет зрелость и способность к конструктивному диалогу (Section 2027 clash of perspectives).",
		Tip:      "Используй формулу STAR: Situation (15 сек), Task (15 сек), Action (60 сек — твоя дипломатия и компромисс), Result (30 сек — общий результат проекта).",
	},
	{
		Question: "Why did you choose the SPARK exchange program, and what unique perspective will you bring to an American campus?",
		Context:  "Комиссия отсекает туристов. Важен ответ про civic engagement и культурный обмен.",
		Tip:      "Расскажи про свой конкретный опыт в родном вузе (проекты, волонтерство) и как ты познакомишь американцев с реальной жизнью и культурой своего региона.",
	},
	{
		Question: "Describe a project you initiated from scratch where you faced unexpected failure or obstacles. What did you learn?",
		Context:  "Проверяет устойчивость к неудачам (resilience) и ответственность.",
		Tip:      "Не обвиняй других. Честно покажи, где ошибся в планировании, как перестроил процесс и какой вывод сделал на будущее.",
	},
	{
		Question: "How do you plan to implement the knowledge and network gained in the US after returning to your home community?",
		Context:  "Проверяет соблюдение правила двух лет визы J-1 (Section 212(e)) и возвращение на родину.",
		Tip:      "Назови конкретную инициативу: открытый воркшоп на кафедре, развитие студенческого медиа или совместный проект со своим вузом.",
	},
}

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
			tu.InlineKeyboardButton("🎙 Тренажер интервью (/interview)").WithCallbackData("action_interview"),
		},
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

// BookingSlot represents a concrete scheduled Zoom appointment slot with specific calendar date and time.
type BookingSlot struct {
	Key     string    // e.g. "20261027_1800"
	Display string    // e.g. "🗓 Вт, 27 октября в 18:00 МСК"
	Time    time.Time // absolute timestamp in MSK
}

// getUpcomingSlots calculates dynamic concrete calendar dates for upcoming interview slots.
func getUpcomingSlots(now time.Time) []BookingSlot {
	msk := time.FixedZone("MSK", 3*3600)
	nowMSK := now.In(msk)

	type recurringDef struct {
		weekday time.Weekday
		hour    int
		minute  int
		nameRu  string
	}

	defs := []recurringDef{
		{weekday: time.Tuesday, hour: 18, minute: 0, nameRu: "Вторник"},
		{weekday: time.Thursday, hour: 19, minute: 30, nameRu: "Четверг"},
		{weekday: time.Saturday, hour: 14, minute: 0, nameRu: "Суббота"},
	}

	monthsRu := map[time.Month]string{
		time.January:   "января",
		time.February:  "февраля",
		time.March:     "марта",
		time.April:     "апреля",
		time.May:       "мая",
		time.June:      "июня",
		time.July:      "июля",
		time.August:    "августа",
		time.September: "сентября",
		time.October:   "октября",
		time.November:  "ноября",
		time.December:  "декабря",
	}

	var slots []BookingSlot
	// Look up to 14 days ahead
	for d := 0; d < 14 && len(slots) < 4; d++ {
		candidateDay := nowMSK.AddDate(0, 0, d)
		for _, def := range defs {
			if candidateDay.Weekday() == def.weekday {
				slotTime := time.Date(
					candidateDay.Year(), candidateDay.Month(), candidateDay.Day(),
					def.hour, def.minute, 0, 0, msk,
				)
				// Must be at least 2 hours in the future
				if slotTime.After(nowMSK.Add(2 * time.Hour)) {
					key := fmt.Sprintf("%04d%02d%02d_%02d%02d",
						slotTime.Year(), slotTime.Month(), slotTime.Day(), slotTime.Hour(), slotTime.Minute())
					display := fmt.Sprintf("🗓 %s, %d %s в %02d:%02d МСК",
						def.nameRu, slotTime.Day(), monthsRu[slotTime.Month()], slotTime.Hour(), slotTime.Minute())

					slots = append(slots, BookingSlot{
						Key:     key,
						Display: display,
						Time:    slotTime,
					})
					if len(slots) >= 4 {
						break
					}
				}
			}
		}
	}
	return slots
}

// BookCallDialog offers available Zoom simulation interview slots for Accelerator & VIP students.
// Prevents concurrency race conditions, double booking, and unauthorized bookings.
func (s *BotService) BookCallDialog(chatID telego.ChatID, from *telego.User, slotKey string) {
	if from == nil {
		return
	}

	if s.db == nil {
		m := tu.Message(chatID, "❌ База данных временно недоступна. Попробуйте позже.")
		s.send(m)
		return
	}

	var user models.User
	_ = s.db.Where("telegram_id = ?", from.ID).First(&user)

	// Entitlement check: Only Accelerator and VIP have 1-on-1 calls included
	userTier := strings.ToLower(strings.TrimSpace(user.Tier))
	if !user.HasAccess || (userTier != "accelerator" && userTier != "vip") {
		text := `📅 <b>Запись на личный Zoom-созвон 1-на-1</b>

Индивидуальная 45–60 минутная симуляция 20-минутного интервью с комиссией SPARK и подробным разбором ошибок включена в тарифы <b>Акселератор</b> и <b>VIP</b>.

Пожалуйста, обновите тариф до Акселератора или VIP, чтобы открыть доступ к слотам куратора.`
		btnUpgrade := tu.InlineKeyboardButton("🚀 Выбрать Акселератор / VIP").WithCallbackData("action_payment")
		btnBack := tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnUpgrade), tu.InlineKeyboardRow(btnBack)))
		s.send(m)
		return
	}

	upcomingSlots := getUpcomingSlots(time.Now())

	// If slotKey is provided, attempt atomic reservation
	if slotKey != "" {
		// Find matching display text
		var targetSlot *BookingSlot
		for _, sl := range upcomingSlots {
			if sl.Key == slotKey {
				targetSlot = &sl
				break
			}
		}
		if targetSlot == nil {
			m := tu.Message(chatID, "❌ Этот слот уже недействителен или прошёл. Пожалуйста, выберите актуальное время.")
			s.send(m)
			s.BookCallDialog(chatID, from, "")
			return
		}

		// Atomic database transaction to prevent double-booking and concurrent collision
		tx := s.db.Begin()
		if tx.Error != nil {
			m := tu.Message(chatID, "❌ Не удалось начать бронирование из-за ошибки базы данных. Попробуйте снова.")
			s.send(m)
			return
		}
		defer func() {
			if r := recover(); r != nil {
				tx.Rollback()
			}
		}()

		// 1. Check if user already holds an active confirmed booking
		var existingUserBooking models.CallBooking
		if err := tx.Where("telegram_id = ? AND status = ? AND cancelled_at IS NULL", from.ID, "confirmed").First(&existingUserBooking).Error; err == nil {
			tx.Rollback()
			zoomInfo := "Ссылка придёт за 1 час до начала"
			if existingUserBooking.ZoomLink != "" && strings.HasPrefix(existingUserBooking.ZoomLink, "http") {
				zoomInfo = existingUserBooking.ZoomLink
			}
			text := fmt.Sprintf(`⚠️ <b>У вас уже есть активная запись на Zoom-симуляцию!</b>

📅 Слот: <b>%s</b>
Ссылка на созвон: %s

В один момент времени может быть активна только одна запись. Если вы хотите перенести время, сначала отмените текущую запись:`,
				existingUserBooking.SlotTime, zoomInfo)
			kb := tu.InlineKeyboard(
				tu.InlineKeyboardRow(tu.InlineKeyboardButton("❌ Отменить текущую запись").WithCallbackData(fmt.Sprintf("cancel_booking_%d", existingUserBooking.ID))),
				tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
			)
			m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
			s.send(m)
			return
		}

		// 2. Check if this specific slot is already reserved by another student
		var existingSlotBooking models.CallBooking
		if err := tx.Where("slot_key = ? AND status = ? AND cancelled_at IS NULL", slotKey, "confirmed").First(&existingSlotBooking).Error; err == nil {
			tx.Rollback()
			text := fmt.Sprintf("❌ <b>Слот «%s» уже занят другим студентом!</b>\n\nПожалуйста, выберите другое свободное время из списка ниже:", targetSlot.Display)
			m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML)
			s.send(m)
			s.BookCallDialog(chatID, from, "")
			return
		}

		// 3. Create confirmed reservation
		zoomMeetingURL := strings.TrimSpace(s.cfg.ZoomMeetingURL)

		booking := models.CallBooking{
			UserID:     user.ID,
			TelegramID: from.ID,
			Username:   from.Username,
			SlotKey:    slotKey,
			SlotTime:   targetSlot.Display,
			CallType:   "zoom_simulation",
			Status:     "confirmed",
			ZoomLink:   zoomMeetingURL,
			CreatedAt:  time.Now(),
		}

		if err := tx.Create(&booking).Error; err != nil {
			tx.Rollback()
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "idx_active_slot_key") || strings.Contains(errStr, "slot_key") || strings.Contains(errStr, "unique") {
				m := tu.Message(chatID, fmt.Sprintf("❌ <b>Слот «%s» уже занят другим студентом!</b>\n\nПожалуйста, выберите другое свободное время.", targetSlot.Display)).WithParseMode(telego.ModeHTML)
				s.send(m)
				s.BookCallDialog(chatID, from, "")
			} else if strings.Contains(errStr, "idx_active_user_booking") || strings.Contains(errStr, "telegram_id") {
				m := tu.Message(chatID, "⚠️ У вас уже есть активная запись на Zoom-симуляцию. Сначала отмените её через /cancel_call.")
				s.send(m)
			} else {
				m := tu.Message(chatID, "❌ Не удалось подтвердить бронирование из-за ошибки базы данных. Попробуйте снова.")
				s.send(m)
			}
			return
		}

		if err := tx.Commit().Error; err != nil {
			m := tu.Message(chatID, "❌ Не удалось завершить сохранение записи. Попробуйте снова.")
			s.send(m)
			return
		}

		zoomText := "🔗 <i>Ссылка на Zoom-конференцию будет сформирована и отправлена куратором за 1 час до начала созвона.</i>"
		if zoomMeetingURL != "" && strings.HasPrefix(zoomMeetingURL, "http") {
			zoomText = fmt.Sprintf(`🔗 Ссылка на Zoom: <a href="%s">%s</a>`, zoomMeetingURL, zoomMeetingURL)
		}

		confirmText := fmt.Sprintf(`✅ <b>Вы успешно записаны на Zoom-симуляцию! (#%d)</b>

📅 Выбранный слот: <b>%s</b>
⏱ Формат: 45–60 минут (20 мин боевая симуляция на английском + разбор)
%s

Перед созвоном повторите технику STAR и подготовьте вопросы комиссии.
Если ваши планы изменятся, отмените запись заранее через команду <code>/cancel_call</code>.`,
			booking.ID, targetSlot.Display, zoomText)

		kb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("❌ Отменить запись").WithCallbackData(fmt.Sprintf("cancel_booking_%d", booking.ID))),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("🎙 Потренироваться в боте (/interview)").WithCallbackData("action_interview")),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
		)

		m := tu.Message(chatID, confirmText).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		s.send(m)

		if s.cfg.AdminTelegramID > 0 {
			alert := fmt.Sprintf("📅 <b>Новая запись на Zoom-созвон! (#%d)</b>\nСтудент: @%s (ID: %d, %s)\nСлот: <b>%s</b>",
				booking.ID, from.Username, from.ID, userTier, targetSlot.Display)
			adminMsg := tu.Message(tu.ID(s.cfg.AdminTelegramID), alert).WithParseMode(telego.ModeHTML)
			s.send(adminMsg)
		}
		return
	}

	// If no slotKey provided, render upcoming slots with availability check
	var activeBooking models.CallBooking
	if err := s.db.Where("telegram_id = ? AND status = ? AND cancelled_at IS NULL", from.ID, "confirmed").First(&activeBooking).Error; err == nil {
		// User already has a booking
		text := fmt.Sprintf(`📅 <b>Ваша текущая запись на Zoom-симуляцию:</b>

• Выбранное время: <b>%s</b>
• Ссылка на Zoom: <a href="%s">%s</a>
• Статус: <b>Подтверждено ✅</b>

Перед созвоном повторите технику STAR по команде /interview.
Чтобы освободить слот или выбрать другое время, отмените текущую запись:`,
			activeBooking.SlotTime, activeBooking.ZoomLink, activeBooking.ZoomLink)

		kb := tu.InlineKeyboard(
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("❌ Отменить запись").WithCallbackData(fmt.Sprintf("cancel_booking_%d", activeBooking.ID))),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("🎙 Тренажёр интервью (/interview)").WithCallbackData("action_interview")),
			tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
		)

		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
		s.send(m)
		return
	}

	// Query already booked slots
	var bookedBookings []models.CallBooking
	s.db.Where("status = ? AND cancelled_at IS NULL", "confirmed").Find(&bookedBookings)
	bookedMap := make(map[string]bool)
	for _, b := range bookedBookings {
		bookedMap[b.SlotKey] = true
	}

	slotsText := `📅 <b>Выберите удобный слот для личной Zoom-симуляции:</b>

Мы проводим боевые созвоны в формате 1-на-1:
1. 20 минут строго на английском по регламенту комиссии SPARK.
2. 25–40 минут детальный разбор структуры ответов, вербальных хуков и жестов.

<i>Выберите свободный слот:</i>`

	rows := [][]telego.InlineKeyboardButton{}
	availableCount := 0
	for _, slot := range upcomingSlots {
		if bookedMap[slot.Key] {
			// Slot is taken
			rows = append(rows, []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton(fmt.Sprintf("🔒 Занято: %s", slot.Display)).WithCallbackData("slot_already_taken"),
			})
		} else {
			availableCount++
			rows = append(rows, []telego.InlineKeyboardButton{
				tu.InlineKeyboardButton(slot.Display).WithCallbackData(fmt.Sprintf("book_slot_%s", slot.Key)),
			})
		}
	}

	if availableCount == 0 {
		slotsText = `📅 <b>Все ближайшие слоты на Zoom-симуляции сейчас заняты.</b>

Новые слоты открываются каждую неделю. Вы также можете написать куратору в поддержку для согласования времени:`
	}

	rows = append(rows, []telego.InlineKeyboardButton{
		tu.InlineKeyboardButton("💬 Согласовать другое время в поддержке").WithURL("https://t.me/spark_prep_bot"),
	})
	rows = append(rows, []telego.InlineKeyboardButton{
		tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
	})

	m := tu.Message(chatID, slotsText).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(rows...))
	s.send(m)
}

// CancelBooking cancels a student's call booking, frees up the slot, and alerts the curator.
func (s *BotService) CancelBooking(chatID telego.ChatID, from *telego.User, bookingID uint) {
	if from == nil || s.db == nil {
		return
	}

	var booking models.CallBooking
	var err error
	if bookingID > 0 {
		err = s.db.Where("id = ? AND telegram_id = ? AND status = ?", bookingID, from.ID, "confirmed").First(&booking).Error
	} else {
		err = s.db.Where("telegram_id = ? AND status = ? AND cancelled_at IS NULL", from.ID, "confirmed").First(&booking).Error
	}

	if err != nil {
		m := tu.Message(chatID, "ℹ️ У вас нет активных записей на созвон для отмены.")
		s.send(m)
		return
	}

	now := time.Now()
	res := s.db.Model(&models.CallBooking{}).
		Where("id = ? AND status = ? AND cancelled_at IS NULL", booking.ID, "confirmed").
		Updates(map[string]interface{}{"status": "cancelled", "cancelled_at": &now})
	if res.Error != nil {
		m := tu.Message(chatID, "❌ Не удалось отменить запись из-за ошибки базы данных. Попробуйте позже.")
		s.send(m)
		return
	}
	if res.RowsAffected != 1 {
		m := tu.Message(chatID, "ℹ️ Эта запись уже отменена или изменена. Обновите список записей командой /book_call.")
		s.send(m)
		return
	}

	text := fmt.Sprintf(`✅ <b>Запись на созвон успешно отменена</b>

Слот <b>«%s»</b> освобождён. Вы можете выбрать другое удобное время по команде <code>/book_call</code>.`, booking.SlotTime)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("📅 Выбрать другое время").WithCallbackData("action_book_call")),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
	)
	m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	s.send(m)

	if s.cfg.AdminTelegramID > 0 {
		alert := fmt.Sprintf("ℹ️ <b>Запись на созвон отменена (#%d)</b>\nСтудент: @%s (ID: %d)\nОсвобождён слот: <b>%s</b>",
			booking.ID, from.Username, from.ID, booking.SlotTime)
		adminMsg := tu.Message(tu.ID(s.cfg.AdminTelegramID), alert).WithParseMode(telego.ModeHTML)
		s.send(adminMsg)
	}
}

// SendInterviewTrainer presents random mock interview question with STAR guidance.
func (s *BotService) SendInterviewTrainer(chatID telego.ChatID) {
	if s.bot == nil {
		return
	}

	idx := rand.Intn(len(interviewQuestions))
	item := interviewQuestions[idx]

	text := fmt.Sprintf(`🎙 <b>Тренажёр интервью SPARK (Раунд 2)</b>

❓ <b>Вопрос комиссии:</b>
<i>«%s»</i>

💡 <b>Контекст вопроса:</b>
%s

⚡️ <b>Как отвечать по технике STAR:</b>
%s

<i>Вы можете надиктовать голосовое сообщение или отправить текст прямо в ответ на это сообщение!</i>`, item.Question, item.Context, item.Tip)

	kb := tu.InlineKeyboard(
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("🔄 Другой вопрос").WithCallbackData("action_interview")),
		// [Временно приостановлено: Сценарий 3 (Запись на Zoom-созвон 1-на-1)]
		// tu.InlineKeyboardRow(tu.InlineKeyboardButton("📅 Записаться на Zoom-симуляцию").WithCallbackData("action_book_call")),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")),
	)

	m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(kb)
	s.send(m)
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
