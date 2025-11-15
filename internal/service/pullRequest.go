package service

import (
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/metrics"
	"avitoTechAutumn2025/internal/storage"
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"github.com/rs/zerolog/log"
)

// secureRandomInt возвращает криптографически безопасное случайное число от 0 до max-1
func secureRandomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

// CreatePullRequest создаёт новый pull request с автоматическим назначением ревьюверов
func (s *Service) CreatePullRequest(outerCtx context.Context, input *domain.CreatePullRequestInput) (*domain.PullRequest, error) {
	const op = "service.CreatePullRequest"
	requestID := logger.GetRequestID(outerCtx)
	var pr *domain.PullRequest

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("create_pull_request").Observe(time.Since(start).Seconds())
		metrics.PRCreationDuration.Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("author_id", input.AuthorID).
		Msg("creating pull request with transaction")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		// Проверяем что PR с таким ID еще не существует
		existingPR, err := tx.PullRequestRepo().GetByID(ctx, input.PullRequestID)
		if err == nil && existingPR != nil {
			return storage.ErrAlreadyExists
		}
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return err
		}

		// Получаем список активных членов команды автора (исключая самого автора)
		activeUsers, err := tx.UserRepo().GetActiveTeamMembers(ctx, input.AuthorID)
		if err != nil {
			return err
		}

		// Выбираем до 2 случайных ревьюверов из доступных
		reviewers := make([]string, 0, 2)
		lenActive := len(activeUsers)
		selectionStart := time.Now()
		for i := 0; i < min(2, lenActive); i++ {
			index, err := secureRandomInt(len(activeUsers))
			if err != nil {
				return err
			}
			userID := activeUsers[index].UserID
			activeUsers = append(activeUsers[:index], activeUsers[index+1:]...)
			reviewers = append(reviewers, userID)
		}
		metrics.RandomReviewerSelectionDuration.Observe(time.Since(selectionStart).Seconds())
		metrics.PRReviewersAssigned.Observe(float64(len(reviewers)))

		log.Info().
			Str("request_id", requestID).
			Str("layer", "service").
			Str("pull_request_id", input.PullRequestID).
			Any("selected_reviewers", reviewers).
			Int("selected_reviewers_count", len(reviewers)).
			Msg("selected reviewers randomly")

		// Создаем pull request со статусом OPEN
		pr = &domain.PullRequest{
			ID:                input.PullRequestID,
			Name:              input.PullRequestName,
			AuthorID:          input.AuthorID,
			Status:            domain.PullRequestStatusOpen,
			AssignedReviewers: reviewers,
		}

		if err := tx.PullRequestRepo().Create(ctx, pr); err != nil {
			return err
		}

		// Создаем записи связи PR с ревьюверами в таблице reviewers
		for _, reviewerID := range reviewers {
			if err := tx.PullRequestRepo().AssignReviewer(ctx, pr.ID, reviewerID); err != nil {
				return err
			}
		}

		log.Info().
			Str("request_id", requestID).
			Str("layer", "service").
			Str("pull_request_id", pr.ID).
			Int("reviewers_count", len(reviewers)).
			Msg("successfully created pull request with reviewers in transaction")

		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Увеличиваем счетчики метрик
	metrics.PRCreatedTotal.Inc()
	metrics.PROpenCount.Inc()

	return pr, nil
}

// MergePullRequest выполняет merge pull request
func (s *Service) MergePullRequest(outerCtx context.Context, input *domain.MergePullRequestInput) (*domain.PullRequest, error) {
	const op = "service.MergePullRequest"
	requestID := logger.GetRequestID(outerCtx)
	var pr *domain.PullRequest

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("merge_pull_request").Observe(time.Since(start).Seconds())
		metrics.PRMergeDuration.Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("pull_request_id", input.PullRequestID).
		Msg("merging pull request")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		existingPR, err := tx.PullRequestRepo().GetByID(ctx, input.PullRequestID)
		if err != nil {
			return err
		}

		// Обеспечиваем идемпотентность - повторный merge не изменяет данные
		if existingPR.Status == domain.PullRequestStatusMerged {
			log.Info().
				Str("request_id", requestID).
				Str("layer", "service").
				Str("pull_request_id", input.PullRequestID).
				Msg("PR already merged, returning current state (idempotent)")
			pr = existingPR
			return nil
		}

		// Обновляем статус на MERGED
		existingPR.Status = domain.PullRequestStatusMerged
		now := time.Now()
		existingPR.MergedAt = &now

		if err := tx.PullRequestRepo().Update(ctx, existingPR); err != nil {
			return err
		}

		pr = existingPR
		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Увеличиваем счетчики метрик
	metrics.PRMergedTotal.Inc()
	metrics.PROpenCount.Dec()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("pull_request_id", pr.ID).
		Str("status", string(pr.Status)).
		Msg("successfully merged pull request")

	return pr, nil
}

