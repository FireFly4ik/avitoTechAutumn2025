package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
				c.Abort()
				return
			}

			token := parts[1]

			adminToken := os.Getenv("ADMIN_TOKEN")
			userToken := os.Getenv("USER_TOKEN")

			switch token {
			case adminToken:
				c.Set("role", "admin")
			case userToken:
				c.Set("role", "user")
			default:
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "admin token required"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "user" && role != "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user token required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
