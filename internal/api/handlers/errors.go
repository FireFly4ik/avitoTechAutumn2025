package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"avitoTechAutumn2025/internal/api"
	"avitoTechAutumn2025/internal/domain"
)

// handleDomainError обрабатывает domain ошибки и возвращает правильный HTTP response
func handleDomainError(c *gin.Context, err error) {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		c.JSON(domainErr.Status, api.ErrorResponse{
			Error: api.Error{
				Code:    string(domainErr.Code),
				Message: domainErr.Message,
			},
		})
		return
	}

	// Fallback на internal error
	c.JSON(http.StatusInternalServerError, api.ErrorResponse{
		Error: api.Error{
			Code:    api.ErrCodeInternalError,
			Message: "internal server error",
		},
	})
}