// ReassignPullRequest переназначает ревьювера на другого члена команды
func (s *Service) ReassignPullRequest(outerCtx context.Context, input *domain.ReassignPullRequestInput) (*domain.ReassignPullRequestResult, error) {
	const op = "service.ReassignPullRequest"
	requestID := logger.GetRequestID(outerCtx)
	var result *domain.ReassignPullRequestResult

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("reassign_pull_request").Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("pull_request_id", input.PullRequestID).
		Str("old_reviewer_id", input.OldUserID).
		Msg("reassigning pull request reviewer")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		pr, err := tx.PullRequestRepo().GetByID(ctx, input.PullRequestID)
		if err != nil {
			return err
		}

		// Запрещаем переназначение для уже смерженных PR
		if pr.Status == domain.PullRequestStatusMerged {
			return domain.ErrReassignOnMerged
		}

		// Проверяем что указанный пользователь действительно является ревьювером этого PR
		isReviewer := false
		for _, r := range pr.AssignedReviewers {
			if r == input.OldUserID {
				isReviewer = true
				break
			}
		}
		if !isReviewer {
			return domain.ErrReviewerMissing
		}

		// Получаем активных членов команды заменяемого ревьювера
		activeUsers, err := tx.UserRepo().GetActiveTeamMembers(ctx, input.OldUserID)
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
			metrics.UserNoCandidatesErrors.Inc()
			return domain.ErrNoCandidate
		}

		// Случайно выбираем нового ревьювера из списка кандидатов
		index, err := secureRandomInt(len(candidates))
		if err != nil {
			return err
		}
		newReviewer := candidates[index].UserID

		log.Info().
			Str("request_id", requestID).
			Str("layer", "service").
			Str("pull_request_id", input.PullRequestID).
			Str("new_reviewer_id", newReviewer).
			Msg("selected new reviewer")

		// Удаляем старого ревьювера и добавляем нового
		if err := tx.PullRequestRepo().UnassignReviewer(ctx, input.PullRequestID, input.OldUserID); err != nil {
			return err
		}

		if err := tx.PullRequestRepo().AssignReviewer(ctx, input.PullRequestID, newReviewer); err != nil {
			return err
		}

		// Обновляем список ревьюверов в памяти для ответа
		newReviewers := make([]string, 0, len(pr.AssignedReviewers))
		for _, r := range pr.AssignedReviewers {
			if r == input.OldUserID {
				newReviewers = append(newReviewers, newReviewer)
			} else {
				newReviewers = append(newReviewers, r)
			}
		}
		pr.AssignedReviewers = newReviewers

		result = &domain.ReassignPullRequestResult{
			PullRequest: *pr,
			ReplacedBy:  newReviewer,
		}

		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Увеличиваем счетчик переназначений
	metrics.PRReassignedTotal.Inc()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("pull_request_id", result.PullRequest.ID).
		Str("old_reviewer_id", input.OldUserID).
		Str("new_reviewer_id", result.ReplacedBy).
		Msg("successfully reassigned reviewer")

	return result, nil
}

