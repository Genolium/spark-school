package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestIPRateLimiter_Allow(t *testing.T) {
	limiter := NewIPRateLimiter(2, 50*time.Millisecond)

	// 1st request - allowed
	allowed, _ := limiter.Allow("1.2.3.4")
	if !allowed {
		t.Errorf("expected 1st request to be allowed")
	}

	// 2nd request - allowed
	allowed, _ = limiter.Allow("1.2.3.4")
	if !allowed {
		t.Errorf("expected 2nd request to be allowed")
	}

	// 3rd request - blocked
	allowed, remaining := limiter.Allow("1.2.3.4")
	if allowed {
		t.Errorf("expected 3rd request to be blocked")
	}
	if remaining <= 0 {
		t.Errorf("expected positive remaining duration")
	}

	// Different IP - allowed
	allowed, _ = limiter.Allow("5.6.7.8")
	if !allowed {
		t.Errorf("expected different IP to be allowed")
	}

	// Wait for window expiration
	time.Sleep(60 * time.Millisecond)
	allowed, _ = limiter.Allow("1.2.3.4")
	if !allowed {
		t.Errorf("expected request after window reset to be allowed")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimitMiddleware(2, 50*time.Millisecond, "Rate limit reached"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Req 1: 200
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("req1: got %d, want 200", w1.Code)
	}

	// Req 2: 200
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("req2: got %d, want 200", w2.Code)
	}

	// Req 3: 429
	req3 := httptest.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("req3: got %d, want 429", w3.Code)
	}
}
