package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
)

// AnalyticsService предоставляет сервисы для аналитики и отчетности
type AnalyticsService struct {
	analyticsRepo AnalyticsRepo
	standardRepo  StandardRepo
	log           *zap.Logger
}

// NewAnalyticsService создает новый сервис аналитики
func NewAnalyticsService(
	analyticsRepo AnalyticsRepo,
	standardRepo StandardRepo,
	log *zap.Logger,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		standardRepo:  standardRepo,
		log:           log,
	}
}

// GetDashboardData возвращает данные для дашборда
func (s *AnalyticsService) GetDashboardData(ctx context.Context, params models.DashboardQueryParams) (models.DashboardData, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetDashboardData"),
	)
	log.Info("fetching dashboard data")

	dateFrom, dateTo, err := parseDateRange(params.DateFrom, params.DateTo)
	if err != nil {
		return models.DashboardData{}, fmt.Errorf("invalid date range: %w", err)
	}

	dashboard := models.DashboardData{}

	// Заказы за период
	log.Debug("fetching orders chart data")
	dashboard.OrdersChart, err = s.analyticsRepo.GetOrdersChartData(ctx, dateFrom, dateTo, params.OrganizationID)
	if err != nil {
		log.Warn("failed to fetch orders chart", zap.Error(err))
		dashboard.OrdersChart = []models.TimeSeriesData{}
	}

	// Выручка за период
	log.Debug("fetching revenue chart data")
	dashboard.RevenueChart, err = s.analyticsRepo.GetRevenueChartData(ctx, dateFrom, dateTo, params.OrganizationID)
	if err != nil {
		log.Warn("failed to fetch revenue chart", zap.Error(err))
		dashboard.RevenueChart = []models.TimeSeriesData{}
	}

	// Топ клиентов
	log.Debug("fetching top customers")
	dashboard.TopCustomers, err = s.analyticsRepo.GetTopCustomers(ctx, dateFrom, dateTo, params.OrganizationID, 10)
	if err != nil {
		log.Warn("failed to fetch top customers", zap.Error(err))
		dashboard.TopCustomers = []models.TopCustomer{}
	}

	// Топ тестов
	log.Debug("fetching top tests")
	dashboard.TopTests, err = s.analyticsRepo.GetTopTests(ctx, dateFrom, dateTo, params.OrganizationID, 10)
	if err != nil {
		log.Warn("failed to fetch top tests", zap.Error(err))
		dashboard.TopTests = []models.TopTest{}
	}

	// Загрузка лаборатории
	log.Debug("fetching lab utilization")
	dashboard.LabUtilization, err = s.analyticsRepo.GetLabUtilization(ctx, params.OrganizationID)
	if err != nil {
		log.Warn("failed to fetch lab utilization", zap.Error(err))
	}

	log.Info("dashboard data fetched successfully")
	return dashboard, nil
}

// GetOrdersAnalytics возвращает аналитику по заказам
func (s *AnalyticsService) GetOrdersAnalytics(ctx context.Context, params models.OrdersAnalyticsQueryParams) (models.OrdersAnalytics, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetOrdersAnalytics"),
	)
	log.Info("fetching orders analytics")

	dateFrom, dateTo, err := parseDateRange(params.DateFrom, params.DateTo)
	if err != nil {
		return models.OrdersAnalytics{}, fmt.Errorf("invalid date range: %w", err)
	}

	analytics := models.OrdersAnalytics{}

	// Конверсия заявок
	log.Debug("fetching conversion stats")
	funnel, err := s.analyticsRepo.GetOrderConversionStats(ctx, dateFrom, dateTo)
	if err != nil {
		log.Warn("failed to fetch conversion stats", zap.Error(err))
		funnel = models.ApplicationFunnel{}
	}
	analytics.ApplicationFunnel = funnel

	// Расчет конверсии
	if funnel.TotalApplications > 0 {
		analytics.ConversionRate = float64(funnel.Completed) / float64(funnel.TotalApplications)
	}

	// Среднее время выполнения
	log.Debug("fetching avg completion time")
	avgTime, err := s.analyticsRepo.GetAvgCompletionTime(ctx, dateFrom, dateTo)
	if err != nil {
		log.Warn("failed to fetch avg completion time", zap.Error(err))
		avgTime = 0
	}
	analytics.AvgCompletionTime = avgTime
	analytics.AvgCompletionHours = avgTime.Hours()

	// Причины отклонений
	log.Debug("fetching rejection reasons")
	analytics.RejectionReasons, err = s.analyticsRepo.GetRejectionReasons(ctx, dateFrom, dateTo)
	if err != nil {
		log.Warn("failed to fetch rejection reasons", zap.Error(err))
		analytics.RejectionReasons = []models.RejectionReason{}
	}

	log.Info("orders analytics fetched successfully")
	return analytics, nil
}

