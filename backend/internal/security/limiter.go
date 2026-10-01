package security

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// clientRecord tracks requests from a specific IP within a rolling window.
type clientRecord struct {
	count       int
	windowStart time.Time
}

// IPRateLimiter provides thread-safe sliding-window rate limiting per IP address.
type IPRateLimiter struct {
	mu          sync.Mutex
	records     map[string]*clientRecord
	limit       int           // Max requests allowed per window
	window      time.Duration // Time window
	cleanupTick time.Duration
}

// NewIPRateLimiter creates an IPRateLimiter instance.
func NewIPRateLimiter(limit int, window time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		records:     make(map[string]*clientRecord),
		limit:       limit,
		window:      window,
		cleanupTick: 5 * time.Minute,
	}

	// Background cleaner goroutine to prevent memory leaks from inactive IPs
	go limiter.cleanupLoop()

	return limiter
}

// Allow checks if the given IP is within rate limits.
func (rl *IPRateLimiter) Allow(ip string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	rec, exists := rl.records[ip]

	if !exists || now.Sub(rec.windowStart) > rl.window {
		// New window
		rl.records[ip] = &clientRecord{
			count:       1,
			windowStart: now,
		}
		return true, 0
	}

	if rec.count >= rl.limit {
		// Exceeded limit: return remaining time in window
		remaining := rl.window - now.Sub(rec.windowStart)
		return false, remaining
	}

	rec.count++
	return true, 0
}

func (rl *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, rec := range rl.records {
			if now.Sub(rec.windowStart) > rl.window*2 {
				delete(rl.records, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware returns a Gin middleware enforcing the specified rate limit.
func RateLimitMiddleware(limit int, window time.Duration, customMsg string) gin.HandlerFunc {
	limiter := NewIPRateLimiter(limit, window)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown"
		}

		allowed, remaining := limiter.Allow(ip)
		if !allowed {
			retrySeconds := int(remaining.Seconds())
			if retrySeconds < 1 {
				retrySeconds = 1
			}

			c.Header("Retry-After", strconv.Itoa(retrySeconds))
			msg := customMsg
			if msg == "" {
				msg = "Превышен лимит запросов. Пожалуйста, подождите перед следующим действием."
			}

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               msg,
				"retry_after_seconds": retrySeconds,
			})
			return
		}

		c.Next()
	}
}
