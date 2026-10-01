package security

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware injects enterprise-grade HTTP security headers into every response.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "SAMEORIGIN")

		// Legacy XSS protection filter
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Restrict dangerous browser features
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

		// Content Security Policy permitting YouTube embeds
		csp := "default-src 'self'; " +
			"img-src 'self' data: https: https://*.telegram.org; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://telegram.org; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' data: https://fonts.gstatic.com; " +
			"frame-src 'self' https://www.youtube.com https://www.youtube-nocookie.com; " +
			"frame-ancestors 'self'; " +
			"connect-src 'self' http://localhost:* https://*.sparkprep.ru https://sparkprep.ru;"
		c.Header("Content-Security-Policy", csp)

		c.Next()
	}
}

// MaxBodySizeMiddleware limits the maximum payload size (in bytes) to prevent memory exhaustion DoS attacks.
func MaxBodySizeMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
