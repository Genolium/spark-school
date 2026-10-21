package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// ClientError accepts sanitized browser errors, logs them and forwards them to
// Sentry when configured. It never accepts form values or cookies.
func ClientError(c *gin.Context) {
	var req struct {
		Message string `json:"message"`
		Stack   string `json:"stack"`
		URL     string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message is required"})
		return
	}
	if len(message) > 500 {
		message = message[:500]
	}
	stack := req.Stack
	if len(stack) > 8000 {
		stack = stack[:8000]
	}
	url := req.URL
	if len(url) > 500 {
		url = url[:500]
	}

	event := sentry.NewEvent()
	event.Level = sentry.LevelError
	event.Message = message
	event.Tags = map[string]string{"source": "frontend"}
	event.Contexts = map[string]sentry.Context{
		"frontend": {"url": url, "stack": stack},
	}
	sentry.CaptureEvent(event)
	log.Printf("[ClientError] %s (%s)", message, url)

	c.JSON(http.StatusAccepted, gin.H{"ok": true})
}
