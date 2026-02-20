package handlers

import (
	"avitoTechAutumn2025/internal/domain"
	"time"
)

// --- Типизированные response-структуры ---

// PullRequestResponse — полный ответ по PR
type PullRequestResponse struct {
	PullRequestID     string     `json:"pull_request_id"`
	PullRequestName   string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            string     `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         *time.Time `json:"created_at"`
	MergedAt          *time.Time `json:"merged_at"`
}

// PullRequestShortResponse — краткий ответ по PR
type PullRequestShortResponse struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

// TeamResponse — ответ по команде
type TeamResponse struct {
	TeamName string               `json:"team_name"`
	Members  []TeamMemberResponse `json:"members"`
}

// TeamMemberResponse — ответ по участнику команды
type TeamMemberResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

// UserResponse — ответ по пользователю
type UserResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

// ReassignmentDetailResponse — детали переназначения ревьювера
type ReassignmentDetailResponse struct {
	OldReviewerID string `json:"old_reviewer_id"`
	NewReviewerID string `json:"new_reviewer_id"`
	WasRemoved    bool   `json:"was_removed"`
}

// --- Маппер-функции ---

func mapPullRequestToAPI(pr *domain.PullRequest) PullRequestResponse {
	return PullRequestResponse{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}

func mapPullRequestShortToAPI(pr domain.PullRequestShort) PullRequestShortResponse {
	return PullRequestShortResponse{
		PullRequestID:   pr.ID,
		PullRequestName: pr.Name,
		AuthorID:        pr.AuthorID,
		Status:          string(pr.Status),
	}
}

func mapTeamToAPI(team *domain.Team) TeamResponse {
	members := make([]TeamMemberResponse, len(team.Members))
	for i, m := range team.Members {
		members[i] = TeamMemberResponse{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}
	return TeamResponse{
		TeamName: team.Name,
		Members:  members,
	}
}

func mapUserToAPI(user *domain.User) UserResponse {
	return UserResponse{
		UserID:   user.UserID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func mapReassignmentDetailsToAPI(details []domain.ReviewerReassignment) []ReassignmentDetailResponse {
	result := make([]ReassignmentDetailResponse, len(details))
	for i, d := range details {
		result[i] = ReassignmentDetailResponse{
			OldReviewerID: d.OldReviewerID,
			NewReviewerID: d.NewReviewerID,
			WasRemoved:    d.WasRemoved,
		}
	}
	return result
}
