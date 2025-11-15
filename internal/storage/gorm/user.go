package gorm

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/storage"
)

type userRepository struct {
	db *gorm.DB
}

// NewUserRepository создаёт новый репозиторий пользователей
func NewUserRepository(db *gorm.DB) storage.UserRepository {
	return &userRepository{db: db}
}

// GetByID получает пользователя по ID
func (r *userRepository) GetByID(ctx context.Context, userID string) (*domain.User, error) {
	var dbUser User
	result := r.db.WithContext(ctx).First(&dbUser, "user_id = ?", userID)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, storage.ErrNotFound
		}
		return nil, result.Error
	}

	return &domain.User{
		UserID:   dbUser.UserID,
		Username: dbUser.Username,
		TeamName: dbUser.TeamName,
		IsActive: dbUser.IsActive,
	}, nil
}

// Update обновляет пользователя (включая is_active)
func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	result := r.db.WithContext(ctx).
		Model(&User{}).
		Where("user_id = ?", user.UserID).
		Updates(map[string]interface{}{
			"is_active": user.IsActive,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return storage.ErrNotFound
	}

	// Перечитываем актуальные данные пользователя из БД
	var dbUser User
	if err := r.db.WithContext(ctx).First(&dbUser, "user_id = ?", user.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return storage.ErrNotFound
		}
		return err
	}

	// Обновляем domain модель актуальными данными из БД
	user.Username = dbUser.Username
	user.TeamName = dbUser.TeamName
	user.IsActive = dbUser.IsActive

	return nil
}

// GetActiveTeamMembers получает активных членов команды по ID пользователя
func (r *userRepository) GetActiveTeamMembers(ctx context.Context, userID string) ([]domain.User, error) {
	// Сначала получаем пользователя чтобы узнать его команду
	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Получаем всех активных членов команды кроме самого пользователя
	var dbUsers []User
	result := r.db.WithContext(ctx).
		Where("team_name = ? AND is_active = ? AND user_id != ?", user.TeamName, true, userID).
		Find(&dbUsers)

	if result.Error != nil {
		return nil, result.Error
	}

	users := make([]domain.User, len(dbUsers))
	for i, dbUser := range dbUsers {
		users[i] = domain.User{
			UserID:   dbUser.UserID,
			Username: dbUser.Username,
			TeamName: dbUser.TeamName,
			IsActive: dbUser.IsActive,
		}
	}

	return users, nil
}

// CreateBatch создаёт несколько пользователей
func (r *userRepository) CreateBatch(ctx context.Context, users []domain.User) error {
	for _, user := range users {
		if err := r.db.WithContext(ctx).Exec(
			"INSERT INTO users (user_id, username, team_name, is_active) VALUES (?, ?, ?, ?)",
			user.UserID, user.Username, user.TeamName, user.IsActive,
		).Error; err != nil {
			return err
		}
	}

	return nil
}
