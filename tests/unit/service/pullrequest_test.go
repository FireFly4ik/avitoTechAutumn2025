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

func TestCreatePullRequest_Success(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)
	mockUserRepo := mocks.NewUserRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.CreatePullRequestInput{
		PullRequestID:   "pr-001",
		PullRequestName: "Add feature",
		AuthorID:        "user-1",
	}

	activeUsers := []domain.User{
		{UserID: "user-2", Username: "Alice", IsActive: true},
		{UserID: "user-3", Username: "Bob", IsActive: true},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)
			mockTx.On("UserRepo").Return(mockUserRepo)

			// PR не существует
			mockPRRepo.On("GetByID", mock.Anything, "pr-001").
				Return(nil, storage.ErrNotFound)

			// Возвращаем активных пользователей команды
			mockUserRepo.On("GetActiveTeamMembers", mock.Anything, "user-1").
				Return(activeUsers, nil)

			// Ожидаем создание PR
			mockPRRepo.On("Create", mock.Anything, mock.MatchedBy(func(pr *domain.PullRequest) bool {
				return pr.ID == "pr-001" &&
					pr.AuthorID == "user-1" &&
					pr.Status == domain.PullRequestStatusOpen &&
					len(pr.AssignedReviewers) <= 2
			})).Return(nil)

			// Ожидаем назначение ревьюверов (до 2)
			mockPRRepo.On("AssignReviewer", mock.Anything, "pr-001", mock.AnythingOfType("string")).
				Return(nil).Maybe()

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.CreatePullRequest(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "pr-001", result.ID)
	assert.Equal(t, domain.PullRequestStatusOpen, result.Status)
	assert.LessOrEqual(t, len(result.AssignedReviewers), 2)

	// Автор не должен быть ревьювером
	for _, reviewerID := range result.AssignedReviewers {
		assert.NotEqual(t, "user-1", reviewerID)
	}
}

func TestCreatePullRequest_NoActiveMembers(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)
	mockUserRepo := mocks.NewUserRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.CreatePullRequestInput{
		PullRequestID:   "pr-002",
		PullRequestName: "Solo PR",
		AuthorID:        "user-1",
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)
			mockTx.On("UserRepo").Return(mockUserRepo)

			mockPRRepo.On("GetByID", mock.Anything, "pr-002").
				Return(nil, storage.ErrNotFound)

			// Нет активных членов команды (кроме автора)
			mockUserRepo.On("GetActiveTeamMembers", mock.Anything, "user-1").
				Return([]domain.User{}, nil)

			// Ожидаем создание PR без ревьюверов
			mockPRRepo.On("Create", mock.Anything, mock.MatchedBy(func(pr *domain.PullRequest) bool {
				return pr.ID == "pr-002" && len(pr.AssignedReviewers) == 0
			})).Return(nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.CreatePullRequest(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.AssignedReviewers))
}

func TestCreatePullRequest_AlreadyExists(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.CreatePullRequestInput{
		PullRequestID:   "pr-existing",
		PullRequestName: "Duplicate",
		AuthorID:        "user-1",
	}

	existingPR := &domain.PullRequest{
		ID:       "pr-existing",
		AuthorID: "user-1",
		Status:   domain.PullRequestStatusOpen,
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			// PR уже существует
			mockPRRepo.On("GetByID", mock.Anything, "pr-existing").
				Return(existingPR, nil)

			fn(context.Background(), mockTx)
		}).Return(storage.ErrAlreadyExists)

	// Act
	result, err := svc.CreatePullRequest(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestMergePullRequest_Success(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.MergePullRequestInput{
		PullRequestID: "pr-001",
	}

	existingPR := &domain.PullRequest{
		ID:       "pr-001",
		AuthorID: "user-1",
		Status:   domain.PullRequestStatusOpen,
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			mockPRRepo.On("GetByID", mock.Anything, "pr-001").
				Return(existingPR, nil)

			mockPRRepo.On("Update", mock.Anything, mock.MatchedBy(func(pr *domain.PullRequest) bool {
				return pr.ID == "pr-001" &&
					pr.Status == domain.PullRequestStatusMerged &&
					pr.MergedAt != nil
			})).Return(nil)

			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.MergePullRequest(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.PullRequestStatusMerged, result.Status)
	assert.NotNil(t, result.MergedAt)
}

func TestMergePullRequest_Idempotent(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.MergePullRequestInput{
		PullRequestID: "pr-001",
	}

	// PR уже смержен
	alreadyMergedPR := &domain.PullRequest{
		ID:       "pr-001",
		AuthorID: "user-1",
		Status:   domain.PullRequestStatusMerged,
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			mockPRRepo.On("GetByID", mock.Anything, "pr-001").
				Return(alreadyMergedPR, nil)

			// Update НЕ должен вызываться
			fn(context.Background(), mockTx)
		}).Return(nil)

	// Act
	result, err := svc.MergePullRequest(context.Background(), input)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.PullRequestStatusMerged, result.Status)
}

func TestReassignPullRequest_AfterMerge_ShouldFail(t *testing.T) {
	// Arrange
	mockTxMgr := mocks.NewTxManager(t)
	mockTx := mocks.NewTx(t)
	mockPRRepo := mocks.NewPullRequestRepository(t)

	svc := service.New(mockTxMgr)

	input := &domain.ReassignPullRequestInput{
		PullRequestID: "pr-001",
		OldUserID:     "user-2",
	}

	mergedPR := &domain.PullRequest{
		ID:                "pr-001",
		AuthorID:          "user-1",
		Status:            domain.PullRequestStatusMerged,
		AssignedReviewers: []string{"user-2"},
	}

	// Setup expectations
	mockTxMgr.On("Do", mock.Anything, mock.AnythingOfType("func(context.Context, storage.Tx) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context, storage.Tx) error)

			mockTx.On("PullRequestRepo").Return(mockPRRepo)

			mockPRRepo.On("GetByID", mock.Anything, "pr-001").
				Return(mergedPR, nil)

			fn(context.Background(), mockTx)
		}).Return(domain.ErrReassignOnMerged)

	// Act
	result, err := svc.ReassignPullRequest(context.Background(), input)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, domain.ErrReassignOnMerged)
}
