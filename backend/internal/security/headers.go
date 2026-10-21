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

		// Content Security Policy. 'unsafe-eval' is only needed by the Next.js
		// development runtime; production builds work without it.
		scriptSrc := "'self' 'unsafe-inline' https://telegram.org https://yookassa.ru https://*.yookassa.ru"
		if gin.Mode() != gin.ReleaseMode {
			scriptSrc += " 'unsafe-eval'"
		}
		csp := "default-src 'self'; " +
			"base-uri 'self'; object-src 'none'; " +
			"img-src 'self' data: https: https://*.telegram.org https://*.yoomoney.ru https://*.yookassa.ru; " +
			"script-src " + scriptSrc + "; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' data: https://fonts.gstatic.com; " +
			"frame-src 'self' https://www.youtube.com https://www.youtube-nocookie.com https://telegram.org https://oauth.telegram.org https://yookassa.ru https://*.yookassa.ru; " +
			"frame-ancestors 'self'; " +
			"form-action 'self' https://yookassa.ru https://*.yookassa.ru; " +
			"connect-src 'self' http://localhost:* https://*.sparkprep.ru https://sparkprep.ru https://so-called-spark.ru https://*.so-called-spark.ru https://api.yookassa.ru https://yookassa.ru https://api.telegram.org;"
		c.Header("Content-Security-Policy", csp)
		c.Header("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
		c.Header("Cross-Origin-Resource-Policy", "same-site")
		if gin.Mode() == gin.ReleaseMode {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			c.Header("Content-Security-Policy", csp+"; upgrade-insecure-requests")
		}

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
