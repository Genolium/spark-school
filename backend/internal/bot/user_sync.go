package bot

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	"github.com/spark-school/backend/internal/models"
)

func (s *BotService) syncTelegramUser(from *telego.User) *models.User {
	if s.db == nil || from == nil || from.ID <= 0 {
		return nil
	}

	tgID := from.ID
	cleanUsername := strings.TrimPrefix(strings.TrimSpace(from.Username), "@")
	firstName := strings.TrimSpace(from.FirstName)
	if firstName == "" {
		firstName = "Студент"
	}
	lastName := strings.TrimSpace(from.LastName)

	var user models.User
	err := s.db.Where("telegram_id = ?", tgID).First(&user).Error
	if err != nil {
		role := "student"
		hasAccess := false
		if s.cfg != nil && s.cfg.AdminTelegramID != 0 && tgID == s.cfg.AdminTelegramID {
			role = "admin"
			hasAccess = true
		}

		user = models.User{
			TelegramID: tgID,
			Username:   cleanUsername,
			FirstName:  firstName,
			LastName:   lastName,
			Role:       role,
			HasAccess:  hasAccess,
			Tier:       "accelerator",
			PaidAmount: 0,
			CreatedAt:  time.Now(),
		}
		if err := s.db.Create(&user).Error; err != nil {
			log.Printf("[Bot User Sync] Failed creating user %d: %v", tgID, err)
			return nil
		}

		refCode := strings.ToLower(cleanUsername)
		if refCode == "" {
			refCode = fmt.Sprintf("id%d", tgID)
		}
		affProfile := models.AffiliateProfile{
			UserID:       user.ID,
			ReferralCode: refCode,
			CreatedAt:    time.Now(),
		}
		_ = s.db.Create(&affProfile).Error

		log.Printf("[Bot User Sync] New lead registered in DB: @%s (ID: %d)", cleanUsername, tgID)
		return &user
	}

	// Never resurrect personal data after an erasure request.
	if user.AnonymizedAt != nil {
		return &user
	}

	// Keep names up to date
	changed := false
	if cleanUsername != "" && user.Username != cleanUsername {
		user.Username = cleanUsername
		changed = true
	}
	if firstName != "" && user.FirstName != firstName {
		user.FirstName = firstName
		changed = true
	}
	if lastName != "" && user.LastName != lastName {
		user.LastName = lastName
		changed = true
	}
	if changed {
		s.db.Save(&user)
	}

	return &user
}

func (s *BotService) associateBotReferral(from *telego.User, inputUsername, refCode string) {
	if s.db == nil || from == nil {
		return
	}

	user := s.syncTelegramUser(from)
	if user == nil {
		return
	}

	if refCode != "" {
		_ = s.affiliateService.TrackReferral(user.ID, refCode)
	}
}

func (s *BotService) attributeLead(from *telego.User, token string) error {
	if s.db == nil || from == nil || token == "" {
		return nil
	}
	var user models.User
	if err := s.db.Where("telegram_id = ?", from.ID).First(&user).Error; err != nil {
		return err
	}
	if user.AttributionToken != "" {
		return nil
	}
	var landing models.AnalyticsEvent
	err := s.db.Where("attribution_token = ? AND visitor_id <> ''", token).Order("created_at DESC").First(&landing).Error
	visitorID := ""
	if err == nil {
		visitorID = landing.VisitorID
	}
	if err := s.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"attribution_token": token,
		"visitor_id":        visitorID,
	}).Error; err != nil {
		return err
	}
	meta := fmt.Sprintf(`{"telegram_id":%d,"attribution_token":%q}`, from.ID, token)
	return s.db.Create(&models.AnalyticsEvent{
		VisitorID:        visitorID,
		AttributionToken: token,
		Step:             "bot_lead",
		Source:           landing.Source,
		Campaign:         landing.Campaign,
		Tier:             user.Tier,
		Metadata:         meta,
		CreatedAt:        time.Now(),
	}).Error
}

func sanitizeAttributionToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "_"); idx != -1 {
		raw = raw[:idx]
	}
	if len(raw) > 32 {
		raw = raw[:32]
	}
	for _, r := range raw {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return ""
		}
	}
	return raw
}

func extractTelegramLink(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "https://t.me/") || strings.HasPrefix(trimmed, "http://t.me/") {
			return trimmed
		}
	}
	return ""
}

func updateChatID(update *telego.Update) int64 {
	if update == nil {
		return 0
	}
	if update.Message != nil {
		return update.Message.Chat.ID
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		return update.CallbackQuery.Message.GetChat().ID
	}
	return 0
}
