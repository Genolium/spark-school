package models

import (
	"time"
)

// User represents a registered student or administrator.
type User struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TelegramID      int64      `gorm:"unique;not null" json:"telegram_id"`
	Username        string     `gorm:"size:64" json:"username"`
	FirstName       string     `gorm:"size:128;not null" json:"first_name"`
	LastName        string     `gorm:"size:128" json:"last_name"`
	PhotoURL        string     `json:"photo_url"`
	Role            string     `gorm:"size:20;default:'student'" json:"role"` // 'student', 'admin'
	HasAccess       bool       `gorm:"default:false" json:"has_access"`
	AccessGrantedAt *time.Time `json:"access_granted_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	AffiliateProfile *AffiliateProfile `gorm:"foreignKey:UserID" json:"affiliate_profile,omitempty"`
}

// Referral establishes the multi-tier relationship between students.
type Referral struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	ReferredByID     *uint     `json:"referred_by_id"`     // Level 1 referrer
	ParentReferrerID *uint     `json:"parent_referrer_id"` // Level 2 referrer
	CreatedAt        time.Time `json:"created_at"`
}

// AffiliateProfile maintains balance and partner tracking code.
type AffiliateProfile struct {
	UserID          uint      `gorm:"primaryKey" json:"user_id"`
	ReferralCode    string    `gorm:"uniqueIndex;size:32;not null" json:"referral_code"`
	CurrentBalance  float64   `gorm:"type:numeric(10,2);default:0.00" json:"current_balance"`
	TotalEarned     float64   `gorm:"type:numeric(10,2);default:0.00" json:"total_earned"`
	TotalWithdrawn  float64   `gorm:"type:numeric(10,2);default:0.00" json:"total_withdrawn"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ReferralTransaction logs commission payouts for transparency.
type ReferralTransaction struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	AffiliateID uint      `gorm:"not null;index" json:"affiliate_id"`
	BuyerID     uint      `gorm:"not null" json:"buyer_id"`
	Level       int       `gorm:"not null" json:"level"` // 1 or 2
	Amount      float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	CreatedAt   time.Time `json:"created_at"`
}

// PayoutRequest represents an affiliate withdrawal request.
type PayoutRequest struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Amount         float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	PaymentDetails string     `gorm:"type:text;not null" json:"payment_details"`
	Status         string     `gorm:"size:20;default:'pending'" json:"status"` // 'pending', 'paid', 'rejected'
	CreatedAt      time.Time  `json:"created_at"`
	ResolvedAt     *time.Time `json:"resolved_at"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// PromoCode represents a 5% discount promo code tied to an affiliate/partner.
type PromoCode struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Code            string    `gorm:"uniqueIndex;size:32;not null" json:"code"`
	DiscountPercent int       `gorm:"default:5" json:"discount_percent"`
	OwnerTelegramID int64     `json:"owner_telegram_id"`
	OwnerUsername   string    `gorm:"size:64" json:"owner_username"`
	UsesCount       int       `gorm:"default:0" json:"uses_count"`
	RewardAmount    float64   `gorm:"type:numeric(10,2);default:1000.00" json:"reward_amount"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// PaymentReceipt stores submitted payment receipts with hash deduplication to prevent fraud/reuse.
type PaymentReceipt struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	FileHash     string    `gorm:"uniqueIndex;size:64;not null" json:"file_hash"`
	FileUniqueID string    `gorm:"index;size:64" json:"file_unique_id"`
	TelegramID   int64     `gorm:"index" json:"telegram_id"`
	Username     string    `gorm:"size:64" json:"username"`
	FileID       string    `gorm:"size:255" json:"file_id"`
	Tier         string    `gorm:"size:32;default:'accelerator'" json:"tier"` // 'basic', 'accelerator', 'vip'
	Amount       float64   `gorm:"type:numeric(10,2)" json:"amount"`
	Status       string    `gorm:"size:20;default:'pending'" json:"status"` // 'pending', 'approved', 'rejected'
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}


