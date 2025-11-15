package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"avitoTechAutumn2025/internal/api/handlers"
	"avitoTechAutumn2025/internal/config"
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/service"
	storageGorm "avitoTechAutumn2025/internal/storage/gorm"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	gormlib "gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var (
	testDB      *gormlib.DB
	testService domain.AssignmentService
	testRouter  *gin.Engine
)

// TestMain настраивает тестовое окружение
func TestMain(m *testing.M) {
	// Настройка логгера для тестов
	logger.Setup(&config.Config{
		ProductionType: "test",
	})

	// Используем тестовую БД (можно переопределить через ENV)
	_ = os.Setenv("DB_HOST", getEnv("TEST_DB_HOST", "localhost"))
	_ = os.Setenv("DB_PORT", getEnv("TEST_DB_PORT", "5436"))
	_ = os.Setenv("DB_NAME", getEnv("TEST_DB_NAME", "avito_test"))
	_ = os.Setenv("DB_USER", getEnv("TEST_DB_USER", "avito"))
	_ = os.Setenv("DB_PASSWORD", getEnv("TEST_DB_PASSWORD", "avito_password"))
	_ = os.Setenv("DB_SSLMODE", "disable")
	_ = os.Setenv("PORT", "8080")
	_ = os.Setenv("PRODUCTION_TYPE", "test")
	_ = os.Setenv("ADMIN_TOKEN", "admin")
	_ = os.Setenv("USER_TOKEN", "user")

	// Загружаем конфигурацию
	cfg := config.NewEnvConfig()

	// Подключаемся к БД и применяем миграции
	var err error
	testDB, err = connectTestDB(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to test DB: %v", err))
	}

	// Создаём TxManager и сервис
	txManager, err := storageGorm.NewTxManager(testDB)
	if err != nil {
		panic(fmt.Sprintf("failed to create tx manager: %v", err))
	}
	testService = service.New(txManager)

	// Создаём роутер
	gin.SetMode(gin.TestMode)
	handler := handlers.NewHandler(testService)
	testRouter = handler.InitRoutes()

	// Запускаем тесты
	code := m.Run()

	os.Exit(code)
}

// connectTestDB подключается к тестовой БД и применяет миграции
func connectTestDB(cfg *config.Config) (*gormlib.DB, error) {
	connectionString := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Database.Host,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.Port,
		cfg.Database.SSLMode,
	)

	gormConfig := &gormlib.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Silent), // Тихий режим для тестов
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gormlib.Open(postgres.Open(connectionString), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from gorm DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Применяем миграции если нужно
	if err := applyMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return db, nil
}

// applyMigrations применяет SQL миграции напрямую
func applyMigrations(db *gormlib.DB) error {
	// Проверяем, нужно ли применять миграции
	var exists bool
	err := db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'teams')").Scan(&exists).Error
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	// Применяем миграцию
	migrationSQL := `
CREATE TABLE IF NOT EXISTS teams (
    team_name TEXT PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    user_id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    team_name TEXT NOT NULL REFERENCES teams(team_name) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now()
);

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'pr_status') THEN
        CREATE TYPE pr_status AS ENUM ('OPEN', 'MERGED');
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id TEXT PRIMARY KEY,
    pull_request_name TEXT NOT NULL,
    author_id TEXT NOT NULL REFERENCES users(user_id),
    status pr_status NOT NULL DEFAULT 'OPEN',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    merged_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS pull_request_reviewers (
    pull_request_id TEXT NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    reviewer_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    PRIMARY KEY (pull_request_id, reviewer_id)
);

CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer ON pull_request_reviewers(reviewer_id);
CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);
CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active);
CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = now();
   RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_teams_updated_at ON teams;
CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_pull_requests_updated_at ON pull_requests;
CREATE TRIGGER update_pull_requests_updated_at BEFORE UPDATE ON pull_requests FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
`

	return db.Exec(migrationSQL).Error
}

