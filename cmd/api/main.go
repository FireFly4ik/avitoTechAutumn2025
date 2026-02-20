package main

import (
	"avitoTechAutumn2025/internal/api/handlers"
	"avitoTechAutumn2025/internal/api/server"
	"avitoTechAutumn2025/internal/config"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/metrics"
	"avitoTechAutumn2025/internal/service"
	storageGorm "avitoTechAutumn2025/internal/storage/gorm"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := godotenv.Load(".env"); err != nil {
		fmt.Println("No .env file found")
	}

	envConfig := config.NewEnvConfig()

	// Валидируем обязательные параметры конфигурации
	if err := envConfig.Validate(); err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	envConfig.PrintConfigWithHiddenSecrets()
	logger.Setup(envConfig)

	// Подключаемся к БД (с retry при недоступности)
	database, err := storageGorm.ConnectDB(envConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	txManager := storageGorm.NewTxManager(database)

	// Запускаем фоновый сбор метрик (отдельно от TxManager)
	reconciler, err := metrics.NewReconciler(database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create metrics reconciler")
	}
	reconciler.Start(5 * time.Second)

	appService := service.New(txManager)
	appHandler := handlers.NewHandler(appService)
	apiServer := server.NewServer(envConfig, appHandler)

	go apiServer.Run()

	// Ожидаем сигнал завершения
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Info().Str("signal", s.String()).Msg("signal received — starting graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Останавливаем HTTP-сервер (перестаём принимать новые запросы, дожидаем текущие)
	apiServer.Shutdown(ctx)

	// 2. Останавливаем фоновые горутины метрик
	reconciler.Stop()

	// 3. Закрываем DB connection pool
	sqlDB, err := database.DB()
	if err != nil {
		log.Error().Err(err).Msg("failed to get sql.DB for closing")
	} else {
		if err := sqlDB.Close(); err != nil {
			log.Error().Err(err).Msg("error closing database connection pool")
		} else {
			log.Info().Msg("database connection pool closed")
		}
	}

	log.Info().Msg("service shutdown gracefully")
}
