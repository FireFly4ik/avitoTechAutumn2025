package api

const (
	ErrCodeTeamExists        = "TEAM_EXISTS"
	ErrCodePullRequestExists = "PR_EXISTS"
	ErrCodePullRequestMerged = "PR_MERGED"
	ErrCodeNotAssigned       = "NOT_ASSIGNED"
	ErrCodeNoCandidate       = "NO_CANDIDATE"
	ErrCodeNotFound          = "NOT_FOUND"

	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeInvalidRequest = "INVALID_REQUEST"
)

// Error represents a standardized error structure
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error Error `json:"error"`
}
