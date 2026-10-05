package bot

import (
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// HandleCalculatorCallback routes calculator interactive steps.
func (s *BotService) HandleCalculatorCallback(query *telego.CallbackQuery) bool {
	if s.bot == nil || query == nil || query.Message == nil {
		return false
	}

	data := query.Data
	chatID := tu.ID(query.Message.GetChat().ID)
	session := s.getSession(query.Message.GetChat().ID)

	_ = s.bot.AnswerCallbackQuery(&telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	switch data {
	case "calc_start":
		session.State = StateCalcQ1
		text := `🎯 <b>Расчёт шансов на грант $20,000 (SPARK 2027)</b>

<b>Шаг 1 из 3: Возраст и гражданство</b>

Для участия в программе действуют базовые критерии:
• Возраст: 18–21 год на момент подачи
• Гражданство: РФ с постоянным проживанием в России

Подходишь под эти критерии?`
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ1Keyboard())
		_, _ = s.bot.SendMessage(m)
		return true

	case "calc_q1_yes":
		session.CalcAgeCitizenship = true
		session.State = StateCalcQ2
		text := `📚 <b>Шаг 2 из 3: Курс обучения в университете</b>

По правилам визы J-1 после стажировки участник обязан вернуться минимум на 1 академический год в свой вуз в РФ.

На каком курсе ты сейчас учишься?`
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ2Keyboard())
		_, _ = s.bot.SendMessage(m)
		return true

	case "calc_q1_no":
		session.State = StateDefault
		text := `⚠️ <b>Возрастное или территориальное ограничение</b>

Программа финансируется Госдепартаментом США со строгими критериями: возраст 18–21 год на момент отбора и гражданство РФ.

Если тебе скоро исполнится 18 или есть особые обстоятельства — напиши так называемому Илю (@ilyan_vas) для личного разбора.`
		btnChat := tu.InlineKeyboardButton("💬 Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
		btnMenu := tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnChat), tu.InlineKeyboardRow(btnMenu)))
		_, _ = s.bot.SendMessage(m)
		return true

	case "calc_q2_12", "calc_q2_3":
		session.CalcStudyYear = "undergraduate"
		session.State = StateCalcQ3
		text := `🗣 <b>Шаг 3 из 3: Уровень английского языка</b>

Оцени свой текущий разговорный и письменный английский:
(Международные сертификаты IELTS/TOEFL не требуются — отбор оценивает реальную речь)`
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ3Keyboard())
		_, _ = s.bot.SendMessage(m)
		return true

	case "calc_q2_senior":
		session.CalcStudyYear = "senior"
		session.State = StateCalcQ3
		text := `⚠️ <b>Нюанс для выпускного курса</b>

Студенты выпускного курса могут участвовать в гранте, если планируют поступление в магистратуру в РФ (это подтверждает обязательство возвращения по J-1).

<b>Оцени свой текущий уровень английского языка:</b>`
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ3Keyboard())
		_, _ = s.bot.SendMessage(m)
		return true

	case "calc_q3_b1", "calc_q3_b2", "calc_q3_c1":
		session.State = StateDefault
		score := 88
		scoreZone := "Отличные шансы (High Potential)"
		langLabel := "B2 (Upper-Intermediate)"

		if data == "calc_q3_c1" {
			score = 96
			scoreZone = "Максимальные шансы (Elite Zone)"
			langLabel = "C1 (Advanced)"
		} else if data == "calc_q3_b1" {
			score = 75
			scoreZone = "Хороший потенциал (Needs Practice)"
			langLabel = "B1 (Intermediate)"
		}

		session.CalcEnglishLevel = data

		reply := fmt.Sprintf(`📊 <b>Твой расчет шансов на грант $20,000:</b>

Оценка профиля: <b>%d/100</b> 🔥
Уровень английского: <b>%s</b>
Статус: <b>%s</b>

✅ <b>Ты полностью проходишь базовые фильтры проекта «так называемый SPARK»!</b>

💡 <b>В чём главный нюанс?</b>
Формальные критерии — это лишь 10%% отбора. Главная битва начинается в <b>Google XYZ резюме</b>, <b>эссе на 500 слов</b> и <b>20-минутном Zoom-интервью</b> с американской комиссией.

В проекте «так называемый SPARK» мы упакуем твою заявку до идеала под ключ от 6 900 ₽ (от 6 555 ₽ по промокоду).

🔥 <i>Места на поток строго ограничены.</i>`, score, langLabel, scoreZone)

		m := tu.Message(chatID, reply).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcResultKeyboard())
		_, _ = s.bot.SendMessage(m)
		return true
	}

	return false
}
