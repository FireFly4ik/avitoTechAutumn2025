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

// Create создаёт новую команду вместе с пользователями (batch upsert)
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

	// Batch upsert участников команды
	if len(users) > 0 {
		dbUsers := make([]User, len(users))
		for i, u := range users {
			dbUsers[i] = User{
				UserID:   u.UserID,
				Username: u.Username,
				TeamName: team.Name,
				IsActive: boolPtr(u.IsActive),
			}
		}

		if err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"username", "team_name", "is_active"}),
			}).
			Create(&dbUsers).Error; err != nil {
			return err
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
			IsActive: derefBool(member.IsActive),
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
			IsActive: derefBool(member.IsActive),
		}
	}

	return &domain.Team{
		Name:    dbTeam.TeamName,
		Members: members,
	}, nil
}

// DeactivateAllMembers деактивирует всех участников команды (batch update)
func (r *teamRepository) DeactivateAllMembers(ctx context.Context, teamName string) (int, error) {
	result := r.db.WithContext(ctx).
		Model(&User{}).
		Where("team_name = ? AND is_active = ?", teamName, true).
		Update("is_active", false)

	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}

// DeactivateMembers деактивирует указанных участников команды
func (r *teamRepository) DeactivateMembers(ctx context.Context, teamName string, userIDs []string) (int, error) {
	if len(userIDs) == 0 {
		return 0, nil
	}

	result := r.db.WithContext(ctx).
		Model(&User{}).
		Where("team_name = ? AND user_id IN ? AND is_active = ?", teamName, userIDs, true).
		Update("is_active", false)

	if result.Error != nil {
		return 0, result.Error
	}

	return int(result.RowsAffected), nil
}
