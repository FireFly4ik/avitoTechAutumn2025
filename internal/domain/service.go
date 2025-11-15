package domain

import "context"

// AssignmentService - интерфейс бизнес-логики сервиса управления PR и командами
//
//go:generate mockery --name=AssignmentService --output=../mocks --outpkg=mocks --filename=assignment_service_mock.go
type AssignmentService interface {
	// CreatePullRequest создаёт новый pull request с автоматическим назначением ревьюверов
	CreatePullRequest(ctx context.Context, input *CreatePullRequestInput) (*PullRequest, error)

	// MergePullRequest выполняет merge pull request
	MergePullRequest(ctx context.Context, input *MergePullRequestInput) (*PullRequest, error)

	// ReassignPullRequest переназначает ревьювера на другого члена команды
	ReassignPullRequest(ctx context.Context, input *ReassignPullRequestInput) (*ReassignPullRequestResult, error)

	// ReassignInactiveReviewers переназначает всех неактивных ревьюверов на активных членов команды на определённом PR
	ReassignInactiveReviewers(ctx context.Context, input *ReassignInactiveInput) (*ReassignInactiveResult, error)

	// CreateTeam создаёт новую команду с участниками
	CreateTeam(ctx context.Context, team *Team) (*Team, error)

	// GetTeam возвращает информацию о команде по имени
	GetTeam(ctx context.Context, teamName string) (*Team, error)

	// DeactivateTeamMembers деактивирует всех пользователей в команде
	DeactivateTeamMembers(ctx context.Context, input *DeactivateTeamInput) (*DeactivateTeamResult, error)

	// SetUserIsActive изменяет статус активности пользователя
	SetUserIsActive(ctx context.Context, userID string, isActive bool) (*User, error)

	// GetReviewerAssignments возвращает список PR, где пользователь назначен ревьювером
	GetReviewerAssignments(ctx context.Context, userID string) ([]PullRequestShort, error)
}
