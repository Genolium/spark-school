package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/spark-school/backend/internal/diagnostics"
	"github.com/spark-school/backend/internal/models"
)

// handleDiagnosticPlan delivers the personalized plan created on the landing.
func (s *BotService) handleDiagnosticPlan(chatID telego.ChatID, from *telego.User, token string) error {
	if from == nil || s.db == nil {
		return fmt.Errorf("diagnostic plan is unavailable")
	}
	var record models.Diagnostic
	if err := s.db.Where("token = ?", token).First(&record).Error; err != nil {
		return s.send(tu.Message(chatID, "❌ План не найден или ссылка устарела. Пройди диагностику на сайте ещё раз."))
	}

	if record.TelegramID == 0 {
		now := time.Now()
		_ = s.db.Model(&models.Diagnostic{}).Where("id = ? AND telegram_id = 0", record.ID).
			Updates(map[string]interface{}{"telegram_id": from.ID, "linked_at": &now}).Error
		if record.AttributionToken != "" {
			_ = s.attributeLead(from, record.AttributionToken)
		}
	}

	plan := diagnostics.Generate(record.AgeCitizenship, record.UniversityYear, record.EnglishLevel)
	rows := [][]telego.InlineKeyboardButton{}
	if record.AgeCitizenship {
		rows = append(rows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("💳 Выбрать тариф").WithCallbackData("action_payment"),
		})
	} else {
		rows = append(rows, []telego.InlineKeyboardButton{
			tu.InlineKeyboardButton("💬 Обсудить альтернативы с Илем").WithURL("https://t.me/ilyan_vas"),
		})
	}
	rows = append(rows, []telego.InlineKeyboardButton{
		tu.InlineKeyboardButton("⬅️ В главное меню").WithCallbackData("action_menu"),
	})

	return s.send(tu.Message(chatID, diagnostics.TelegramText(plan)).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(tu.InlineKeyboard(rows...)))
}

func diagnosticTokenFromStart(param string) string {
	param = strings.TrimSpace(param)
	if !strings.HasPrefix(param, "plan_") {
		return ""
	}
	token := strings.TrimPrefix(param, "plan_")
	if len(token) > 32 {
		token = token[:32]
	}
	for _, r := range token {
		if !((r >= 'a' && r <= 'f') || (r >= '0' && r <= '9')) {
			return ""
		}
	}
	return token
}
