package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) initProtocolRoutes(api *gin.RouterGroup) {
	group := api.Group("/protocol")
	{
		group.POST("/", h.createProtocol)
		group.GET("/", h.getProtocolList)
		group.GET("/full/:id", h.getProtocolFull)
		group.GET("/groups/:id", h.getProtocolsByGroupID)
		group.GET("/summary/:id", h.getGroupSummary)
		group.PUT("/:id/status", h.updateProtocolStatus)
		group.DELETE("/:id", h.deleteProtocol)
	}
}

func (h *Handler) createProtocol(c *gin.Context) {
	var req models.CreateProtocolRequest
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	protocol, err := h.protocol.CreateProtocolWithSample(c.Request.Context(), req)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": protocol.ID})
}

func (h *Handler) getProtocolList(c *gin.Context) {
	var p models.Paginated
	if err := c.BindQuery(&p); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	data, err := h.protocol.GetList(c.Request.Context(), p.Limit, p.Offset)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getProtocolFull(c *gin.Context) {
	id := c.Param("id")

	data, err := h.protocol.GetProtocolFull(c.Request.Context(), id)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getProtocolsByGroupID(c *gin.Context) {
	groupID := c.Param("id")
	data, err := h.protocol.GetProtocolsByGroupID(c.Request.Context(), groupID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getGroupSummary(c *gin.Context) {
	groupID := c.Param("group_id")
	data, err := h.protocol.GetGroupSummary(c.Request.Context(), groupID)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) updateProtocolStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required,oneof=draft completed archived"`
	}
	if err := c.BindJSON(&req); err != nil {
		newErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.protocol.UpdateProtocolStatus(c.Request.Context(), id, req.Status); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) deleteProtocol(c *gin.Context) {
	id := c.Param("id")

	if err := h.protocol.DeleteProtocol(c.Request.Context(), id); err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, nil)
}
