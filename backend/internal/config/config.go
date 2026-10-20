package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	DatabaseURL        string
	TelegramBotToken   string
	TelegramAPIServer  string
	TelegramInviteLink string
	TelegramChannelID  int64
	JWTSecret          string
	AdminTelegramID    int64
	AdminUsername      string
	AdminPassword      string
	Domain             string
	FrontendURL        string
	AntilopayPublicKey string
	AntilopaySecretID  string
	TributeAPIKey      string
	EnotSecretKey      string
	EnotShopID         string
	YooKassaShopID     string
	YooKassaSecretKey  string
	TelegramBotUsername string
	TelegramWebhookSecret string
	DigestHour         int
	ZoomMeetingURL     string
}

func Load() *Config {
	// Attempt to load .env file if it exists
	_ = godotenv.Load()

	adminTGID, _ := strconv.ParseInt(getEnv("ADMIN_TELEGRAM_ID", "123456789"), 10, 64)
	tgChannelID, _ := strconv.ParseInt(getEnv("TELEGRAM_CHANNEL_ID", "0"), 10, 64)
	digestHour, _ := strconv.Atoi(getEnv("DIGEST_HOUR", "21"))
	if digestHour < 0 || digestHour > 23 {
		digestHour = 21
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://spark_user:StrongPassword2026@postgres:5432/spark_db?sslmode=disable"),
		TelegramBotToken:   getEnv("TELEGRAM_BOT_TOKEN", ""),
		TelegramBotUsername: getEnv("NEXT_PUBLIC_BOT_USERNAME", getEnv("TELEGRAM_BOT_USERNAME", "spark_prep_bot")),
		TelegramAPIServer:  getEnv("TELEGRAM_API_SERVER", ""),
		TelegramInviteLink: getEnv("TELEGRAM_INVITE_LINK", ""),
		TelegramChannelID:  tgChannelID,
		JWTSecret:          getEnv("JWT_SECRET", "super-secret-spark-jwt-key-2027"),
		AdminTelegramID:    adminTGID,
		AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "SparkAdmin2026!"),
		Domain:             getEnv("DOMAIN", "sparkprep.ru"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
		AntilopayPublicKey: getEnv("ANTILOPAY_PUBLIC_KEY", ""),
		AntilopaySecretID:  getEnv("ANTILOPAY_SECRET_ID", ""),
		TributeAPIKey:      getEnv("TRIBUTE_API_KEY", ""),
		EnotSecretKey:      getEnv("ENOT_SECRET_KEY", ""),
		EnotShopID:         getEnv("ENOT_SHOP_ID", ""),
		YooKassaShopID:     getEnv("YOOKASSA_SHOP_ID", "1491540"),
		YooKassaSecretKey:  getEnv("YOOKASSA_SECRET_KEY", ""),
		TelegramWebhookSecret: getEnv("TELEGRAM_WEBHOOK_SECRET", ""),
		DigestHour:         digestHour,
		ZoomMeetingURL:     getEnv("ZOOM_MEETING_URL", "https://us05web.zoom.us/j/spark2027"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// Validate checks for critical security misconfigurations in production environments.
func (c *Config) Validate() []string {
	var warnings []string
	isProd := os.Getenv("ENV") == "production" || os.Getenv("GIN_MODE") == "release" || (c.Domain != "localhost" && c.Domain != "127.0.0.1")

	if isProd {
		if c.JWTSecret == "super-secret-spark-jwt-key-2027" || len(c.JWTSecret) < 32 {
			warnings = append(warnings, "SECURITY WARNING: JWT_SECRET is using the insecure default or is shorter than 32 characters! Set a strong random secret.")
		}
		if c.AdminPassword == "SparkAdmin2026!" {
			warnings = append(warnings, "SECURITY WARNING: ADMIN_PASSWORD is using the default value! Set a strong unique password or bcrypt hash.")
		}
		if c.TelegramBotToken == "" {
			warnings = append(warnings, "SECURITY WARNING: TELEGRAM_BOT_TOKEN is empty! Telegram authentication and notifications will not function.")
		}
	}
	return warnings
}