// setupTest очищает БД перед каждым тестом
func setupTest(t *testing.T) {
	t.Helper()

	// Очищаем таблицы в правильном порядке (FK constraints)
	tables := []string{
		"pull_request_reviewers",
		"pull_requests",
		"users",
		"teams",
	}

	for _, table := range tables {
		err := testDB.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)).Error
		require.NoError(t, err, "failed to truncate table %s", table)
	}
} // getEnv возвращает переменную окружения или дефолт
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// createTestTeam создаёт тестовую команду с участниками
func createTestTeam(t *testing.T, teamName string, userCount int) []string {
	t.Helper()

	ctx := context.Background()
	userIDs := make([]string, userCount)
	members := make([]domain.TeamMember, userCount)

	for i := 0; i < userCount; i++ {
		userID := fmt.Sprintf("test-user-%s-%d", teamName, i)
		userIDs[i] = userID
		members[i] = domain.TeamMember{
			UserID:   userID,
			Username: fmt.Sprintf("User %d", i),
			IsActive: true,
		}
	}

	team := &domain.Team{
		Name:    teamName,
		Members: members,
	}

	_, err := testService.CreateTeam(ctx, team)
	require.NoError(t, err, "failed to create test team")

	return userIDs
}

// createTestPR создаёт тестовый PR
func createTestPR(t *testing.T, prID, authorID string) *domain.PullRequest {
	t.Helper()

	ctx := context.Background()
	input := &domain.CreatePullRequestInput{
		PullRequestID:   prID,
		PullRequestName: "Test PR",
		AuthorID:        authorID,
	}

	pr, err := testService.CreatePullRequest(ctx, input)
	require.NoError(t, err, "failed to create test PR")

	return pr
}

// TestTeamAdd_CreateNew проверяет создание новой команды
func TestTeamAdd_CreateNew(t *testing.T) {
	setupTest(t)

	reqBody := map[string]interface{}{
		"team_name": "backend",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice", "is_active": true},
			{"user_id": "u2", "username": "Bob", "is_active": true},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "backend", response["team_name"])

	members := response["members"].([]interface{})
	assert.Equal(t, 2, len(members))
}

// TestTeamAdd_UpsertUsers проверяет upsert логику для пользователей
func TestTeamAdd_UpsertUsers(t *testing.T) {
	setupTest(t)

	// Создаём команду с пользователем
	reqBody1 := map[string]interface{}{
		"team_name": "team1",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice", "is_active": true},
		},
	}
	body1, _ := json.Marshal(reqBody1)
	req1 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	testRouter.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusCreated, w1.Code)

	// Создаём другую команду с тем же пользователем (должен переместиться)
	reqBody2 := map[string]interface{}{
		"team_name": "team2",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "Alice Updated", "is_active": false},
		},
	}
	body2, _ := json.Marshal(reqBody2)
	req2 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	testRouter.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusCreated, w2.Code)

	// Проверяем что пользователь обновился
	ctx := context.Background()
	team, err := testService.GetTeam(ctx, "team2")
	require.NoError(t, err)
	require.Equal(t, 1, len(team.Members))

	member := team.Members[0]
	assert.Equal(t, "u1", member.UserID)
	assert.Equal(t, "Alice Updated", member.Username)
	assert.Equal(t, false, member.IsActive) // Критично! Проверяем is_active=false
}

