package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spark-school/backend/internal/bot"
	"github.com/spark-school/backend/internal/config"
	"github.com/spark-school/backend/internal/database"
)

func main() {
	log.Println("Starting Standalone проект «так называемый SPARK» Telegram Bot Service...")

	cfg := config.Load()
	if cfg.TelegramBotToken == "" {
		log.Println("WARNING: TELEGRAM_BOT_TOKEN is empty. Set it in .env or environment variables.")
	}

	// Initialize Database connection for bot handlers
	db := database.Init(cfg)

	// Create Bot Service
	botService := bot.NewBotService(cfg, db)

	// Start bot polling in a goroutine
	go botService.Start()

	log.Println("Standalone Telegram Bot service is running. Waiting for events...")

	// Listen for OS signals to stop gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Standalone Telegram Bot service...")
}
