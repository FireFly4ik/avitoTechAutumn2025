package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"avitoTechAutumn2025/internal/api/handlers"
	"avitoTechAutumn2025/internal/api/server"
	"avitoTechAutumn2025/internal/config"
	"avitoTechAutumn2025/internal/domain"
	"avitoTechAutumn2025/internal/logger"
	"avitoTechAutumn2025/internal/service"
	storageGorm "avitoTechAutumn2025/internal/storage/gorm"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	gormlib "gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var (
	baseURL     string
	testDB      *gormlib.DB
	testService domain.AssignmentService
	apiServer   *server.Server
	httpClient  *http.Client
)

// TestMain настраивает E2E тестовое окружение
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
	_ = os.Setenv("APP_PORT", "8082") // Другой порт для E2E тестов
	_ = os.Setenv("PRODUCTION_TYPE", "test")
	_ = os.Setenv("ADMIN_TOKEN", "admin-token-e2e")
	_ = os.Setenv("USER_TOKEN", "user-token-e2e")

	// Загружаем конфигурацию
	cfg := config.NewEnvConfig()
	baseURL = fmt.Sprintf("http://localhost:%s", cfg.Port)

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

	// Создаём и запускаем HTTP сервер в отдельной горутине
	handler := handlers.NewHandler(testService)
	apiServer = server.NewServer(cfg, handler)
	go apiServer.Run()

	// Ждём пока сервер запустится
	if err := waitForServer(baseURL, 30); err != nil {
		panic(fmt.Sprintf("server did not start: %v", err))
	}

	// HTTP клиент с таймаутом
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
	}

	// Запускаем тесты
	code := m.Run()

	// Останавливаем сервер
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	apiServer.Shutdown(ctx)

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
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
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
	var exists bool
	err := db.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'teams')").Scan(&exists).Error
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

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
}

// waitForServer ждёт готовности HTTP сервера
func waitForServer(url string, maxAttempts int) error {
	for i := 0; i < maxAttempts; i++ {
		resp, err := http.Get(url + "/metrics")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("server not ready after %d attempts", maxAttempts)
}

// getEnv возвращает переменную окружения или дефолт
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// httpRequest выполняет HTTP запрос и возвращает ответ
func httpRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	t.Helper()

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, baseURL+path, reqBody)
	require.NoError(t, err)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	require.NoError(t, err)

	return resp
}

// parseJSON парсит JSON ответ
func parseJSON(t *testing.T, resp *http.Response) map[string]interface{} {
	t.Helper()

	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)

	return result
}

