package domain

import "time"

// PullRequestStatus - статус pull request
type PullRequestStatus string

const (
	PullRequestStatusOpen   PullRequestStatus = "OPEN"
	PullRequestStatusMerged PullRequestStatus = "MERGED"
)

// PullRequest - domain модель pull request
type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            PullRequestStatus
	AssignedReviewers []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

// PullRequestShort - краткая информация о PR для списков
type PullRequestShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   PullRequestStatus
}

// Team - domain модель команды
type Team struct {
	Name    string
	Members []TeamMember
}

// TeamMember - член команды
type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

// User - domain модель пользователя
type User struct {
	UserID   string
	Username string
	TeamName string
	IsActive bool
}

// Input/Output DTOs для методов сервиса

// CreatePullRequestInput - входные данные для создания PR
type CreatePullRequestInput struct {
	PullRequestID   string
	PullRequestName string
	AuthorID        string
}

// MergePullRequestInput - входные данные для merge PR
type MergePullRequestInput struct {
	PullRequestID string
}

// ReassignPullRequestInput - входные данные для переназначения ревьювера
type ReassignPullRequestInput struct {
	PullRequestID string
	OldUserID     string
}

// ReassignPullRequestResult - результат переназначения ревьювера
type ReassignPullRequestResult struct {
	PullRequest PullRequest
	ReplacedBy  string
}

// DeactivateTeamInput - входные данные для массовой деактивации команды
type DeactivateTeamInput struct {
	TeamName string
}

// DeactivateTeamResult - результат массовой деактивации команды
type DeactivateTeamResult struct {
	TeamName             string
	DeactivatedUserCount int
}

// ReassignInactiveInput - входные данные для переназначения неактивных ревьюверов PR
type ReassignInactiveInput struct {
	PullRequestID string
}

// ReassignInactiveResult - результат переназначения неактивных ревьюверов
type ReassignInactiveResult struct {
	PullRequestID       string
	ReassignmentDetails []ReviewerReassignment
}

// ReviewerReassignment - детали переназначения одного ревьювера
type ReviewerReassignment struct {
	OldReviewerID string
	NewReviewerID string // пустая строка если не удалось найти замену
	WasRemoved    bool   // true если ревьювер был просто удален без замены
}
