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
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	group, err := h.group.Create(c, input.Name, input.ProjectName, input.Location, input.MaterialID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	newSuccessResponse(c, http.StatusOK, "id", group.ID)
}

func (h *Handler) getGroupList(c *gin.Context) {
	var p models.Paginated
	if err := c.BindQuery(&p); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.group.GetList(c.Request.Context(), p.Limit, p.Offset)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getGroupByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.group.GetByID(c.Request.Context(), id.String())
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) deleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.group.DeleteGroupByID(c.Request.Context(), id.String()); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
