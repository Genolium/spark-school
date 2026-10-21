package auth

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/spark-school/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestVerifyTOTP(t *testing.T) {
	// RFC 6238 test secret (base32 of "12345678901234567890").
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Unix(59, 0)
	code := totpCode([]byte("12345678901234567890"), uint64(now.Unix()/30))
	if !VerifyTOTP(secret, code, now) {
		t.Fatalf("expected %s to be valid", code)
	}
	if VerifyTOTP(secret, "000000", now) && code != "000000" {
		t.Fatal("invalid code accepted")
	}
	if VerifyTOTP("not-base32!", code, now) {
		t.Fatal("invalid secret accepted")
	}
}

func TestRevocation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:auth_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.RevokedToken{}); err != nil {
		t.Fatal(err)
	}
	user := &models.User{ID: 1, TelegramID: 42, Role: "admin", HasAccess: true}
	token, err := GenerateAdminJWT(user, "test-secret-key-32-chars-spark-2027")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := parseTokenString(token, "test-secret-key-32-chars-spark-2027")
	if err != nil {
		t.Fatal(err)
	}
	if claims.ID == "" {
		t.Fatal("token must have jti")
	}
	if isRevoked(db, claims.ID) {
		t.Fatal("new token must not be revoked")
	}
	if err := RevokeJWT(db, claims); err != nil {
		t.Fatal(err)
	}
	if !isRevoked(db, claims.ID) {
		t.Fatal("revoked token must be rejected")
	}
	_ = token
}
