package models

import (
	"time"
)

// User represents a registered student or administrator.
type User struct {
	ID                              uint       `gorm:"primaryKey" json:"id"`
	TelegramID                      int64      `gorm:"unique;not null" json:"telegram_id"`
	Username                        string     `gorm:"size:64" json:"username"`
	FirstName                       string     `gorm:"size:128;not null" json:"first_name"`
	LastName                        string     `gorm:"size:128" json:"last_name"`
	PhotoURL                        string     `json:"photo_url"`
	Role                            string     `gorm:"size:20;default:'student'" json:"role"` // 'student', 'admin'
	HasAccess                       bool       `gorm:"default:false" json:"has_access"`
	AccessGrantedAt                 *time.Time `json:"access_granted_at"`
	Tier                            string     `gorm:"size:32;default:'accelerator'" json:"tier"` // 'accelerator', 'vip'
	PaidAmount                      float64    `gorm:"type:numeric(10,2);default:0.00" json:"paid_amount"`
	PaidAmountKopecks               int64      `gorm:"default:0" json:"paid_amount_kopecks"`
	PromoCode                       string     `gorm:"size:32" json:"promo_code"`
	ReceiptStatus                   string     `gorm:"-" json:"receipt_status,omitempty"`
	ReceiptIssued                   bool       `gorm:"default:false;index" json:"receipt_issued"`
	ReceiptIssuedAt                 *time.Time `json:"receipt_issued_at,omitempty"`
	ReceiptURL                      string     `gorm:"size:512" json:"receipt_url,omitempty"`
	ReceiptNumber                   string     `gorm:"size:64" json:"receipt_number,omitempty"`
	UpsellAuditSent                 bool       `gorm:"default:false;index" json:"upsell_audit_sent"`
	UpsellAuditSentAt               *time.Time `json:"upsell_audit_sent_at,omitempty"`
	HasAuditUpsell                  bool       `gorm:"default:false;index" json:"has_audit_upsell"`
	AuditCredits                    int        `gorm:"default:0" json:"audit_credits"`
	AttributionToken                string     `gorm:"size:64;index" json:"attribution_token,omitempty"`
	VisitorID                       string     `gorm:"size:64;index" json:"visitor_id,omitempty"`
	RemindersOptOut                 bool       `gorm:"default:false;index" json:"reminders_opt_out"`
	DiagnosticStartedReminderSent   bool       `gorm:"default:false;index" json:"diagnostic_started_reminder_sent"`
	DiagnosticCompletedReminderSent bool       `gorm:"default:false;index" json:"diagnostic_completed_reminder_sent"`
	AnonymizedAt                    *time.Time `json:"anonymized_at,omitempty"`
	CreatedAt                       time.Time  `json:"created_at"`
	UpdatedAt                       time.Time  `json:"updated_at"`

	AffiliateProfile *AffiliateProfile `gorm:"foreignKey:UserID" json:"affiliate_profile,omitempty"`
}

// Referral establishes the multi-tier relationship between students.
type Referral struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"index;not null" json:"user_id"`
	ReferredByID     *uint     `json:"referred_by_id"`     // Level 1 referrer
	ParentReferrerID *uint     `json:"parent_referrer_id"` // Level 2 referrer
	CreatedAt        time.Time `json:"created_at"`
}

// AffiliateProfile maintains balance and partner tracking code.
type AffiliateProfile struct {
	UserID         uint      `gorm:"primaryKey" json:"user_id"`
	ReferralCode   string    `gorm:"uniqueIndex;size:32;not null" json:"referral_code"`
	CurrentBalance float64   `gorm:"type:numeric(10,2);default:0.00" json:"current_balance"`
	TotalEarned    float64   `gorm:"type:numeric(10,2);default:0.00" json:"total_earned"`
	TotalWithdrawn float64   `gorm:"type:numeric(10,2);default:0.00" json:"total_withdrawn"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ReferralTransaction logs commission payouts for transparency.
