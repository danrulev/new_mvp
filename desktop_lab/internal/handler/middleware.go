package handler

import (
	"context"
	contextkeys "desktop_lab/internal/contextKey"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Константы для контекста и cookie.
const (
	userIDKey      = "user_id"
	roleKey        = "role"
	adminKey       = "admin"
	userKey        = "user"
	refreshToken   = "refresh_token"
	accessToken    = "access_token"
	authHeader     = "Authorization"
	requestHeader  = "X-Request-ID"
	requestContext = "request_id"
	requestIDKey   = "request_id"
)

// logging создает middleware для логирования HTTP-запросов.
func (h *Handler) logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()

		c.Set(requestIDKey, requestID)
		ctx := context.WithValue(c.Request.Context(), contextkeys.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()

		h.log.Debug("HTTP request started",
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
			logFn = h.log.Error
		case statusCode >= 400:
			logFn = h.log.Warn
		default:
			logFn = h.log.Info
		}

		logFn("http_request_completed", fields...)
	}
}

// loggerWith создает логгер с полями запроса.
func (h *Handler) loggerWith(c *gin.Context, fields ...zap.Field) *zap.Logger {
	base := []zap.Field{
		zap.String("request_id", h.getRequestID(c)),
		zap.String("client_ip", c.ClientIP()),
	}
	return h.log.With(append(base, fields...)...)
}

// authMiddleware проверяет аутентификацию пользователя.
func (h *Handler) authMiddleware(c *gin.Context) {
	_, err := getRefreshToken(c)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/api/auth/login")
		c.Abort()
		return
	}

	accessToken, err := getAccessToken(c)
	if err != nil {
		c.Redirect(http.StatusUnauthorized, "/api/auth/login")
		c.Abort()
		return
	}

	userID, err := h.auth.ParseToken(c.Request.Context(), accessToken)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/api/auth/login")
		c.Abort()
		return
	}

	c.Set(userIDKey, userID)
	c.Next()
}

// getRequestID извлекает ID запроса из контекста.
func (h *Handler) getRequestID(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

// getAccessToken извлекает access токен из заголовка Authorization.
func getAccessToken(c *gin.Context) (string, error) {
	token := c.GetHeader(authHeader)
	if token == "" {
		return "", errors.New("empty authorization header")
	}

	tokenPaths := strings.Split(token, " ")
	if len(tokenPaths) != 2 || tokenPaths[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}

	return tokenPaths[1], nil
}

// getUserID извлекает ID пользователя из контекста.
func getUserID(c *gin.Context) (string, error) {
	id, exists := c.Get(userIDKey)
	if !exists {
		return "", fmt.Errorf("user id not found in context")
	}

	userID, ok := id.(string)
	if !ok {
		return "", fmt.Errorf("invalid user id format")
	}

	if userID == "" {
		return "", fmt.Errorf("invalid user id")
	}

	return userID, nil
}

// getRefreshToken извлекает refresh токен из cookie.
func getRefreshToken(c *gin.Context) (string, error) {
	tokenID, err := c.Cookie(refreshToken)
	if err != nil || tokenID == "" {
		return "", fmt.Errorf("token ID not found in cookie: %v", err)
	}

	return tokenID, nil
}
