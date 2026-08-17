package mysql_repo

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"fmt"
	"time"
)

// AnalyticsRepositoryImpl реализует интерфейс AnalyticsRepository
type AnalyticsRepositoryImpl struct {
	db *sql.DB
}

// NewAnalyticsRepository создает новый экземпляр репозитория аналитики
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepositoryImpl {
	return &AnalyticsRepositoryImpl{db: db}
}

// GetOrdersChartData возвращает данные по заказам за период
func (r *AnalyticsRepositoryImpl) GetOrdersChartData(ctx context.Context, dateFrom, dateTo time.Time, organizationID string) ([]models.TimeSeriesData, error) {
	query := `
		SELECT DATE(created_at) as date, COUNT(*) as count
		FROM orders
		WHERE created_at >= $1 AND created_at <= $2
	`
	args := []interface{}{dateFrom, dateTo}
	
	if organizationID != "" {
		query += " AND organization_id = $3"
		args = append(args, organizationID)
	}
	
	query += " GROUP BY DATE(created_at) ORDER BY date"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query orders chart data: %w", err)
	}
	defer rows.Close()

	var results []models.TimeSeriesData
	for rows.Next() {
		var tsd models.TimeSeriesData
		err := rows.Scan(&tsd.Date, &tsd.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order chart row: %w", err)
		}
		results = append(results, tsd)
	}

	return results, rows.Err()
}

// GetRevenueChartData возвращает данные по выручке за период
func (r *AnalyticsRepositoryImpl) GetRevenueChartData(ctx context.Context, dateFrom, dateTo time.Time, organizationID string) ([]models.TimeSeriesData, error) {
	query := `
		SELECT DATE(o.created_at) as date, SUM(o.total_amount) as revenue
		FROM orders o
		WHERE o.created_at >= $1 AND o.created_at <= $2 AND o.status NOT IN ('cancelled', 'draft')
	`
	args := []interface{}{dateFrom, dateTo}
	
	if organizationID != "" {
		query += " AND o.organization_id = $3"
		args = append(args, organizationID)
	}
	
	query += " GROUP BY DATE(o.created_at) ORDER BY date"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query revenue chart data: %w", err)
	}
	defer rows.Close()

	var results []models.TimeSeriesData
	for rows.Next() {
		var tsd models.TimeSeriesData
		err := rows.Scan(&tsd.Date, &tsd.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan revenue chart row: %w", err)
		}
		results = append(results, tsd)
	}

	return results, rows.Err()
}

// GetTopCustomers возвращает топ клиентов
func (r *AnalyticsRepositoryImpl) GetTopCustomers(ctx context.Context, dateFrom, dateTo time.Time, organizationID string, limit int) ([]models.TopCustomer, error) {
	query := `
		SELECT 
			o.customer_id,
			o.customer_name,
			COUNT(DISTINCT o.id) as order_count,
			SUM(o.total_amount) as total_amount
		FROM orders o
		WHERE o.created_at >= $1 AND o.created_at <= $2 AND o.status NOT IN ('cancelled', 'draft')
	`
	args := []interface{}{dateFrom, dateTo}
	
	if organizationID != "" {
		query += " AND o.organization_id = $3"
		args = append(args, organizationID)
	}
	
	query += " GROUP BY o.customer_id, o.customer_name ORDER BY total_amount DESC LIMIT $4"
	if organizationID != "" {
		query = query[:len(query)-1] + ", $4" // replace last placeholder
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top customers: %w", err)
	}
	defer rows.Close()

	var results []models.TopCustomer
	for rows.Next() {
		var tc models.TopCustomer
		err := rows.Scan(&tc.CustomerID, &tc.CustomerName, &tc.OrderCount, &tc.TotalAmount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top customer row: %w", err)
		}
		results = append(results, tc)
	}

	return results, rows.Err()
}

