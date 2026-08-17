package models

import "time"

// DashboardData представляет данные для дашборда аналитики
type DashboardData struct {
	OrdersChart      []TimeSeriesData `json:"orders_chart"`
	RevenueChart     []TimeSeriesData `json:"revenue_chart"`
	TopCustomers     []TopCustomer    `json:"top_customers"`
	TopTests         []TopTest        `json:"top_tests"`
	LabUtilization   LabUtilization   `json:"lab_utilization"`
}

// TimeSeriesData представляет данные временного ряда
type TimeSeriesData struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
	Label string    `json:"label,omitempty"`
}

// TopCustomer представляет топ клиента
type TopCustomer struct {
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	OrderCount   int     `json:"order_count"`
	TotalAmount  float64 `json:"total_amount"`
}

// TopTest представляет топ теста
type TopTest struct {
	TestMethodID   string  `json:"test_method_id"`
	TestMethodName string  `json:"test_method_name"`
	ExecutionCount int     `json:"execution_count"`
	AvgPrice       float64 `json:"avg_price,omitempty"`
}

// LabUtilization представляет загрузку лаборатории
type LabUtilization struct {
	CurrentLoad      float64   `json:"current_load"` // 0.0 - 1.0
	ActiveOrders     int       `json:"active_orders"`
	CompletedToday   int       `json:"completed_today"`
	PendingSamples   int       `json:"pending_samples"`
	CapacityUsage    float64   `json:"capacity_usage"` // 0.0 - 1.0
}

// OrdersAnalytics представляет аналитику по заказам
type OrdersAnalytics struct {
	ConversionRate       float64              `json:"conversion_rate"` // 0.0 - 1.0
	ApplicationFunnel    ApplicationFunnel    `json:"application_funnel"`
	AvgCompletionTime    time.Duration        `json:"avg_completion_time"`
	AvgCompletionHours   float64              `json:"avg_completion_hours"`
	RejectionReasons     []RejectionReason    `json:"rejection_reasons"`
}

// ApplicationFunnel представляет воронку конверсии заявок
type ApplicationFunnel struct {
	TotalApplications int `json:"total_applications"`
	Pending           int `json:"pending"`
	Accepted          int `json:"accepted"`
	InProgress        int `json:"in_progress"`
	Completed         int `json:"completed"`
	Rejected          int `json:"rejected"`
	Cancelled         int `json:"cancelled"`
}

// RejectionReason представляет причину отклонения
type RejectionReason struct {
	Reason     string `json:"reason"`
	Count      int    `json:"count"`
	Percentage float64 `json:"percentage"` // 0.0 - 100.0
}

// QualityAnalytics представляет аналитику по качеству
type QualityAnalytics struct {
	NonConformanceRate    float64           `json:"non_conformance_rate"` // 0.0 - 1.0
	NonConformancePercent string            `json:"non_conformance_percent"`
	ControlCharts         []ControlChart    `json:"control_charts"`
	MeasurementUncertainty []MeasurementUncertainty `json:"measurement_uncertainty"`
}

// ControlChart представляет контрольную карту
type ControlChart struct {
	TestMethodID   string             `json:"test_method_id"`
	TestMethodName string             `json:"test_method_name"`
	Unit           string             `json:"unit"`
	DataPoints     []ControlChartData `json:"data_points"`
	ControlLimits  ControlLimits      `json:"control_limits"`
}

// ControlChartData представляет точку данных контрольной карты
type ControlChartData struct {
	SampleNumber string    `json:"sample_number"`
	Value        float64   `json:"value"`
	TestDate     time.Time `json:"test_date"`
	InControl    bool      `json:"in_control"`
}

// ControlLimits представляет контрольные пределы
type ControlLimits struct {
	UCL  float64 `json:"ucl"`  // Upper Control Limit
	CL   float64 `json:"cl"`   // Center Line (mean)
	LCL  float64 `json:"lcl"`  // Lower Control Limit
	UWL  float64 `json:"uwl"`  // Upper Warning Limit
	LWL  float64 `json:"lwl"`  // Lower Warning Limit
}

// MeasurementUncertainty представляет неопределенность измерений
type MeasurementUncertainty struct {
	TestMethodID    string  `json:"test_method_id"`
	TestMethodName  string  `json:"test_method_name"`
	Unit            string  `json:"unit"`
	Uncertainty     float64 `json:"uncertainty"`
	CoverageFactor  float64 `json:"coverage_factor"` // k-factor
	ConfidenceLevel float64 `json:"confidence_level"` // 0.95 for 95%
	ExpandedUncertainty float64 `json:"expanded_uncertainty"`
}

// FinancialReport представляет финансовый отчет
type FinancialReport struct {
	PLReport              PLReport              `json:"pl_report"`
	AccountsReceivable    []AccountsReceivable  `json:"accounts_receivable"`
	TaxReports            []TaxReport           `json:"tax_reports"`
}

// PLReport представляет отчет о прибылях и убытках
type PLReport struct {
	PeriodStart      time.Time `json:"period_start"`
	PeriodEnd        time.Time `json:"period_end"`
	Revenue          float64   `json:"revenue"`
	CostOfGoodsSold  float64   `json:"cost_of_goods_sold"`
	GrossProfit      float64   `json:"gross_profit"`
	OperatingExpenses float64  `json:"operating_expenses"`
	OperatingIncome  float64   `json:"operating_income"`
	NetIncome        float64   `json:"net_income"`
	Currency         string    `json:"currency"`
}

// AccountsReceivable представляет дебиторскую задолженность
type AccountsReceivable struct {
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	Amount           float64   `json:"amount"`
	DueDate          time.Time `json:"due_date"`
	OverdueDays      int       `json:"overdue_days"`
	Status           string    `json:"status"` // current, overdue, paid
}

// TaxReport представляет налоговый отчет
type TaxReport struct {
	ReportType     string    `json:"report_type"` // VAT, Profit, etc.
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	TaxBase        float64   `json:"tax_base"`
	TaxRate        float64   `json:"tax_rate"`
	TaxAmount      float64   `json:"tax_amount"`
	Submitted      bool      `json:"submitted"`
	SubmittedDate  *time.Time `json:"submitted_date,omitempty"`
}

// AnalyticsDateRange представляет диапазон дат для аналитики
type AnalyticsDateRange struct {
	DateFrom time.Time `json:"date_from"`
	DateTo   time.Time `json:"date_to"`
}

// DashboardQueryParams представляет параметры запроса дашборда
type DashboardQueryParams struct {
	DateFrom       string `form:"date_from"`
	DateTo         string `form:"date_to"`
	OrganizationID string `form:"organization_id"`
}

// OrdersAnalyticsQueryParams представляет параметры запроса аналитики заказов
type OrdersAnalyticsQueryParams struct {
	DateFrom string `form:"date_from"`
	DateTo   string `form:"date_to"`
}

// QualityAnalyticsQueryParams представляет параметры запроса аналитики качества
type QualityAnalyticsQueryParams struct {
	DateFrom     string `form:"date_from"`
	DateTo       string `form:"date_to"`
	TestMethodID string `form:"test_method_id"`
}

// FinancialReportQueryParams представляет параметры запроса финансового отчета
type FinancialReportQueryParams struct {
	PeriodStart string `form:"period_start"`
	PeriodEnd   string `form:"period_end"`
}
