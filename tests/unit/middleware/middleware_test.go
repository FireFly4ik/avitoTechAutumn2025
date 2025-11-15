package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"avitoTechAutumn2025/internal/api/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_AdminToken(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	os.Setenv("USER_TOKEN", "test-user-token")
	defer os.Unsetenv("ADMIN_TOKEN")
	defer os.Unsetenv("USER_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/admin", middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"role": "admin"})
	})

	// Act
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"role":"admin"`)
}

func TestAuthMiddleware_UserToken(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	os.Setenv("USER_TOKEN", "test-user-token")
	defer os.Unsetenv("ADMIN_TOKEN")
	defer os.Unsetenv("USER_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/data", middleware.RequireUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"role": "user"})
	})

	// Act - User token для user endpoint
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer test-user-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"role":"user"`)
}

func TestAuthMiddleware_AdminTokenForUserEndpoint(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	os.Setenv("USER_TOKEN", "test-user-token")
	defer os.Unsetenv("ADMIN_TOKEN")
	defer os.Unsetenv("USER_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/data", middleware.RequireUser(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"role": "admin"})
	})

	// Act - Admin token должен работать для user endpoint
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"role":"admin"`)
}

func TestAuthMiddleware_UserTokenForAdminEndpoint_Forbidden(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	os.Setenv("USER_TOKEN", "test-user-token")
	defer os.Unsetenv("ADMIN_TOKEN")
	defer os.Unsetenv("USER_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/admin", middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Act - User token для admin endpoint должен быть запрещён
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer test-user-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	defer os.Unsetenv("ADMIN_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/admin", middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Act - Без токена
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	os.Setenv("USER_TOKEN", "test-user-token")
	defer os.Unsetenv("ADMIN_TOKEN")
	defer os.Unsetenv("USER_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/admin", middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Act - Неверный токен
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidAuthorizationFormat(t *testing.T) {
	// Arrange
	os.Setenv("ADMIN_TOKEN", "test-admin-token")
	defer os.Unsetenv("ADMIN_TOKEN")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.AuthMiddleware())

	router.GET("/admin", middleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Act - Неверный формат Authorization (без Bearer)
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "test-admin-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCORSMiddleware(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORSMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Act
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.CORSMiddleware())

	router.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Act - OPTIONS preflight request
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRecoveryMiddleware(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.RecoveryMiddleware())

	router.GET("/panic", func(c *gin.Context) {
		panic(fmt.Errorf("simulated panic"))
	})

	// Act
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}
