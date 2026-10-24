package bot

import (
	"fmt"
	"log"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// HandleCalculatorCallback routes calculator interactive steps.
func (s *BotService) HandleCalculatorCallback(query *telego.CallbackQuery) error {
	if s.bot == nil || query == nil || query.Message == nil {
		return nil
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
		pin := TgEmoji(EmojiPinRed, "📍")

		text := fmt.Sprintf(`%s <b>Диагностика готовности к SPARK 2027</b>

<b>Шаг 1 из 3: Возраст и гражданство</b>

Для участия в программе действуют базовые критерии:
• Возраст: 18–21 год на момент подачи
• Гражданство: РФ с постоянным проживанием в России

Подходишь под эти критерии?`, pin)

		photoFile := s.ResolvePhotoFile(MediaGrantChances)
		photoMsg := tu.Photo(chatID, photoFile).
			WithCaption(text).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(CalcQ1Keyboard())

		sentMsg, err := s.bot.SendPhoto(photoMsg)
		if err == nil && sentMsg != nil {
			s.CacheSentPhoto(MediaGrantChances, sentMsg)
		} else {
			log.Printf("[Bot Error] calc_start SendPhoto failed: %v. Retrying text message...", err)
			m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ1Keyboard())
			if _, msgErr := s.bot.SendMessage(m); msgErr != nil {
				log.Printf("[Bot Error] calc_start SendMessage failed: %v. Retrying stripped...", msgErr)
				cleanText := StripTgEmoji(text)
				cleanMsg := tu.Message(chatID, cleanText).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ1Keyboard())
				_, _ = s.bot.SendMessage(cleanMsg)
			}
		}
		return nil

	case "calc_q1_yes":
		session.CalcAgeCitizenship = true
		session.State = StateCalcQ2
		books := TgEmoji(EmojiBooks, "📚")
		text := fmt.Sprintf(`%s <b>Шаг 2 из 3: Курс обучения в университете</b>

По правилам визы J-1 после стажировки участник обязан вернуться минимум на 1 академический год в свой вуз в РФ.

На каком курсе ты сейчас учишься?`, books)
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ2Keyboard())
		_, _ = s.bot.SendMessage(m)
		return nil

	case "calc_q1_no":
		session.State = StateDefault
		warning := TgEmoji(EmojiWarning, "⚠️")
		text := fmt.Sprintf(`%s <b>Возрастное или территориальное ограничение</b>

Программа финансируется Госдепартаментом США со строгими критериями: возраст 18–21 год на момент отбора и гражданство РФ.

Если тебе скоро исполнится 18 или есть особые обстоятельства — напиши так называемому Илю (@ilyan_vas) для личного разбора.`, warning)
		btnChat := tu.InlineKeyboardButton("💬 Написать так называемому Илю").WithURL("https://t.me/ilyan_vas")
		btnMenu := tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu")
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(tu.InlineKeyboard(tu.InlineKeyboardRow(btnChat), tu.InlineKeyboardRow(btnMenu)))
		_, _ = s.bot.SendMessage(m)
		return nil

	case "calc_q2_12", "calc_q2_3":
		session.CalcStudyYear = "undergraduate"
		session.State = StateCalcQ3
		speech := TgEmoji(EmojiSpeechGreen, "🗣")
		text := fmt.Sprintf(`%s <b>Шаг 3 из 3: Уровень английского языка</b>

Оцени свой текущий разговорный и письменный английский:
(Международные сертификаты IELTS/TOEFL не требуются — отбор оценивает реальную речь)`, speech)
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ3Keyboard())
		_, _ = s.bot.SendMessage(m)
		return nil

	case "calc_q2_senior":
		session.CalcStudyYear = "senior"
		session.State = StateCalcQ3
		warning := TgEmoji(EmojiWarning, "⚠️")
		speech := TgEmoji(EmojiSpeechGreen, "🗣")
		text := fmt.Sprintf(`%s <b>Нюанс для выпускного курса</b>

Студенты выпускного курса могут участвовать в гранте, если планируют поступление в магистратуру в РФ (это подтверждает обязательство возвращения по J-1).

%s <b>Оцени свой текущий уровень английского языка:</b>`, warning, speech)
		m := tu.Message(chatID, text).WithParseMode(telego.ModeHTML).WithReplyMarkup(CalcQ3Keyboard())
		_, _ = s.bot.SendMessage(m)
		return nil

	case "calc_q3_b1", "calc_q3_b2", "calc_q3_c1":
		session.State = StateDefault
		score := 88
		scoreZone := "Достаточный уровень для интервью"
		langLabel := "B2 (Upper-Intermediate)"

		if data == "calc_q3_c1" {
			score = 96
			scoreZone = "Высокая готовность по языку"
			langLabel = "C1 (Advanced)"
		} else if data == "calc_q3_b1" {
			score = 75
			scoreZone = "Стартовый уровень, есть запас по языку"
			langLabel = "B1 (Intermediate)"
		}

		session.CalcEnglishLevel = data
		s.RegisterOrUpdateDripLead(chatID.ID, query.From.Username, query.From.FirstName, score)

		star := TgEmoji(EmojiSparkleGreen, "✨")
		sun := TgEmoji(EmojiSunGreen, "☀️")
		spark := TgEmoji(EmojiStarGreen, "⭐")
		chatBubbles := TgEmoji(EmojiChatBubbles, "💬")
		fire := TgEmoji(EmojiFire, "🔥")

		pricing := s.getPricing()
		accPriceStr := formatPriceRub(int(pricing.AcceleratorPrice))
		accPromoStr := formatPriceRub(int(pricing.AcceleratorPrice * 0.95))

		reply := fmt.Sprintf(`%s <b>Диагностика готовности к SPARK:</b>

Оценка готовности: <b>%d/100</b> %s
Уровень английского: <b>%s</b>
Статус: <b>%s</b>

%s <b>Это оценка готовности к подготовке, а не вероятность гранта.</b> Решение принимает комиссия программы.

%s <b>Дальше решают документы:</b>
<b>резюме американского стандарта</b>, <b>эссе на 500 слов</b> и <b>20-минутное Zoom-интервью</b>.

В проекте «так называемый SPARK» разбираем заявку под ключ от %s ₽ (от %s ₽ по промокоду).

%s <i>Оплата только по СБП.</i>`, star, score, spark, langLabel, scoreZone, sun, chatBubbles, accPriceStr, accPromoStr, fire)

		calcKb := CalcResultKeyboard(pricing)

		photoFile := s.ResolvePhotoFile(MediaCalcResult)
		photoMsg := tu.Photo(chatID, photoFile).
			WithCaption(reply).
			WithParseMode(telego.ModeHTML).
			WithReplyMarkup(calcKb)

		sentMsg, err := s.bot.SendPhoto(photoMsg)
		if err == nil && sentMsg != nil {
			s.CacheSentPhoto(MediaCalcResult, sentMsg)
		} else if err != nil {
			log.Printf("[Bot Error] CalcResult SendPhoto failed: %v. Sending fallback...", err)
			cleanReply := StripTgEmoji(reply)
			m := tu.Message(chatID, cleanReply).WithParseMode(telego.ModeHTML).WithReplyMarkup(calcKb)
			if _, textErr := s.bot.SendMessage(m); textErr != nil {
				plainMsg := tu.Message(chatID, cleanReply).WithReplyMarkup(calcKb)
				_, _ = s.bot.SendMessage(plainMsg)
			}
		}
		return nil
	}

	return nil
}