// GetQualityAnalytics возвращает аналитику по качеству
func (s *AnalyticsService) GetQualityAnalytics(ctx context.Context, params models.QualityAnalyticsQueryParams) (models.QualityAnalytics, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetQualityAnalytics"),
	)
	log.Info("fetching quality analytics")

	dateFrom, dateTo, err := parseDateRange(params.DateFrom, params.DateTo)
	if err != nil {
		return models.QualityAnalytics{}, fmt.Errorf("invalid date range: %w", err)
	}

	analytics := models.QualityAnalytics{}

	// Процент несоответствий
	log.Debug("fetching non-conformance rate")
	rate, err := s.analyticsRepo.GetNonConformanceRate(ctx, dateFrom, dateTo, params.TestMethodID)
	if err != nil {
		log.Warn("failed to fetch non-conformance rate", zap.Error(err))
		rate = 0
	}
	analytics.NonConformanceRate = rate
	analytics.NonConformancePercent = fmt.Sprintf("%.2f%%", rate*100)

	// Контрольные карты
	if params.TestMethodID != "" {
		log.Debug("fetching control chart data", zap.String("test_method_id", params.TestMethodID))

		// Получаем информацию о методе
		methodInfo, err := s.standardRepo.GetTestMethod(ctx, params.TestMethodID)
		if err != nil {
			log.Warn("failed to fetch method info", zap.Error(err))
		}

		// Получаем данные для контрольной карты
		dataPoints, err := s.analyticsRepo.GetControlChartData(ctx, dateFrom, dateTo, params.TestMethodID)
		if err != nil {
			log.Warn("failed to fetch control chart data", zap.Error(err))
			dataPoints = []models.ControlChartData{}
		}

		// Рассчитываем контрольные пределы
		controlLimits := s.calculateControlLimits(dataPoints)

		analytics.ControlCharts = []models.ControlChart{
			{
				TestMethodID:   params.TestMethodID,
				TestMethodName: methodInfo.Name,
				Unit:           methodInfo.Unit,
				DataPoints:     dataPoints,
				ControlLimits:  controlLimits,
			},
		}
	}

	// Неопределенность измерений
	if params.TestMethodID != "" {
		log.Debug("fetching measurement uncertainty", zap.String("test_method_id", params.TestMethodID))
		mu, err := s.analyticsRepo.GetMeasurementUncertainty(ctx, params.TestMethodID)
		if err != nil {
			log.Warn("failed to fetch measurement uncertainty", zap.Error(err))
		} else {
			// Расчет расширенной неопределенности
			mu.ExpandedUncertainty = mu.Uncertainty * mu.CoverageFactor
			analytics.MeasurementUncertainty = []models.MeasurementUncertainty{mu}
		}
	}

	log.Info("quality analytics fetched successfully")
	return analytics, nil
}

// GetFinancialReport возвращает финансовый отчет
func (s *AnalyticsService) GetFinancialReport(ctx context.Context, params models.FinancialReportQueryParams) (models.FinancialReport, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetFinancialReport"),
	)
	log.Info("fetching financial report")

	periodStart, periodEnd, err := parseDateRange(params.PeriodStart, params.PeriodEnd)
	if err != nil {
		return models.FinancialReport{}, fmt.Errorf("invalid period: %w", err)
	}

	report := models.FinancialReport{}

	// P&L отчет
	log.Debug("fetching P&L report")
	report.PLReport, err = s.analyticsRepo.GetPLReport(ctx, periodStart, periodEnd)
	if err != nil {
		log.Warn("failed to fetch P&L report", zap.Error(err))
		report.PLReport = models.PLReport{
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Currency:    "RUB",
		}
	}

	// Дебиторская задолженность
	log.Debug("fetching accounts receivable")
	report.AccountsReceivable, err = s.analyticsRepo.GetAccountsReceivable(ctx, periodEnd)
	if err != nil {
		log.Warn("failed to fetch accounts receivable", zap.Error(err))
		report.AccountsReceivable = []models.AccountsReceivable{}
	}

	// Налоговые отчеты
	log.Debug("fetching tax reports")
	report.TaxReports, err = s.analyticsRepo.GetTaxReports(ctx, periodStart, periodEnd)
	if err != nil {
		log.Warn("failed to fetch tax reports", zap.Error(err))
		report.TaxReports = []models.TaxReport{}
	}

	log.Info("financial report fetched successfully")
	return report, nil
}

// calculateControlLimits рассчитывает контрольные пределы для контрольной карты
func (s *AnalyticsService) calculateControlLimits(dataPoints []models.ControlChartData) models.ControlLimits {
	if len(dataPoints) == 0 {
		return models.ControlLimits{}
	}

	// Расчет среднего значения (CL)
	var sum float64
	for _, dp := range dataPoints {
		sum += dp.Value
	}
	cl := sum / float64(len(dataPoints))

	// Расчет стандартного отклонения
	var varianceSum float64
	for _, dp := range dataPoints {
		diff := dp.Value - cl
		varianceSum += diff * diff
	}
	stdDev := math.Sqrt(varianceSum / float64(len(dataPoints)))

	// Контрольные пределы (3 сигмы)
	ucl := cl + 3*stdDev
	lcl := cl - 3*stdDev

	// Предупредительные пределы (2 сигмы)
	uwl := cl + 2*stdDev
	lwl := cl - 2*stdDev

	return models.ControlLimits{
		UCL: ucl,
		CL:  cl,
		LCL: lcl,
		UWL: uwl,
		LWL: lwl,
	}
}

// parseDateRange парсит диапазон дат из строк
func parseDateRange(dateFromStr, dateToStr string) (time.Time, time.Time, error) {
	var dateFrom, dateTo time.Time
	var err error

	if dateFromStr == "" {
		// По умолчанию - последний месяц
		dateFrom = time.Now().AddDate(0, -1, 0)
	} else {
		dateFrom, err = time.Parse("2006-01-02", dateFromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date_from format: %w", err)
		}
	}

	if dateToStr == "" {
		dateTo = time.Now()
	} else {
		dateTo, err = time.Parse("2006-01-02", dateToStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid date_to format: %w", err)
		}
	}

	return dateFrom, dateTo, nil
}
