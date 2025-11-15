package metrics

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"
)

// StartDBStatsCollector запускает горутину для периодического сбора статистики connection pool
func StartDBStatsCollector(sqlDB *sql.DB, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := sqlDB.Stats()

			// Обновляем метрики connection pool
			DBConnectionPoolActive.Set(float64(stats.InUse))
			DBConnectionPoolIdle.Set(float64(stats.Idle))

			log.Debug().
				Int("in_use", stats.InUse).
				Int("idle", stats.Idle).
				Int("max_open", stats.MaxOpenConnections).
				Msg("updated db connection pool metrics")

		case <-stopCh:
			log.Info().Msg("stopping db stats collector")
			return
		}
	}
}