// TestTeamAdd_IsActiveFalse проверяет что is_active=false сохраняется правильно
func TestTeamAdd_IsActiveFalse(t *testing.T) {
	setupTest(t)

	reqBody := map[string]interface{}{
		"team_name": "test",
		"members": []map[string]interface{}{
			{"user_id": "active", "username": "Active", "is_active": true},
			{"user_id": "inactive", "username": "Inactive", "is_active": false},
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)

	// Проверяем через GetTeam
	ctx := context.Background()
	team, err := testService.GetTeam(ctx, "test")
	require.NoError(t, err)
	require.Equal(t, 2, len(team.Members))

	// Находим inactive пользователя
	var inactiveMember *domain.TeamMember
	for _, m := range team.Members {
		if m.UserID == "inactive" {
			inactiveMember = &m
			break
		}
	}

	require.NotNil(t, inactiveMember, "Inactive пользователь должен существовать")
	assert.False(t, inactiveMember.IsActive, "is_active должен быть false")
}

// TestTeamAdd_Duplicate проверяет ошибку при дубликате команды
func TestTeamAdd_Duplicate(t *testing.T) {
	setupTest(t)

	reqBody := map[string]interface{}{
		"team_name": "duplicate",
		"members": []map[string]interface{}{
			{"user_id": "u1", "username": "User", "is_active": true},
		},
	}
	body, _ := json.Marshal(reqBody)

	// Первый запрос - успех
	req1 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	testRouter.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Второй запрос - ошибка
	req2 := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	testRouter.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusBadRequest, w2.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "TEAM_EXISTS", errorObj["code"])
}

// TestTeamDeactivate_Success проверяет массовую деактивацию команды
func TestTeamDeactivate_Success(t *testing.T) {
	setupTest(t)

	// Создаём команду с 5 активными участниками
	ctx := context.Background()
	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "User 1", IsActive: true},
			{UserID: "u2", Username: "User 2", IsActive: true},
			{UserID: "u3", Username: "User 3", IsActive: true},
			{UserID: "u4", Username: "User 4", IsActive: true},
			{UserID: "u5", Username: "User 5", IsActive: true},
		},
	}
	_, err := testService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Деактивируем всю команду
	reqBody := map[string]interface{}{
		"team_name": "backend",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "backend", response["team_name"])
	assert.Equal(t, float64(5), response["deactivated_user_count"])

	// Проверяем что все действительно деактивированы
	updatedTeam, err := testService.GetTeam(ctx, "backend")
	require.NoError(t, err)

	for _, member := range updatedTeam.Members {
		assert.False(t, member.IsActive, "Все члены команды должны быть неактивны")
	}
}

// TestTeamDeactivate_PartiallyInactive проверяет деактивацию когда часть уже неактивна
func TestTeamDeactivate_PartiallyInactive(t *testing.T) {
	setupTest(t)

	ctx := context.Background()
	team := &domain.Team{
		Name: "mixed",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "User 1", IsActive: true},
			{UserID: "u2", Username: "User 2", IsActive: true},
			{UserID: "u3", Username: "User 3", IsActive: false}, // Уже неактивен
			{UserID: "u4", Username: "User 4", IsActive: false}, // Уже неактивен
		},
	}
	_, err := testService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Деактивируем
	reqBody := map[string]interface{}{
		"team_name": "mixed",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/team/deactivate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	// Должно быть деактивировано только 2 (которые были активны)
	assert.Equal(t, float64(2), response["deactivated_user_count"])
}

// TestTeamGet_Success проверяет получение команды
func TestTeamGet_Success(t *testing.T) {
	setupTest(t)

	// Создаём команду
	createTestTeam(t, "backend", 3)

	// Получаем
	req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=backend", nil)
	req.Header.Set("Authorization", "Bearer user")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var team map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &team)

	assert.Equal(t, "backend", team["team_name"])
	members := team["members"].([]interface{})
	assert.Equal(t, 3, len(members))
}

// TestTeamGet_NotFound проверяет ошибку 404 для несуществующей команды
func TestTeamGet_NotFound(t *testing.T) {
	setupTest(t)

	req := httptest.NewRequest(http.MethodGet, "/team/get?team_name=nonexistent", nil)
	req.Header.Set("Authorization", "Bearer user")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "NOT_FOUND", errorObj["code"])
}

