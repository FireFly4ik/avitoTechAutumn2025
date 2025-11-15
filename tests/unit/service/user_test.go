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

func TestSetUserIsActive_Activate(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockUserRepo := mocks.NewUserRepository(t)

	svc := service.New(mockTxMgr)

	existingUser := &domain.User{
		UserID:   "user-1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: false,
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("UserRepo").Return(mockUserRepo)

			// Пользователь существует
			mockUserRepo.On("GetByID", mock.Anything, "user-1").
				Return(existingUser, nil)

			// Обновляем пользователя
			mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
				return user.UserID == "user-1" && user.IsActive == true
			})).Return(nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.SetUserIsActive(context.Background(), "user-1", true)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user-1", result.UserID)
	assert.True(t, result.IsActive)
}

func TestSetUserIsActive_Deactivate(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockUserRepo := mocks.NewUserRepository(t)

	svc := service.New(mockTxMgr)

	existingUser := &domain.User{
		UserID:   "user-1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: true,
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("UserRepo").Return(mockUserRepo)

			mockUserRepo.On("GetByID", mock.Anything, "user-1").
				Return(existingUser, nil)

			mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
				return user.UserID == "user-1" && user.IsActive == false
			})).Return(nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.SetUserIsActive(context.Background(), "user-1", false)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user-1", result.UserID)
	assert.False(t, result.IsActive)
}

func TestSetUserIsActive_UserNotFound(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockUserRepo := mocks.NewUserRepository(t)

	svc := service.New(mockTxMgr)

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("UserRepo").Return(mockUserRepo)

			// Пользователь не найден
			mockUserRepo.On("GetByID", mock.Anything, "nonexistent").
				Return(nil, storage.ErrNotFound)

			fn(context.Background(), mockTx)
		}).Return(storage.ErrNotFound)

	// Act
	result, err := svc.SetUserIsActive(context.Background(), "nonexistent", true)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrResourceNotFound)
}

func TestSetUserIsActive_Idempotent(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockUserRepo := mocks.NewUserRepository(t)

	svc := service.New(mockTxMgr)

	// Пользователь уже активен
	existingUser := &domain.User{
		UserID:   "user-1",
		Username: "Alice",
		TeamName: "backend",
		IsActive: true,
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("UserRepo").Return(mockUserRepo)

			mockUserRepo.On("GetByID", mock.Anything, "user-1").
				Return(existingUser, nil)

			// Update всё равно вызывается (идемпотентность на уровне данных)
			mockUserRepo.On("Update", mock.Anything, mock.MatchedBy(func(user *domain.User) bool {
				return user.UserID == "user-1" && user.IsActive == true
			})).Return(nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act - Активируем уже активного пользователя
	result, err := svc.SetUserIsActive(context.Background(), "user-1", true)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsActive)
}

func TestGetReviewerAssignments_Success(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	expectedPRs := []domain.PullRequestShort{
		{
			ID:       "pr-001",
			Name:     "Feature A",
			AuthorID: "author-1",
			Status:   domain.PullRequestStatusOpen,
		},
		{
			ID:       "pr-002",
			Name:     "Feature B",
			AuthorID: "author-2",
			Status:   domain.PullRequestStatusMerged,
		},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			mockPRRepo.On("GetPRsReviewedByUser", mock.Anything, "reviewer-1").
				Return(expectedPRs, nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.GetReviewerAssignments(context.Background(), "reviewer-1")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "pr-001", result[0].ID)
	assert.Equal(t, "pr-002", result[1].ID)
}

func TestGetReviewerAssignments_EmptyList(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			// Пользователь не назначен ни на один PR
			mockPRRepo.On("GetPRsReviewedByUser", mock.Anything, "user-without-prs").
				Return([]domain.PullRequestShort{}, nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.GetReviewerAssignments(context.Background(), "user-without-prs")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result))
}

func TestGetReviewerAssignments_IncludesMergedPRs(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	expectedPRs := []domain.PullRequestShort{
		{
			ID:       "pr-open",
			Name:     "Open PR",
			AuthorID: "author-1",
			Status:   domain.PullRequestStatusOpen,
		},
		{
			ID:       "pr-merged",
			Name:     "Merged PR",
			AuthorID: "author-2",
			Status:   domain.PullRequestStatusMerged,
		},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			mockPRRepo.On("GetPRsReviewedByUser", mock.Anything, "reviewer-1").
				Return(expectedPRs, nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.GetReviewerAssignments(context.Background(), "reviewer-1")

	// Assert - Важно: должны возвращаться и OPEN и MERGED PR
	require.NoError(t, err)
	assert.Equal(t, 2, len(result))

	// Проверяем что есть оба статуса
	statuses := make(map[domain.PullRequestStatus]bool)
	for _, pr := range result {
		statuses[pr.Status] = true
	}
	assert.True(t, statuses[domain.PullRequestStatusOpen])
	assert.True(t, statuses[domain.PullRequestStatusMerged])
}

func TestDeactivateTeamMembers_WithOpenPRs(t *testing.T) {
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

			// Деактивируем 3 участников
			mockTeamRepo.On("DeactivateAllMembers", mock.Anything, "backend").
				Return(3, nil)

			fn(context.Background(), mockTx)
		}).Return(nil) // Act
	result, err := svc.DeactivateTeamMembers(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "backend", result.TeamName)
	assert.Equal(t, 3, result.DeactivatedUserCount)
}

func TestDeactivateTeamMembers_TeamNotFound(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.DeactivateTeamInput{
		TeamName: "nonexistent",
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			// GetByName возвращает ошибку - команда не найдена
			mockTeamRepo.On("GetByName", mock.Anything, "nonexistent").
				Return(nil, storage.ErrNotFound)

			fn(context.Background(), mockTx)
		}).Return(storage.ErrNotFound) // Act
	result, err := svc.DeactivateTeamMembers(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestDeactivateTeamMembers_AlreadyAllInactive(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockTeamRepo := mocks.NewTeamRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.DeactivateTeamInput{
		TeamName: "inactive-team",
	}

	existingTeam := &domain.Team{
		Name:    "inactive-team",
		Members: []domain.TeamMember{},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("TeamRepo").Return(mockTeamRepo)

			// Проверяем существование команды
			mockTeamRepo.On("GetByName", mock.Anything, "inactive-team").
				Return(existingTeam, nil)

			// Все уже неактивны - деактивировано 0
			mockTeamRepo.On("DeactivateAllMembers", mock.Anything, "inactive-team").
				Return(0, nil)

			fn(context.Background(), mockTx)
		}).Return(nil) // Act
	result, err := svc.DeactivateTeamMembers(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "inactive-team", result.TeamName)
	assert.Equal(t, 0, result.DeactivatedUserCount)
}
