package service

import (
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/storage"
	"context"
	"github.com/rs/zerolog/log"
)

// SetUserIsActive изменяет статус активности пользователя
func (s *Service) SetUserIsActive(outerCtx context.Context, userID string, isActive bool) (*domain.User, error) {
	const op = "service.SetUserIsActive"
	requestID := logger.GetRequestID(outerCtx)
	var user *domain.User

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

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("user_id", userID).
		Int("pr_count", len(prs)).
		Msg("successfully fetched reviewed PRs")

	return prs, nil
}
