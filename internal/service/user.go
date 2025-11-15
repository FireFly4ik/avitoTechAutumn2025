package service

import (
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/metrics"
	"avitoTechAutumn2025/internal/storage"
	"context"
	"time"

	"github.com/rs/zerolog/log"
)

// SetUserIsActive изменяет статус активности пользователя
func (s *Service) SetUserIsActive(outerCtx context.Context, userID string, isActive bool) (*domain.User, error) {
	const op = "service.SetUserIsActive"
	requestID := logger.GetRequestID(outerCtx)
	var user *domain.User

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("set_user_active").Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("user_id", userID).
		Bool("is_active", isActive).
		Msg("setting user active status")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		u, err := tx.UserRepo().GetByID(ctx, userID)
		if err != nil {
			return err
		}

		u.IsActive = isActive
		if err := tx.UserRepo().Update(ctx, u); err != nil {
			return err
		}

		user = u
		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Обновляем метрики статуса пользователя
	status := "inactive"
	if user.IsActive {
		status = "active"
	}
	metrics.UserActiveStatusChanged.WithLabelValues(status).Inc()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("user_id", user.UserID).
		Bool("is_active", user.IsActive).
		Msg("successfully updated user active status")

	return user, nil
}

// GetReviewerAssignments возвращает список PR, где пользователь назначен ревьювером
func (s *Service) GetReviewerAssignments(outerCtx context.Context, userID string) ([]domain.PullRequestShort, error) {
	const op = "service.GetReviewerAssignments"
	requestID := logger.GetRequestID(outerCtx)
	var prs []domain.PullRequestShort

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("get_reviewer_assignments").Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("user_id", userID).
		Msg("fetching PRs reviewed by user")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		result, err := tx.PullRequestRepo().GetPRsReviewedByUser(ctx, userID)
		if err != nil {
			return err
		}
		prs = result
		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Метрику количества назначений обновляет reconcile-горутина автоматически

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("user_id", userID).
		Int("pr_count", len(prs)).
		Msg("successfully fetched reviewed PRs")

	return prs, nil
}

// DeactivateTeamMembers массово деактивирует всех участников команды
func (s *Service) DeactivateTeamMembers(outerCtx context.Context, input *domain.DeactivateTeamInput) (*domain.DeactivateTeamResult, error) {
	const op = "service.DeactivateTeamMembers"
	requestID := logger.GetRequestID(outerCtx)
	var result *domain.DeactivateTeamResult

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("deactivate_team_members").Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("team_name", input.TeamName).
		Msg("deactivating all team members")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		// Проверяем существование команды
		_, err := tx.TeamRepo().GetByName(ctx, input.TeamName)
		if err != nil {
			return err
		}

		// Деактивируем всех участников команды одним batch update
		deactivatedCount, err := tx.TeamRepo().DeactivateAllMembers(ctx, input.TeamName)
		if err != nil {
			return err
		}

		result = &domain.DeactivateTeamResult{
			TeamName:             input.TeamName,
			DeactivatedUserCount: deactivatedCount,
		}

		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Обновляем метрики
	metrics.UserActiveStatusChanged.WithLabelValues("inactive").Add(float64(result.DeactivatedUserCount))

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("team_name", result.TeamName).
		Int("deactivated_count", result.DeactivatedUserCount).
		Msg("successfully deactivated all team members")

	return result, nil
}
