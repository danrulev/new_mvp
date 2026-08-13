package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) initGroupRoutes(api *gin.RouterGroup) {
	// Группы экспериментов требуют аутентификации
	auth := api.Group("/group")
	auth.Use(h.authMiddleware)
	{
		// Создание группы - техник, инженер, админ
		auth.POST("/", h.permissionMiddleware(models.PermGroupCreate), h.createGroup)
		// Чтение списка - все аутентифицированные
		auth.GET("/", h.permissionMiddleware(models.PermGroupRead), h.getGroupList)
		// Чтение по ID - все аутентифицированные
		auth.GET("/:id", h.permissionMiddleware(models.PermGroupRead), h.getGroupByID)
		// Обновление группы - техник, инженер, админ
		auth.PUT("/:id", h.permissionMiddleware(models.PermGroupUpdate), h.updateGroup)
		// Удаление группы - только инженер и админ
		auth.DELETE("/:id", h.permissionMiddleware(models.PermGroupDelete), h.deleteGroup)
		
		// Добавление пробы в группу - техник, инженер, админ
		auth.PUT("/:id/samples/:sampleID", h.permissionMiddleware(models.PermGroupUpdate), h.addSampleToGroup)
		// Удаление пробы из группы - техник, инженер, админ
		auth.DELETE("/:id/samples/:sampleID", h.permissionMiddleware(models.PermGroupUpdate), h.removeSampleFromGroup)
		// Получение проб группы - все аутентифицированные
		auth.GET("/:id/samples", h.permissionMiddleware(models.PermGroupRead), h.getSamplesByGroupID)
	}
}

type CreateGroupRequest struct {
	Name        string `json:"name"`
	ProjectName string `json:"project_name"`
	Location    string `json:"location"`
	MaterialID  string `json:"material_id"`
}

func (h *Handler) createGroup(c *gin.Context) {
	var input CreateGroupRequest

	if err := c.BindJSON(&input); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createGroup", "invalid data", err)
		return
	}

	group, err := h.group.Create(c, input.Name, input.ProjectName, input.Location, input.MaterialID)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createGroup", "service error", err)
		return
	}

	newSuccessResponse(c, http.StatusOK, "id", group.ID)
}

func (h *Handler) getGroupList(c *gin.Context) {
	var filter models.GroupListFilter
	if err := c.BindQuery(&filter); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getGroupList", "invalid data", err)
		return
	}

	data, err := h.group.GetList(c.Request.Context(), filter)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getGroupList", "service error", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getGroupByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getGroupByID", "invalid id param", err)
		return
	}

	data, err := h.group.GetByID(c.Request.Context(), id.String())
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getGroupByID", "service error", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) updateGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "updateGroup", "id param is empty", nil)
		return
	}

	var req models.UpdateExperimentGroup
	if err := c.BindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "updateGroup", "invalid request body", err)
		return
	}

	err := h.group.UpdateGroupByID(c.Request.Context(), id, req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "updateGroup", "failed to update group", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) deleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "deleteGroup", "invalid id param", err)
		return
	}

	if err := h.group.DeleteGroupByID(c.Request.Context(), id.String()); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "deleteGroup", "service error", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// addSampleToGroup добавляет пробу в группу
func (h *Handler) addSampleToGroup(c *gin.Context) {
	groupID := c.Param("id")
	sampleID := c.Param("sampleID")

	if groupID == "" || sampleID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "addSampleToGroup", "group_id and sample_id are required", nil)
		return
	}

	if err := h.group.AddSampleToGroup(c.Request.Context(), sampleID, groupID); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "addSampleToGroup", "failed to add sample to group", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sample added to group"})
}

// removeSampleFromGroup удаляет пробу из группы
func (h *Handler) removeSampleFromGroup(c *gin.Context) {
	sampleID := c.Param("sampleID")

	if sampleID == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "removeSampleFromGroup", "sample_id is required", nil)
		return
	}

	if err := h.group.RemoveSampleFromGroup(c.Request.Context(), sampleID); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "removeSampleFromGroup", "failed to remove sample from group", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "sample removed from group"})
}

// getSamplesByGroupID возвращает все пробы группы
func (h *Handler) getSamplesByGroupID(c *gin.Context) {
	groupID := c.Param("id")
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