// TestUserSetIsActive_Success проверяет изменение статуса активности
func TestUserSetIsActive_Success(t *testing.T) {
	setupTest(t)

	// Создаём пользователя
	userIDs := createTestTeam(t, "backend", 1)
	userID := userIDs[0]

	// Деактивируем
	reqBody := map[string]interface{}{
		"user_id":   userID,
		"is_active": false,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/users/setIsActive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, userID, response["user_id"])
	assert.False(t, response["is_active"].(bool))
}

// TestUserGetReview_Success проверяет получение PR'ов пользователя
func TestUserGetReview_Success(t *testing.T) {
	setupTest(t)

	// Создаём команду и PR
	userIDs := createTestTeam(t, "backend", 3)
	pr := createTestPR(t, "pr-001", userIDs[0])

	require.NotEmpty(t, pr.AssignedReviewers, "PR должен иметь ревьюеров")
	reviewerID := pr.AssignedReviewers[0]

	// Получаем PR'ы ревьювера
	req := httptest.NewRequest(http.MethodGet, "/users/getReview?user_id="+reviewerID, nil)
	req.Header.Set("Authorization", "Bearer user")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, reviewerID, response["user_id"])

	prs := response["pull_requests"].([]interface{})
	assert.Equal(t, 1, len(prs))

	prData := prs[0].(map[string]interface{})
	assert.Equal(t, "pr-001", prData["pull_request_id"])
}

