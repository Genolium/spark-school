package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spark-school/backend/internal/models"
)

type JWTClaims struct {
	UserID     uint   `json:"user_id"`
	TelegramID int64  `json:"telegram_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	HasAccess  bool   `json:"has_access"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a signed token with 30-day validity.
func GenerateJWT(user *models.User, secret string) (string, error) {
	claims := JWTClaims{
		UserID:     user.ID,
		TelegramID: user.TelegramID,
		Username:   user.Username,
		Role:       user.Role,
		HasAccess:  user.HasAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "spark-school-auth",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// VerifyTelegramWidget validates data received from Telegram Login Widget.
func VerifyTelegramWidget(data map[string]string, botToken string) bool {
	hashReceived, ok := data["hash"]
	if !ok || hashReceived == "" {
		return false
	}

	var keys []string
	for k := range data {
		if k != "hash" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var checkStringParts []string
	for _, k := range keys {
		checkStringParts = append(checkStringParts, fmt.Sprintf("%s=%s", k, data[k]))
	}
	dataCheckString := strings.Join(checkStringParts, "\n")

	// Secret key for widget is sha256(botToken)
	secretKey := sha256.Sum256([]byte(botToken))

	h := hmac.New(sha256.New, secretKey[:])
	h.Write([]byte(dataCheckString))
	calculatedHash := hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(calculatedHash), []byte(hashReceived)) {
		return false
	}

	// Verify that auth_date is not older than 24 hours (anti-replay attack)
	if authDateStr, ok := data["auth_date"]; ok {
		if authDate, err := strconv.ParseInt(authDateStr, 10, 64); err == nil {
			if time.Now().Unix()-authDate > 86400 {
				return false
			}
		}
	}

	return true
}

func parseToken(c *gin.Context, secret string) (*JWTClaims, error) {
	tokenStr := ""

	// Check Authorization header
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Fallback to cookie
	if tokenStr == "" {
		if cookie, err := c.Cookie("spark_session"); err == nil {
			tokenStr = cookie
		}
	}

	if tokenStr == "" {
		return nil, fmt.Errorf("missing session token")
	}

	claims := &JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid session token")
	}

	return claims, nil
}

// AuthMiddleware validates the JWT token in Authorization header or Cookie.
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := parseToken(c, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: " + err.Error()})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("telegram_id", claims.TelegramID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("has_access", claims.HasAccess)
		c.Next()
	}
}

// AdminMiddleware ensures current user has admin role from their JWT.
func AdminMiddleware(jwtSecret string) gin.HandlerFunc {
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

		c.Set("user_id", claims.UserID)
		c.Set("telegram_id", claims.TelegramID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("has_access", claims.HasAccess)
		c.Next()
	}
}
