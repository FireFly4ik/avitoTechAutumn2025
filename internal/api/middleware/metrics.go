package middleware

import (
	"avitoTechAutumn2025/internal/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// MetricsMiddleware собирает HTTP метрики для Prometheus
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем /metrics endpoint чтобы не учитывать его в статистике
		if c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		start := time.Now()

		// Записываем размер запроса
		if c.Request.ContentLength > 0 {
			metrics.HTTPRequestSize.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
			).Observe(float64(c.Request.ContentLength))
		}

		// Оборачиваем ResponseWriter для отслеживания размера ответа
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			statusCode:     200,
		}
		c.Writer = writer

		c.Next()

		// Вычисляем длительность запроса
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(writer.statusCode)
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		// Записываем метрики
		metrics.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			status,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			c.Request.Method,
			path,
			status,
		).Observe(duration)

		metrics.HTTPResponseSize.WithLabelValues(
			c.Request.Method,
			path,
		).Observe(float64(writer.size))
	}
}

// responseWriter оборачивает gin.ResponseWriter для отслеживания размера и статуса
type responseWriter struct {
	gin.ResponseWriter
	statusCode int
	size       int
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.size += size
	return size, err
}
