package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestVerifyTelegramWebApp(t *testing.T) {
	botToken := "123456:test-bot-token"
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	values.Set("query_id", "AAExample")
	values.Set("user", `{"id":42,"first_name":"Test","username":"test"}`)

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(botToken))
	mac := hmac.New(sha256.New, secret.Sum(nil))
	mac.Write([]byte(strings.Join(parts, "\n")))
	values.Set("hash", hex.EncodeToString(mac.Sum(nil)))

	if _, ok := VerifyTelegramWebApp(values.Encode(), botToken); !ok {
		t.Fatal("valid initData rejected")
	}
	if _, ok := VerifyTelegramWebApp(values.Encode(), "wrong-token"); ok {
		t.Fatal("invalid token accepted")
	}
	tampered := values.Encode() + "x"
	if _, ok := VerifyTelegramWebApp(tampered, botToken); ok {
		t.Fatal("tampered initData accepted")
	}
}
