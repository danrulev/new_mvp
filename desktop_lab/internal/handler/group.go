package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) initGroupRoutes(api *gin.RouterGroup) {
	group := api.Group("/group")
	{
		group.POST("/", h.createGroup)
		group.GET("/", h.getGroupList)
		group.GET("/:id", h.getGroupByID)
		group.PUT("/:id", h.updateGroup)
		group.DELETE("/:id", h.deleteGroup)
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
