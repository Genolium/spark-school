package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env                   string
	Port                  string
	DatabaseURL           string
	TelegramBotToken      string
	TelegramAPIServer     string
	TelegramInviteLink    string
	TelegramChannelID     int64
	JWTSecret             string
	AdminTelegramID       int64
	AdminUsername         string
	AdminPassword         string
	Domain                string
	FrontendURL           string
	AntilopayPublicKey    string
	AntilopaySecretID     string
	TributeAPIKey         string
	EnotSecretKey         string
	EnotShopID            string
	YooKassaShopID        string
	YooKassaSecretKey     string
	TelegramBotUsername   string
	TelegramWebhookSecret string
	DigestHour            int
	ZoomMeetingURL        string
}

// Known insecure production defaults that must never be used.
var knownInsecureSecrets = map[string]struct{}{
	"super-secret-spark-jwt-key-2027": {},
	"SparkAdmin2026!":                 {},
	"StrongPassword2026":              {},
	"admin":                           {},
	"password":                        {},
	"secret":                          {},
}

func Load() *Config {
	// Attempt to load .env file if it exists
	_ = godotenv.Load()

	adminTGID, _ := strconv.ParseInt(getEnv("ADMIN_TELEGRAM_ID", "0"), 10, 64)
	tgChannelID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHANNEL_ID", "0"), 10, 64)
	digestHour, _ := strconv.Atoi(getEnv("DIGEST_HOUR", "21"))
	if digestHour < 0 || digestHour > 23 {
		digestHour = 21
	}

	env := strings.ToLower(strings.TrimSpace(getEnv("ENV", "development")))
	if env == "" {
		env = "development"
	}

	return &Config{
		Env:                   env,
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           strings.TrimSpace(os.Getenv("DATABASE_URL")),
		TelegramBotToken:      strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")),
		TelegramBotUsername:   getEnv("NEXT_PUBLIC_BOT_USERNAME", getEnv("TELEGRAM_BOT_USERNAME", "spark_prep_bot")),
		TelegramAPIServer:     getEnv("TELEGRAM_API_SERVER", ""),
		TelegramInviteLink:    getEnv("TELEGRAM_INVITE_LINK", ""),
		TelegramChannelID:     tgChannelID,
		JWTSecret:             strings.TrimSpace(os.Getenv("JWT_SECRET")),
		AdminTelegramID:       adminTGID,
		AdminUsername:         getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:         os.Getenv("ADMIN_PASSWORD"),
		Domain:                getEnv("DOMAIN", "localhost"),
		FrontendURL:           getEnv("FRONTEND_URL", "http://localhost:3000"),
		AntilopayPublicKey:    getEnv("ANTILOPAY_PUBLIC_KEY", ""),
		AntilopaySecretID:     getEnv("ANTILOPAY_SECRET_ID", ""),
		TributeAPIKey:         getEnv("TRIBUTE_API_KEY", ""),
		EnotSecretKey:         getEnv("ENOT_SECRET_KEY", ""),
		EnotShopID:            getEnv("ENOT_SHOP_ID", ""),
		YooKassaShopID:        getEnv("YOOKASSA_SHOP_ID", "1491540"),
		YooKassaSecretKey:     getEnv("YOOKASSA_SECRET_KEY", ""),
		TelegramWebhookSecret: strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_SECRET")),
		DigestHour:            digestHour,
		ZoomMeetingURL:        getEnv("ZOOM_MEETING_URL", ""),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// IsProduction reports whether the process must run with production safeguards.
// It is intentionally explicit: the deployment sets ENV=production (or GIN_MODE=release).
func (c *Config) IsProduction() bool {
	if c == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(c.Env), "production") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("GIN_MODE")), "release")
}

