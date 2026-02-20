package metrics

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Reconciler периодически пересчитывает метрики из БД.
// Выделен в отдельный компонент, чтобы не создавать побочных эффектов
// в конструкторах (например, в NewTxManager).
type Reconciler struct {
	db     *gorm.DB
	sqlDB  *sql.DB
	stopCh chan struct{}
}

// NewReconciler создаёт новый Reconciler для периодического обновления метрик
func NewReconciler(db *gorm.DB) (*Reconciler, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	return &Reconciler{
		db:     db,
		sqlDB:  sqlDB,
		stopCh: make(chan struct{}),
	}, nil
}

// Start запускает фоновые горутины для сбора метрик.
// Вызывать из main.go после создания всех зависимостей.
func (r *Reconciler) Start(interval time.Duration) {
	// Горутина для метрик connection pool
	go StartDBStatsCollector(r.sqlDB, interval, r.stopCh)

	// Горутина для пересчёта бизнес-метрик из БД
	go r.reconcileLoop(interval)
}

// Stop останавливает все фоновые горутины.
// Вызывать при graceful shutdown.
func (r *Reconciler) Stop() {
	close(r.stopCh)
	log.Info().Msg("metrics reconciler stopped")
}

func (r *Reconciler) reconcileLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
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
			if err := r.db.Raw(`SELECT team_name FROM teams`).Scan(&allTeamNames).Error; err != nil {
				log.Error().Err(err).Msg("failed to query all team names")
				continue
			}

			teamData := make(map[string]teamRow)

			var teams []teamRow
			err := r.db.Raw(`SELECT team_name, COUNT(*) as count, SUM(CASE WHEN is_active THEN 1 ELSE 0 END) as active FROM users GROUP BY team_name`).Scan(&teams).Error
			if err != nil {
				log.Error().Err(err).Msg("failed to query team membership counts")
				continue
			}

			for _, t := range teams {
				teamData[t.TeamName] = t
			}

			for _, teamName := range allTeamNames {
				if data, exists := teamData[teamName]; exists {
					TeamMembersCount.WithLabelValues(teamName).Set(float64(data.Count))
					TeamActiveMembersCount.WithLabelValues(teamName).Set(float64(data.Active))
				} else {
					TeamMembersCount.WithLabelValues(teamName).Set(0)
					TeamActiveMembersCount.WithLabelValues(teamName).Set(0)
				}
			}

			var users []userRow
			err = r.db.Raw(`SELECT reviewer_id as user_id, COUNT(*) as count FROM pull_request_reviewers GROUP BY reviewer_id`).Scan(&users).Error
			if err != nil {
				log.Error().Err(err).Msg("failed to query review assignments counts")
			} else {
				UserReviewAssignmentsCount.Reset()
				for _, u := range users {
					UserReviewAssignmentsCount.WithLabelValues(u.UserID).Set(float64(u.Count))
				}
			}

		case <-r.stopCh:
			log.Info().Msg("stopping metrics reconciliation goroutine")
			return
		}
	}
}
