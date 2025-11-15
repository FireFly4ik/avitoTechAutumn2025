package service_test

import (
	"context"
	"testing"

	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/mocks"
	"avitoTechAutumn2025/internal/service"
	"avitoTechAutumn2025/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateTeam_Success(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			// Ожидаем создание команды (проверка дубликата на уровне БД)
			mockTeamRepo.On("Create", mock.Anything, team, mock.MatchedBy(func(users []domain.User) bool {
				return len(users) == 2 &&
					users[0].UserID == "u1" &&
					users[1].UserID == "u2"
			})).Return(nil)

			fn(context.Background(), mockTx)
		}).Return(nil) // Act
	result, err := svc.CreateTeam(context.Background(), team)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", result.Name)
	assert.Equal(t, 2, len(result.Members))
}

func TestCreateTeam_AlreadyExists(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
		},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			// Create возвращает ошибку дубликата (constraint на уровне БД)
			mockTeamRepo.On("Create", mock.Anything, team, mock.Anything).
				Return(storage.ErrAlreadyExists)

			fn(context.Background(), mockTx)
		}).Return(storage.ErrAlreadyExists) // Act
	result, err := svc.CreateTeam(context.Background(), team)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrTeamExists)
}

func TestGetTeam_Success(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	expectedTeam := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: false},
		},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			mockTeamRepo.On("GetByName", mock.Anything, "backend").
				Return(expectedTeam, nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.GetTeam(context.Background(), "backend")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", result.Name)
	assert.Equal(t, 2, len(result.Members))

	// Проверяем что неактивные пользователи тоже возвращаются
	assert.True(t, result.Members[0].IsActive)
	assert.False(t, result.Members[1].IsActive)
}

func TestGetTeam_NotFound(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			mockTeamRepo.On("GetByName", mock.Anything, "nonexistent").
				Return(nil, storage.ErrNotFound)

			fn(context.Background(), mockTx)
		}).Return(storage.ErrNotFound)

	// Act
	result, err := svc.GetTeam(context.Background(), "nonexistent")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrResourceNotFound)
}

func TestDeactivateTeamMembers_Success(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.DeactivateTeamInput{
		TeamName: "backend",
	}

	existingTeam := &domain.Team{
		Name:    "backend",
		Members: []domain.TeamMember{},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			// Проверяем существование команды
			mockTeamRepo.On("GetByName", mock.Anything, "backend").
				Return(existingTeam, nil)

			// Деактивируем 5 участников
			mockTeamRepo.On("DeactivateAllMembers", mock.Anything, "backend").
				Return(5, nil)

			fn(context.Background(), mockTx)
		}).Return(nil) // Act
	result, err := svc.DeactivateTeamMembers(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", result.TeamName)
	assert.Equal(t, 5, result.DeactivatedUserCount)
}
