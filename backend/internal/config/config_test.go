package config

import (
	"os"
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	// Clear any overrides
	os.Unsetenv("TELEGRAM_INVITE_LINK")
	os.Unsetenv("PORT")

	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.TelegramInviteLink != "" {
		t.Errorf("expected empty default invite link, got %s", cfg.TelegramInviteLink)
	}
}

func TestConfigCustomInviteLink(t *testing.T) {
	customLink := "https://t.me/+custom_private_invite_2027"
	os.Setenv("TELEGRAM_INVITE_LINK", customLink)
	defer os.Unsetenv("TELEGRAM_INVITE_LINK")

	cfg := Load()
	if cfg.TelegramInviteLink != customLink {
		t.Errorf("expected %s, got %s", customLink, cfg.TelegramInviteLink)
	}
}

func TestConfigValidateProduction(t *testing.T) {
	cfg := &Config{
		Domain:           "sparkprep.ru",
		JWTSecret:        "super-secret-spark-jwt-key-2027",
		AdminPassword:    "SparkAdmin2026!",
		TelegramBotToken: "",
	}

	warnings := cfg.Validate()
	if len(warnings) == 0 {
		t.Fatal("expected security warnings for default credentials in production domain")
	}

	hasJWTWarning := false
	hasPassWarning := false
	hasBotWarning := false
	for _, w := range warnings {
		if len(w) > 0 {
			if w[:24] == "SECURITY WARNING: JWT_SE" {
				hasJWTWarning = true
			}
			if w[:24] == "SECURITY WARNING: ADMIN_" {
				hasPassWarning = true
			}
			if w[:24] == "SECURITY WARNING: TELEGR" {
				hasBotWarning = true
			}
		}
	}

	if !hasJWTWarning || !hasPassWarning || !hasBotWarning {
		t.Errorf("missing expected warnings: JWT=%v, Pass=%v, Bot=%v", hasJWTWarning, hasPassWarning, hasBotWarning)
	}
}
