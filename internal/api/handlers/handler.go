package handlers

import (
	"avitoTechAutumn2025/internal/api/middleware"
	"avitoTechAutumn2025/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	TeamPathRoute   = "/team"
	AddTeamRoute    = "/add"
	GetTeamRoute    = "/get"
	DeactivateRoute = "/deactivate"

	UserPathRoute    = "/users"
	SetIsActiveRoute = "/setIsActive"
	GetReviewRoute   = "/getReview"

	PullRequestPathRoute           = "/pullRequest"
	CreatePullRequestRoute         = "/create"
	MergePullRequestRoute          = "/merge"
	ReassignPullRequestRoute       = "/reassign"
	ReassignInactiveReviewersRoute = "/reassignInactive"
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
		middleware.MetricsMiddleware(),
		middleware.RecoveryMiddleware(),
		middleware.CORSMiddleware(),
		middleware.AuthMiddleware(),
	)

	// Prometheus metrics endpoint (без аутентификации для scraping)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	teamGroup := r.Group(TeamPathRoute)
	{
		teamGroup.POST(AddTeamRoute, h.AddTeam)
		teamGroup.GET(GetTeamRoute, middleware.RequireUser(), h.GetTeam)
		teamGroup.POST(DeactivateRoute, middleware.RequireAdmin(), h.DeactivateTeam)
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
		prGroup.POST(ReassignInactiveReviewersRoute, middleware.RequireAdmin(), h.ReassignInactiveReviewers)
	}

	return r
}
