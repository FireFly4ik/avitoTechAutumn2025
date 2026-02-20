package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"avitoTechAutumn2025/internal/api"
	"avitoTechAutumn2025/internal/api/middleware"
	"avitoTechAutumn2025/internal/domain"
)

// getRequestID извлекает request_id из gin.Context
func getRequestID(c *gin.Context) string {
	if rid, exists := c.Get(middleware.RequestIDKey); exists {
		if s, ok := rid.(string); ok {
			return s
		}
	}
	return ""
}

// handleDomainError обрабатывает domain ошибки и возвращает правильный HTTP response
func handleDomainError(c *gin.Context, err error) {
	requestID := getRequestID(c)

	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		c.JSON(domainErr.Status, api.ErrorResponse{
			Error: api.Error{
				Code:      string(domainErr.Code),
				Message:   domainErr.Message,
				RequestID: requestID,
			},
		})
		return
	}

	// Fallback на internal error
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error: api.Error{
			Code:      api.ErrCodeInternalError,
			Message:   "internal server error",
			RequestID: requestID,
		},
	})
}

// handleValidationError возвращает 400 Bad Request с request_id
func handleValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, api.ErrorResponse{
		Error: api.Error{
			Code:      api.ErrCodeInvalidRequest,
			Message:   message,
			RequestID: getRequestID(c),
		},
	})
}
