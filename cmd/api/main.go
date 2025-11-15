package main

import (
	"avitoTechAutumn2025/internal/api/handlers"
	"avitoTechAutumn2025/internal/api/server"
	"avitoTechAutumn2025/internal/config"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/service"
	storageGorm "avitoTechAutumn2025/internal/storage/gorm"
	"context"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("No .env file found")
	}
	envConfig := config.NewEnvConfig()
	envConfig.PrintConfigWithHiddenSecrets()

	logger.Setup(envConfig)

	// Подключаемся к БД и создаём менеджер транзакций
	database, err := storageGorm.ConnectDB(envConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	txManager, err := storageGorm.NewTxManager(database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	appService := service.New(txManager)
	appHandler := handlers.NewHandler(appService)
	apiServer := server.NewServer(envConfig, appHandler)

	go apiServer.Run()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Info().Msg(fmt.Sprintf("signal received: %s — starting graceful shutdown", s))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	apiServer.Shutdown(ctx)

	log.Info().Msg("service shutdown gracefully")
}
