package storage

import (
	"context"

	"avitoTechAutumn2025/internal/domain"
)

// TxManager управляет транзакциями базы данных
type TxManager interface {
	// Do выполняет функцию fn внутри транзакции
	// Если fn возвращает ошибку, транзакция откатывается
	// Иначе транзакция коммитится
	Do(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
}

// Tx представляет транзакцию с доступом к репозиториям
type Tx interface {
	PullRequestRepo() PullRequestRepository
	UserRepo() UserRepository
	TeamRepo() TeamRepository
}

// PullRequestRepository определяет операции с pull requests
type PullRequestRepository interface {
	// Create создаёт новый pull request
	Create(ctx context.Context, pr *domain.PullRequest) error

	// GetByID возвращает pull request по ID
	GetByID(ctx context.Context, id string) (*domain.PullRequest, error)

	// Update обновляет pull request
	Update(ctx context.Context, pr *domain.PullRequest) error

	// AssignReviewer назначает ревьювера на PR
	AssignReviewer(ctx context.Context, prID, reviewerID string) error

	// UnassignReviewer удаляет ревьювера с PR
	UnassignReviewer(ctx context.Context, prID, reviewerID string) error

	// GetReviewers возвращает список ревьюверов PR
	GetReviewers(ctx context.Context, prID string) ([]string, error)

	// GetPRsReviewedByUser возвращает список PR где пользователь является ревьювером
	GetPRsReviewedByUser(ctx context.Context, userID string) ([]domain.PullRequestShort, error)

	// GetInactiveReviewers возвращает список неактивных ревьюверов для данного PR
	GetInactiveReviewers(ctx context.Context, prID string) ([]string, error)
}

// UserRepository определяет операции с пользователями
type UserRepository interface {
	// GetByID возвращает пользователя по ID
	GetByID(ctx context.Context, userID string) (*domain.User, error)

	// Update обновляет пользователя
	Update(ctx context.Context, user *domain.User) error

	// GetActiveTeamMembers возвращает активных членов команды (исключая указанного пользователя)
	GetActiveTeamMembers(ctx context.Context, excludeUserID string) ([]domain.User, error)

	// CreateBatch создаёт нескольких пользователей за раз
	CreateBatch(ctx context.Context, users []domain.User) error
}

// TeamRepository определяет операции с командами
type TeamRepository interface {
	// Create создаёт команду и её участников (с upsert логикой)
	Create(ctx context.Context, team *domain.Team, users []domain.User) error

	// GetByName возвращает команду по имени с её участниками
	GetByName(ctx context.Context, name string) (*domain.Team, error)

	// DeactivateAllMembers деактивирует всех участников команды (batch update)
	DeactivateAllMembers(ctx context.Context, teamName string) (int, error)
}