// GetTopTests возвращает топ тестов
func (r *AnalyticsRepositoryImpl) GetTopTests(ctx context.Context, dateFrom, dateTo time.Time, organizationID string, limit int) ([]models.TopTest, error) {
	query := `
		SELECT 
			oi.test_method_id,
			oi.test_method_name,
			COUNT(oi.id) as execution_count,
			AVG(oi.unit_price) as avg_price
		FROM order_items oi
		JOIN orders o ON oi.order_id = o.id
		WHERE o.created_at >= $1 AND o.created_at <= $2 AND o.status NOT IN ('cancelled', 'draft')
	`
	args := []interface{}{dateFrom, dateTo}
	
	if organizationID != "" {
		query += " AND o.organization_id = $3"
		args = append(args, organizationID)
	}
	
	query += " GROUP BY oi.test_method_id, oi.test_method_name ORDER BY execution_count DESC LIMIT $4"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query top tests: %w", err)
	}
	defer rows.Close()

	var results []models.TopTest
	for rows.Next() {
		var tt models.TopTest
		var avgPrice sql.NullFloat64
		err := rows.Scan(&tt.TestMethodID, &tt.TestMethodName, &tt.ExecutionCount, &avgPrice)
		if err != nil {
			return nil, fmt.Errorf("failed to scan top test row: %w", err)
		}
		if avgPrice.Valid {
			tt.AvgPrice = avgPrice.Float64
		}
		results = append(results, tt)
	}

	return results, rows.Err()
}

// GetLabUtilization возвращает данные по загрузке лаборатории
func (r *AnalyticsRepositoryImpl) GetLabUtilization(ctx context.Context, organizationID string) (models.LabUtilization, error) {
	utilization := models.LabUtilization{}

	// Активные заказы
	query := `SELECT COUNT(*) FROM orders WHERE status IN ('pending', 'accepted', 'in_progress')`
	if organizationID != "" {
		query += fmt.Sprintf(" AND organization_id = '%s'", organizationID)
	}
	err := r.db.QueryRowContext(ctx, query).Scan(&utilization.ActiveOrders)
	if err != nil {
		return utilization, fmt.Errorf("failed to get active orders: %w", err)
	}

	// Завершено сегодня
	today := time.Now().Truncate(24 * time.Hour)
	query = `SELECT COUNT(*) FROM orders WHERE status = 'completed' AND completed_at >= $1`
	err = r.db.QueryRowContext(ctx, query, today).Scan(&utilization.CompletedToday)
	if err != nil {
		return utilization, fmt.Errorf("failed to get completed today: %w", err)
	}

	// Ожидающие образцы
	query = `SELECT COUNT(*) FROM samples WHERE status = 'pending'`
	if organizationID != "" {
		query += fmt.Sprintf(" AND organization_id = '%s'", organizationID)
	}
	err = r.db.QueryRowContext(ctx, query).Scan(&utilization.PendingSamples)
	if err != nil {
		return utilization, fmt.Errorf("failed to get pending samples: %w", err)
	}

	// Расчет загрузки (упрощенно)
	if utilization.ActiveOrders > 0 {
		utilization.CurrentLoad = float64(utilization.ActiveOrders) / 100.0 // нормализация
		if utilization.CurrentLoad > 1.0 {
			utilization.CurrentLoad = 1.0
		}
		utilization.CapacityUsage = utilization.CurrentLoad
	}

	return utilization, nil
}

// GetOrderConversionStats возвращает статистику конверсии заявок
func (r *AnalyticsRepositoryImpl) GetOrderConversionStats(ctx context.Context, dateFrom, dateTo time.Time) (models.ApplicationFunnel, error) {
	funnel := models.ApplicationFunnel{}

	query := `
		SELECT 
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'accepted') as accepted,
			COUNT(*) FILTER (WHERE status = 'in_progress') as in_progress,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'rejected') as rejected,
			COUNT(*) FILTER (WHERE status = 'cancelled') as cancelled
		FROM orders
		WHERE created_at >= $1 AND created_at <= $2
	`

	err := r.db.QueryRowContext(ctx, query, dateFrom, dateTo).Scan(
		&funnel.Pending, &funnel.Accepted, &funnel.InProgress,
		&funnel.Completed, &funnel.Rejected, &funnel.Cancelled,
	)
	if err != nil {
		return funnel, fmt.Errorf("failed to get conversion stats: %w", err)
	}

	funnel.TotalApplications = funnel.Pending + funnel.Accepted + funnel.InProgress +
		funnel.Completed + funnel.Rejected + funnel.Cancelled

	return funnel, nil
}

