package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"desktop_lab/internal/models"
	"desktop_lab/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// QualityControlHandler обработчик для контроля качества
type QualityControlHandler struct {
	service *service.QualityControlService
	log     *zap.Logger
}

// NewQualityControlHandler создает новый обработчик
func NewQualityControlHandler(service *service.QualityControlService, log *zap.Logger) *QualityControlHandler {
	return &QualityControlHandler{service: service, log: log}
}

// RegisterRoutes регистрирует маршруты
func (h *QualityControlHandler) RegisterRoutes(rg *gin.RouterGroup) {
	qualityGroup := rg.Group("/quality")
	{
		// Audit logs
		qualityGroup.GET("/audit", h.getAuditLogs)
		qualityGroup.GET("/audit/:id", h.getAuditLogByID)

		// Protocol versions
		qualityGroup.GET("/protocols/:protocol_id/versions", h.getProtocolVersions)
		qualityGroup.GET("/protocol-versions/:id", h.getProtocolVersionByID)
		qualityGroup.GET("/protocol-versions/:id/content", h.getProtocolVersionContent)
		qualityGroup.GET("/protocol-versions/:id/pdf", h.getProtocolVersionPDF)
		qualityGroup.POST("/protocols/:protocol_id/versions", h.createProtocolVersion)
		qualityGroup.DELETE("/protocol-versions/:id", h.deleteProtocolVersion)

		// Protocol templates
		qualityGroup.GET("/templates", h.getProtocolTemplates)
		qualityGroup.GET("/templates/:id", h.getProtocolTemplate)
		qualityGroup.POST("/templates", h.createProtocolTemplate)
		qualityGroup.PUT("/templates/:id", h.updateProtocolTemplate)
		qualityGroup.DELETE("/templates/:id", h.deleteProtocolTemplate)
		qualityGroup.POST("/templates/:id/activate", h.activateProtocolTemplate)
	}
}

// ============================================================================
// AUDIT LOGS
// ============================================================================

// getAuditLogs GET /api/v1/quality/audit
func (h *QualityControlHandler) getAuditLogs(c *gin.Context) {
	filter := models.AuditLogFilter{}

	// Парсинг параметров
	if userID := c.Query("user_id"); userID != "" {
		if id, err := strconv.ParseInt(userID, 10, 64); err == nil {
			filter.UserID = &id
		}
	}
	if resourceType := c.Query("resource_type"); resourceType != "" {
		filter.ResourceType = &resourceType
	}
	if resourceID := c.Query("resource_id"); resourceID != "" {
		if id, err := strconv.ParseInt(resourceID, 10, 64); err == nil {
			filter.ResourceID = &id
		}
	}
	if action := c.Query("action"); action != "" {
		actionType := models.ActionType(action)
		filter.Action = &actionType
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			filter.Offset = o
		}
	}

	audits, total, err := h.service.GetAuditLogs(c.Request.Context(), filter)
	if err != nil {
		h.log.Error("failed to get audit logs", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   audits,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// getAuditLogByID GET /api/v1/quality/audit/:id
func (h *QualityControlHandler) getAuditLogByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	audit, err := h.service.GetAuditLogByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get audit log", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if audit == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": audit})
}

// ============================================================================
// PROTOCOL VERSIONS
// ============================================================================

// getProtocolVersions GET /api/v1/quality/protocols/:protocol_id/versions
func (h *QualityControlHandler) getProtocolVersions(c *gin.Context) {
	protocolID, err := strconv.ParseInt(c.Param("protocol_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid protocol_id"})
		return
	}

	versions, err := h.service.GetProtocolVersions(c.Request.Context(), protocolID)
	if err != nil {
		h.log.Error("failed to get protocol versions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": versions})
}

// getProtocolVersionByID GET /api/v1/quality/protocol-versions/:id
func (h *QualityControlHandler) getProtocolVersionByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	version, err := h.service.GetProtocolVersionByID(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get protocol version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if version == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "version not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": version})
}

