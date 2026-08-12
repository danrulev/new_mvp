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
		
		// Получение пробы по ID - техник, инженер, админ, клиент (только чтение)
		auth.GET("/:id", h.permissionMiddleware(models.PermSampleRead), h.getSampleByID)
		
		// Обновление пробы - техник, инженер, админ
		auth.PUT("/:id", h.permissionMiddleware(models.PermSampleUpdate), h.updateSample)
		
		// Удаление пробы - только админ и инженер
		auth.DELETE("/:id", h.permissionMiddleware(models.PermSampleDelete), h.deleteSample)
		
		// Получение проб группы - техник, инженер, админ
		auth.GET("/group/:groupID", h.permissionMiddleware(models.PermSampleRead), h.getSamplesByGroupID)
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

// getSampleByID возвращает пробу по ID
func (h *Handler) getSampleByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getSampleByID", "sample id is required", nil)
		return
	}

	sample, err := h.sample.GetSampleByID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getSampleByID", "service error", err)
		return
	}
	if sample.ID == "" {
		h.newErrorResponse(c, http.StatusNotFound, "getSampleByID", "sample not found", nil)
		return
	}

	c.JSON(http.StatusOK, sample)
}

// updateSample обновляет пробу
func (h *Handler) updateSample(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "updateSample", "sample id is required", nil)
		return
	}

	var dto models.UpdateSampleDTO
	if err := c.BindJSON(&dto); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "updateSample", "invalid data", err)
		return
	}

	sample, err := h.sample.UpdateSample(c.Request.Context(), id, dto)
	if err != nil {
		if err.Error() == "sample not found" {
			h.newErrorResponse(c, http.StatusNotFound, "updateSample", "sample not found", err)
			return
		}
		h.newErrorResponse(c, http.StatusInternalServerError, "updateSample", "service error", err)
		return
	}

	c.JSON(http.StatusOK, sample)
}

// deleteSample удаляет пробу
func (h *Handler) deleteSample(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "deleteSample", "sample id is required", nil)
		return
	}

	err := h.sample.DeleteSample(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "sample not found" {
			h.newErrorResponse(c, http.StatusNotFound, "deleteSample", "sample not found", err)
			return
		}
		h.newErrorResponse(c, http.StatusInternalServerError, "deleteSample", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "sample deleted successfully"})
}

// getSamplesByGroupID возвращает все пробы группы
func (h *Handler) getSamplesByGroupID(c *gin.Context) {
	groupID := c.Param("groupID")
	if groupID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getSamplesByGroupID", "group id is required", nil)
		return
	}

	samples, err := h.sample.GetSamplesByGroupID(c.Request.Context(), groupID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getSamplesByGroupID", "service error", err)
		return
	}

	if samples == nil {
		samples = []models.Sample{}
	}

	c.JSON(http.StatusOK, samples)
}
