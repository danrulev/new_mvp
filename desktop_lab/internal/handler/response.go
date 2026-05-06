package handler

import (
	contextkeys "desktop_lab/internal/contextKey"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *Handler) newErrorResponse(c *gin.Context, statusCode int, handler, message string, err error) {
	requestID := c.Value(contextkeys.RequestIDKey)
	if requestID == nil {
		requestID = "unknown"
	}

	h.log.Error(handler, zap.String("request_id", requestID.(string)), zap.String("message", message), zap.Error(err))
	c.AbortWithStatusJSON(statusCode, gin.H{"error": message})
}

func newSuccessResponse(c *gin.Context, statusCode int, field string, data interface{}) {
	c.JSON(statusCode, gin.H{field: data})
}
