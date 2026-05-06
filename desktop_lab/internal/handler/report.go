package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// initReportRoutes регистрирует маршруты для генерации отчётов
func (h *Handler) initReportRoutes(api *gin.RouterGroup) {
	group := api.Group("/report")
	{
		group.GET("/protocol/:id/pdf", h.downloadProtocolPDF)
		group.GET("/group/:id/pdf", h.downloadGroupSummaryPDF)
	}
}

// downloadProtocolPDF генерирует и отдаёт PDF протокола
func (h *Handler) downloadProtocolPDF(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		newErrorResponse(c, http.StatusBadRequest, "protocol ID is required")
		return
	}

	pdfBytes, err := h.report.GenerateProtocolPDF(c.Request.Context(), id)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
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
		newErrorResponse(c, http.StatusBadRequest, "group ID is required")
		return
	}

	pdfBytes, err := h.report.GenerateGroupSummaryPDF(c.Request.Context(), id)
	if err != nil {
		newErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("group_summary_%s.pdf", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