// ReassignInactiveReviewers переназначает всех неактивных ревьюверов PR
func (s *Service) ReassignInactiveReviewers(outerCtx context.Context, input *domain.ReassignInactiveInput) (*domain.ReassignInactiveResult, error) {
	const op = "service.ReassignInactiveReviewers"
	requestID := logger.GetRequestID(outerCtx)
	var result *domain.ReassignInactiveResult

	start := time.Now()
	defer func() {
		metrics.ServiceOperationDuration.WithLabelValues("reassign_inactive_reviewers").Observe(time.Since(start).Seconds())
	}()

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("pull_request_id", input.PullRequestID).
		Msg("reassigning all inactive reviewers")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		// Получаем PR
		pr, err := tx.PullRequestRepo().GetByID(ctx, input.PullRequestID)
		if err != nil {
			return err
		}

		// Запрещаем переназначение для уже смерженных PR
		if pr.Status == domain.PullRequestStatusMerged {
			return domain.ErrReassignOnMerged
		}

		// Получаем список неактивных ревьюверов
		inactiveReviewers, err := tx.PullRequestRepo().GetInactiveReviewers(ctx, input.PullRequestID)
		if err != nil {
			return err
		}

		if len(inactiveReviewers) == 0 {
			// Нет неактивных ревьюверов - возвращаем пустой результат
			result = &domain.ReassignInactiveResult{
				PullRequestID:       input.PullRequestID,
				ReassignmentDetails: []domain.ReviewerReassignment{},
			}
			return nil
		}

		log.Info().
			Str("request_id", requestID).
			Str("pull_request_id", input.PullRequestID).
			Int("inactive_count", len(inactiveReviewers)).
			Any("inactive_reviewers", inactiveReviewers).
			Msg("found inactive reviewers")

		reassignments := make([]domain.ReviewerReassignment, 0, len(inactiveReviewers))

		// Для каждого неактивного ревьювера пытаемся найти замену
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
				// Нет кандидатов - просто удаляем ревьювера
				if err := tx.PullRequestRepo().UnassignReviewer(ctx, input.PullRequestID, oldReviewerID); err != nil {
					return err
				}

				reassignments = append(reassignments, domain.ReviewerReassignment{
					OldReviewerID: oldReviewerID,
					NewReviewerID: "",
					WasRemoved:    true,
				})

				// Удаляем из списка ревьюверов в памяти
				newReviewers := make([]string, 0, len(pr.AssignedReviewers)-1)
				for _, r := range pr.AssignedReviewers {
					if r != oldReviewerID {
						newReviewers = append(newReviewers, r)
					}
				}
				pr.AssignedReviewers = newReviewers

				log.Info().
					Str("request_id", requestID).
					Str("pull_request_id", input.PullRequestID).
					Str("old_reviewer_id", oldReviewerID).
					Msg("removed inactive reviewer (no candidates)")

				continue
			}

			// Случайно выбираем нового ревьювера
			index, err := secureRandomInt(len(candidates))
			if err != nil {
				return err
			}
			newReviewer := candidates[index].UserID

			// Удаляем старого и назначаем нового
			if err := tx.PullRequestRepo().UnassignReviewer(ctx, input.PullRequestID, oldReviewerID); err != nil {
				return err
			}

			if err := tx.PullRequestRepo().AssignReviewer(ctx, input.PullRequestID, newReviewer); err != nil {
				return err
			}

			reassignments = append(reassignments, domain.ReviewerReassignment{
				OldReviewerID: oldReviewerID,
				NewReviewerID: newReviewer,
				WasRemoved:    false,
			})

			// Обновляем список ревьюверов в памяти
			newReviewers := make([]string, 0, len(pr.AssignedReviewers))
			for _, r := range pr.AssignedReviewers {
				if r == oldReviewerID {
					newReviewers = append(newReviewers, newReviewer)
				} else {
					newReviewers = append(newReviewers, r)
				}
			}
			pr.AssignedReviewers = newReviewers

			// Добавляем нового ревьювера в excludeMap для следующих итераций
			excludeMap[newReviewer] = true

			log.Info().
				Str("request_id", requestID).
				Str("pull_request_id", input.PullRequestID).
				Str("old_reviewer_id", oldReviewerID).
				Str("new_reviewer_id", newReviewer).
				Msg("reassigned inactive reviewer")
		}

		result = &domain.ReassignInactiveResult{
			PullRequestID:       input.PullRequestID,
			ReassignmentDetails: reassignments,
		}

		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	// Обновляем метрики
	for _, detail := range result.ReassignmentDetails {
		if detail.WasRemoved {
			metrics.UserNoCandidatesErrors.Inc()
		} else {
			metrics.PRReassignedTotal.Inc()
		}
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("pull_request_id", result.PullRequestID).
		Int("reassigned_count", len(result.ReassignmentDetails)).
		Msg("successfully reassigned all inactive reviewers")

	return result, nil
}
