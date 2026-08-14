package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// newErrorResponse отправляет JSON-ответ с ошибкой и логирует её.
func (h *Handler) newErrorResponse(c *gin.Context, statusCode int, handler, message string, err error) {
	requestID := h.getRequestID(c)
	if requestID == "" {
		requestID = "unknown"
	}

	h.log.Error(handler, zap.String("request_id", requestID), zap.String("message", message), zap.Error(err))
	c.AbortWithStatusJSON(statusCode, gin.H{"error": message})
}

// newSuccessResponse отправляет успешный JSON-ответ.
func newSuccessResponse(c *gin.Context, statusCode int, field string, data interface{}) {
	c.JSON(statusCode, gin.H{field: data})
}
