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

	err := s.txmgr.DoRead(outerCtx, func(ctx context.Context, tx storage.Tx) error {
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

// DeactivateTeamMembers массово деактивирует указанных участников команды и переназначает их ревью
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
		Any("user_ids", input.UserIDs).
		Msg("deactivating team members")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		// Проверяем существование команды
		_, err := tx.TeamRepo().GetByName(ctx, input.TeamName)
		if err != nil {
			return err
		}

		// Деактивируем участников команды
		var deactivatedCount int
		if len(input.UserIDs) > 0 {
			// Деактивируем только указанных пользователей
			deactivatedCount, err = tx.TeamRepo().DeactivateMembers(ctx, input.TeamName, input.UserIDs)
		} else {
			// Если user_ids пустой — деактивируем всех
			deactivatedCount, err = tx.TeamRepo().DeactivateAllMembers(ctx, input.TeamName)
		}
		if err != nil {
			return err
		}

		// Находим все открытые PR, где деактивированные пользователи являются ревьюверами
		userIDsToReassign := input.UserIDs
		if len(userIDsToReassign) == 0 {
			// Если деактивировали всех — получаем список всех участников команды
			team, err := tx.TeamRepo().GetByName(ctx, input.TeamName)
			if err != nil {
				return err
			}
			for _, member := range team.Members {
				userIDsToReassign = append(userIDsToReassign, member.UserID)
			}
		}

		openPRIDs, err := tx.PullRequestRepo().GetOpenPRsByReviewers(ctx, userIDsToReassign)
		if err != nil {
			return err
		}

		// Переназначаем неактивных ревьюверов на каждом затронутом PR
		reassignedCount := 0
		for _, prID := range openPRIDs {
			pr, err := tx.PullRequestRepo().GetByID(ctx, prID)
			if err != nil {
				return err
			}

			// Получаем список неактивных ревьюверов на этом PR
			inactiveReviewers, err := tx.PullRequestRepo().GetInactiveReviewers(ctx, prID)
			if err != nil {
				return err
			}

			for _, oldReviewerID := range inactiveReviewers {
				// Получаем активных членов команды заменяемого ревьювера
				activeUsers, err := tx.UserRepo().GetActiveTeamMembers(ctx, oldReviewerID)
				if err != nil {
					return err
				}

				// Формируем список кандидатов, исключая автора PR и текущих ревьюверов
				excludeMap := make(map[string]bool)
				excludeMap[pr.AuthorID] = true
				for _, r := range pr.AssignedReviewers {
					excludeMap[r] = true
				}

				candidates := make([]domain.User, 0)
				for _, user := range activeUsers {
					if !excludeMap[user.UserID] {
						candidates = append(candidates, user)
					}
				}

				if len(candidates) == 0 {
					// Нет кандидатов — просто удаляем ревьювера
					if err := tx.PullRequestRepo().UnassignReviewer(ctx, prID, oldReviewerID); err != nil {
						return err
					}
					// Обновляем список в памяти
					newReviewers := make([]string, 0, len(pr.AssignedReviewers)-1)
					for _, r := range pr.AssignedReviewers {
						if r != oldReviewerID {
							newReviewers = append(newReviewers, r)
						}
					}
					pr.AssignedReviewers = newReviewers
					reassignedCount++

					log.Info().
						Str("request_id", requestID).
						Str("pull_request_id", prID).
						Str("old_reviewer_id", oldReviewerID).
						Msg("removed inactive reviewer during team deactivation (no candidates)")
					continue
				}

				// Случайно выбираем нового ревьювера
				index, err := secureRandomInt(len(candidates))
				if err != nil {
					return err
				}
				newReviewer := candidates[index].UserID

				// Удаляем старого и назначаем нового
				if err := tx.PullRequestRepo().UnassignReviewer(ctx, prID, oldReviewerID); err != nil {
					return err
				}
				if err := tx.PullRequestRepo().AssignReviewer(ctx, prID, newReviewer); err != nil {
					return err
				}

				// Обновляем список в памяти
				newReviewers := make([]string, 0, len(pr.AssignedReviewers))
				for _, r := range pr.AssignedReviewers {
					if r == oldReviewerID {
						newReviewers = append(newReviewers, newReviewer)
					} else {
						newReviewers = append(newReviewers, r)
					}
				}
				pr.AssignedReviewers = newReviewers
				reassignedCount++

				log.Info().
					Str("request_id", requestID).
					Str("pull_request_id", prID).
					Str("old_reviewer_id", oldReviewerID).
					Str("new_reviewer_id", newReviewer).
					Msg("reassigned reviewer during team deactivation")
			}
		}

		result = &domain.DeactivateTeamResult{
			TeamName:             input.TeamName,
			DeactivatedUserCount: deactivatedCount,
			ReassignedCount:      reassignedCount,
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
		Int("reassigned_count", result.ReassignedCount).
		Msg("successfully deactivated team members and reassigned reviews")

	return result, nil
}