// Validate returns a hard error when the configuration is not safe to run.
// Development may use local values, but it must still provide explicit secrets:
// there are no working defaults baked into the binary.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("configuration is nil")
	}

	var problems []string

	if strings.TrimSpace(c.DatabaseURL) == "" {
		problems = append(problems, "DATABASE_URL is required")
	} else if _, err := url.Parse(c.DatabaseURL); err != nil {
		problems = append(problems, "DATABASE_URL is not a valid URL")
	}

	if strings.TrimSpace(c.JWTSecret) == "" {
		problems = append(problems, "JWT_SECRET is required")
	} else if len(c.JWTSecret) < 32 {
		problems = append(problems, "JWT_SECRET must be at least 32 characters")
	} else if isKnownInsecure(c.JWTSecret) {
		problems = append(problems, "JWT_SECRET must not be a known/default value")
	}

	if strings.TrimSpace(c.AdminPassword) == "" {
		problems = append(problems, "ADMIN_PASSWORD is required")
	} else if isKnownInsecure(c.AdminPassword) {
		problems = append(problems, "ADMIN_PASSWORD must not be a known/default value")
	} else if c.IsProduction() && len(c.AdminPassword) < 12 {
		problems = append(problems, "ADMIN_PASSWORD must be at least 12 characters in production")
	}

	if c.IsProduction() {
		if pw := databasePassword(c.DatabaseURL); pw == "" {
			problems = append(problems, "POSTGRES_PASSWORD (embedded in DATABASE_URL) is required")
		} else if isKnownInsecure(pw) || len(pw) < 16 {
			problems = append(problems, "POSTGRES_PASSWORD must be strong (at least 16 characters) and not a known default")
		}

		if strings.TrimSpace(c.TelegramWebhookSecret) == "" {
			problems = append(problems, "TELEGRAM_WEBHOOK_SECRET is required in production")
		} else if len(c.TelegramWebhookSecret) < 16 {
			problems = append(problems, "TELEGRAM_WEBHOOK_SECRET must be at least 16 characters in production")
		}

		if strings.TrimSpace(c.TelegramBotToken) == "" {
			problems = append(problems, "TELEGRAM_BOT_TOKEN is required in production")
		}

		if strings.TrimSpace(c.Domain) == "" {
			problems = append(problems, "DOMAIN is required in production")
		}

		// At least one real payment provider must be fully configured.
		providers := 0
		if strings.TrimSpace(c.YooKassaSecretKey) != "" {
			providers++
			if strings.TrimSpace(c.YooKassaShopID) == "" {
				problems = append(problems, "YOOKASSA_SHOP_ID is required when YOOKASSA_SECRET_KEY is set")
			}
		}
		if strings.TrimSpace(c.TributeAPIKey) != "" {
			providers++
		}
		if strings.TrimSpace(c.AntilopayPublicKey) != "" {
			providers++
		}
		if providers == 0 {
			problems = append(problems, "at least one payment provider key must be configured in production (YOOKASSA_SECRET_KEY, TRIBUTE_API_KEY or ANTILOPAY_PUBLIC_KEY)")
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration: %s", strings.Join(problems, "; "))
	}
	return nil
}

// Warnings returns non-fatal security notices for development environments.
func (c *Config) Warnings() []string {
	var warnings []string
	if c == nil {
		return warnings
	}
	if !c.IsProduction() {
		if len(c.JWTSecret) > 0 && len(c.JWTSecret) < 32 {
			warnings = append(warnings, "JWT_SECRET is shorter than 32 characters")
		}
		if strings.TrimSpace(c.TelegramWebhookSecret) == "" {
			warnings = append(warnings, "TELEGRAM_WEBHOOK_SECRET is empty; the webhook endpoint is only safe for local development")
		}
		if strings.TrimSpace(c.AdminPassword) == "" {
			warnings = append(warnings, "ADMIN_PASSWORD is empty; admin login is disabled")
		}
	}
	return warnings
}

func isKnownInsecure(value string) bool {
	_, found := knownInsecureSecrets[strings.TrimSpace(value)]
	return found
}

func databasePassword(databaseURL string) string {
	u, err := url.Parse(databaseURL)
	if err != nil || u.User == nil {
		return ""
	}
	pw, _ := u.User.Password()
	return pw
}

// GenerateSecret is a convenience for explicit local bootstrap only. Production
// deployments must inject secrets, never generate them at startup.
func GenerateSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
