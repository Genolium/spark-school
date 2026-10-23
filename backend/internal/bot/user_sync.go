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
	if user.AttributionToken != "" && user.VisitorID != "" {
		return nil
	}

	// First touch wins: keep the token the visitor arrived with, but retry the
	// landing lookup when the matching event has not been ingested yet.
	attributionToken := user.AttributionToken
	if attributionToken == "" {
		attributionToken = token
	}
	var landing models.AnalyticsEvent
	err := s.db.Where("attribution_token = ? AND visitor_id <> ''", attributionToken).
		Order("created_at DESC").First(&landing).Error
	visitorID := user.VisitorID
	if err == nil && landing.VisitorID != "" {
		visitorID = landing.VisitorID
	}

	updates := map[string]interface{}{"attribution_token": attributionToken}
	if visitorID != "" {
		updates["visitor_id"] = visitorID
	}
	if err := s.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		return err
	}
	if visitorID == "" {
		// No landing event yet. A later /start will finish the attribution.
		return nil
	}

	var existing int64
	if err := s.db.Model(&models.AnalyticsEvent{}).
		Where("step = ? AND visitor_id = ?", "bot_lead", visitorID).
		Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}

	meta := fmt.Sprintf(`{"telegram_id":%d,"attribution_token":%q}`, from.ID, attributionToken)
	return s.db.Create(&models.AnalyticsEvent{
		VisitorID:        visitorID,
		AttributionToken: attributionToken,
		Step:             "bot_lead",
		Source:           landing.Source,
		Campaign:         landing.Campaign,
		Tier:             user.Tier,
		Metadata:         meta,
		CreatedAt:        time.Now(),
	}).Error
}

// splitAttributionToken pulls the short landing token out of a Telegram start
// parameter. Every bot entry point can carry it: pay links, the free audit
// magnet, calculator results and a bare bot link.
func splitAttributionToken(param string) (string, string) {
	param = strings.TrimSpace(param)
	if idx := strings.Index(param, "_a_"); idx != -1 {
		if token := sanitizeAttributionToken(param[idx+len("_a_"):]); token != "" {
			return param[:idx], token
		}
	}
	if strings.HasPrefix(param, "a_") {
		if token := sanitizeAttributionToken(strings.TrimPrefix(param, "a_")); token != "" {
			return "", token
		}
	}
	return param, ""
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
