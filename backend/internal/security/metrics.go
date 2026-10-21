package security

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// Metrics holds process-local counters used by the monitoring worker.
type Metrics struct {
	webhook5xx        atomic.Int64
	readinessFailures atomic.Int64
}

func NewMetrics() *Metrics { return &Metrics{} }

func (m *Metrics) RecordReadinessFailure() {
	if m != nil {
		m.readinessFailures.Add(1)
	}
}

func (m *Metrics) RecordWebhook5xx() {
	if m != nil {
		m.webhook5xx.Add(1)
	}
}

func (m *Metrics) SwapReadinessFailures() int64 {
	if m == nil {
		return 0
	}
	return m.readinessFailures.Swap(0)
}

func (m *Metrics) SwapWebhook5xx() int64 {
	if m == nil {
		return 0
	}
	return m.webhook5xx.Swap(0)
}

// WebhookMetricsMiddleware counts 5xx responses produced by payment/Telegram
// webhook routes. It must wrap the handler (be registered before it).
func WebhookMetricsMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if m != nil && c.Writer.Status() >= 500 {
			m.webhook5xx.Add(1)
		}
	}
}