type ReferralTransaction struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	AffiliateID   uint      `gorm:"not null;index" json:"affiliate_id"`
	BuyerID       uint      `gorm:"not null" json:"buyer_id"`
	Level         int       `gorm:"not null" json:"level"` // 1 or 2
	PaymentID     string    `gorm:"size:64;index" json:"payment_id,omitempty"`
	Amount        float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	AmountKopecks int64     `gorm:"default:0" json:"amount_kopecks"`
	CreatedAt     time.Time `json:"created_at"`
}

// PayoutRequest represents an affiliate withdrawal request.
type PayoutRequest struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"not null;index" json:"user_id"`
	Amount         float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	AmountKopecks  int64      `gorm:"default:0" json:"amount_kopecks"`
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
	ID            uint       `gorm:"primaryKey" json:"id"`
	FileHash      string     `gorm:"uniqueIndex;size:64;not null" json:"file_hash"`
	FileUniqueID  string     `gorm:"index;size:64" json:"file_unique_id"`
	TelegramID    int64      `gorm:"index" json:"telegram_id"`
	Username      string     `gorm:"size:64" json:"username"`
	FileID        string     `gorm:"size:255" json:"file_id"`
	Tier          string     `gorm:"size:32;default:'accelerator'" json:"tier"` // 'accelerator', 'vip'
	Amount        float64    `gorm:"type:numeric(10,2)" json:"amount"`
	AmountKopecks int64      `gorm:"default:0" json:"amount_kopecks"`
	Status        string     `gorm:"size:20;default:'pending'" json:"status"` // 'pending', 'approved', 'rejected'
	PromoCode     string     `gorm:"size:32" json:"promo_code,omitempty"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
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
	AvailableAt *time.Time `gorm:"index" json:"available_at,omitempty"`
	LockedUntil *time.Time `gorm:"index" json:"locked_until,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}

// PaymentTransaction logs verified Antilopay/gateway payments with idempotency guarantee.
type PaymentTransaction struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	PaymentID        string     `gorm:"uniqueIndex;size:64;not null" json:"payment_id"`
	OrderID          string     `gorm:"index;size:100" json:"order_id"`
	TelegramID       int64      `gorm:"index" json:"telegram_id"`
	Username         string     `gorm:"size:64" json:"username"`
	Tier             string     `gorm:"size:32;default:'accelerator'" json:"tier"`
	Amount           float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	AmountKopecks    int64      `gorm:"default:0" json:"amount_kopecks"`
	OriginalAmount   float64    `gorm:"type:numeric(10,2)" json:"original_amount"`
	Fee              float64    `gorm:"type:numeric(10,2)" json:"fee"`
	Status           string     `gorm:"size:32;not null" json:"status"` // 'SUCCESS', 'FAIL', 'PENDING'
	Currency         string     `gorm:"size:10;default:'RUB'" json:"currency"`
	PayMethod        string     `gorm:"size:32" json:"pay_method"`
	ProductName      string     `gorm:"size:128" json:"product_name"`
	PromoCode        string     `gorm:"size:32" json:"promo_code"`
	CustomerEmail    string     `gorm:"size:128" json:"customer_email"`
	CustomerPhone    string     `gorm:"size:64" json:"customer_phone"`
	RawCallback      string     `gorm:"type:text" json:"raw_callback"`
	Source           string     `gorm:"size:32" json:"source,omitempty"`
	VisitorID        string     `gorm:"size:64;index" json:"visitor_id,omitempty"`
	AttributionToken string     `gorm:"size:64;index" json:"attribution_token,omitempty"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// PaymentOrder is a server-side checkout intent. The expected amount and tier
