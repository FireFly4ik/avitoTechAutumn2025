package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"avitoTechAutumn2025/internal/api/handlers"
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupTestRouter(mockService *mocks.AssignmentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	_ = os.Setenv("ADMIN_TOKEN", "test-admin-token")
	handler := handlers.NewHandler(mockService)
	return handler.InitRoutes()
}

func TestCreatePullRequestHandler_Success(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id":   "pr-001",
		"pull_request_name": "Add feature",
		"author_id":         "user-1",
	}

	expectedPR := &domain.PullRequest{
		ID:                "pr-001",
		Name:              "Add feature",
		AuthorID:          "user-1",
		Status:            domain.PullRequestStatusOpen,
		AssignedReviewers: []string{"user-2", "user-3"},
	}

	mockService.On("CreatePullRequest", mock.Anything, mock.MatchedBy(func(input *domain.CreatePullRequestInput) bool {
		return input.PullRequestID == "pr-001" &&
			input.PullRequestName == "Add feature" &&
			input.AuthorID == "user-1"
	})).Return(expectedPR, nil)

	// Act
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	assert.Equal(t, "pr-001", pr["pull_request_id"])
	assert.Equal(t, "OPEN", pr["status"])

	mockService.AssertExpectations(t)
}

func TestCreatePullRequestHandler_InvalidRequest(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	// Невалидный запрос (пустой pull_request_id)
	requestBody := map[string]interface{}{
		"pull_request_id":   "",
		"pull_request_name": "Add feature",
		"author_id":         "user-1",
	}

	// Act
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "INVALID_REQUEST", errorObj["code"])
}

func TestCreatePullRequestHandler_DuplicatePR(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id":   "pr-001",
		"pull_request_name": "Duplicate",
		"author_id":         "user-1",
	}

	mockService.On("CreatePullRequest", mock.Anything, mock.Anything).
		Return(nil, domain.ErrPRExists)

	// Act
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "PR_EXISTS", errorObj["code"])

	mockService.AssertExpectations(t)
}

func TestMergePullRequestHandler_Success(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-001",
	}

	expectedPR := &domain.PullRequest{
		ID:       "pr-001",
		AuthorID: "user-1",
		Status:   domain.PullRequestStatusMerged,
	}

	mockService.On("MergePullRequest", mock.Anything, mock.MatchedBy(func(input *domain.MergePullRequestInput) bool {
		return input.PullRequestID == "pr-001"
	})).Return(expectedPR, nil)

	// Act
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	assert.Equal(t, "pr-001", pr["pull_request_id"])
	assert.Equal(t, "MERGED", pr["status"])

	mockService.AssertExpectations(t)
}

func TestAddTeamHandler_Success(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice", "is_active": true},
			{"user_id": "u2", "username": "Bob", "is_active": true},
		},
	}

	expectedTeam := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}

	mockService.On("CreateTeam", mock.Anything, mock.MatchedBy(func(team *domain.Team) bool {
		return team.Name == "backend" && len(team.Members) == 2
	})).Return(expectedTeam, nil)

	// Act
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "backend", response["team_name"])
	members := response["members"].([]interface{})
	assert.Equal(t, 2, len(members))

	mockService.AssertExpectations(t)
}

func TestGetTeamHandler_Success(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	expectedTeam := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: false},
		},
	}

	mockService.On("GetTeam", mock.Anything, "backend").
		Return(expectedTeam, nil)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=backend", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "backend", response["team_name"])
	members := response["members"].([]interface{})
	assert.Equal(t, 2, len(members))

	mockService.AssertExpectations(t)
}

func TestGetTeamHandler_NotFound(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	mockService.On("GetTeam", mock.Anything, "nonexistent").
		Return(nil, domain.ErrResourceNotFound)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errorObj["code"])

	mockService.AssertExpectations(t)
}

func TestSetUserIsActiveHandler_Success(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"user_id":   "u1",
		"is_active": false,
	}

	expectedUser := &domain.User{
		UserID:   "u1",
		Username: "Alice",
		IsActive: false,
	}

	mockService.On("SetUserIsActive", mock.Anything, "u1", false).
		Return(expectedUser, nil)

	// Act
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "u1", response["user_id"])
	assert.False(t, response["is_active"].(bool))

	mockService.AssertExpectations(t)
}

func TestGetReviewHandler_Success(t *testing.T) {
	// Arrange
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

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
			Status:   domain.PullRequestStatusOpen,
		},
	}

	mockService.On("GetReviewerAssignments", mock.Anything, "reviewer-1").
		Return(expectedPRs, nil)

	// Act
	req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id=reviewer-1", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "reviewer-1", response["user_id"])
	prs := response["pull_requests"].([]interface{})
	assert.Equal(t, 2, len(prs))

	mockService.AssertExpectations(t)
}
