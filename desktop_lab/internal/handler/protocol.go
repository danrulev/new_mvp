package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *Handler) initProtocolRoutes(api *gin.RouterGroup) {
	// Протоколы требуют аутентификации
	auth := api.Group("/protocol")
	auth.Use(h.authMiddleware)
	{
		// Создание протокола - инженер, админ
		auth.POST("/", h.permissionMiddleware(models.PermProtocolCreate), h.createProtocol)
		// Чтение списка - все аутентифицированные
		auth.GET("/", h.permissionMiddleware(models.PermProtocolRead), h.getProtocolList)
		// Полное чтение - все аутентифицированные
		auth.GET("/full/:id", h.permissionMiddleware(models.PermProtocolRead), h.getProtocolFull)
		// По группе - все аутентифицированные
		auth.GET("/groups/:id", h.permissionMiddleware(models.PermProtocolRead), h.getProtocolsByGroupID)
		// Сводка группы - все аутентифицированные
		auth.GET("/summary/:id", h.permissionMiddleware(models.PermReportRead), h.getGroupSummary)
		// Обновление статуса - инженер, админ
		auth.PUT("/:id/status", h.permissionMiddleware(models.PermProtocolUpdate), h.updateProtocolStatus)
		// Обновление протокола - инженер, админ
		auth.PUT("/:id", h.permissionMiddleware(models.PermProtocolUpdate), h.updateProtocol)
		// Удаление - только инженер и админ
		auth.DELETE("/:id", h.permissionMiddleware(models.PermProtocolDelete), h.deleteProtocol)
	}
}

func (h *Handler) createProtocol(c *gin.Context) {
	var req models.CreateProtocolRequest
	if err := c.BindJSON(&req); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createProtocol", "invalid data", err)
		return
	}

	h.log.Debug("create protocol", zap.Any("req", req))
	protocol, err := h.protocol.CreateProtocolWithSample(c.Request.Context(), req)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "createProtocol", "service error", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": protocol.ID})
}

func (h *Handler) getProtocolList(c *gin.Context) {
	var p models.ProtocolListFilter
	if err := c.BindQuery(&p); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getProtocolList", "invalid data", err)
		return
	}

	data, err := h.protocol.GetList(c.Request.Context(), p)
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

func (h *Handler) updateProtocol(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "getGroupSummary", "id param is empty", nil)
		return
	}
	var upd models.UpdateProtocolRequest
	if err := c.BindJSON(&upd); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "updateProtocol", "invalid data", err)
		return
	}

	if err := h.protocol.UpdateProtocol(c.Request.Context(), id, upd); err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "updateProtocol", "service error", err)
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
