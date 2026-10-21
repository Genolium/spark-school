package bot

import (
	"sync"
	"time"

	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BotState represents the current step/state of a user in the bot.
type BotState string

const (
	StateDefault        BotState = "default"
	StateCalcQ1         BotState = "calc_q1"         // Age & Russian citizenship
	StateCalcQ2         BotState = "calc_q2"         // University study year
	StateCalcQ3         BotState = "calc_q3"         // English proficiency level
	StateWaitingPromo   BotState = "waiting_promo"   // User clicked "Enter promo code"
	StateWaitingReceipt BotState = "waiting_receipt" // User clicked "Pay" and is expected to send screenshot
	StateAdminBroadcast BotState = "admin_broadcast" // Curator typed broadcast command and is entering message text
)

// UserSession holds state machine and temporary calculator or payment data.
type UserSession struct {
	State        BotState
	SelectedTier string // "accelerator" or "vip"
	PromoCode    string
	DiscountPct  int
	LastActivity time.Time

	// Upgrade fields
	IsUpgrade       bool
	UpgradeFromTier string
	UpgradeToTier   string
	UpgradeAmount   float64

	// Calculator answers
	CalcAgeCitizenship bool
	CalcStudyYear      string // "1-2", "3", "senior"
	CalcEnglishLevel   string // "b1", "b2", "c1"
}

// SessionManager is a thread-safe in-memory session manager with TTL auto-cleanup.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
	ttl      time.Duration
	db       *gorm.DB
}

// NewSessionManager creates a new session manager instance. When db is
// provided, sessions survive process restarts and are shared by instances.
func NewSessionManager(ttl time.Duration, dbs ...*gorm.DB) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[int64]*UserSession),
		ttl:      ttl,
	}
	if len(dbs) > 0 {
		sm.db = dbs[0]
	}

	// Background worker to clean up stale sessions every 10 minutes
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			sm.cleanup()
		}
	}()

	return sm
}

// Get returns existing user session or creates a new one in StateDefault.
func (sm *SessionManager) Get(chatID int64) *UserSession {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[chatID]
	if !exists {
		session = sm.loadFromDB(chatID)
		if session != nil {
			sm.sessions[chatID] = session
			exists = true
		}
	}
	if !exists || time.Since(session.LastActivity) > sm.ttl {
		session = &UserSession{
			State:        StateDefault,
			SelectedTier: "accelerator",
			LastActivity: time.Now(),
		}
		sm.sessions[chatID] = session
	} else {
		session.LastActivity = time.Now()
	}

	return session
}

// Peek returns the session without creating one.
func (sm *SessionManager) Peek(chatID int64) *UserSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.sessions[chatID]
}

func (sm *SessionManager) loadFromDB(chatID int64) *UserSession {
	if sm.db == nil {
		return nil
	}
	var record models.BotSession
	if err := sm.db.Where("telegram_id = ? AND expires_at > ?", chatID, time.Now()).First(&record).Error; err != nil {
		return nil
	}
	return &UserSession{
		State:              BotState(record.State),
		SelectedTier:       record.SelectedTier,
		PromoCode:          record.PromoCode,
		DiscountPct:        record.DiscountPct,
		LastActivity:       record.LastActivity,
		IsUpgrade:          record.IsUpgrade,
		UpgradeFromTier:    record.UpgradeFromTier,
		UpgradeToTier:      record.UpgradeToTier,
		UpgradeAmount:      record.UpgradeAmount,
		CalcAgeCitizenship: record.CalcAgeCitizenship,
		CalcStudyYear:      record.CalcStudyYear,
		CalcEnglishLevel:   record.CalcEnglishLevel,
	}
}

// Save persists the current session (write-through). It is safe to call when
// no database is configured.
func (sm *SessionManager) Save(chatID int64, session *UserSession) {
	if sm == nil || session == nil {
		return
	}
	sm.mu.Lock()
	sm.sessions[chatID] = session
	db := sm.db
	sm.mu.Unlock()
	if db == nil {
		return
	}
	record := models.BotSession{
		TelegramID:         chatID,
		State:              string(session.State),
		SelectedTier:       session.SelectedTier,
		PromoCode:          session.PromoCode,
		DiscountPct:        session.DiscountPct,
		IsUpgrade:          session.IsUpgrade,
		UpgradeFromTier:    session.UpgradeFromTier,
		UpgradeToTier:      session.UpgradeToTier,
		UpgradeAmount:      session.UpgradeAmount,
		CalcAgeCitizenship: session.CalcAgeCitizenship,
		CalcStudyYear:      session.CalcStudyYear,
		CalcEnglishLevel:   session.CalcEnglishLevel,
		LastActivity:       session.LastActivity,
		ExpiresAt:          time.Now().Add(sm.ttl),
	}
	_ = db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&record).Error
}

// SetState updates the state of a user.
func (sm *SessionManager) SetState(chatID int64, state BotState) {
	sm.mu.Lock()
	session, exists := sm.sessions[chatID]
	if !exists {
		session = &UserSession{
			SelectedTier: "accelerator",
		}
		sm.sessions[chatID] = session
	}
	session.State = state
	session.LastActivity = time.Now()
	sm.mu.Unlock()
	sm.Save(chatID, session)
}

// Reset clears session state back to default.
func (sm *SessionManager) Reset(chatID int64) {
	sm.mu.Lock()
	delete(sm.sessions, chatID)
	db := sm.db
	sm.mu.Unlock()
	if db != nil {
		_ = db.Where("telegram_id = ?", chatID).Delete(&models.BotSession{}).Error
	}
}

func (sm *SessionManager) cleanup() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	for id, session := range sm.sessions {
		if now.Sub(session.LastActivity) > sm.ttl {
			delete(sm.sessions, id)
		}
	}
}
