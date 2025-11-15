package handlers

import (
	"avitoTechAutumn2025/internal/api"
	"avitoTechAutumn2025/internal/api/middleware"
	"avitoTechAutumn2025/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

// AddTeam обрабатывает создание команды с участниками (upsert пользователей)
func (h *Handler) AddTeam(c *gin.Context) {
	var req struct {
		TeamName string `json:"team_name" binding:"required"`
		Members  []struct {
			UserID   string `json:"user_id" binding:"required"`
			Username string `json:"username" binding:"required"`
			IsActive bool   `json:"is_active"`
		} `json:"members" binding:"required"`
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
		Str("team_name", req.TeamName).
		Msg("creating team")

	members := make([]domain.TeamMember, len(req.Members))
	for i, m := range req.Members {
		members[i] = domain.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		}
	}

	team := &domain.Team{
		Name:    req.TeamName,
		Members: members,
	}

	createdTeam, err := h.service.CreateTeam(c.Request.Context(), team)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("team_name", createdTeam.Name).
		Msg("successfully created team")

	c.JSON(http.StatusCreated, mapTeamToAPI(createdTeam))
}

// GetTeam обрабатывает получение информации о команде по имени
func (h *Handler) GetTeam(c *gin.Context) {
	teamName := c.Query("team_name")
	if teamName == "" {
		log.Warn().
			Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
			Str("layer", "handler").
			Msg("missing team_name parameter")

		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Error: api.Error{
				Code:    api.ErrCodeInvalidRequest,
				Message: "team_name parameter is required",
			},
		})
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("team_name", teamName).
		Msg("getting team information")

	team, err := h.service.GetTeam(c.Request.Context(), teamName)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("team_name", team.Name).
		Int("user_count", len(team.Members)).
		Msg("successfully retrieved team information")

	c.JSON(http.StatusOK, mapTeamToAPI(team))
}

// DeactivateTeam обрабатывает массовую деактивацию всех участников команды
func (h *Handler) DeactivateTeam(c *gin.Context) {
	var req struct {
		TeamName string `json:"team_name" binding:"required"`
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
		Str("team_name", req.TeamName).
		Msg("deactivating team members")

	input := &domain.DeactivateTeamInput{
		TeamName: req.TeamName,
	}

	result, err := h.service.DeactivateTeamMembers(c.Request.Context(), input)
	if err != nil {
		handleDomainError(c, err)
		return
	}

	log.Info().
		Str("request_id", c.MustGet(middleware.RequestIDKey).(string)).
		Str("layer", "handler").
		Str("team_name", result.TeamName).
		Int("deactivated_count", result.DeactivatedUserCount).
		Msg("successfully deactivated team members")

	c.JSON(http.StatusOK, gin.H{
		"team_name":              result.TeamName,
		"deactivated_user_count": result.DeactivatedUserCount,
	})
}
