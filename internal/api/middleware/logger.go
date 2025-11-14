package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	RequestIDKey string = "request_id"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()

		c.Set(RequestIDKey, requestID)

		ctx := context.WithValue(c.Request.Context(), RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()

		log.Info().
			Str("request_id", requestID).
			Str("layer", "middleware").
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Msg("request started")

		c.Next()

		latency := time.Since(start)

		log.Info().
			Str("request_id", requestID).
			Str("layer", "middleware").
			Dur("latency", latency).
			Int("status", c.Writer.Status()).
			Msg("request completed")
	}
}
