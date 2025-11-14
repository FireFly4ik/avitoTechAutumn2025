package handlers

import (
	"avitoTechAutumn2025/internal/api/middleware"
	"avitoTechAutumn2025/internal/domain"
	"github.com/gin-gonic/gin"
)

const (
	TeamPathRoute = "/team"
	AddTeamRoute  = "/add"
	GetTeamRoute  = "/get"

	UserPathRoute    = "/users"
	SetIsActiveRoute = "/setIsActive"
	GetReviewRoute   = "/getReview"

	PullRequestPathRoute     = "/pullRequest"
	CreatePullRequestRoute   = "/create"
	MergePullRequestRoute    = "/merge"
	ReassignPullRequestRoute = "/reassign"
)

type Handler struct {
	service domain.AssignmentService
}

func NewHandler(service domain.AssignmentService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) InitRoutes() *gin.Engine {
	r := gin.New()

	r.Use(
		middleware.LoggerMiddleware(),
		middleware.RecoveryMiddleware(),
		middleware.CORSMiddleware(),
		middleware.AuthMiddleware(),
	)

	teamGroup := r.Group(TeamPathRoute)
	{
		teamGroup.POST(AddTeamRoute, h.AddTeam)
		teamGroup.GET(GetTeamRoute, middleware.RequireUser(), h.GetTeam)
	}

	userGroup := r.Group(UserPathRoute)
	{
		userGroup.POST(SetIsActiveRoute, middleware.RequireAdmin(), h.SetIsActive)
		userGroup.GET(GetReviewRoute, middleware.RequireUser(), h.GetReview)
	}

	prGroup := r.Group(PullRequestPathRoute)
	{
		prGroup.POST(CreatePullRequestRoute, middleware.RequireAdmin(), h.CreatePullRequest)
		prGroup.POST(MergePullRequestRoute, middleware.RequireAdmin(), h.MergePullRequest)
		prGroup.POST(ReassignPullRequestRoute, middleware.RequireAdmin(), h.ReassignPullRequest)
	}

	return r
}
