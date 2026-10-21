package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
)

var validOnboardingStatuses = map[string]struct{}{
	"pending_intro":  {},
	"awaiting_reply": {},
	"completed":      {},
	"skipped":        {},
}

// ListOnboarding returns new students that need a manual introduction.
func (h *AdminHandler) ListOnboarding(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	query := h.db.Model(&models.User{}).Where("role != ?", "admin")
	if status != "" {
		if _, ok := validOnboardingStatuses[status]; !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown onboarding status"})
			return
		}
		query = query.Where("onboarding_status = ?", status)
	} else {
		query = query.Where("onboarding_status IN ?", []string{"pending_intro", "awaiting_reply"})
	}
	var users []models.User
	if err := query.Order("access_granted_at DESC").Limit(200).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load onboarding queue"})
		return
	}
	var pending, awaiting, completed int64
	h.db.Model(&models.User{}).Where("onboarding_status = ?", "pending_intro").Count(&pending)
	h.db.Model(&models.User{}).Where("onboarding_status = ?", "awaiting_reply").Count(&awaiting)
	h.db.Model(&models.User{}).Where("onboarding_status = ?", "completed").Count(&completed)

	c.JSON(http.StatusOK, gin.H{
		"users":  users,
		"counts": gin.H{"pending_intro": pending, "awaiting_reply": awaiting, "completed": completed},
	})
}

// UpdateOnboarding moves a student through the manual introduction flow.
func (h *AdminHandler) UpdateOnboarding(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid onboarding payload"})
		return
	}
	status := strings.TrimSpace(req.Status)
	if _, ok := validOnboardingStatuses[status]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown onboarding status"})
		return
	}
	if h.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database is unavailable"})
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
		return
	}
	now := time.Now()
	updates := map[string]interface{}{
		"onboarding_status": status,
		"onboarding_notes":  strings.TrimSpace(req.Notes),
	}
	step := ""
	switch status {
	case "awaiting_reply":
		updates["intro_sent_at"] = &now
		step = "onboarding_intro_sent"
	case "completed":
		updates["intro_sent_at"] = firstTime(user.IntroSentAt, now)
		updates["onboarding_replied_at"] = &now
		updates["onboarding_completed_at"] = &now
		step = "onboarding_replied"
	}
	if err := h.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update onboarding"})
		return
	}
	if step != "" {
		_ = h.db.Create(&models.AnalyticsEvent{
			VisitorID:        user.VisitorID,
			AttributionToken: user.AttributionToken,
			Step:             step,
			Source:           "onboarding",
			Tier:             user.Tier,
			Metadata:         `{"manual":true}`,
			CreatedAt:        now,
		}).Error
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "status": status})
}

func firstTime(existing *time.Time, fallback time.Time) *time.Time {
	if existing != nil {
		return existing
	}
	return &fallback
}