// TestFullWorkflow проверяет полный сценарий работы с приложением
func TestFullWorkflow(t *testing.T) {
	setupTest(t)

	// Шаг 1: Создаём команду разработки
	t.Log("Step 1: Creating backend team")
	teamReq := map[string]interface{}{
		"team_name": "backend-team",
		"members": []map[string]interface{}{
			{"user_id": "dev1", "username": "Alice", "is_active": true},
			{"user_id": "dev2", "username": "Bob", "is_active": true},
			{"user_id": "dev3", "username": "Charlie", "is_active": true},
		},
	}
	resp := httpRequest(t, http.MethodPost, "/team/add", teamReq, "")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	team := parseJSON(t, resp)
	assert.Equal(t, "backend-team", team["team_name"])

	// Шаг 2: Создаём PR от Alice
	t.Log("Step 2: Alice creates a pull request")
	prReq := map[string]interface{}{
		"pull_request_id":   "pr-001",
		"pull_request_name": "Add authentication",
		"author_id":         "dev1",
	}
	resp = httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "admin-token-e2e")
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	prResult := parseJSON(t, resp)
	pr := prResult["pr"].(map[string]interface{})
	assert.Equal(t, "pr-001", pr["pull_request_id"])
	assert.Equal(t, "OPEN", pr["status"])
	reviewers := pr["assigned_reviewers"].([]interface{})
	assert.NotEmpty(t, reviewers, "PR should have reviewers assigned")
	t.Logf("Assigned reviewers: %v", reviewers)

	// Шаг 3: Проверяем что у ревьювера есть этот PR в списке
	t.Log("Step 3: Checking reviewer's assignments")
	reviewerID := reviewers[0].(string)
	resp = httpRequest(t, http.MethodGet, fmt.Sprintf("/users/getReview?user_id=%s", reviewerID), nil, "user-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assignments := parseJSON(t, resp)
	prs := assignments["pull_requests"].([]interface{})
	assert.NotEmpty(t, prs, "Reviewer should have PR assignments")
	firstPR := prs[0].(map[string]interface{})
	assert.Equal(t, "pr-001", firstPR["pull_request_id"])

	// Шаг 4: Bob становится неактивным
	t.Log("Step 4: Deactivating Bob")
	deactivateReq := map[string]interface{}{
		"user_id":   "dev2",
		"is_active": false,
	}
	resp = httpRequest(t, http.MethodPost, "/users/setIsActive", deactivateReq, "admin-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	user := parseJSON(t, resp)
	assert.Equal(t, "dev2", user["user_id"])
	assert.False(t, user["is_active"].(bool))

	// Шаг 5: Если Bob был ревьювером - переназначаем неактивных
	t.Log("Step 5: Reassigning inactive reviewers")
	if reviewerID == "dev2" {
		reassignReq := map[string]interface{}{
			"pull_request_id": "pr-001",
		}
		resp = httpRequest(t, http.MethodPost, "/pullRequest/reassignInactive", reassignReq, "admin-token-e2e")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		result := parseJSON(t, resp)
		details := result["reassignment_details"].([]interface{})
		assert.NotEmpty(t, details, "Should have reassignment details")
		t.Logf("Reassignment details: %v", details)
	}

	// Шаг 6: Мержим PR
	t.Log("Step 6: Merging pull request")
	mergeReq := map[string]interface{}{
		"pull_request_id": "pr-001",
	}
	resp = httpRequest(t, http.MethodPost, "/pullRequest/merge", mergeReq, "admin-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	mergeResult := parseJSON(t, resp)
	mergedPR := mergeResult["pr"].(map[string]interface{})
	assert.Equal(t, "MERGED", mergedPR["status"])
	assert.NotNil(t, mergedPR["merged_at"], "merged_at should be set")

	// Шаг 7: Проверяем идемпотентность merge
	t.Log("Step 7: Testing merge idempotency")
	resp = httpRequest(t, http.MethodPost, "/pullRequest/merge", mergeReq, "admin-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Second merge should be idempotent")

	// Шаг 8: Получаем информацию о команде
	t.Log("Step 8: Getting team information")
	resp = httpRequest(t, http.MethodGet, "/team/get?team_name=backend-team", nil, "user-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	teamInfo := parseJSON(t, resp)
	assert.Equal(t, "backend-team", teamInfo["team_name"])
	members := teamInfo["members"].([]interface{})
	assert.Equal(t, 3, len(members))
}

// TestAuthenticationFlow проверяет аутентификацию
func TestAuthenticationFlow(t *testing.T) {
	setupTest(t)

	// Создаём команду для тестов
	teamReq := map[string]interface{}{
		"team_name": "test-team",
		"members": []map[string]interface{}{
			{"user_id": "user1", "username": "User One", "is_active": true},
		},
	}
	resp := httpRequest(t, http.MethodPost, "/team/add", teamReq, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	t.Run("AdminEndpoint_WithoutToken", func(t *testing.T) {
		prReq := map[string]interface{}{
			"pull_request_id":   "pr-auth-1",
			"pull_request_name": "Test",
			"author_id":         "user1",
		}
		resp := httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("AdminEndpoint_WithUserToken", func(t *testing.T) {
		prReq := map[string]interface{}{
			"pull_request_id":   "pr-auth-2",
			"pull_request_name": "Test",
			"author_id":         "user1",
		}
		resp := httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "user-token-e2e")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("AdminEndpoint_WithAdminToken", func(t *testing.T) {
		prReq := map[string]interface{}{
			"pull_request_id":   "pr-auth-3",
			"pull_request_name": "Test",
			"author_id":         "user1",
		}
		resp := httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "admin-token-e2e")
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("UserEndpoint_WithUserToken", func(t *testing.T) {
		resp := httpRequest(t, http.MethodGet, "/team/get?team_name=test-team", nil, "user-token-e2e")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestTeamDeactivationFlow проверяет массовую деактивацию команды
func TestTeamDeactivationFlow(t *testing.T) {
	setupTest(t)

	// Создаём команду
	t.Log("Creating team with 5 members")
	teamReq := map[string]interface{}{
		"team_name": "dying-team",
		"members": []map[string]interface{}{
			{"user_id": "m1", "username": "Member 1", "is_active": true},
			{"user_id": "m2", "username": "Member 2", "is_active": true},
			{"user_id": "m3", "username": "Member 3", "is_active": true},
			{"user_id": "m4", "username": "Member 4", "is_active": true},
			{"user_id": "m5", "username": "Member 5", "is_active": true},
		},
	}
	resp := httpRequest(t, http.MethodPost, "/team/add", teamReq, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Деактивируем всю команду
	t.Log("Deactivating entire team")
	deactivateReq := map[string]interface{}{
		"team_name": "dying-team",
	}
	resp = httpRequest(t, http.MethodPost, "/team/deactivate", deactivateReq, "admin-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	result := parseJSON(t, resp)
	assert.Equal(t, "dying-team", result["team_name"])
	assert.Equal(t, float64(5), result["deactivated_user_count"])

	// Проверяем что все деактивированы
	t.Log("Verifying all members are inactive")
	resp = httpRequest(t, http.MethodGet, "/team/get?team_name=dying-team", nil, "user-token-e2e")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	team := parseJSON(t, resp)
	members := team["members"].([]interface{})
	for _, m := range members {
		member := m.(map[string]interface{})
		assert.False(t, member["is_active"].(bool), "All members should be inactive")
	}
}

// TestErrorHandling проверяет обработку ошибок
func TestErrorHandling(t *testing.T) {
	setupTest(t)

	t.Run("CreatePR_NonexistentAuthor", func(t *testing.T) {
		prReq := map[string]interface{}{
			"pull_request_id":   "pr-error-1",
			"pull_request_name": "Test",
			"author_id":         "nonexistent-user",
		}
		resp := httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "admin-token-e2e")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("GetTeam_Nonexistent", func(t *testing.T) {
		resp := httpRequest(t, http.MethodGet, "/team/get?team_name=nonexistent", nil, "user-token-e2e")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("MergePR_Nonexistent", func(t *testing.T) {
		mergeReq := map[string]interface{}{
			"pull_request_id": "nonexistent-pr",
		}
		resp := httpRequest(t, http.MethodPost, "/pullRequest/merge", mergeReq, "admin-token-e2e")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("DuplicatePR", func(t *testing.T) {
		// Создаём команду
		teamReq := map[string]interface{}{
			"team_name": "dup-team",
			"members": []map[string]interface{}{
				{"user_id": "dup1", "username": "Dup User", "is_active": true},
			},
		}
		resp := httpRequest(t, http.MethodPost, "/team/add", teamReq, "")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		// Создаём PR
		prReq := map[string]interface{}{
			"pull_request_id":   "pr-duplicate",
			"pull_request_name": "Test",
			"author_id":         "dup1",
		}
		resp = httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "admin-token-e2e")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		// Пытаемся создать дубликат
		resp = httpRequest(t, http.MethodPost, "/pullRequest/create", prReq, "admin-token-e2e")
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}