// TestPullRequestCreate_Success проверяет успешное создание PR с назначением ревьюеров
func TestPullRequestCreate_Success(t *testing.T) {
	setupTest(t)

	// Создаём команду с 3 участниками
	userIDs := createTestTeam(t, "backend", 3)
	authorID := userIDs[0]

	// Создаём PR
	reqBody := map[string]interface{}{
		"pull_request_id":   "pr-001",
		"pull_request_name": "Add feature",
		"author_id":         authorID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	// Проверяем ответ
	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	pr := response["pr"].(map[string]interface{})
	assert.Equal(t, "pr-001", pr["pull_request_id"])
	assert.Equal(t, "OPEN", pr["status"])

	// Проверяем что назначены до 2 ревьюеров
	reviewers := pr["assigned_reviewers"].([]interface{})
	assert.LessOrEqual(t, len(reviewers), 2)
	assert.GreaterOrEqual(t, len(reviewers), 1) // Минимум 1 (есть 2 кандидата кроме автора)

	// Проверяем что автор не назначен сам себе
	for _, reviewer := range reviewers {
		assert.NotEqual(t, authorID, reviewer)
	}
}

// TestPullRequestCreate_NoReviewers проверяет создание PR когда нет кандидатов
func TestPullRequestCreate_NoReviewers(t *testing.T) {
	setupTest(t)

	// Создаём команду с 1 участником (только автор)
	userIDs := createTestTeam(t, "solo", 1)
	authorID := userIDs[0]

	reqBody := map[string]interface{}{
		"pull_request_id":   "pr-002",
		"pull_request_name": "Solo PR",
		"author_id":         authorID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	// Должно быть 0 ревьюеров
	assert.Equal(t, 0, len(reviewers))
}

// TestPullRequestCreate_OnlyActiveReviewers проверяет что назначаются только активные
func TestPullRequestCreate_OnlyActiveReviewers(t *testing.T) {
	setupTest(t)

	ctx := context.Background()

	// Создаём команду
	team := &domain.Team{
		Name: "mixed",
		Members: []domain.TeamMember{
			{UserID: "active-1", Username: "Active 1", IsActive: true},
			{UserID: "active-2", Username: "Active 2", IsActive: true},
			{UserID: "inactive-1", Username: "Inactive 1", IsActive: false},
			{UserID: "inactive-2", Username: "Inactive 2", IsActive: false},
		},
	}
	_, err := testService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Создаём PR от active-1
	reqBody := map[string]interface{}{
		"pull_request_id":   "pr-003",
		"pull_request_name": "Test active only",
		"author_id":         "active-1",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	pr := response["pr"].(map[string]interface{})
	reviewers := pr["assigned_reviewers"].([]interface{})

	// Должен быть назначен только active-2 (1 активный кандидат)
	assert.Equal(t, 1, len(reviewers))
	assert.Equal(t, "active-2", reviewers[0])
}

// TestPullRequestCreate_Duplicate проверяет ошибку при дубликате PR
func TestPullRequestCreate_Duplicate(t *testing.T) {
	setupTest(t)

	userIDs := createTestTeam(t, "backend", 2)
	authorID := userIDs[0]

	reqBody := map[string]interface{}{
		"pull_request_id":   "pr-duplicate",
		"pull_request_name": "Test",
		"author_id":         authorID,
	}
	body, _ := json.Marshal(reqBody)

	// Первый запрос - успех
	req1 := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer admin")
	w1 := httptest.NewRecorder()
	testRouter.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	// Второй запрос - ошибка
	req2 := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer admin")
	w2 := httptest.NewRecorder()
	testRouter.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "PR_EXISTS", errorObj["code"])
}

// TestPullRequestMerge_Idempotent проверяет идемпотентность merge
func TestPullRequestMerge_Idempotent(t *testing.T) {
	setupTest(t)

	// Создаём PR
	userIDs := createTestTeam(t, "backend", 3)
	pr := createTestPR(t, "pr-merge", userIDs[0])

	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
	}
	body, _ := json.Marshal(reqBody)

	// Первый merge
	req1 := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer admin")
	w1 := httptest.NewRecorder()
	testRouter.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	var response1 map[string]interface{}
	_ = json.Unmarshal(w1.Body.Bytes(), &response1)
	pr1 := response1["pr"].(map[string]interface{})
	assert.Equal(t, "MERGED", pr1["status"])
	assert.NotNil(t, pr1["merged_at"], "merged_at должен быть установлен после merge")

	// Второй merge - должен быть идемпотентным (200 OK, не ошибка!)
	req2 := httptest.NewRequest(http.MethodPost, "/pullRequest/merge", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer admin")
	w2 := httptest.NewRecorder()
	testRouter.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code, "Merge должен быть идемпотентным")

	var response2 map[string]interface{}
	_ = json.Unmarshal(w2.Body.Bytes(), &response2)
	pr2 := response2["pr"].(map[string]interface{})
	assert.Equal(t, "MERGED", pr2["status"])

	// merged_at должен остаться тем же
	assert.Equal(t, pr1["merged_at"], pr2["merged_at"])
}

// TestPullRequestReassign_Success проверяет переназначение ревьювера
func TestPullRequestReassign_Success(t *testing.T) {
	setupTest(t)

	// Создаём команду с 4 участниками
	userIDs := createTestTeam(t, "backend", 4)

	// Создаём PR
	pr := createTestPR(t, "pr-reassign", userIDs[0])
	require.NotEmpty(t, pr.AssignedReviewers, "PR должен иметь ревьюеров")

	oldReviewerID := pr.AssignedReviewers[0]

	// Переназначаем
	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
		"old_reviewer_id": oldReviewerID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	newReviewerID := response["replaced_by"].(string)
	assert.NotEmpty(t, newReviewerID)
	assert.NotEqual(t, oldReviewerID, newReviewerID)
	assert.NotEqual(t, userIDs[0], newReviewerID, "Новый ревьювер не должен быть автором")
}

// TestPullRequestReassign_AfterMerge проверяет запрет переназначения после merge
func TestPullRequestReassign_AfterMerge(t *testing.T) {
	setupTest(t)

	userIDs := createTestTeam(t, "backend", 3)
	pr := createTestPR(t, "pr-merged", userIDs[0])
	require.NotEmpty(t, pr.AssignedReviewers)

	// Мержим PR
	ctx := context.Background()
	mergeInput := &domain.MergePullRequestInput{PullRequestID: pr.ID}
	_, err := testService.MergePullRequest(ctx, mergeInput)
	require.NoError(t, err)

	// Пытаемся переназначить
	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
		"old_reviewer_id": pr.AssignedReviewers[0],
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "PR_MERGED", errorObj["code"])
}

