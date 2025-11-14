package logger

import (
	"avitoTechAutumn2025/internal/api/middleware"
	"avitoTechAutumn2025/internal/config"
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Setup инициализирует и настраивает логгер zerolog в зависимости от окружения.
// В режиме debug логи выводятся в stdout, в production - в файл.
// Возвращает настроенный экземпляр логгера.
func Setup(envConf *config.Config) *zerolog.Logger {
	// Установка уровня логирования в зависимости от окружения
	if envConf.ProductionType == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel) // Все логи, включая debug
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel) // Только info и выше
	}

	// Формат времени в логах: часы:минуты:секунды день.месяц.год
	zerolog.TimeFieldFormat = "15:04:05 02.01.2006"

	// Кастомизация отображения информации о вызывающем коде
	// Показываем только последние 2 части пути к файлу для краткости
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		parts := strings.Split(file, "/")
		if len(parts) > 2 {
			file = strings.Join(parts[len(parts)-2:], "/")
		}
		return fmt.Sprintf("%s:%d", file, line)
	}

	var writer io.Writer

	// Выбор выходного потока для логов
	if envConf.ProductionType == "prod" {
		// В production логи пишутся в файл
		logPath := envConf.LogPath

		// Создание директории для логов, если она не существует
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			log.Fatal().Err(err).Msg("failed to create logger directory")
		}

		// Открытие файла логов (создание или дополнение)
		logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open logger file")
		}
		writer = logFile
	} else {
		// В debug режиме логи выводятся в консоль
		writer = os.Stdout
	}

	// Создание логгера с добавлением информации о вызывающем коде и timestamp
	loggerContext := zerolog.New(writer).
		With().
		Caller().    // Добавляет информацию о файле и строке вызова
		Timestamp(). // Добавляет метку времени
		Logger()

	// Устанавливаем глобальный логгер, чтобы все вызовы log.Info() использовали его
	log.Logger = loggerContext

	log.Info().Msg("logger setup complete")
	return &loggerContext
}

func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(middleware.RequestIDKey).(string); ok {
		return requestID
	}
	return "unknown"
}
