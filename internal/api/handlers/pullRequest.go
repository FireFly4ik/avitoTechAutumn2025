package handlers

import (
	"avitoTechAutumn2025/internal/api"
	"avitoTechAutumn2025/internal/api/middleware"
	"avitoTechAutumn2025/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

// CreatePullRequest обрабатывает создание PR с автоматическим назначением ревьюверов
func (h *Handler) CreatePullRequest(c *gin.Context) {
	var req struct {
		PullRequestID   string `json:"pull_request_id" binding:"required"`
		PullRequestName string `json:"pull_request_name" binding:"required"`
		AuthorID        string `json:"author_id" binding:"required"`
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
		Str("pull_request_id", req.PullRequestID).
		Str("author_id", req.AuthorID).
		Msg("creating pull request")

	input := &domain.CreatePullRequestInput{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	}

	pr, err := h.service.CreatePullRequest(c.Request.Context(), input)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("pull_request_id", pr.ID).
		Str("author_id", pr.AuthorID).
		Int("reviewers_assigned", len(pr.AssignedReviewers)).
		Msg("successfully created pull request")

	c.JSON(http.StatusCreated, map[string]interface{}{
		"pr": mapPullRequestToAPI(pr),
	})
}

// MergePullRequest обрабатывает merge PR (идемпотентная операция)
func (h *Handler) MergePullRequest(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
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
		Str("pull_request_id", req.PullRequestID).
		Msg("merging pull request")

	input := &domain.MergePullRequestInput{
		PullRequestID: req.PullRequestID,
	}

	pr, err := h.service.MergePullRequest(c.Request.Context(), input)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("pull_request_id", pr.ID).
		Str("status", string(pr.Status)).
		Msg("successfully merged pull request")

	c.JSON(http.StatusOK, map[string]interface{}{
		"pr": mapPullRequestToAPI(pr),
	})
}

// ReassignPullRequest обрабатывает переназначение ревьювера на другого члена команды
func (h *Handler) ReassignPullRequest(c *gin.Context) {
	var req struct {
		PullRequestID string `json:"pull_request_id" binding:"required"`
		OldUserID     string `json:"old_reviewer_id" binding:"required"`
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
		Str("pull_request_id", req.PullRequestID).
		Str("old_reviewer_id", req.OldUserID).
		Msg("reassigning pull request reviewer")

	input := &domain.ReassignPullRequestInput{
		PullRequestID: req.PullRequestID,
		OldUserID:     req.OldUserID,
	}

	result, err := h.service.ReassignPullRequest(c.Request.Context(), input)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("pull_request_id", result.PullRequest.ID).
		Str("old_reviewer_id", req.OldUserID).
		Str("new_reviewer_id", result.ReplacedBy).
		Msg("successfully reassigned pull request reviewer")

	c.JSON(http.StatusOK, map[string]interface{}{
		"pr":          mapPullRequestToAPI(&result.PullRequest),
		"replaced_by": result.ReplacedBy,
	})
}
