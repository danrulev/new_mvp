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

func (s *Handler) getRequestID(c *gin.Context) string {
	return c.GetString(requestIDKey)
}

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

func getUserID(c *gin.Context) (uuid.UUID, error) {
	id, exists := c.Get(userIDKey)
	if !exists {
		return uuid.Nil, fmt.Errorf("user id not found in context")
	}

	t, ok := id.(string)
	if !ok {
		return uuid.Nil, fmt.Errorf("invalid user id format")
	}

	userID, err := uuid.Parse(t)
	if err != nil {
		return uuid.Nil, fmt.Errorf("user id is not uuid")
	}

	if userID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("invalid user id")
	}

	return userID, nil
}

func getRefreshToken(c *gin.Context) (string, error) {
	tokenID, err := c.Cookie(refreshToken)
	if err != nil || tokenID == "" {
		return "", fmt.Errorf("token ID not found in cookie: %v", err)
	}

	return tokenID, nil
}
