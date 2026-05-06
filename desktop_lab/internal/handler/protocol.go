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
		h.newErrorResponse(c, http.StatusInternalServerError, "createProtocol", "invalid data", err)
		return
	}

	protocol, err := h.protocol.CreateProtocolWithSample(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createProtocol", "service error", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": protocol.ID})
}

func (h *Handler) getProtocolList(c *gin.Context) {
	var p models.Paginated
	if err := c.BindQuery(&p); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getProtocolList", "invalid data", err)
		return
	}

	data, err := h.protocol.GetList(c.Request.Context(), p.Limit, p.Offset)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getProtocolList", "service error", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getProtocolFull(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getProtocolFull", "id param is empty", nil)
		return
	}

	data, err := h.protocol.GetProtocolFull(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getProtocolFull", "service error", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getProtocolsByGroupID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getProtocolsByGroupID", "id param is empty", nil)
		return
	}

	data, err := h.protocol.GetProtocolsByGroupID(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getProtocolsByGroupID", "service error", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) getGroupSummary(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getGroupSummary", "id param is empty", nil)
		return
	}

	data, err := h.protocol.GetGroupSummary(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getGroupSummary", "service error", err)
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *Handler) updateProtocolStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getGroupSummary", "id param is empty", nil)
		return
	}

	var req struct {
		Status string `json:"status" binding:"required,oneof=draft completed archived"`
	}
	if err := c.BindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "updateProtocolStatus", "invalid data", err)
		return
	}

	if err := h.protocol.UpdateProtocolStatus(c.Request.Context(), id, req.Status); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "updateProtocolStatus", "service error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "updated"})
}

func (h *Handler) deleteProtocol(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "deleteProtocol", "id param is empty", nil)
		return
	}

	if err := h.protocol.DeleteProtocol(c.Request.Context(), id); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "deleteProtocol", "service error", err)
		return
	}

	c.JSON(http.StatusOK, nil)
}
