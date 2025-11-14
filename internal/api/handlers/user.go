package handlers

import (
	"avitoTechAutumn2025/internal/api"
	"avitoTechAutumn2025/internal/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

// SetIsActive обрабатывает изменение статуса активности пользователя
func (h *Handler) SetIsActive(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id" binding:"required"`
		IsActive bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error().
			Err(err).
			Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
			Str("layer", "handler").
			Msg("failed to parse request")

		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error: api.Error{
				Code:    api.ErrCodeInvalidRequest,
				Message: "Failed to parse request: " + err.Error(),
			},
		})
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("user_id", req.UserID).
		Bool("is_active", req.IsActive).
		Msg("setting user active status")

	user, err := h.service.SetUserIsActive(c.Request.Context(), req.UserID, req.IsActive)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("user_id", req.UserID).
		Bool("is_active", user.IsActive).
		Msg("successfully updated user active status")

	c.JSON(http.StatusOK, mapUserToAPI(user))
}

// GetReview обрабатывает получение списка pull request-ов, в которых пользователь является ревьювером
func (h *Handler) GetReview(c *gin.Context) {
	userId := c.Query("user_id")
	if userId == "" {
		log.Warn().
			Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
			Str("layer", "handler").
			Msg("missing user_id parameter")

		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error: api.Error{
				Code:    api.ErrCodeInvalidRequest,
				Message: "user_id parameter is required",
			},
		})
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("user_id", userId).
		Msg("getting pull requests reviewed by user")

	prs, err := h.service.GetReviewerAssignments(c.Request.Context(), userId)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	prList := make([]map[string]interface{}, len(prs))
	for i, pr := range prs {
		prList[i] = mapPullRequestShortToAPI(pr)
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("user_id", userId).
		Int("pr_count", len(prs)).
		Msg("successfully retrieved reviewed PRs")

	c.JSON(http.StatusOK, gin.H{
		"user_id":       userId,
		"pull_requests": prList,
	})
}
