package config

import (
	"strings"
	"testing"
)

func TestConfigLoadDefaults_NoWorkingSecrets(t *testing.T) {
	t.Setenv("TELEGRAM_INVITE_LINK", "")
	t.Setenv("PORT", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ADMIN_PASSWORD", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.TelegramInviteLink != "" {
		t.Errorf("expected empty default invite link, got %s", cfg.TelegramInviteLink)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("JWT_SECRET must not have a built-in default, got %q", cfg.JWTSecret)
	}
	if cfg.AdminPassword != "" {
		t.Errorf("ADMIN_PASSWORD must not have a built-in default, got %q", cfg.AdminPassword)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DATABASE_URL must not have a built-in default, got %q", cfg.DatabaseURL)
	}
}

func TestConfigCustomInviteLink(t *testing.T) {
	customLink := "https://t.me/+custom_private_invite_2027"
	t.Setenv("TELEGRAM_INVITE_LINK", customLink)

	cfg := Load()
	if cfg.TelegramInviteLink != customLink {
		t.Errorf("expected %s, got %s", customLink, cfg.TelegramInviteLink)
	}
}

func validProductionConfig() *Config {
	return &Config{
		Env:                   "production",
		DatabaseURL:           "postgres://spark_user:a-very-strong-postgres-password@postgres:5432/spark_db?sslmode=disable",
		JWTSecret:             strings.Repeat("a", 48),
		AdminPassword:         strings.Repeat("b", 20),
		TelegramBotToken:      "123456:AAExampleTelegramBotTokenHere",
		TelegramWebhookSecret: strings.Repeat("c", 32),
		Domain:                "sparkprep.ru",
		YooKassaShopID:        "1491540",
		YooKassaSecretKey:     "live_example_key",
	}
}

func TestConfigValidate_ProductionAcceptsStrongConfig(t *testing.T) {
	cfg := validProductionConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid production config, got: %v", err)
	}
}

func TestConfigValidate_ProductionRejectsDefaultsAndMissingSecrets(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
		want   string
	}{
		{"default jwt", func(c *Config) { c.JWTSecret = "super-secret-spark-jwt-key-2027" }, "JWT_SECRET"},
		{"short jwt", func(c *Config) { c.JWTSecret = "short" }, "JWT_SECRET"},
		{"missing admin password", func(c *Config) { c.AdminPassword = "" }, "ADMIN_PASSWORD"},
		{"default admin password", func(c *Config) { c.AdminPassword = "SparkAdmin2026!" }, "ADMIN_PASSWORD"},
		{"missing postgres password", func(c *Config) {
			c.DatabaseURL = "postgres://spark_user@postgres:5432/spark_db?sslmode=disable"
		}, "POSTGRES_PASSWORD"},
		{"default postgres password", func(c *Config) {
			c.DatabaseURL = "postgres://spark_user:StrongPassword2026@postgres:5432/spark_db?sslmode=disable"
		}, "POSTGRES_PASSWORD"},
		{"missing webhook secret", func(c *Config) { c.TelegramWebhookSecret = "" }, "TELEGRAM_WEBHOOK_SECRET"},
		{"missing bot token", func(c *Config) { c.TelegramBotToken = "" }, "TELEGRAM_BOT_TOKEN"},
		{"no payment provider", func(c *Config) {
			c.YooKassaSecretKey = ""
			c.TributeAPIKey = ""
			c.AntilopayPublicKey = ""
		}, "payment provider"},
		{"yookassa without shop id", func(c *Config) { c.YooKassaShopID = "" }, "YOOKASSA_SHOP_ID"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validProductionConfig()
			tc.mutate(cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestConfigValidate_DevelopmentAllowsMissingPaymentProviders(t *testing.T) {
	cfg := &Config{
		Env:           "development",
		DatabaseURL:   "postgres://spark_user:localdev@localhost:5432/spark_db?sslmode=disable",
		JWTSecret:     strings.Repeat("a", 40),
		AdminPassword: "local-dev-password",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development config should be accepted: %v", err)
	}
}

func TestDatabasePassword(t *testing.T) {
	cases := map[string]string{
		"postgres://user:pass@host:5432/db": "pass",
		"postgres://user@host:5432/db":      "",
		"not-a-url":                         "",
	}
	for input, want := range cases {
		if got := databasePassword(input); got != want {
			t.Errorf("databasePassword(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestConfigTrustedProxiesAndRetention(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1, 172.16.0.0/12, ::1")
	t.Setenv("ANALYTICS_RETENTION_DAYS", "90")
	cfg := Load()
	if len(cfg.TrustedProxies) != 3 {
		t.Fatalf("expected three trusted proxies, got %v", cfg.TrustedProxies)
	}
	if cfg.AnalyticsRetentionDays != 90 {
		t.Fatalf("expected 90-day retention, got %d", cfg.AnalyticsRetentionDays)
	}
	t.Setenv("ANALYTICS_RETENTION_DAYS", "invalid")
	if got := Load().AnalyticsRetentionDays; got != 180 {
		t.Fatalf("invalid retention must fall back to 180, got %d", got)
	}
}
