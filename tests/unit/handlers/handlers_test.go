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

// === Тесты DeactivateTeam handler ===

func TestDeactivateTeamHandler_Success(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"team_name": "backend",
	}

	expectedResult := &domain.DeactivateTeamResult{
		TeamName:             "backend",
		DeactivatedUserCount: 5,
		ReassignedCount:      2,
	}

	mockService.On("DeactivateTeamMembers", mock.Anything, mock.MatchedBy(func(input *domain.DeactivateTeamInput) bool {
		return input.TeamName == "backend" && len(input.UserIDs) == 0
	})).Return(expectedResult, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "backend", response["team_name"])
	assert.Equal(t, float64(5), response["deactivated_user_count"])
	assert.Equal(t, float64(2), response["reassigned_count"])

	mockService.AssertExpectations(t)
}

func TestDeactivateTeamHandler_WithUserIDs(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"team_name": "backend",
		"user_ids":  []string{"u1", "u2"},
	}

	expectedResult := &domain.DeactivateTeamResult{
		TeamName:             "backend",
		DeactivatedUserCount: 2,
		ReassignedCount:      0,
	}

	mockService.On("DeactivateTeamMembers", mock.Anything, mock.MatchedBy(func(input *domain.DeactivateTeamInput) bool {
		return input.TeamName == "backend" && len(input.UserIDs) == 2 &&
			input.UserIDs[0] == "u1" && input.UserIDs[1] == "u2"
	})).Return(expectedResult, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "backend", response["team_name"])
	assert.Equal(t, float64(2), response["deactivated_user_count"])

	mockService.AssertExpectations(t)
}

func TestDeactivateTeamHandler_TeamNotFound(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"team_name": "nonexistent",
	}

	mockService.On("DeactivateTeamMembers", mock.Anything, mock.Anything).
		Return(nil, domain.ErrResourceNotFound)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestDeactivateTeamHandler_MissingTeamName(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Тесты ReassignPullRequest handler ===

func TestReassignPullRequestHandler_Success(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-001",
		"old_reviewer_id": "user-2",
	}

	expectedResult := &domain.ReassignPullRequestResult{
		PullRequest: domain.PullRequest{
			ID:                "pr-001",
			AuthorID:          "user-1",
			Status:            domain.PullRequestStatusOpen,
			AssignedReviewers: []string{"user-3", "user-4"},
		},
		ReplacedBy: "user-4",
	}

	mockService.On("ReassignPullRequest", mock.Anything, mock.MatchedBy(func(input *domain.ReassignPullRequestInput) bool {
		return input.PullRequestID == "pr-001" && input.OldUserID == "user-2"
	})).Return(expectedResult, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "user-4", response["replaced_by"])
	pr := response["pr"].(map[string]interface{})
	assert.Equal(t, "pr-001", pr["pull_request_id"])

	mockService.AssertExpectations(t)
}

func TestReassignPullRequestHandler_MergedPR(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-001",
		"old_reviewer_id": "user-2",
	}

	mockService.On("ReassignPullRequest", mock.Anything, mock.Anything).
		Return(nil, domain.ErrReassignOnMerged)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "PR_MERGED", errorObj["code"])

	mockService.AssertExpectations(t)
}

func TestReassignPullRequestHandler_NotAssigned(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-001",
		"old_reviewer_id": "user-99",
	}

	mockService.On("ReassignPullRequest", mock.Anything, mock.Anything).
		Return(nil, domain.ErrReviewerMissing)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestReassignPullRequestHandler_InvalidRequest(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	// Отсутствует pull_request_id
	requestBody := map[string]interface{}{
		"old_reviewer_id": "user-2",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// === Тесты ReassignInactiveReviewers handler ===

func TestReassignInactiveHandler_Success(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-001",
	}

	expectedResult := &domain.ReassignInactiveResult{
		PullRequestID: "pr-001",
		ReassignmentDetails: []domain.ReviewerReassignment{
			{OldReviewerID: "user-2", NewReviewerID: "user-4", WasRemoved: false},
			{OldReviewerID: "user-5", NewReviewerID: "", WasRemoved: true},
		},
	}

	mockService.On("ReassignInactiveReviewers", mock.Anything, mock.MatchedBy(func(input *domain.ReassignInactiveInput) bool {
		return input.PullRequestID == "pr-001"
	})).Return(expectedResult, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "pr-001", response["pull_request_id"])
	details := response["reassignment_details"].([]interface{})
	assert.Equal(t, 2, len(details))

	// Проверяем первый — замена
	detail0 := details[0].(map[string]interface{})
	assert.Equal(t, "user-2", detail0["old_reviewer_id"])
	assert.Equal(t, "user-4", detail0["new_reviewer_id"])
	assert.False(t, detail0["was_removed"].(bool))

	// Проверяем второй — удаление без замены
	detail1 := details[1].(map[string]interface{})
	assert.Equal(t, "user-5", detail1["old_reviewer_id"])
	assert.True(t, detail1["was_removed"].(bool))

	mockService.AssertExpectations(t)
}

func TestReassignInactiveHandler_NoInactive(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-001",
	}

	expectedResult := &domain.ReassignInactiveResult{
		PullRequestID:       "pr-001",
		ReassignmentDetails: []domain.ReviewerReassignment{},
	}

	mockService.On("ReassignInactiveReviewers", mock.Anything, mock.Anything).
		Return(expectedResult, nil)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	details := response["reassignment_details"].([]interface{})
	assert.Equal(t, 0, len(details))

	mockService.AssertExpectations(t)
}

func TestReassignInactiveHandler_MergedPR(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "pr-merged",
	}

	mockService.On("ReassignInactiveReviewers", mock.Anything, mock.Anything).
		Return(nil, domain.ErrReassignOnMerged)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockService.AssertExpectations(t)
}

func TestReassignInactiveHandler_PRNotFound(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "nonexistent",
	}

	mockService.On("ReassignInactiveReviewers", mock.Anything, mock.Anything).
		Return(nil, domain.ErrResourceNotFound)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

// === Тесты авторизации ===

func TestDeactivateTeamHandler_NoAuth(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"team_name": "backend",
	}

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// НЕ устанавливаем Authorization
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMergePullRequestHandler_InvalidRequest(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	// Пустой body
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestMergePullRequestHandler_NotFound(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"pull_request_id": "nonexistent",
	}

	mockService.On("MergePullRequest", mock.Anything, mock.Anything).
		Return(nil, domain.ErrResourceNotFound)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestSetUserIsActiveHandler_UserNotFound(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"user_id":   "nonexistent",
		"is_active": false,
	}

	mockService.On("SetUserIsActive", mock.Anything, "nonexistent", false).
		Return(nil, domain.ErrResourceNotFound)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockService.AssertExpectations(t)
}

func TestGetReviewHandler_MissingUserID(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	// Нет query param user_id
	req := httptest.NewRequest(http.MethodGet, "/users/getReview", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAddTeamHandler_DuplicateTeam(t *testing.T) {
	mockService := mocks.NewAssignmentService(t)
	router := setupTestRouter(mockService)

	requestBody := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice", "is_active": true},
		},
	}

	mockService.On("CreateTeam", mock.Anything, mock.Anything).
		Return(nil, domain.ErrTeamExists)

	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "TEAM_EXISTS", errorObj["code"])

	mockService.AssertExpectations(t)
}
