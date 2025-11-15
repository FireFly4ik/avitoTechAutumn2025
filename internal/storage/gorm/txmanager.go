package gorm

import (
	"avitoTechAutumn2025/internal/config"
	"avitoTechAutumn2025/internal/metrics"
	"avitoTechAutumn2025/internal/storage"
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// txManager реализует storage.TxManager для GORM
type txManager struct {
	db *gorm.DB
}

// NewTxManager создаёт новый менеджер транзакций для GORM
func NewTxManager(envConf *config.Config) (storage.TxManager, error) {
	db, err := ConnectDB(envConf)
	if err != nil {
		return nil, err
	}

	// Получаем *sql.DB для мониторинга connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Запускаем коллектор метрик connection pool
	stopCh := make(chan struct{})
	go metrics.StartDBStatsCollector(sqlDB, 5*time.Second, stopCh)

	// Запускаем горутину для пересчёта метрик команды/юзера из БД
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		type teamRow struct {
			TeamName string
			Count    int
			Active   int
		}
		type userRow struct {
			UserID string
			Count  int
		}

		for {
			select {
			case <-ticker.C:
				// Получаем список всех команд из таблицы teams
				var allTeamNames []string
				if err := db.Raw(`SELECT team_name FROM teams`).Scan(&allTeamNames).Error; err != nil {
					log.Error().Err(err).Msg("failed to query all team names")
					continue
				}

				// Создаём map для быстрого доступа к актуальным данным команд
				teamData := make(map[string]teamRow)

				// Пересчитываем членов команд и активных
				var teams []teamRow
				// SQL: count and sum active per team
				err := db.Raw(`SELECT team_name, COUNT(*) as count, SUM(CASE WHEN is_active THEN 1 ELSE 0 END) as active FROM users GROUP BY team_name`).Scan(&teams).Error
				if err != nil {
					log.Error().Err(err).Msg("failed to query team membership counts")
					continue
				}

				// Заполняем map актуальными данными
				for _, t := range teams {
					teamData[t.TeamName] = t
				}

				// Обновляем метрики для всех команд из таблицы teams
				for _, teamName := range allTeamNames {
					if data, exists := teamData[teamName]; exists {
						// У команды есть пользователи
						metrics.TeamMembersCount.WithLabelValues(teamName).Set(float64(data.Count))
						metrics.TeamActiveMembersCount.WithLabelValues(teamName).Set(float64(data.Active))
						log.Debug().
							Str("team_name", teamName).
							Int("total", data.Count).
							Int("active", data.Active).
							Msg("updated team metrics")
					} else {
						// У команды нет пользователей - устанавливаем 0
						metrics.TeamMembersCount.WithLabelValues(teamName).Set(0)
						metrics.TeamActiveMembersCount.WithLabelValues(teamName).Set(0)
						log.Debug().
							Str("team_name", teamName).
							Msg("set team metrics to 0 (no members)")
					}
				}

				// Пересчитываем назначения review по пользователям
				var users []userRow
				err = db.Raw(`SELECT reviewer_id as user_id, COUNT(*) as count FROM pull_request_reviewers GROUP BY reviewer_id`).Scan(&users).Error
				if err != nil {
					log.Error().Err(err).Msg("failed to query review assignments counts")
				} else {
					// Сбрасываем метрику перед обновлением
					metrics.UserReviewAssignmentsCount.Reset()

					for _, u := range users {
						metrics.UserReviewAssignmentsCount.WithLabelValues(u.UserID).Set(float64(u.Count))
					}
				}

			case <-stopCh:
				log.Info().Msg("stopping metrics reconciliation goroutine")
				return
			}
		}
	}()

	return &txManager{db: db}, nil
}

// Do выполняет функцию внутри транзации с автоматическим commit/rollback
func (tm *txManager) Do(ctx context.Context, fn func(ctx context.Context, tx storage.Tx) error) error {
	start := time.Now()

	err := tm.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txWrapper := &transaction{
			db: tx,
		}

		err := fn(ctx, txWrapper)
		if err != nil {
			// GORM автоматически сделает ROLLBACK
			metrics.DBTransactionTotal.WithLabelValues("error").Inc()
			return err
		}

		// GORM автоматически сделает COMMIT
		metrics.DBTransactionTotal.WithLabelValues("success").Inc()
		return nil
	})

	// Записываем длительность транзакции
	metrics.DBTransactionDuration.Observe(time.Since(start).Seconds())

	return err
}

// transaction - обёртка над gorm.DB, реализует storage.Tx
type transaction struct {
	db *gorm.DB
}

// PullRequestRepo возвращает репозиторий PR в рамках транзакции
func (t *transaction) PullRequestRepo() storage.PullRequestRepository {
	return NewPullRequestRepository(t.db)
}

// UserRepo возвращает репозиторий пользователей в рамках транзакции
func (t *transaction) UserRepo() storage.UserRepository {
	return NewUserRepository(t.db)
}

// TeamRepo возвращает репозиторий команд в рамках транзакции
func (t *transaction) TeamRepo() storage.TeamRepository {
	return NewTeamRepository(t.db)
}

// Commit не нужен, так как GORM автоматически коммитит
func (t *transaction) Commit() error {
	return nil
}

// Rollback не нужен, так как GORM автоматически откатывает при ошибке
func (t *transaction) Rollback() error {
	return nil
}