// getProtocolVersionContent GET /api/v1/quality/protocol-versions/:id/content
func (h *QualityControlHandler) getProtocolVersionContent(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	content, err := h.service.GetProtocolVersionContent(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get protocol version content", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": content})
}

// getProtocolVersionPDF GET /api/v1/quality/protocol-versions/:id/pdf
func (h *QualityControlHandler) getProtocolVersionPDF(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	pdfData, err := h.service.GetProtocolVersionPDF(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get protocol version PDF", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if pdfData == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "PDF not found"})
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename=protocol_version.pdf")
	c.Data(http.StatusOK, "application/pdf", pdfData)
}

// createProtocolVersion POST /api/v1/quality/protocols/:protocol_id/versions
func (h *QualityControlHandler) createProtocolVersion(c *gin.Context) {
	protocolID, err := strconv.ParseInt(c.Param("protocol_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid protocol_id"})
		return
	}

	var req models.ProtocolVersionCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ProtocolID = protocolID

	// Получаем пользователя из контекста (должно быть добавлено middleware)
	userID, _ := c.Get("user_id")
	userName, _ := c.Get("user_name")

	if id, ok := userID.(int64); ok {
		req.ChangedBy = id
	}
	if name, ok := userName.(string); ok {
		req.ChangedByName = name
	}

	version, err := h.service.CreateProtocolVersion(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("failed to create protocol version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": version})
}

// deleteProtocolVersion DELETE /api/v1/quality/protocol-versions/:id
func (h *QualityControlHandler) deleteProtocolVersion(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	userID, _ := c.Get("user_id")
	deletedBy, _ := userID.(int64)

	err = h.service.DeleteProtocolVersion(c.Request.Context(), id, deletedBy)
	if err != nil {
		h.log.Error("failed to delete protocol version", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "version deleted successfully"})
}

// ============================================================================
// PROTOCOL TEMPLATES
// ============================================================================

// getProtocolTemplates GET /api/v1/quality/templates
func (h *QualityControlHandler) getProtocolTemplates(c *gin.Context) {
	orgID, _ := c.Get("organization_id")
	activeOnly := c.Query("active_only") == "true"

	var orgIDInt int64
	if id, ok := orgID.(int64); ok {
		orgIDInt = id
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization_id required"})
		return
	}

	templates, err := h.service.GetProtocolTemplatesByOrganization(c.Request.Context(), orgIDInt, activeOnly)
	if err != nil {
		h.log.Error("failed to get protocol templates", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": templates})
}

// getProtocolTemplate GET /api/v1/quality/templates/:id
func (h *QualityControlHandler) getProtocolTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	template, err := h.service.GetProtocolTemplate(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get protocol template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if template == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// createProtocolTemplate POST /api/v1/quality/templates
func (h *QualityControlHandler) createProtocolTemplate(c *gin.Context) {
	var req models.ProtocolTemplateCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Получаем organization_id и created_by из контекста
	orgID, _ := c.Get("organization_id")
	userID, _ := c.Get("user_id")

	if id, ok := orgID.(int64); ok {
		req.OrganizationID = id
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "organization_id required"})
		return
	}

	if id, ok := userID.(int64); ok {
		req.CreatedBy = id
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	template, err := h.service.CreateProtocolTemplate(c.Request.Context(), &req)
	if err != nil {
		h.log.Error("failed to create protocol template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": template})
}

// updateProtocolTemplate PUT /api/v1/quality/templates/:id
func (h *QualityControlHandler) updateProtocolTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req models.ProtocolTemplateUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template, err := h.service.UpdateProtocolTemplate(c.Request.Context(), id, &req)
	if err != nil {
		h.log.Error("failed to update protocol template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": template})
}

// deleteProtocolTemplate DELETE /api/v1/quality/templates/:id
func (h *QualityControlHandler) deleteProtocolTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.DeleteProtocolTemplate(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to delete protocol template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "template deleted successfully"})
}

// activateProtocolTemplate POST /api/v1/quality/templates/:id/activate
func (h *QualityControlHandler) activateProtocolTemplate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.SetProtocolTemplateAsActive(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to activate protocol template", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "template activated successfully"})
}