// TestPullRequestReassign_NotAssigned проверяет ошибку если пользователь не ревьювер
func TestPullRequestReassign_NotAssigned(t *testing.T) {
	setupTest(t)

	userIDs := createTestTeam(t, "backend", 4)
	pr := createTestPR(t, "pr-not-assigned", userIDs[0])

	// Находим пользователя который НЕ назначен ревьювером
	var notAssignedID string
	for _, uid := range userIDs {
		if uid == userIDs[0] {
			continue // Автор
		}
		isAssigned := false
		for _, reviewerID := range pr.AssignedReviewers {
			if uid == reviewerID {
				isAssigned = true
				break
			}
		}
		if !isAssigned {
			notAssignedID = uid
			break
		}
	}

	if notAssignedID == "" {
		t.Skip("Все пользователи назначены ревьюверами")
	}

	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
		"old_reviewer_id": notAssignedID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "NOT_ASSIGNED", errorObj["code"])
}

// TestReassignInactive_Success проверяет переназначение всех неактивных ревьюверов
func TestReassignInactive_Success(t *testing.T) {
	setupTest(t)

	ctx := context.Background()

	// Создаём команду с активными и неактивными
	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "author", Username: "Author", IsActive: true},
			{UserID: "active-1", Username: "Active 1", IsActive: true},
			{UserID: "active-2", Username: "Active 2", IsActive: true},
			{UserID: "inactive-1", Username: "Inactive 1", IsActive: true}, // Сделаем неактивным после
			{UserID: "inactive-2", Username: "Inactive 2", IsActive: true}, // Сделаем неактивным после
		},
	}
	_, err := testService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Создаём PR
	prInput := &domain.CreatePullRequestInput{
		PullRequestID:   "pr-reassign-inactive",
		PullRequestName: "Test PR",
		AuthorID:        "author",
	}
	pr, err := testService.CreatePullRequest(ctx, prInput)
	require.NoError(t, err)
	require.NotEmpty(t, pr.AssignedReviewers)

	// Деактивируем некоторых ревьюверов
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == "active-1" || reviewerID == "active-2" {
			// Деактивируем
			_, err := testService.SetUserIsActive(ctx, reviewerID, false)
			require.NoError(t, err)
		}
	}

	// Переназначаем неактивных
	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	details := response["reassignment_details"].([]interface{})
	assert.NotEmpty(t, details, "Должны быть детали переназначения")

	// Проверяем что каждый неактивный либо заменён, либо удалён
	for _, detail := range details {
		d := detail.(map[string]interface{})
		oldID := d["old_reviewer_id"].(string)
		newID := d["new_reviewer_id"]
		wasRemoved := d["was_removed"].(bool)

		// Старый должен быть неактивным
		assert.Contains(t, []string{"active-1", "active-2"}, oldID)

		// Если не удалён - должен быть новый
		if !wasRemoved {
			assert.NotEmpty(t, newID)
			assert.NotEqual(t, oldID, newID)
		}
	}
}

// TestReassignInactive_NoInactive проверяет что ничего не происходит если нет неактивных
func TestReassignInactive_NoInactive(t *testing.T) {
	setupTest(t)

	// Создаём PR с активными ревьюверами
	userIDs := createTestTeam(t, "backend", 3)
	pr := createTestPR(t, "pr-all-active", userIDs[0])

	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	details := response["reassignment_details"].([]interface{})
	assert.Empty(t, details, "Не должно быть переназначений")
}

