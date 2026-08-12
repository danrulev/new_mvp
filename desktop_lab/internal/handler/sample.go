package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initSampleRoutes(api *gin.RouterGroup) {
	// Проба требуют аутентификации
	auth := api.Group("/sample")
	auth.Use(h.authMiddleware)
	{
		// Создание пробы - техник, инженер, админ
		auth.POST("/", h.permissionMiddleware(models.PermSampleCreate), h.createSample)
	}
}

func (h *Handler) createSample(c *gin.Context) {
	var dto models.CreateSampleDTO
	if err := c.BindJSON(&dto); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "createSample", "invalid data", err)
		return
	}

	groupID := c.PostForm("group_id")
	if groupID == "" {
		// Пробуем получить из JSON если не в form
		var req struct {
			GroupID string `json:"group_id"`
		}
		if err := c.BindJSON(&req); err != nil || req.GroupID == "" {
			h.newErrorResponse(c, http.StatusBadRequest, "createSample", "group_id is required", err)
			return
		}
		groupID = req.GroupID
	}

	sample, err := h.sample.CreateSample(c.Request.Context(), dto, groupID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createSample", "service error", err)
		return
	}

	c.JSON(http.StatusCreated, sample)
}
