package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"

	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/storage"
)

// Service реализует domain.AssignmentService используя storage.TxManager
type Service struct {
	txmgr storage.TxManager
}

// Проверка что Service реализует интерфейс domain.AssignmentService
var _ domain.AssignmentService = (*Service)(nil)

// New создаёт новый Service с TxManager
func New(txmgr storage.TxManager) *Service {
	return &Service{
		txmgr: txmgr,
	}
}

// formatError преобразует ошибки storage слоя в доменные ошибки с правильными HTTP кодами.
// Оригинальная ошибка сохраняется через Wrap для целей дебага.
func (s *Service) formatError(ctx context.Context, op string, err error) error {
	switch {
	case errors.Is(err, storage.ErrNotFound):
		return fmt.Errorf("%s: %w", op, domain.ErrResourceNotFound)
	case errors.Is(err, storage.ErrAlreadyExists):
		switch op {
		case "service.CreatePullRequest":
			return fmt.Errorf("%s: %w", op, domain.ErrPRExists)
		case "service.CreateTeam":
			return fmt.Errorf("%s: %w", op, domain.ErrTeamExists)
		}
		return fmt.Errorf("%s: %w", op, domain.ErrInternal)
	case errors.Is(err, storage.ErrConflict):
		return fmt.Errorf("%s: %w", op, domain.ErrReassignOnMerged)
	case domain.IsDomainError(err):
		return err
	case errors.Is(err, ctx.Err()):
		return ctx.Err()
	default:
		log.Error().Err(err).Str("operation", op).Msg("operation failed")
		return fmt.Errorf("%s: %w", op, domain.ErrInternal)
	}
}