// TestReassignInactive_AfterMerge проверяет запрет переназначения после merge
func TestReassignInactive_AfterMerge(t *testing.T) {
	setupTest(t)

	ctx := context.Background()

	// Создаём и мержим PR
	userIDs := createTestTeam(t, "backend", 3)
	pr := createTestPR(t, "pr-merged", userIDs[0])

	mergeInput := &domain.MergePullRequestInput{PullRequestID: pr.ID}
	_, err := testService.MergePullRequest(ctx, mergeInput)
	require.NoError(t, err)

	// Пытаемся переназначить неактивных
	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	errorObj := response["error"].(map[string]interface{})
	assert.Equal(t, "PR_MERGED", errorObj["code"])
}

// TestReassignInactive_RemovesWhenNoCandidates проверяет удаление когда нет кандидатов
func TestReassignInactive_RemovesWhenNoCandidates(t *testing.T) {
	setupTest(t)

	ctx := context.Background()

	// Создаём команду где все станут неактивными
	team := &domain.Team{
		Name: "dying-team",
		Members: []domain.TeamMember{
			{UserID: "author", Username: "Author", IsActive: true},
			{UserID: "reviewer", Username: "Reviewer", IsActive: true},
		},
	}
	_, err := testService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Создаём PR
	prInput := &domain.CreatePullRequestInput{
		PullRequestID:   "pr-no-candidates",
		PullRequestName: "Test PR",
		AuthorID:        "author",
	}
	pr, err := testService.CreatePullRequest(ctx, prInput)
	require.NoError(t, err)
	require.Equal(t, 1, len(pr.AssignedReviewers))
	require.Equal(t, "reviewer", pr.AssignedReviewers[0])

	// Деактивируем ревьювера
	_, err = testService.SetUserIsActive(ctx, "reviewer", false)
	require.NoError(t, err)

	// Переназначаем - должен быть удалён (нет кандидатов)
	reqBody := map[string]interface{}{
		"pull_request_id": pr.ID,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassignInactive", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin")

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &response)

	details := response["reassignment_details"].([]interface{})
	require.Equal(t, 1, len(details))

	detail := details[0].(map[string]interface{})
	assert.Equal(t, "reviewer", detail["old_reviewer_id"])
	assert.Equal(t, "", detail["new_reviewer_id"])
	assert.True(t, detail["was_removed"].(bool))
}

// TestAuth_RequiresAdmin проверяет что административные эндпоинты требуют admin токен
func TestAuth_RequiresAdmin(t *testing.T) {
	setupTest(t)

	userIDs := createTestTeam(t, "backend", 2)

	endpoints := []struct {
		method string
		path   string
		body   map[string]interface{}
	}{
		{
			method: http.MethodPost,
			path:   "/pullRequest/create",
			body: map[string]interface{}{
				"pull_request_id":   "pr-test",
				"pull_request_name": "Test",
				"author_id":         userIDs[0],
			},
		},
		{
			method: http.MethodPost,
			path:   "/team/deactivate",
			body:   map[string]interface{}{"team_name": "backend"},
		},
		{
			method: http.MethodPost,
			path:   "/users/setIsActive",
			body:   map[string]interface{}{"user_id": userIDs[0], "is_active": false},
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.path, func(t *testing.T) {
			body, _ := json.Marshal(endpoint.body)

			// Без токена
			req1 := httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewReader(body))
			req1.Header.Set("Content-Type", "application/json")
			w1 := httptest.NewRecorder()
			testRouter.ServeHTTP(w1, req1)
			assert.Equal(t, http.StatusUnauthorized, w1.Code, "Должен требовать токен")

			// С user токеном (недостаточно прав)
			req2 := httptest.NewRequest(endpoint.method, endpoint.path, bytes.NewReader(body))
			req2.Header.Set("Content-Type", "application/json")
			req2.Header.Set("Authorization", "Bearer user")
			w2 := httptest.NewRecorder()
			testRouter.ServeHTTP(w2, req2)
			assert.Equal(t, http.StatusUnauthorized, w2.Code, "User токен недостаточен")
		})
	}
}
