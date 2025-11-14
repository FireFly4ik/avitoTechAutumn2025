package server

import (
	"avitoTechAutumn2025/internal/api/handlers"
	"avitoTechAutumn2025/internal/config"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"net/http"
)

type Server struct {
	envConfig *config.Config
	handler   *handlers.Handler
	server    *http.Server
}

func NewServer(envConfig *config.Config, handler *handlers.Handler) *Server {
	return &Server{
		envConfig: envConfig,
		handler:   handler,
	}
}

func (s *Server) Run() {
	if s.envConfig.ProductionType != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	s.server = &http.Server{
		Handler: s.handler.InitRoutes(),
		Addr:    ":" + s.envConfig.Port,
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("failed to start HTTP server")
	}
}

func (s *Server) Shutdown(ctx context.Context) {
	if err := s.server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("error during server shutdown")
		return
	}
	log.Info().Msg("Server shutdown gracefully")
}