// are fixed at creation time and verified again when the provider webhook
// arrives, so a client cannot change the price or product after checkout.
type PaymentOrder struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	OrderID           string    `gorm:"uniqueIndex;size:100;not null" json:"order_id"`
	TelegramID        int64     `gorm:"index;not null" json:"telegram_id"`
	Username          string    `gorm:"size:64" json:"username"`
	Tier              string    `gorm:"size:32;not null" json:"tier"`
	Amount            float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	AmountKopecks     int64     `gorm:"default:0" json:"amount_kopecks"`
	Currency          string    `gorm:"size:10;default:'RUB'" json:"currency"`
	PromoCode         string    `gorm:"size:32" json:"promo_code,omitempty"`
	Provider          string    `gorm:"size:32;not null" json:"provider"`
	ProviderPaymentID string    `gorm:"size:100;index" json:"provider_payment_id,omitempty"`
	Status            string    `gorm:"size:20;default:'pending';index" json:"status"` // pending, paid, canceled, failed
	VisitorID         string    `gorm:"size:64;index" json:"visitor_id,omitempty"`
	AttributionToken  string    `gorm:"size:64;index" json:"attribution_token,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// DripLead tracks progress of 72-hour automated nurture sequences for prospective students.
type DripLead struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TelegramID      int64      `gorm:"uniqueIndex;not null" json:"telegram_id"`
	Username        string     `gorm:"size:64" json:"username"`
	FirstName       string     `gorm:"size:128" json:"first_name"`
	Stage           int        `gorm:"default:0" json:"stage"` // 0: new, 1: 2h sent, 2: 24h sent, 3: 48h sent, -1: cancelled/purchased
	CalculatedScore int        `gorm:"default:0" json:"calculated_score"`
	NextTriggerAt   time.Time  `gorm:"index;not null" json:"next_trigger_at"`
	IsCompleted     bool       `gorm:"default:false;index" json:"is_completed"`
	LockedUntil     *time.Time `gorm:"index" json:"locked_until,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// FiscalReceipt stores issued official tax/self-employed receipts sent to students.
type FiscalReceipt struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	TelegramID    int64      `gorm:"index;not null" json:"telegram_id"`
	Amount        float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	Tier          string     `gorm:"size:32" json:"tier"`
	ReceiptURL    string     `gorm:"size:512" json:"receipt_url"`
	ReceiptNumber string     `gorm:"size:64" json:"receipt_number"`
	ServiceName   string     `gorm:"size:255" json:"service_name"`
	SentToBot     bool       `gorm:"default:false" json:"sent_to_bot"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// StudentSubmission tracks essays, resumes, and video pitches submitted for curator review.
type StudentSubmission struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	TelegramID   int64      `gorm:"index;not null" json:"telegram_id"`
	Username     string     `gorm:"size:64" json:"username"`
	WorkType     string     `gorm:"size:32;not null" json:"work_type"` // 'essay', 'resume', 'video', 'positioning'
	Content      string     `gorm:"type:text;not null" json:"content"` // Google Doc link or submitted text
	Version      int        `gorm:"default:1" json:"version"`
	Status       string     `gorm:"size:32;default:'pending';index" json:"status"` // 'pending', 'reviewing', 'reviewed', 'changes_requested'
	CuratorNotes string     `gorm:"type:text" json:"curator_notes,omitempty"`
	SubmittedAt  time.Time  `json:"submitted_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
}

// StudentProgress tracks completion milestones across the 6-step preparation journey.
type StudentProgress struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	TelegramID int64     `gorm:"uniqueIndex;not null" json:"telegram_id"`
	Step1Done  bool      `gorm:"default:false" json:"step1_done"` // 1. Narrative Spine & Positioning
	Step2Done  bool      `gorm:"default:false" json:"step2_done"` // 2. 1-Page US Resume (Action Verbs)
	Step3Done  bool      `gorm:"default:false" json:"step3_done"` // 3. 500-Word Essay
	Step4Done  bool      `gorm:"default:false" json:"step4_done"` // 4. Video Pitch (80-100s)
	Step5Done  bool      `gorm:"default:false" json:"step5_done"` // 5. Academic Transcripts & TAL Check
	Step6Done  bool      `gorm:"default:false" json:"step6_done"` // 6. 20-min Zoom Interview Simulation & DET
	UpdatedAt  time.Time `json:"updated_at"`
}

