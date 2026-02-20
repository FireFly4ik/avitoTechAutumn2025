package api

// HTTP-слоевые коды ошибок.
// Доменные коды ошибок (TEAM_EXISTS, PR_EXISTS, NOT_FOUND и т.д.) определены
// в domain/errors.go и маппятся автоматически через handleDomainError.
const (
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeInvalidRequest = "INVALID_REQUEST"
)

// Error represents a standardized error structure
type Error struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Error Error `json:"error"`
}
