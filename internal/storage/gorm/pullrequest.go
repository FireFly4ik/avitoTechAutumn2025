package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/storage"
)

type pullRequestRepository struct {
	db *gorm.DB
}

// NewPullRequestRepository создаёт новый репозиторий PR
func NewPullRequestRepository(db *gorm.DB) storage.PullRequestRepository {
	return &pullRequestRepository{db: db}
}

// Create создаёт новый pull request
func (r *pullRequestRepository) Create(ctx context.Context, pr *domain.PullRequest) error {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", pr.ID).
		Msg("creating pull request in database")

	dbPR := &PullRequest{
		PullRequestID:   pr.ID,
		PullRequestName: pr.Name,
		AuthorID:        pr.AuthorID,
		Status:          string(pr.Status),
	}

	result := r.db.WithContext(ctx).Clauses(clause.Returning{}).Create(dbPR)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == storage.UniqueViolation {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", pr.ID).
				Msg("pull request already exists")
			return storage.ErrAlreadyExists
		}
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", pr.ID).
				Msg("pull request already exists (gorm)")
			return storage.ErrAlreadyExists
		}
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", pr.ID).
			Msg("error creating pull request")
		return result.Error
	}

	// Обновляем CreatedAt в domain модели
	pr.CreatedAt = &dbPR.CreatedAt

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", pr.ID).
		Msg("successfully created pull request")

	return nil
}

// GetByID получает pull request по ID
func (r *pullRequestRepository) GetByID(ctx context.Context, pullRequestID string) (*domain.PullRequest, error) {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", pullRequestID).
		Msg("fetching pull request by ID")

	var dbPR PullRequest
	result := r.db.WithContext(ctx).First(&dbPR, "pull_request_id = ?", pullRequestID)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", pullRequestID).
				Msg("pull request not found")
			return nil, storage.ErrNotFound
		}
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", pullRequestID).
			Msg("error fetching pull request")
		return nil, result.Error
	}

	// Получаем ревьюверов
	reviewers, err := r.GetReviewers(ctx, pullRequestID)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", pullRequestID).
		Msg("successfully fetched pull request")

	return &domain.PullRequest{
		ID:                dbPR.PullRequestID,
		Name:              dbPR.PullRequestName,
		AuthorID:          dbPR.AuthorID,
		Status:            domain.PullRequestStatus(dbPR.Status),
		AssignedReviewers: reviewers,
		CreatedAt:         &dbPR.CreatedAt,
		MergedAt:          dbPR.MergedAt,
	}, nil
}

// Update обновляет pull request (используется для merge)
func (r *pullRequestRepository) Update(ctx context.Context, pr *domain.PullRequest) error {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", pr.ID).
		Msg("updating pull request")

	// Сначала проверяем существование
	var existingPR PullRequest
	result := r.db.WithContext(ctx).Where("pull_request_id = ?", pr.ID).First(&existingPR)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", pr.ID).
				Msg("pull request not found")
			return storage.ErrNotFound
		}
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", pr.ID).
			Msg("error fetching pull request")
		return result.Error
	}

	// Обновляем поля
	existingPR.Status = string(pr.Status)
	if pr.Status == domain.PullRequestStatusMerged && pr.MergedAt == nil {
		now := time.Now()
		existingPR.MergedAt = &now
		pr.MergedAt = &now
	} else {
		existingPR.MergedAt = pr.MergedAt
	}

	result = r.db.WithContext(ctx).Save(&existingPR)
	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", pr.ID).
			Msg("error saving pull request")
		return result.Error
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", pr.ID).
		Msg("successfully updated pull request")

	return nil
}

// AssignReviewer назначает ревьювера на PR
func (r *pullRequestRepository) AssignReviewer(ctx context.Context, prID, reviewerID string) error {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", prID).
		Str("reviewer_id", reviewerID).
		Msg("assigning reviewer to pull request")

	// Проверяем существование PR
	var pr PullRequest
	if err := r.db.WithContext(ctx).Where("pull_request_id = ?", prID).First(&pr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", prID).
				Msg("pull request not found")
			return storage.ErrNotFound
		}
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Msg("error fetching pull request")
		return err
	}

	// Проверяем существование пользователя
	var user User
	if err := r.db.WithContext(ctx).Where("user_id = ?", reviewerID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("reviewer_id", reviewerID).
				Msg("reviewer not found")
			return storage.ErrNotFound
		}
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("reviewer_id", reviewerID).
			Msg("error fetching reviewer")
		return err
	}

	// Проверяем не назначен ли уже
	var existing Reviewer
	err := r.db.WithContext(ctx).
		Where("pull_request_id = ? AND reviewer_id = ?", prID, reviewerID).
		First(&existing).Error

	if err == nil {
		log.Warn().
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Str("reviewer_id", reviewerID).
			Msg("reviewer already assigned")
		return storage.ErrAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Str("reviewer_id", reviewerID).
			Msg("error checking existing assignment")
		return err
	}

	// Создаём назначение
	reviewer := &Reviewer{
		PullRequestID: prID,
		ReviewerID:    reviewerID,
	}

	result := r.db.WithContext(ctx).Create(reviewer)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == storage.UniqueViolation {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", prID).
				Str("reviewer_id", reviewerID).
				Msg("reviewer already assigned (unique constraint)")
			return storage.ErrAlreadyExists
		}
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return storage.ErrAlreadyExists
		}
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Str("reviewer_id", reviewerID).
			Msg("error creating reviewer assignment")
		return result.Error
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", prID).
		Str("reviewer_id", reviewerID).
		Msg("successfully assigned reviewer")

	return nil
}