// CallBooking tracks scheduled Zoom consultation and simulation interviews.
type CallBooking struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"index;not null" json:"user_id"`
	TelegramID     int64      `gorm:"index;not null" json:"telegram_id"`
	Username       string     `gorm:"size:64" json:"username"`
	SlotKey        string     `gorm:"size:64;not null;index" json:"slot_key"`
	SlotTime       string     `gorm:"size:128;not null" json:"slot_time"`
	CallType       string     `gorm:"size:64;default:'zoom_simulation'" json:"call_type"`
	Status         string     `gorm:"size:32;default:'confirmed';index" json:"status"` // 'confirmed', 'completed', 'cancelled'
	ZoomLink       string     `gorm:"size:255" json:"zoom_link"`
	Reminder24Sent bool       `gorm:"column:reminder_24_sent;default:false;index" json:"reminder_24_sent"`
	Reminder1Sent  bool       `gorm:"column:reminder_1_sent;default:false;index" json:"reminder_1_sent"`
	CreatedAt      time.Time  `json:"created_at"`
	CancelledAt    *time.Time `json:"cancelled_at,omitempty"`
}

// ProcessedTelegramUpdate is the persistent claim/lease record for a Telegram
// update. The primary key makes the initial claim atomic across all instances.
type ProcessedTelegramUpdate struct {
	UpdateID    int        `gorm:"primaryKey" json:"update_id"`
	Status      string     `gorm:"size:20;default:'processing';index" json:"status"` // processing, completed, failed
	Attempts    int        `gorm:"default:1" json:"attempts"`
	LockedUntil *time.Time `gorm:"index" json:"locked_until,omitempty"`
	LastError   string     `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// OrderIntent records an unpaid checkout intention so abandoned orders can be
// recovered with a bounded number of reminders and an opt-out.
type OrderIntent struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         uint       `gorm:"index" json:"user_id"`
	TelegramID     int64      `gorm:"index;not null" json:"telegram_id"`
	Tier           string     `gorm:"size:32;not null" json:"tier"`
	Amount         float64    `gorm:"type:numeric(10,2);not null" json:"amount"`
	PromoCode      string     `gorm:"size:32" json:"promo_code,omitempty"`
	Source         string     `gorm:"size:32" json:"source,omitempty"`
	Status         string     `gorm:"size:20;default:'pending';index" json:"status"` // pending, paid, superseded, cancelled
	PaymentID      string     `gorm:"size:64;index" json:"payment_id,omitempty"`
	ReminderCount  int        `gorm:"default:0" json:"reminder_count"`
	LastReminderAt *time.Time `json:"last_reminder_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ExperimentAssignment is a sticky A/B bucket for an anonymous visitor.
type ExperimentAssignment struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Experiment       string    `gorm:"size:64;not null;index" json:"experiment"`
	Variant          string    `gorm:"size:64;not null" json:"variant"`
	VisitorID        string    `gorm:"size:64;index" json:"visitor_id,omitempty"`
	AttributionToken string    `gorm:"size:64;index" json:"attribution_token,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// AnalyticsEvent stores full-funnel conversion events.
type AnalyticsEvent struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	VisitorID        string    `gorm:"size:64;index" json:"visitor_id,omitempty"`
	AttributionToken string    `gorm:"size:64;index" json:"attribution_token,omitempty"`
	Step             string    `gorm:"size:64;index;not null" json:"step"`
	Source           string    `gorm:"size:64;index" json:"source,omitempty"`
	Campaign         string    `gorm:"size:80;index" json:"campaign,omitempty"`
	Tier             string    `gorm:"size:32;index" json:"tier,omitempty"`
	PromoCode        string    `gorm:"size:32;index" json:"promo_code,omitempty"`
	PaymentID        string    `gorm:"size:64;index" json:"payment_id,omitempty"`
	SchemaVersion    int       `gorm:"default:1" json:"schema_version"`
	Metadata         string    `gorm:"type:text" json:"metadata,omitempty"`
	IP               string    `gorm:"size:45" json:"ip,omitempty"`
	UserAgent        string    `gorm:"size:255" json:"user_agent,omitempty"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

// BackupHeartbeat is written by scripts/backup.sh after a verified dump.
type BackupHeartbeat struct {
	ID            int       `gorm:"primaryKey" json:"id"`
	LastSuccessAt time.Time `json:"last_success_at"`
	FileName      string    `gorm:"size:255" json:"file_name,omitempty"`
	SizeBytes     int64     `json:"size_bytes,omitempty"`
}

// TableName matches the table created by migration 00008 and scripts/backup.sh.
func (BackupHeartbeat) TableName() string { return "backup_heartbeat" }

// RevokedToken is a JWT ID that must no longer be accepted.
type RevokedToken struct {
	JTI       string    `gorm:"primaryKey;size:64" json:"jti"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminAuditLog records privileged mutations.
