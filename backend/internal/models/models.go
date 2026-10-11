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
	Tier            string     `gorm:"size:32;default:'accelerator'" json:"tier"` // 'accelerator', 'vip'
	PaidAmount      float64    `gorm:"type:numeric(10,2);default:0.00" json:"paid_amount"`
	PromoCode       string     `gorm:"size:32" json:"promo_code"`
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

// OutboxEvent provides reliable asynchronous delivery of critical system events (notifications, invite links).
type OutboxEvent struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	EventType   string     `gorm:"size:64;not null" json:"event_type"` // 'telegram_notification', 'admin_alert'
	Payload     string     `gorm:"type:jsonb;not null" json:"payload"`
	Status      string     `gorm:"size:20;default:'pending';index" json:"status"` // 'pending', 'processing', 'completed', 'failed'
	RetryCount  int        `gorm:"default:0" json:"retry_count"`
	MaxRetries  int        `gorm:"default:5" json:"max_retries"`
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// PaymentTransaction logs verified Antilopay/gateway payments with idempotency guarantee.
type PaymentTransaction struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	PaymentID      string    `gorm:"uniqueIndex;size:64;not null" json:"payment_id"`
	OrderID        string    `gorm:"index;size:100" json:"order_id"`
	TelegramID     int64     `gorm:"index" json:"telegram_id"`
	Username       string    `gorm:"size:64" json:"username"`
	Tier           string    `gorm:"size:32;default:'accelerator'" json:"tier"`
	Amount         float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	OriginalAmount float64   `gorm:"type:numeric(10,2)" json:"original_amount"`
	Fee            float64   `gorm:"type:numeric(10,2)" json:"fee"`
	Status         string    `gorm:"size:32;not null" json:"status"` // 'SUCCESS', 'FAIL', 'PENDING'
	Currency       string    `gorm:"size:10;default:'RUB'" json:"currency"`
	PayMethod      string    `gorm:"size:32" json:"pay_method"`
	ProductName    string    `gorm:"size:128" json:"product_name"`
	PromoCode      string    `gorm:"size:32" json:"promo_code"`
	CustomerEmail  string    `gorm:"size:128" json:"customer_email"`
	CustomerPhone  string    `gorm:"size:64" json:"customer_phone"`
	RawCallback    string    `gorm:"type:text" json:"raw_callback"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// DripLead tracks progress of 72-hour automated nurture sequences for prospective students.
type DripLead struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	TelegramID      int64     `gorm:"uniqueIndex;not null" json:"telegram_id"`
	Username        string    `gorm:"size:64" json:"username"`
	FirstName       string    `gorm:"size:128" json:"first_name"`
	Stage           int       `gorm:"default:0" json:"stage"` // 0: new, 1: 2h sent, 2: 24h sent, 3: 48h sent, -1: cancelled/purchased
	CalculatedScore int       `gorm:"default:0" json:"calculated_score"`
	NextTriggerAt   time.Time `gorm:"index;not null" json:"next_trigger_at"`
	IsCompleted     bool      `gorm:"default:false;index" json:"is_completed"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
