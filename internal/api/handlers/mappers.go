package handlers

import (
	"avitoTechAutumn2025/internal/domain"
)

// mapPullRequestToAPI конвертирует domain.PullRequest в API response
func mapPullRequestToAPI(pr *domain.PullRequest) map[string]interface{} {
	return map[string]interface{}{
		"pull_request_id":    pr.ID,
		"pull_request_name":  pr.Name,
		"author_id":          pr.AuthorID,
		"status":             string(pr.Status),
		"assigned_reviewers": pr.AssignedReviewers,
		"created_at":         pr.CreatedAt,
		"merged_at":          pr.MergedAt,
	}
}

// mapPullRequestShortToAPI конвертирует domain.PullRequestShort в API response
func mapPullRequestShortToAPI(pr domain.PullRequestShort) map[string]interface{} {
	return map[string]interface{}{
		"pull_request_id":   pr.ID,
		"pull_request_name": pr.Name,
		"author_id":         pr.AuthorID,
		"status":            string(pr.Status),
	}
}

// mapTeamToAPI конвертирует domain.Team в API response
func mapTeamToAPI(team *domain.Team) map[string]interface{} {
	members := make([]map[string]interface{}, len(team.Members))
	for i, m := range team.Members {
		members[i] = map[string]interface{}{
			"user_id":   m.UserID,
			"username":  m.Username,
			"is_active": m.IsActive,
		}
	}

	return map[string]interface{}{
		"team_name": team.Name,
		"members":   members,
	}
}

// mapUserToAPI конвертирует domain.User в API response
func mapUserToAPI(user *domain.User) map[string]interface{} {
	return map[string]interface{}{
		"user_id":   user.UserID,
		"username":  user.Username,
		"team_name": user.TeamName,
		"is_active": user.IsActive,
	}
}
