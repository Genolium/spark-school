package bot

import (
	"sync"
	"time"
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
)

// UserSession holds state machine and temporary calculator or payment data.
type UserSession struct {
	State       BotState
	SelectedTier string // "accelerator" or "vip"
	PromoCode    string
	DiscountPct  int
	LastActivity time.Time

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
}

// NewSessionManager creates a new session manager instance.
func NewSessionManager(ttl time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions: make(map[int64]*UserSession),
		ttl:      ttl,
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

// SetState updates the state of a user.
func (sm *SessionManager) SetState(chatID int64, state BotState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[chatID]
	if !exists {
		session = &UserSession{
			SelectedTier: "accelerator",
		}
		sm.sessions[chatID] = session
	}
	session.State = state
	session.LastActivity = time.Now()
}

// Reset clears session state back to default.
func (sm *SessionManager) Reset(chatID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, chatID)
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
