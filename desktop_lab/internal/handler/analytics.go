package handler

import (
	"desktop_lab/internal/models"
	"net/http"

	"github.com/gin-gonic/gin"
)

// initAnalyticsRoutes регистрирует маршруты для аналитики и отчетов
func (h *Handler) initAnalyticsRoutes(api *gin.RouterGroup) {
	// Все маршруты аналитики требуют аутентификации
	auth := api.Group("")
	auth.Use(h.authMiddleware)
	{
		// Дашборд - доступен всем аутентифицированным пользователям
		auth.GET("/analytics/dashboard", h.permissionMiddleware(models.PermReportRead), h.getDashboard)
		
		// Аналитика заказов
		auth.GET("/analytics/orders", h.permissionMiddleware(models.PermReportRead), h.getOrdersAnalytics)
		
		// Аналитика качества
		auth.GET("/analytics/quality", h.permissionMiddleware(models.PermReportRead), h.getQualityAnalytics)
		
		// Финансовые отчеты - требуют права на финансовые отчеты
		auth.GET("/reports/financial", h.permissionMiddleware(models.PermReportRead), h.getFinancialReport)
	}
}

// getDashboard обрабатывает GET /api/analytics/dashboard
// Возвращает:
//   - заказы за период (график)
//   - выручка (график)
//   - топ клиентов
//   - топ тестов
//   - загрузка лаборатории
func (h *Handler) getDashboard(c *gin.Context) {
	var params models.DashboardQueryParams
	
	if err := c.ShouldBindQuery(&params); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "getDashboard", "invalid query parameters", err)
		return
	}

	dashboard, err := h.analytics.GetDashboardData(c.Request.Context(), params)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getDashboard", "failed to fetch dashboard data", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dashboard,
	})
}

// getOrdersAnalytics обрабатывает GET /api/analytics/orders
// Возвращает:
//   - конверсия заявок
//   - среднее время выполнения
//   - причины отклонений
func (h *Handler) getOrdersAnalytics(c *gin.Context) {
	var params models.OrdersAnalyticsQueryParams
	
	if err := c.ShouldBindQuery(&params); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "getOrdersAnalytics", "invalid query parameters", err)
		return
	}

	analytics, err := h.analytics.GetOrdersAnalytics(c.Request.Context(), params)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getOrdersAnalytics", "failed to fetch orders analytics", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": analytics,
	})
}

// getQualityAnalytics обрабатывает GET /api/analytics/quality
// Возвращает:
//   - процент несоответствий
//   - контрольные карты
//   - неопределенность измерений
func (h *Handler) getQualityAnalytics(c *gin.Context) {
	var params models.QualityAnalyticsQueryParams
	
	if err := c.ShouldBindQuery(&params); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "getQualityAnalytics", "invalid query parameters", err)
		return
	}

	analytics, err := h.analytics.GetQualityAnalytics(c.Request.Context(), params)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getQualityAnalytics", "failed to fetch quality analytics", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": analytics,
	})
}

// getFinancialReport обрабатывает GET /api/reports/financial
// Возвращает:
//   - P&L отчет
//   - дебиторская задолженность
//   - налоговые отчеты
func (h *Handler) getFinancialReport(c *gin.Context) {
	var params models.FinancialReportQueryParams
	
	if err := c.ShouldBindQuery(&params); err != nil {
		h.newErrorResponse(c, http.StatusBadRequest, "getFinancialReport", "invalid query parameters", err)
		return
	}

	report, err := h.analytics.GetFinancialReport(c.Request.Context(), params)
	if err != nil {
		h.newErrorResponse(c, http.StatusInternalServerError, "getFinancialReport", "failed to fetch financial report", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": report,
	})
}