// GetAvgCompletionTime возвращает среднее время выполнения заказов
func (r *AnalyticsRepositoryImpl) GetAvgCompletionTime(ctx context.Context, dateFrom, dateTo time.Time) (time.Duration, error) {
	query := `
		SELECT AVG(EXTRACT(EPOCH FROM (completed_at - created_at))) as avg_seconds
		FROM orders
		WHERE status = 'completed' AND completed_at >= $1 AND completed_at <= $2
	`

	var avgSeconds sql.NullFloat64
	err := r.db.QueryRowContext(ctx, query, dateFrom, dateTo).Scan(&avgSeconds)
	if err != nil {
		return 0, fmt.Errorf("failed to get avg completion time: %w", err)
	}

	if !avgSeconds.Valid {
		return 0, nil
	}

	return time.Duration(avgSeconds.Float64 * float64(time.Second)), nil
}

// GetRejectionReasons возвращает причины отклонений
func (r *AnalyticsRepositoryImpl) GetRejectionReasons(ctx context.Context, dateFrom, dateTo time.Time) ([]models.RejectionReason, error) {
	query := `
		SELECT comment as reason, COUNT(*) as count
		FROM order_workflow_entries
		WHERE to_status = 'rejected' AND created_at >= $1 AND created_at <= $2
		GROUP BY comment
		ORDER BY count DESC
	`

	rows, err := r.db.QueryContext(ctx, query, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("failed to query rejection reasons: %w", err)
	}
	defer rows.Close()

	var results []models.RejectionReason
	var totalCount int
	
	// Сначала получаем общее количество
	countQuery := `SELECT COUNT(*) FROM order_workflow_entries WHERE to_status = 'rejected' AND created_at >= $1 AND created_at <= $2`
	r.db.QueryRowContext(ctx, countQuery, dateFrom, dateTo).Scan(&totalCount)

	for rows.Next() {
		var rr models.RejectionReason
		err := rows.Scan(&rr.Reason, &rr.Count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rejection reason row: %w", err)
		}
		if totalCount > 0 {
			rr.Percentage = float64(rr.Count) / float64(totalCount) * 100.0
		}
		results = append(results, rr)
	}

	return results, rows.Err()
}

