package gorm

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/storage"
)

type teamRepository struct {
	db *gorm.DB
}

// NewTeamRepository создаёт новый репозиторий команд
func NewTeamRepository(db *gorm.DB) storage.TeamRepository {
	return &teamRepository{db: db}
}

// Create создаёт новую команду вместе с пользователями
func (r *teamRepository) Create(ctx context.Context, team *domain.Team, users []domain.User) error {
	dbTeam := &Team{
		TeamName: team.Name,
	}

	// Создаём команду
	result := r.db.WithContext(ctx).Create(dbTeam)
	if result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == storage.UniqueViolation {
			return storage.ErrAlreadyExists
		}
		return result.Error
	}

	// Добавляем участников команды с upsert - обновляем при конфликте
	if len(users) > 0 {
		dbUsers := make([]User, len(users))
		for i, user := range users {
			dbUsers[i] = User{
				UserID:   user.UserID,
				Username: user.Username,
				TeamName: team.Name,
				IsActive: user.IsActive,
			}
		}

		// Используем ON CONFLICT для upsert
		result = r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"username", "is_active", "team_name"}),
		}).Create(&dbUsers)

		if result.Error != nil {
			return result.Error
		}
	}

	// Загружаем созданную команду с участниками
	if err := r.db.WithContext(ctx).Preload("Members").First(dbTeam, "team_name = ?", team.Name).Error; err != nil {
		return err
	}

	// Обновляем domain модель
	team.Members = make([]domain.TeamMember, len(dbTeam.Members))
	for i, member := range dbTeam.Members {
		team.Members[i] = domain.TeamMember{
			UserID:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		}
	}

	return nil
}

// GetByName получает команду по имени
func (r *teamRepository) GetByName(ctx context.Context, teamName string) (*domain.Team, error) {
	var dbTeam Team
	result := r.db.WithContext(ctx).
		Preload("Members").
		First(&dbTeam, "team_name = ?", teamName)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, result.Error
	}

	members := make([]domain.TeamMember, len(dbTeam.Members))
	for i, member := range dbTeam.Members {
		members[i] = domain.TeamMember{
			UserID:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		}
	}

	return &domain.Team{
		Name:    dbTeam.TeamName,
		Members: members,
	}, nil
}
