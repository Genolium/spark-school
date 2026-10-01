package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

type ReceiptHandler struct {
	db                 *gorm.DB
	checkDuplicateFunc func(fileHash, fileUniqueID string) (*models.PaymentReceipt, bool)
	saveReceiptFunc    func(receipt *models.PaymentReceipt) error
}

func NewReceiptHandler(db *gorm.DB) *ReceiptHandler {
	return &ReceiptHandler{db: db}
}

// WithCustomStore sets custom duplicate checker and receipt saver (useful for testing without live db).
func (h *ReceiptHandler) WithCustomStore(checkDuplicate func(fileHash, fileUniqueID string) (*models.PaymentReceipt, bool), saveReceipt func(receipt *models.PaymentReceipt) error) *ReceiptHandler {
	h.checkDuplicateFunc = checkDuplicate
	h.saveReceiptFunc = saveReceipt
	return h
}

type SubmitReceiptRequest struct {
	FileHash     string  `json:"file_hash"`
	FileUniqueID string  `json:"file_unique_id"`
	TelegramID   int64   `json:"telegram_id"`
	Username     string  `json:"username"`
	FileID       string  `json:"file_id"`
	Tier         string  `json:"tier"`
	Amount       float64 `json:"amount"`
}

// ComputeHash computes a SHA256 hex string from raw content or identifier.
func ComputeHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// SubmitReceipt is an alias for VerifyOrSubmit to handle receipt submission.
func (h *ReceiptHandler) SubmitReceipt(c *gin.Context) {
	h.VerifyOrSubmit(c)
}

// VerifyOrSubmit checks if a receipt has already been submitted (anti-fraud deduplication)
// and records it if new.
func (h *ReceiptHandler) VerifyOrSubmit(c *gin.Context) {
	var req SubmitReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	cleanHash := strings.TrimSpace(req.FileHash)
	cleanUniqueID := strings.TrimSpace(req.FileUniqueID)
	cleanFileID := strings.TrimSpace(req.FileID)

	if cleanHash == "" && cleanUniqueID != "" {
		cleanHash = ComputeHash([]byte(cleanUniqueID))
	} else if cleanHash == "" && cleanFileID != "" {
		cleanHash = ComputeHash([]byte(cleanFileID))
	}

	if cleanHash == "" && cleanUniqueID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Необходимо передать file_hash или file_id чека"})
		return
	}

	// 1. Custom store check (e.g. for unit testing without live db)
	if h.checkDuplicateFunc != nil {
		if existing, isDup := h.checkDuplicateFunc(cleanHash, cleanUniqueID); isDup {
			c.JSON(http.StatusConflict, gin.H{
				"duplicate":  true,
				"error":      "Этот чек уже был отправлен ранее и зарегистрирован в системе",
				"receipt_id": existing.ID,
				"status":     existing.Status,
			})
			return
		}
	}

	// 2. Database check
	if h.db != nil {
		var existing models.PaymentReceipt
		if err := h.db.Where("file_hash = ? OR (file_unique_id != '' AND file_unique_id = ?)", cleanHash, cleanUniqueID).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{
				"duplicate":  true,
				"error":      "Этот чек уже был отправлен ранее и зарегистрирован в системе",
				"receipt_id": existing.ID,
				"status":     existing.Status,
			})
			return
		}

		tier := strings.TrimSpace(req.Tier)
		if tier == "" {
			tier = "accelerator"
		}

		receipt := models.PaymentReceipt{
			FileHash:     cleanHash,
			FileUniqueID: cleanUniqueID,
			TelegramID:   req.TelegramID,
			Username:     req.Username,
			FileID:       cleanFileID,
			Tier:         tier,
			Amount:       req.Amount,
			Status:       "pending",
			CreatedAt:    time.Now(),
		}

		if err := h.db.Create(&receipt).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения чека"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"duplicate":  false,
			"status":     "recorded",
			"receipt_id": receipt.ID,
		})
		return
	}

	// 3. Custom store save (if set)
	if h.saveReceiptFunc != nil {
		tier := strings.TrimSpace(req.Tier)
		if tier == "" {
			tier = "accelerator"
		}
		receipt := models.PaymentReceipt{
			FileHash:     cleanHash,
			FileUniqueID: cleanUniqueID,
			TelegramID:   req.TelegramID,
			Username:     req.Username,
			FileID:       cleanFileID,
			Tier:         tier,
			Amount:       req.Amount,
			Status:       "pending",
			CreatedAt:    time.Now(),
		}
		if err := h.saveReceiptFunc(&receipt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения чека"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"duplicate":  false,
			"status":     "recorded",
			"receipt_id": receipt.ID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"duplicate": false,
		"status":    "verified",
	})
}