// GetNonConformanceRate возвращает процент несоответствий
func (r *AnalyticsRepositoryImpl) GetNonConformanceRate(ctx context.Context, dateFrom, dateTo time.Time, testMethodID string) (float64, error) {
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE is_compliant = false) as non_compliant,
			COUNT(*) as total
		FROM test_results tr
		JOIN protocols p ON tr.protocol_id = p.id
		WHERE p.test_date >= $1 AND p.test_date <= $2
	`
	args := []interface{}{dateFrom, dateTo}
	
	if testMethodID != "" {
		query += " AND tr.method_id = $3"
		args = append(args, testMethodID)
	}

	var nonCompliant, total int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&nonCompliant, &total)
	if err != nil {
		return 0, fmt.Errorf("failed to get non-conformance rate: %w", err)
	}

	if total == 0 {
		return 0, nil
	}

	return float64(nonCompliant) / float64(total), nil
}

// GetControlChartData возвращает данные для контрольной карты
func (r *AnalyticsRepositoryImpl) GetControlChartData(ctx context.Context, dateFrom, dateTo time.Time, testMethodID string) ([]models.ControlChartData, error) {
	query := `
		SELECT 
			s.sample_number,
			tr.value,
			p.test_date,
			tr.is_compliant
		FROM test_results tr
		JOIN protocols p ON tr.protocol_id = p.id
		JOIN samples s ON tr.sample_id = s.id
		WHERE tr.method_id = $1 AND p.test_date >= $2 AND p.test_date <= $3
		ORDER BY p.test_date
	`

	rows, err := r.db.QueryContext(ctx, query, testMethodID, dateFrom, dateTo)
	if err != nil {
		return nil, fmt.Errorf("failed to query control chart data: %w", err)
	}
	defer rows.Close()

	var results []models.ControlChartData
	for rows.Next() {
		var ccd models.ControlChartData
		err := rows.Scan(&ccd.SampleNumber, &ccd.Value, &ccd.TestDate, &ccd.InControl)
		if err != nil {
			return nil, fmt.Errorf("failed to scan control chart row: %w", err)
		}
		results = append(results, ccd)
	}

	return results, rows.Err()
}

// GetTestMethodInfo возвращает информацию о методе теста
func (r *AnalyticsRepositoryImpl) GetTestMethodInfo(ctx context.Context, testMethodID string) (models.TestMethodFull, error) {
	// Эта функция должна быть делегирована standardRepo
	// Здесь заглушка для интерфейса
	return models.TestMethodFull{}, nil
}

// GetMeasurementUncertainty возвращает неопределенность измерений
func (r *AnalyticsRepositoryImpl) GetMeasurementUncertainty(ctx context.Context, testMethodID string) (models.MeasurementUncertainty, error) {
	// Заглушка - данные могут храниться в отдельной таблице
	mu := models.MeasurementUncertainty{
		TestMethodID:    testMethodID,
		CoverageFactor:  2.0,
		ConfidenceLevel: 0.95,
	}
	return mu, nil
}

// GetPLReport возвращает отчет P&L
func (r *AnalyticsRepositoryImpl) GetPLReport(ctx context.Context, periodStart, periodEnd time.Time) (models.PLReport, error) {
	pl := models.PLReport{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Currency:    "RUB",
	}

	// Выручка
	revenueQuery := `
		SELECT COALESCE(SUM(total_amount), 0)
		FROM orders
		WHERE status NOT IN ('cancelled', 'draft') AND created_at >= $1 AND created_at <= $2
	`
	err := r.db.QueryRowContext(ctx, revenueQuery, periodStart, periodEnd).Scan(&pl.Revenue)
	if err != nil {
		return pl, fmt.Errorf("failed to get revenue: %w", err)
	}

	// Упрощенный расчет себестоимости (30% от выручки)
	pl.CostOfGoodsSold = pl.Revenue * 0.3
	pl.GrossProfit = pl.Revenue - pl.CostOfGoodsSold
	
	// Операционные расходы (упрощенно 20% от выручки)
	pl.OperatingExpenses = pl.Revenue * 0.2
	pl.OperatingIncome = pl.GrossProfit - pl.OperatingExpenses
	
	// Чистая прибыль (налог 20%)
	tax := pl.OperatingIncome * 0.2
	pl.NetIncome = pl.OperatingIncome - tax

	return pl, nil
}

// GetAccountsReceivable возвращает дебиторскую задолженность
func (r *AnalyticsRepositoryImpl) GetAccountsReceivable(ctx context.Context, periodEnd time.Time) ([]models.AccountsReceivable, error) {
	query := `
		SELECT DISTINCT
			o.organization_id,
			o.customer_name as organization_name,
			SUM(o.total_amount) as amount,
			MAX(o.created_at) as due_date
		FROM orders o
		WHERE o.status IN ('completed', 'in_progress') AND o.created_at <= $1
		GROUP BY o.organization_id, o.customer_name
		ORDER BY amount DESC
	`

	rows, err := r.db.QueryContext(ctx, query, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("failed to query accounts receivable: %w", err)
	}
	defer rows.Close()

	var results []models.AccountsReceivable
	now := time.Now()
	
	for rows.Next() {
		var ar models.AccountsReceivable
		err := rows.Scan(&ar.OrganizationID, &ar.OrganizationName, &ar.Amount, &ar.DueDate)
		if err != nil {
			return nil, fmt.Errorf("failed to scan AR row: %w", err)
		}
		
		// Расчет дней просрочки
		daysOverdue := int(now.Sub(ar.DueDate).Hours() / 24)
		if daysOverdue < 0 {
			daysOverdue = 0
			ar.Status = "current"
		} else {
			ar.Status = "overdue"
		}
		ar.OverdueDays = daysOverdue
		
		results = append(results, ar)
	}

	return results, rows.Err()
}

// GetTaxReports возвращает налоговые отчеты
func (r *AnalyticsRepositoryImpl) GetTaxReports(ctx context.Context, periodStart, periodEnd time.Time) ([]models.TaxReport, error) {
	// Заглушка - реальная реализация зависит от структуры данных
	reports := []models.TaxReport{
		{
			ReportType:  "VAT",
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			TaxBase:     1000000.0,
			TaxRate:     0.20,
			TaxAmount:   200000.0,
			Submitted:   false,
		},
		{
			ReportType:  "Profit Tax",
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			TaxBase:     500000.0,
			TaxRate:     0.20,
			TaxAmount:   100000.0,
			Submitted:   false,
		},
	}
	return reports, nil
}