// UnassignReviewer снимает ревьювера с PR
func (r *pullRequestRepository) UnassignReviewer(ctx context.Context, prID, reviewerID string) error {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", prID).
		Str("reviewer_id", reviewerID).
		Msg("unassigning reviewer from pull request")

	// Проверяем существование и статус PR
	var pr PullRequest
	if err := r.db.WithContext(ctx).Where("pull_request_id = ?", prID).First(&pr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", prID).
				Msg("pull request not found")
			return storage.ErrNotFound
		}
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Msg("error fetching pull request")
		return err
	}

	if pr.Status == "MERGED" {
		log.Warn().
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Msg("pull request already merged")
		return storage.ErrConflict
	}

	// Проверяем существование пользователя
	var user User
	if err := r.db.WithContext(ctx).Where("user_id = ?", reviewerID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("reviewer_id", reviewerID).
				Msg("reviewer not found")
			return storage.ErrNotFound
		}
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("reviewer_id", reviewerID).
			Msg("error fetching reviewer")
		return err
	}

	// Проверяем что ревьювер назначен
	var prReviewer Reviewer
	if err := r.db.WithContext(ctx).
		Where("pull_request_id = ? AND reviewer_id = ?", prID, reviewerID).
		First(&prReviewer).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn().
				Str("request_id", requestID).
				Str("layer", "storage").
				Str("pull_request_id", prID).
				Str("reviewer_id", reviewerID).
				Msg("reviewer not assigned to pull request")
			return storage.ErrNotFound
		}
		log.Error().
			Err(err).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Str("reviewer_id", reviewerID).
			Msg("error checking reviewer assignment")
		return err
	}

	// Удаляем назначение
	result := r.db.WithContext(ctx).Delete(&prReviewer)
	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Str("reviewer_id", reviewerID).
			Msg("error deleting reviewer assignment")
		return result.Error
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", prID).
		Str("reviewer_id", reviewerID).
		Msg("successfully unassigned reviewer")

	return nil
}

// GetReviewers получает список ревьюверов для PR
func (r *pullRequestRepository) GetReviewers(ctx context.Context, prID string) ([]string, error) {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", prID).
		Msg("fetching reviewers for pull request")

	var reviewers []Reviewer
	result := r.db.WithContext(ctx).
		Where("pull_request_id = ?", prID).
		Find(&reviewers)

	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("pull_request_id", prID).
			Msg("error fetching reviewers")
		return nil, result.Error
	}

	reviewerIDs := make([]string, len(reviewers))
	for i, r := range reviewers {
		reviewerIDs[i] = r.ReviewerID
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("pull_request_id", prID).
		Int("reviewers_count", len(reviewerIDs)).
		Msg("successfully fetched reviewers")

	return reviewerIDs, nil
}

// GetPRsReviewedByUser получает список PR, где пользователь является ревьювером
func (r *pullRequestRepository) GetPRsReviewedByUser(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	requestID := logger.GetRequestID(ctx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("user_id", userID).
		Msg("fetching PRs reviewed by user")

	var dbPRs []PullRequest

	result := r.db.WithContext(ctx).
		Joins("JOIN pull_request_reviewers ON pull_request_reviewers.pull_request_id = pull_requests.pull_request_id").
		Where("pull_request_reviewers.reviewer_id = ?", userID).
		Find(&dbPRs)

	if result.Error != nil {
		log.Error().
			Err(result.Error).
			Str("request_id", requestID).
			Str("layer", "storage").
			Str("user_id", userID).
			Msg("error fetching reviewed PRs")
		return nil, result.Error
	}

	prs := make([]domain.PullRequestShort, len(dbPRs))
	for i, dbPR := range dbPRs {
		prs[i] = domain.PullRequestShort{
			ID:       dbPR.PullRequestID,
			Name:     dbPR.PullRequestName,
			AuthorID: dbPR.AuthorID,
			Status:   domain.PullRequestStatus(dbPR.Status),
		}
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "storage").
		Str("user_id", userID).
		Int("pr_count", len(prs)).
		Msg("successfully fetched reviewed PRs")

	return prs, nil
}
