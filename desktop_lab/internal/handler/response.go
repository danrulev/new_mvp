package handler

import "github.com/gin-gonic/gin"

func newErrorResponse(c *gin.Context, statusCode int, message string) {
	c.AbortWithStatusJSON(statusCode, gin.H{"error": message})
}

func newSuccessResponse(c *gin.Context, statusCode int, field string, data interface{}) {
	c.JSON(statusCode, gin.H{field: data})
}
