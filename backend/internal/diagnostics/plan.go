// Package diagnostics builds a personalized preparation plan from the landing
// diagnostic answers. It is deterministic and contains no promises of a grant.
package diagnostics

import (
	"fmt"
	"strings"
)

type Step struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type Plan struct {
	Score           int    `json:"score"`
	Summary         string `json:"summary"`
	Warning         string `json:"warning,omitempty"`
	RecommendedTier string `json:"recommended_tier,omitempty"`
	Steps           []Step `json:"steps"`
}

func ScoreFor(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "c1":
		return 96
	case "b2":
		return 88
	default:
		return 75
	}
}

func Generate(ageCitizenship bool, universityYear, englishLevel string) Plan {
	plan := Plan{Score: ScoreFor(englishLevel)}
	year := strings.ToLower(strings.TrimSpace(universityYear))
	level := strings.ToLower(strings.TrimSpace(englishLevel))

	if !ageCitizenship {
		plan.Summary = "Формальные критерии программы не совпадают с твоим профилем. Но это не значит, что нет других возможностей."
		plan.Warning = "Проверь актуальные требования программы и обсуди альтернативы с Илем."
		plan.Steps = []Step{
			{Title: "Сверить критерии", Description: "Возраст, гражданство и сроки — первое, что проверяет комиссия."},
			{Title: "Найти альтернативу", Description: "Есть другие обменные и грантовые программы с другими требованиями."},
			{Title: "Разобрать профиль", Description: "Напиши Илю — он подскажет, куда двигаться дальше."},
		}
		return plan
	}

	switch level {
	case "c1":
		plan.Summary = "Сильная языковая база. Главный фокус — не язык, а упаковка истории и доказательство результата."
	case "b2":
		plan.Summary = "Хороший уровень для интервью. Нужно отточить структуру заявки и уверенную речь под стрессом."
	default:
		plan.Summary = "Стартовый уровень достаточен. Комиссия смотрит на идею и потенциал, но над языком стоит работать параллельно."
	}

	plan.Steps = []Step{
		{Title: "Нарратив и позиционирование", Description: "Собрать сквозную линию: кто ты, что уже сделал и зачем тебе программа. Без шаблонных фраз и запрещённых тем TAL."},
		{Title: "Американское резюме на 1 страницу", Description: "Action Verbs, измеримые результаты и структура, которую комиссия понимает за 10 секунд."},
		{Title: "Эссе на 500 слов", Description: "Один конкретный конфликт или поворот, личная рефлексия и связь с целями программы."},
		{Title: "Видеовизитка 80–100 секунд", Description: "Динамичная запись без чтения с экрана: хук, история, вывод."},
		{Title: "Интервью 20 минут", Description: "Отработать ответы на behavioural questions и не потерять нить под давлением."},
	}

	switch {
	case level == "b1":
		plan.Steps = append([]Step{{Title: "Английский 20 минут в день", Description: "Короткие ежедневные практики на речь и аудирование — важнее редких марафонов."}}, plan.Steps...)
		plan.RecommendedTier = "basic"
	case level == "c1":
		plan.RecommendedTier = "accelerator"
	default:
		plan.RecommendedTier = "accelerator"
	}

	if year == "senior" {
		plan.Warning = "Выпускной курс: проверь правило J-1 и планы на магистратуру в РФ. Это может влиять на участие."
	}

	return plan
}

func TelegramText(plan Plan) string {
	var b strings.Builder
	b.WriteString("<b>Твой персональный план подготовки</b>\n\n")
	b.WriteString(plan.Summary)
	b.WriteString("\n")
	if plan.Warning != "" {
		b.WriteString("\n⚠️ ")
		b.WriteString(plan.Warning)
		b.WriteString("\n")
	}
	b.WriteString("\n<b>Что делать по шагам:</b>\n")
	for i, step := range plan.Steps {
		b.WriteString(fmt.Sprintf("%d. <b>%s</b> — %s\n", i+1, step.Title, step.Description))
	}
	b.WriteString("\n<i>Это план подготовки, а не гарантия гранта.</i>")
	return b.String()
}
