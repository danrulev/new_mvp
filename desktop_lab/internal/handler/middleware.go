package handler

import (
	"context"
	contextkeys "desktop_lab/internal/contextKey"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	requestIDKey = "request_id"
)

func (s *Handler) logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()

		c.Set(requestIDKey, requestID)
		ctx := context.WithValue(c.Request.Context(), contextkeys.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		s.log.Debug("HTTP request started",
			zap.String("request_id", requestID),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", clientIP),
		)

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()
		if bodySize < 0 {
			bodySize = 0
		}

		fields := []zap.Field{
			zap.String("request_id", requestID),
			zap.Int("status_code", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", clientIP),
			zap.Duration("latency_ms", latency),
			zap.Int("response_size_bytes", bodySize),
		}

		var logFn func(string, ...zap.Field)
		switch {
		case statusCode >= 500:
			logFn = s.log.Error
		case statusCode >= 400:
			logFn = s.log.Warn
		default:
			logFn = s.log.Info
		}

		logFn("http_request_completed", fields...)
	}
}

func (s *Handler) loggerWith(c *gin.Context, fields ...zap.Field) *zap.Logger {
	base := []zap.Field{
		zap.String("request_id", s.getRequestID(c)),
		zap.String("client_ip", c.ClientIP()),
	}
	return s.log.With(append(base, fields...)...)
}

func (s *Handler) getRequestID(c *gin.Context) string {
	return c.GetString(requestIDKey)
}