type AdminAuditLog struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	AdminUsername   string    `gorm:"size:64;index" json:"admin_username"`
	AdminTelegramID int64     `json:"admin_telegram_id"`
	Method          string    `gorm:"size:10;not null" json:"method"`
	Path            string    `gorm:"size:255;not null" json:"path"`
	Status          int       `gorm:"not null" json:"status"`
	IP              string    `gorm:"size:45" json:"ip,omitempty"`
	RequestID       string    `gorm:"size:64" json:"request_id,omitempty"`
	CreatedAt       time.Time `gorm:"index" json:"created_at"`
}

// RateLimit is a shared sliding-window counter for multi-instance deployments.
type RateLimit struct {
	Key         string    `gorm:"primaryKey;size:255" json:"key"`
	WindowStart time.Time `json:"window_start"`
	Count       int       `gorm:"not null;default:0" json:"count"`
}

// BotSession persists the Telegram FSM across restarts.
type BotSession struct {
	TelegramID         int64     `gorm:"primaryKey" json:"telegram_id"`
	State              string    `gorm:"size:32" json:"state"`
	SelectedTier       string    `gorm:"size:32" json:"selected_tier"`
	PromoCode          string    `gorm:"size:32" json:"promo_code"`
	DiscountPct        int       `json:"discount_pct"`
	PriceExperiment    string    `gorm:"size:16" json:"price_experiment,omitempty"`
	VIPPriceExperiment string    `gorm:"size:16" json:"vip_price_experiment,omitempty"`
	IsUpgrade          bool      `json:"is_upgrade"`
	UpgradeFromTier    string    `gorm:"size:32" json:"upgrade_from_tier"`
	UpgradeToTier      string    `gorm:"size:32" json:"upgrade_to_tier"`
	UpgradeAmount      float64   `gorm:"type:numeric(10,2)" json:"upgrade_amount"`
	CalcAgeCitizenship bool      `json:"calc_age_citizenship"`
	CalcStudyYear      string    `gorm:"size:32" json:"calc_study_year"`
	CalcEnglishLevel   string    `gorm:"size:32" json:"calc_english_level"`
	LastActivity       time.Time `gorm:"index" json:"last_activity"`
	ExpiresAt          time.Time `gorm:"index" json:"expires_at"`
}

// SchedulerLease ensures only one instance runs a given background job.
type SchedulerLease struct {
	Name      string    `gorm:"primaryKey;size:64" json:"name"`
	Owner     string    `gorm:"size:128" json:"owner"`
	ExpiresAt time.Time `gorm:"index" json:"expires_at"`
}

// Diagnostic stores a completed landing diagnostic and its public token.
type Diagnostic struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	Token            string     `gorm:"uniqueIndex;size:32;not null" json:"token"`
	VisitorID        string     `gorm:"size:64;index" json:"visitor_id,omitempty"`
	AttributionToken string     `gorm:"size:64;index" json:"attribution_token,omitempty"`
	TelegramID       int64      `gorm:"index" json:"telegram_id,omitempty"`
	AgeCitizenship   bool       `json:"age_citizenship"`
	UniversityYear   string     `gorm:"size:32" json:"university_year"`
	EnglishLevel     string     `gorm:"size:16" json:"english_level"`
	Score            int        `json:"score"`
	Source           string     `gorm:"size:64" json:"source,omitempty"`
	LinkedAt         *time.Time `json:"linked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
