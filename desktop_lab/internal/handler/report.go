package handler

import (
	"desktop_lab/internal/models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// initReportRoutes регистрирует маршруты для генерации отчётов
func (h *Handler) initReportRoutes(api *gin.RouterGroup) {
	// Отчеты требуют аутентификации
	auth := api.Group("/report")
	auth.Use(h.authMiddleware)
	{
		// Генерация PDF - все аутентифицированные с правом чтения отчетов
		auth.GET("/protocol/:id/pdf", h.permissionMiddleware(models.PermReportRead), h.downloadProtocolPDF)
		auth.GET("/group/:id/pdf", h.permissionMiddleware(models.PermReportRead), h.downloadGroupSummaryPDF)
	}
}

// downloadProtocolPDF генерирует и отдаёт PDF протокола
func (h *Handler) downloadProtocolPDF(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "downloadProtocolPDF", "id param is empty", nil)
		return
	}

	pdfBytes, err := h.report.GenerateProtocolPDF(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "downloadProtocolPDF", "service error", err)
		return
	}

	filename := fmt.Sprintf("protocol_%s.pdf", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// downloadGroupSummaryPDF генерирует и отдаёт сводный PDF по группе
func (h *Handler) downloadGroupSummaryPDF(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		h.newErrorResponse(c, http.StatusBadRequest, "downloadGroupSummaryPDF", "id param is empty", nil)
		return
	}

	pdfBytes, err := h.report.GenerateGroupSummaryPDF(c.Request.Context(), id)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "downloadGroupSummaryPDF", "service error", err)
		return
	}

	filename := fmt.Sprintf("group_summary_%s.pdf", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
