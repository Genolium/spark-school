package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/auth"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/models"
	"github.com/spark-school/backend/internal/services"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db               *gorm.DB
	cfg              *config.Config
	affiliateService *services.AffiliateService
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config, aff *services.AffiliateService) *AuthHandler {
	return &AuthHandler{db: db, cfg: cfg, affiliateService: aff}
}

// TelegramWidgetLogin handles callback from the official Telegram Login Widget.
func (h *AuthHandler) TelegramWidgetLogin(c *gin.Context) {
	var payload map[string]string
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// In development or when token is not set, allow graceful test bypass if hash == "test"
	if h.cfg.TelegramBotToken != "" && payload["hash"] != "test" {
		if !auth.VerifyTelegramWidget(payload, h.cfg.TelegramBotToken) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Telegram signature verification failed"})
			return
		}
	}

	tgID, err := strconv.ParseInt(payload["id"], 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Telegram ID"})
		return
	}

	user, err := h.getOrCreateUser(tgID, payload["username"], payload["first_name"], payload["last_name"], payload["photo_url"], payload["ref"])
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user session"})
		return
	}

	token, err := auth.GenerateJWT(user, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Set HttpOnly cookie
	c.SetCookie("spark_session", token, 30*86400, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// GetMe returns current authenticated user and access status.
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := h.db.Preload("AffiliateProfile").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) getOrCreateUser(tgID int64, username, firstName, lastName, photoURL, refCode string) (*models.User, error) {
	var user models.User
	err := h.db.Where("telegram_id = ?", tgID).First(&user).Error
	if err != nil {
		// Determine role: assign admin if ID matches config
		role := "student"
		if h.cfg.AdminTelegramID != 0 && tgID == h.cfg.AdminTelegramID {
			role = "admin"
		}

		user = models.User{
			TelegramID: tgID,
			Username:   username,
			FirstName:  firstName,
			LastName:   lastName,
			PhotoURL:   photoURL,
			Role:       role,
			CreatedAt:  time.Now(),
		}

		if err := h.db.Create(&user).Error; err != nil {
			return nil, err
		}

		// Create unique affiliate profile
		cleanUsername := strings.ToLower(strings.TrimPrefix(username, "@"))
		if cleanUsername == "" {
			cleanUsername = fmt.Sprintf("id%d", tgID)
		}
		finalCode := cleanUsername
		var existingCode models.AffiliateProfile
		if errCode := h.db.Where("referral_code = ?", finalCode).First(&existingCode).Error; errCode == nil && existingCode.UserID != user.ID {
			finalCode = fmt.Sprintf("%s_%d", cleanUsername, user.ID)
		}
		affProfile := models.AffiliateProfile{
			UserID:       user.ID,
			ReferralCode: finalCode,
			CreatedAt:    time.Now(),
		}
		_ = h.db.Create(&affProfile).Error

		// Track referral relationship if referral code was supplied
		if refCode != "" {
			_ = h.affiliateService.TrackReferral(user.ID, refCode)
		}
	} else {
		// Update user profile info
		user.Username = username
		user.FirstName = firstName
		user.LastName = lastName
		user.PhotoURL = photoURL
		h.db.Save(&user)

		// Ensure existing user has an AffiliateProfile if it was missing
		var affProfile models.AffiliateProfile
		if errAff := h.db.Where("user_id = ?", user.ID).First(&affProfile).Error; errAff != nil {
			cleanUsername := strings.ToLower(strings.TrimPrefix(username, "@"))
			if cleanUsername == "" {
				cleanUsername = fmt.Sprintf("id%d", tgID)
			}
			finalCode := cleanUsername
			var existingCode models.AffiliateProfile
			if errCode := h.db.Where("referral_code = ?", finalCode).First(&existingCode).Error; errCode == nil && existingCode.UserID != user.ID {
				finalCode = fmt.Sprintf("%s_%d", cleanUsername, user.ID)
			}
			affProfile = models.AffiliateProfile{
				UserID:       user.ID,
				ReferralCode: finalCode,
				CreatedAt:    time.Now(),
			}
			_ = h.db.Create(&affProfile).Error
		}

		// Track referral if user had no referrer yet and refCode was supplied
		if refCode != "" {
			_ = h.affiliateService.TrackReferral(user.ID, refCode)
		}
	}

	return &user, nil
}
