package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
)

func randomJTI() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// RevokeJWT makes an issued token unusable across all instances.
func RevokeJWT(db *gorm.DB, claims *JWTClaims) error {
	if db == nil || claims == nil || claims.ID == "" {
		return fmt.Errorf("cannot revoke token without jti")
	}
	expires := time.Now().Add(time.Hour)
	if claims.ExpiresAt != nil {
		expires = claims.ExpiresAt.Time
	}
	return db.Create(&models.RevokedToken{JTI: claims.ID, ExpiresAt: expires, CreatedAt: time.Now()}).Error
}

func isRevoked(db *gorm.DB, jti string) bool {
	if db == nil || jti == "" {
		return false
	}
	var count int64
	if err := db.Model(&models.RevokedToken{}).Where("jti = ?", jti).Count(&count).Error; err != nil {
		// Fail closed: if revocation state cannot be checked, reject the token.
		return true
	}
	return count > 0
}

// AdminMiddlewareWithRevocation is AdminMiddleware plus a database-backed
// revocation check, so logout works across backend instances.
func AdminMiddlewareWithRevocation(jwtSecret string, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := parseToken(c, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
			return
		}
		if claims.Role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden: Administrator privileges required"})
			return
		}
		if isRevoked(db, claims.ID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: session revoked"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("telegram_id", claims.TelegramID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("has_access", claims.HasAccess)
		c.Set("jwt_claims", claims)
		c.Next()
	}
}

// VerifyTOTP validates a 6-digit RFC 6238 code with a one-step clock window.
func VerifyTOTP(secretBase32, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secretBase32)))
	if err != nil {
		return false
	}
	step := now.Unix() / 30
	for offset := int64(-1); offset <= 1; offset++ {
		if hmac.Equal([]byte(totpCode(secret, uint64(step+offset))), []byte(code)) {
			return true
		}
	}
	return false
}

func totpCode(secret []byte, counter uint64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, secret)
	mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}
