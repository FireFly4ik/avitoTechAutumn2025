package service

import (
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/storage"
	"context"
	"github.com/rs/zerolog/log"
)

// CreateTeam создаёт новую команду с участниками
func (s *Service) CreateTeam(outerCtx context.Context, team *domain.Team) (*domain.Team, error) {
	const op = "service.CreateTeam"
	requestID := logger.GetRequestID(outerCtx)

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("team_name", team.Name).
		Int("members_count", len(team.Members)).
		Msg("creating team")

	// Преобразуем TeamMember в User для передачи в репозиторий
	users := make([]domain.User, len(team.Members))
	for i, member := range team.Members {
		users[i] = domain.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: team.Name,
			IsActive: member.IsActive,
		}
	}

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		return tx.TeamRepo().Create(ctx, team, users)
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("team_name", team.Name).
		Int("members_count", len(team.Members)).
		Msg("successfully created team")

	return team, nil
}

// GetTeam возвращает информацию о команде по имени
func (s *Service) GetTeam(outerCtx context.Context, teamName string) (*domain.Team, error) {
	const op = "service.GetTeam"
	requestID := logger.GetRequestID(outerCtx)
	var team *domain.Team

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("team_name", teamName).
		Msg("fetching team")

	err := s.txmgr.Do(outerCtx, func(ctx context.Context, tx storage.Tx) error {
		t, err := tx.TeamRepo().GetByName(ctx, teamName)
		if err != nil {
			return err
		}
		team = t
		return nil
	})

	if err != nil {
		return nil, s.formatError(outerCtx, op, err)
	}

	log.Info().
		Str("request_id", requestID).
		Str("layer", "service").
		Str("team_name", team.Name).
		Int("members_count", len(team.Members)).
		Msg("successfully fetched team")

	return team, nil
}
