package mysql_repo

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type ProtocolRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewProtocolRepo(db *sqlx.DB, log *zap.Logger) *ProtocolRepo {
	return &ProtocolRepo{db: db, log: log}
}

// CreateFull создает протокол и результаты в одной транзакции с batch insert
func (r *ProtocolRepo) CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error {
	log := logQuery(ctx, r.log, "INSERT (TX)", "protocols + test_results",
		zap.String("protocol_id", protocol.ID),
		zap.String("protocol_number", protocol.ProtocolNumber),
		zap.String("sample_id", protocol.SampleID),
		zap.Int("results_count", len(results)),
		zap.String("status", protocol.Status),
	)
	log.Info("starting protocol creation transaction")

	// Проверка контекста перед началом операции
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("transaction rollback failed", zap.Error(rbErr))
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	testDate := protocol.TestDate.Format(timeLayout)

	protocol.CreatedAt = nowUTC
	protocol.UpdatedAt = nowUTC

	_, err = tx.ExecContext(ctx,
		`INSERT INTO protocols (id, sample_id, protocol_number, lab_name, operator_name, test_date, status, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		protocol.ID, protocol.SampleID, protocol.ProtocolNumber, protocol.LabName,
		protocol.OperatorName, testDate, protocol.Status, protocol.Note, nowStr, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert protocol: %w", err)
	}

	if len(results) > 0 {
		// Batch insert для результатов - оптимизация
		batchSize := 100
		resultStmt, err := tx.PrepareContext(ctx,
			`INSERT INTO test_results 
			 (id, protocol_id, method_id, input_data, calculated_value, applied_limit_id, is_compliant, deviation_msg, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("failed to prepare result statement: %w", err)
		}
		defer resultStmt.Close()

		for i, res := range results {
			// Проверка контекста в цикле
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during result insertion: %w", ctx.Err())
			default:
			}

			res.ID = uuid.New().String()
			res.ProtocolID = protocol.ID
			res.CreatedAt = nowUTC

			inputJSON := "{}"
			if len(res.InputData) > 0 {
				b, marshalErr := json.Marshal(res.InputData)
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal input data: %w", marshalErr)
				}
				inputJSON = string(b)
			}

			var calcVal *float64 = res.CalculatedValue
			var compliant *int = nil
			if res.IsCompliant != nil {
				v := 0
				if *res.IsCompliant {
					v = 1
				}
				compliant = &v
			}

			_, err = resultStmt.ExecContext(ctx,
				res.ID, res.ProtocolID, res.MethodID, inputJSON, calcVal,
				res.AppliedLimitID, compliant, res.DeviationMsg, res.Note, nowStr,
			)
			if err != nil {
				return fmt.Errorf("failed to insert result for method %s: %w", res.MethodID, err)
			}

			// Флеш каждые batchSize записей
			if (i+1)%batchSize == 0 {
				log.Debug("batch insert progress", zap.Int("inserted", i+1))
			}
		}
	}

	if err = tx.Commit(); err != nil {
		log.Error("transaction commit failed", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("protocol and results created successfully",
		zap.Int("results_inserted", len(results)))
	return nil
}

// CreateWithSample создает пробу и протокол с результатами в одной транзакции.
// Обеспечивает атомарность: либо создаются все записи, либо ни одной.
// Принимает уже подготовленные результаты с вычисленными значениями.
func (r *ProtocolRepo) CreateWithSample(ctx context.Context, sample models.Sample, protocol models.Protocol, results []models.TestResult) error {
	log := logQuery(ctx, r.log, "INSERT (TX)", "samples + protocols + test_results",
		zap.String("sample_id", sample.ID),
		zap.String("protocol_id", protocol.ID),
		zap.String("protocol_number", protocol.ProtocolNumber),
		zap.Int("results_count", len(results)),
	)
	log.Info("starting atomic protocol creation with sample transaction")

	// Проверка контекста перед началом операции
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("transaction rollback failed", zap.Error(rbErr))
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)
	testDate := protocol.TestDate.Format(timeLayout)
	collDate := sample.CollectionDate.Format(timeLayout)

	protocol.CreatedAt = nowUTC
	protocol.UpdatedAt = nowUTC
	sample.CreatedAt = nowUTC
	sample.UpdatedAt = nowUTC

	// 1. Создаем пробу (Sample)
	rawContext, err := sample.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal sample context: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO samples (id, group_id, material_id, sample_number, collection_place, collection_date, context_params, note, 
		                      photo_url, length_mm, width_mm, height_mm, shape, weight_grams, color, batch_number, manufacturer, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sample.ID, sample.GroupID, sample.MaterialID, sample.SampleNumber, sample.CollectionPlace, collDate, rawContext, sample.Note,
		nullString(sample.PhotoURL),
		nullFloat64(sample.LengthMM),
		nullFloat64(sample.WidthMM),
		nullFloat64(sample.HeightMM),
		nullString(sample.Shape),
		nullFloat64(sample.WeightGrams),
		nullString(sample.Color),
		nullString(sample.BatchNumber),
		nullString(sample.Manufacturer),
		nowStr,
		nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert sample: %w", err)
	}
	log.Debug("sample inserted", zap.String("sample_id", sample.ID))

	// 2. Создаем протокол (Protocol)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO protocols (id, sample_id, protocol_number, lab_name, operator_name, test_date, status, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		protocol.ID, protocol.SampleID, protocol.ProtocolNumber, protocol.LabName,
		protocol.OperatorName, testDate, protocol.Status, protocol.Note, nowStr, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert protocol: %w", err)
	}
	log.Debug("protocol inserted", zap.String("protocol_id", protocol.ID))

	// 3. Создаем результаты тестов (TestResults)
	if len(results) > 0 {
		resultStmt, prepErr := tx.PrepareContext(ctx,
			`INSERT INTO test_results 
			 (id, protocol_id, method_id, input_data, calculated_value, applied_limit_id, is_compliant, deviation_msg, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if prepErr != nil {
			return fmt.Errorf("failed to prepare result statement: %w", prepErr)
		}
		defer resultStmt.Close()

		for i, res := range results {
			// Проверка контекста в цикле
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during result insertion: %w", ctx.Err())
			default:
			}

			resultID := uuid.New().String()

			res.ID = resultID
			res.ProtocolID = protocol.ID
			res.CreatedAt = nowUTC

			inputJSON := "{}"
			if len(res.InputData) > 0 {
				b, marshalErr := json.Marshal(res.InputData)
				if marshalErr != nil {
					return fmt.Errorf("failed to marshal input data: %w", marshalErr)
				}
				inputJSON = string(b)
			}

			var calcVal *float64 = res.CalculatedValue
			var compliant *int = nil
			if res.IsCompliant != nil {
				v := 0
				if *res.IsCompliant {
					v = 1
				}
				compliant = &v
			}

			_, execErr := resultStmt.ExecContext(ctx,
				resultID, res.ProtocolID, res.MethodID, inputJSON, calcVal,
				res.AppliedLimitID, compliant, res.DeviationMsg, res.Note, nowStr,
			)
			if execErr != nil {
				return fmt.Errorf("failed to insert result for method %s: %w", res.MethodID, execErr)
			}

			// Лог прогресса
			if (i+1)%100 == 0 {
				log.Debug("batch insert progress", zap.Int("inserted", i+1))
			}
		}
	}

	if err = tx.Commit(); err != nil {
		log.Error("transaction commit failed", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("sample, protocol and results created successfully in single transaction",
		zap.String("protocol_number", protocol.ProtocolNumber),
		zap.Int("results_inserted", len(results)))
	return nil
}

// GetByID загружает протокол с данными пробы
func (r *ProtocolRepo) GetByID(ctx context.Context, id string) (models.Protocol, error) {
	log := logQuery(ctx, r.log, "SELECT", "protocols",
		zap.String("protocol_id", id))
	log.Debug("fetching protocol by ID")

	p := models.Protocol{}
	var testDateStr, createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, sample_id, protocol_number, lab_name, operator_name, test_date, status, note, created_at, updated_at 
		 FROM protocols WHERE id = ?`, id,
	).Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &p.Note, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		log.Debug("protocol not found")
		return models.Protocol{}, nil
	}
	if err != nil {
		return models.Protocol{}, err
	}

	p.TestDate, err = parseTime(testDateStr)
	if err != nil {
		r.log.Warn("failed to parse test_date for protocol", zap.String("id", id), zap.Error(err))
		p.TestDate = time.Now()
	}

	p.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		r.log.Warn("failed to parse created_at", zap.String("val", createdAt), zap.Error(err))
		p.CreatedAt = time.Now() // Fallback
	}

	p.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		r.log.Warn("failed to parse updated_at", zap.String("val", updatedAt), zap.Error(err))
		p.UpdatedAt = time.Now() // Fallback
	}
	log.Debug("protocol retrieved successfully",
		zap.String("protocol_number", p.ProtocolNumber),
		zap.String("status", p.Status))
	return p, nil
}

// GetResultsByProtocolID загружает результаты
func (r *ProtocolRepo) GetResultsByProtocolID(ctx context.Context, protocolID string) ([]models.TestResult, error) {
	log := logQuery(ctx, r.log, "SELECT", "test_results",
		zap.String("protocol_id", protocolID))
	log.Debug("fetching test results for protocol")

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, method_id, input_data, calculated_value, applied_limit_id, is_compliant, deviation_msg, note, created_at 
		 FROM test_results WHERE protocol_id = ?`,
		protocolID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TestResult
	for rows.Next() {
		var r models.TestResult
		var rawJSON, createdAt string
		var compliant *int

		err := rows.Scan(&r.ID, &r.MethodID, &rawJSON, &r.CalculatedValue, &r.AppliedLimitID, &compliant, &r.DeviationMsg, &r.Note, &createdAt)
		if err != nil {
			return nil, err
		}

		if rawJSON != "" && rawJSON != "{}" {
			if err := json.Unmarshal([]byte(rawJSON), &r.InputData); err != nil {
				r.InputData = make(map[string]interface{})
			}
		} else {
			r.InputData = make(map[string]interface{})
		}

		if compliant != nil {
			v := *compliant == 1
			r.IsCompliant = &v
		}

		r.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			r.CreatedAt = time.Now()
		}

		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	log.Debug("test results retrieved successfully",
		zap.Int("count", len(results)))
	return results, nil
}

// GetByGroupID возвращает список протоколов группы с оптимизированным запросом
func (r *ProtocolRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
	log := logQuery(ctx, r.log, "SELECT", "protocols",
		zap.String("group_id", groupID))
	log.Debug("fetching protocols by group ID")

	// Оптимизированный запрос с INNER JOIN и явным указанием полей
	query := `
		SELECT p.id, p.sample_id, p.protocol_number, p.status, p.created_at
		FROM protocols p
		INNER JOIN samples s ON p.sample_id = s.id
		WHERE s.group_id = ?
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Предварительное выделение памяти (оптимизация)
	var protocols []models.Protocol

	for rows.Next() {
		var p models.Protocol
		var createdAt string
		if err := rows.Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.Status, &createdAt); err != nil {
			return nil, err
		}

		p.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			p.CreatedAt = time.Now()
		}
		protocols = append(protocols, p)
	}
	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	log.Debug("protocols retrieved successfully",
		zap.Int("count", len(protocols)))

	return protocols, nil
}

func (r *ProtocolRepo) GetList(ctx context.Context, filter models.ProtocolListFilter) ([]models.Protocol, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "protocols", zap.Int64("limit", filter.Limit), zap.Int64("offset", filter.Offset))
	log.Debug("fetching paginated protocols list")

	// 1. Валидация входных параметров
	if filter.Limit <= 0 || filter.Limit > 1000 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	var (
		filterFields []string
		filterArgs   []interface{}
	)

	// 2. Построение условий фильтрации
	if filter.LabName != nil {
		filterFields = append(filterFields, "lab_name LIKE ?")
		filterArgs = append(filterArgs, "%"+*filter.LabName+"%") // ИСПРАВЛЕНО: % добавляются к аргументу
	}

	if filter.OperatorName != nil {
		filterFields = append(filterFields, "operator_name LIKE ?")
		filterArgs = append(filterArgs, "%"+*filter.OperatorName+"%") // ИСПРАВЛЕНО
	}

	if filter.ProtocolID != nil {
		// РЕКОМЕНДАЦИЯ: Для ID обычно используется точное совпадение (=), а не LIKE.
		// Если вам нужен именно частичный поиск, оставьте LIKE, но синтаксис исправлен.
		filterFields = append(filterFields, "protocol_number LIKE ?")
		filterArgs = append(filterArgs, "%"+*filter.ProtocolID+"%") // ИСПРАВЛЕНО
	}

	if filter.StartTestDate != nil {
		filterFields = append(filterFields, "test_date >= ?")
		// Форматируем время в строку для надежности работы с MySQL DATETIME/DATE
		filterArgs = append(filterArgs, filter.StartTestDate.Format(timeLayout))
	}

	if filter.EndTestDate != nil {
		filterFields = append(filterFields, "test_date <= ?")
		filterArgs = append(filterArgs, filter.EndTestDate.Format(timeLayout))
	}

	if filter.Status != nil {
		filterFields = append(filterFields, "status = ?")
		filterArgs = append(filterArgs, *filter.Status)
	}

	// 3. Формирование WHERE-клаузы
	var whereClause string
	if len(filterFields) > 0 {
		whereClause = " WHERE " + strings.Join(filterFields, " AND ")
	}

	// 4. Запрос общего количества (с учетом фильтров, но БЕЗ LIMIT/OFFSET)
	countQuery := "SELECT COUNT(*) FROM protocols" + whereClause

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count protocols: %w", err)
	}

	// Если записей нет, сразу возвращаем пустой результат, чтобы не делать лишний запрос
	if total == 0 {
		return []models.Protocol{}, 0, nil
	}

	// 5. Запрос данных (с фильтрами И с LIMIT/OFFSET)
	// Копируем filterArgs, чтобы не мутировать исходный слайс перед добавлением limit/offset
	selectArgs := append([]interface{}{}, filterArgs...)
	selectArgs = append(selectArgs, filter.Limit, filter.Offset)

	selectQuery := `
		SELECT id, sample_id, protocol_number, lab_name, operator_name, test_date, status, created_at, updated_at 
		FROM protocols 
	` + whereClause + `
		ORDER BY created_at DESC 
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query protocols list: %w", err)
	}
	defer rows.Close()

	// 6. Чтение результатов
	protocols := make([]models.Protocol, 0, filter.Limit)

	for rows.Next() {
		var p models.Protocol
		var testDateStr, createdAt, updatedAt string

		if scanErr := rows.Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &createdAt, &updatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("failed to scan protocol row: %w", scanErr)
		}

		p.CreatedAt, _ = parseTime(createdAt)
		if p.CreatedAt.IsZero() {
			p.CreatedAt = time.Now()
		}

		p.UpdatedAt, _ = parseTime(updatedAt)
		if p.UpdatedAt.IsZero() {
			p.UpdatedAt = time.Now()
		}

		p.TestDate, _ = parseTime(testDateStr)
		if p.TestDate.IsZero() {
			p.TestDate = time.Now()
		}

		protocols = append(protocols, p)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	log.Debug("protocols retrieved successfully",
		zap.Int("returned_count", len(protocols)),
		zap.Int64("total_count", total),
	)

	return protocols, total, nil
}

// GetProtocolFull загружает полный протокол с использованием одного запроса для результатов
func (r *ProtocolRepo) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
	log := logQuery(ctx, r.log, "SELECT (FULL JOIN)", "protocols + samples + materials + test_results + test_methods",
		zap.String("protocol_id", id))
	log.Debug("fetching full protocol with joins")
	var full models.ProtocolFull

	// Оптимизированный запрос с явным указанием полей и USING для JOIN
	query := `
		SELECT 
			p.id, p.sample_id, p.protocol_number, p.lab_name, p.operator_name, p.test_date, p.status, p.note, p.created_at, p.updated_at,
			s.id, s.group_id, s.material_id, s.sample_number, s.collection_place, s.collection_date, s.context_params, s.note, s.created_at,
			s.photo_url, s.length_mm, s.width_mm, s.height_mm, s.shape, s.weight_grams, s.color, s.batch_number, s.manufacturer,
			m.id, m.name, m.code, m.created_at
		FROM protocols p
		INNER JOIN samples s ON p.sample_id = s.id
		INNER JOIN materials m ON s.material_id = m.id
		WHERE p.id = ?
	`

	var rawContext sql.NullString
	var testDateStr, collDateStr, protCreatedAt, protUpdatedAt, sampCreatedAt, matCreatedAt string
	var photoURL sql.NullString
	var lengthMM, widthMM, heightMM, weightGrams sql.NullFloat64
	var shape, color, batchNumber, manufacturer sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&full.Protocol.ID, &full.Protocol.SampleID, &full.Protocol.ProtocolNumber, &full.Protocol.LabName,
		&full.Protocol.OperatorName, &testDateStr, &full.Protocol.Status, &full.Protocol.Note, &protCreatedAt, &protUpdatedAt,

		&full.Sample.ID, &full.Sample.GroupID, &full.Sample.MaterialID, &full.Sample.SampleNumber, &full.Sample.CollectionPlace, &collDateStr, &rawContext, &full.Sample.Note, &sampCreatedAt,

		&photoURL, &lengthMM, &widthMM, &heightMM, &shape, &weightGrams, &color, &batchNumber, &manufacturer,

		&full.Material.ID, &full.Material.Name, &full.Material.Code, &matCreatedAt,
	)
	if err == sql.ErrNoRows {
		log.Debug("protocol not found")
		return models.ProtocolFull{}, nil
	}
	if err != nil {
		return models.ProtocolFull{}, err
	}

	full.Protocol.TestDate, _ = parseTime(testDateStr)
	full.Sample.CollectionDate, _ = parseTime(collDateStr)

	full.Protocol.CreatedAt, _ = parseTime(protCreatedAt)
	full.Protocol.UpdatedAt, _ = parseTime(protUpdatedAt)
	full.Sample.CreatedAt, _ = parseTime(sampCreatedAt)
	full.Material.CreatedAt, _ = parseTime(matCreatedAt)

	if rawContext.Valid {
		if err := full.Sample.FromJSON(rawContext.String); err != nil {
			r.log.Warn("failed to parse sample context", zap.Error(err))
			full.Sample.ContextParams = make(map[string]string)
		}
	} else {
		full.Sample.ContextParams = make(map[string]string)
	}

	// Заполняем расширенные поля образца из результата запроса
	if photoURL.Valid {
		full.Sample.PhotoURL = photoURL.String
	}
	if lengthMM.Valid {
		full.Sample.LengthMM = &lengthMM.Float64
	}
	if widthMM.Valid {
		full.Sample.WidthMM = &widthMM.Float64
	}
	if heightMM.Valid {
		full.Sample.HeightMM = &heightMM.Float64
	}
	if shape.Valid {
		full.Sample.Shape = shape.String
	}
	if weightGrams.Valid {
		full.Sample.WeightGrams = &weightGrams.Float64
	}

	if color.Valid {
		full.Sample.Color = color.String
	}
	if batchNumber.Valid {
		full.Sample.BatchNumber = batchNumber.String
	}
	if manufacturer.Valid {
		full.Sample.Manufacturer = manufacturer.String
	}

	// Оптимизированный запрос результатов с предварительной подготовкой statement
	// Используем LEFT JOIN для normative_limits чтобы избежать N+1 запросов
	resultsQuery := `
		SELECT 
			tr.id, tr.method_id, tr.input_data,
			tr.calculated_value, tr.applied_limit_id, tr.is_compliant,
			tr.deviation_msg, tr.note, tr.created_at,
			m.name AS method_name,  
			m.unit AS method_unit,
			nl.limit_type, nl.min_value, nl.max_value
		FROM test_results tr
		INNER JOIN test_methods m ON tr.method_id = m.id
		LEFT JOIN normative_limits nl ON tr.applied_limit_id = nl.id
		WHERE tr.protocol_id = ?
		ORDER BY tr.created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, resultsQuery, id)
	if err != nil {
		return models.ProtocolFull{}, err
	}
	defer rows.Close()

	// Предварительное выделение памяти для результатов (оптимизация)
	full.Results = make([]models.TestResultResponse, 0)

	for rows.Next() {
		var res models.TestResultResponse
		var rawJSON, createdAt string
		var compliant *int

		// Переменные для сканирования лимита
		var limitType sql.NullString
		var minVal, maxVal sql.NullFloat64

		if err := rows.Scan(
			&res.ID, &res.MethodID, &rawJSON,
			&res.CalculatedValue, &res.AppliedLimitID, &compliant,
			&res.DeviationMsg, &res.Note, &createdAt,
			&res.MethodName, &res.MethodUnit,
			&limitType, &minVal, &maxVal,
		); err != nil {
			return models.ProtocolFull{}, err
		}

		if rawJSON != "" && rawJSON != "{}" {
			if err := json.Unmarshal([]byte(rawJSON), &res.InputData); err != nil {
				r.log.Warn("failed to unmarshal input data", zap.Error(err))
				res.InputData = make(map[string]interface{})
			}
		} else {
			res.InputData = make(map[string]interface{})
		}

		res.CreatedAt, _ = parseTime(createdAt)

		if compliant != nil {
			v := *compliant == 1
			res.IsCompliant = &v
		}

		// Сохраняем данные лимита в ответ
		if limitType.Valid {
			res.LimitType = limitType.String
		}

		if minVal.Valid {
			res.MinValue = &minVal.Float64
		}
		if maxVal.Valid {
			res.MaxValue = &maxVal.Float64
		}

		full.Results = append(full.Results, res)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error for results", zap.Error(err))
		return models.ProtocolFull{}, err
	}

	log.Info("full protocol loaded successfully",
		zap.String("protocol_number", full.Protocol.ProtocolNumber),
		zap.Int("results_count", len(full.Results)))

	return full, nil
}

func (r *ProtocolRepo) UpdateProtocol(ctx context.Context, id string, req models.UpdateProtocolRequest) error {
	log := logQuery(ctx, r.log, "UPDATE (TX)", "protocols + samples",
		zap.String("protocol_id", id),
		zap.Bool("lab_name_provided", req.LabName != nil),
		zap.Bool("operator_provided", req.OperatorName != nil),
		zap.Bool("test_date_provided", req.TestDate != nil && !req.TestDate.IsZero()),
		zap.Bool("sample_updates", req.GroupID != nil || req.SampleNumber != nil || req.ContextParams != nil),
	)
	log.Debug("starting protocol update transaction")

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("transaction rollback failed", zap.Error(rbErr))
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	var sampleID string
	err = tx.QueryRowContext(ctx, `SELECT sample_id FROM protocols WHERE id = ?`, id).Scan(&sampleID)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Warn("protocol not found for update",
				zap.String("protocol_id", id))
			return fmt.Errorf("protocol not found")
		}
		return fmt.Errorf("failed to fetch sample ID for protocol: %w", err)
	}

	var (
		protocolUpdateFields []string
		protocolUpdateValues []interface{}
	)
	if req.LabName != nil {
		protocolUpdateFields = append(protocolUpdateFields, "lab_name = ?")
		protocolUpdateValues = append(protocolUpdateValues, *req.LabName)
		log.Debug("including protocol field update", zap.String("field", "lab_name"))
	}
	if req.OperatorName != nil {
		protocolUpdateFields = append(protocolUpdateFields, "operator_name = ?")
		protocolUpdateValues = append(protocolUpdateValues, *req.OperatorName)
		log.Debug("including protocol field update", zap.String("field", "operator_name"))
	}
	if req.TestDate != nil && !req.TestDate.IsZero() {
		protocolUpdateFields = append(protocolUpdateFields, "test_date = ?")
		testDate := req.TestDate.Format(timeLayout)
		protocolUpdateValues = append(protocolUpdateValues, testDate)
		log.Debug("including protocol field update", zap.String("field", "test_date"))
	}

	if len(protocolUpdateFields) == 0 {
		log.Debug("no fields to update, skipping query")
	} else {
		protocolUpdateFields = append(protocolUpdateFields, "updated_at = ?")
		protocolUpdateValues = append(protocolUpdateValues, time.Now().Format(timeLayout))

		_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE protocols SET %v WHERE id = ? AND status = 'draft'`, strings.Join(protocolUpdateFields, ", ")), append(protocolUpdateValues, id)...)
		if err != nil {
			return fmt.Errorf("failed to update protocol: %w", err)
		}
	}

	var (
		sampleUpdateFields []string
		sampleUpdateValues []interface{}
	)
	if req.GroupID != nil {
		sampleUpdateFields = append(sampleUpdateFields, "group_id = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.GroupID)
		log.Debug("including sample field update", zap.String("field", "group_id"))
	}
	if req.SampleNumber != nil {
		sampleUpdateFields = append(sampleUpdateFields, "sample_number = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.SampleNumber)
		log.Debug("including sample field update", zap.String("field", "sample_number"))
	}
	if req.CollectionPlace != nil {
		sampleUpdateFields = append(sampleUpdateFields, "collection_place = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.CollectionPlace)
		log.Debug("including sample field update", zap.String("field", "collection_place"))
	}
	if req.CollectionDate != nil {
		sampleUpdateFields = append(sampleUpdateFields, "collection_date = ?")
		collectionDate := req.CollectionDate.Format(timeLayout)
		sampleUpdateValues = append(sampleUpdateValues, collectionDate)
		log.Debug("including sample field update", zap.String("field", "collection_date"))
	}
	if req.Note != nil {
		sampleUpdateFields = append(sampleUpdateFields, "note = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.Note)
		log.Debug("including sample field update", zap.String("field", "note"))
	}
	if req.ContextParams != nil {
		inputJSON := "{}"
		if len(req.ContextParams) > 0 {
			b, marshalErr := json.Marshal(req.ContextParams)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal input data: %w", marshalErr)
			}
			inputJSON = string(b)
		}
		sampleUpdateFields = append(sampleUpdateFields, "context_params = ?")
		sampleUpdateValues = append(sampleUpdateValues, inputJSON)
		log.Debug("including sample field update", zap.String("field", "context_params"))
	}
	if len(sampleUpdateFields) == 0 {
		log.Debug("no fields to update, skipping query")
	} else {
		sampleUpdateFields = append(sampleUpdateFields, "updated_at = ?")
		sampleUpdateValues = append(sampleUpdateValues, time.Now().Format(timeLayout))
		_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE samples SET %v WHERE id = ?`, strings.Join(sampleUpdateFields, ", ")), append(sampleUpdateValues, sampleID)...)
		if err != nil {
			return fmt.Errorf("failed to update sample: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		log.Error("transaction commit failed", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("protocol and sample updated successfully")
	return nil
}

func (r *ProtocolRepo) DeleteProtocol(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "protocols",
		zap.String("protocol_id", id))
	log.Info("deleting protocol")
	_, err := r.db.ExecContext(ctx, `DELETE FROM protocols WHERE id = ?`, id)
	if err != nil {
		log.Error("delete failed", zap.Error(err))
		return err
	}
	log.Info("protocol deleted successfully")
	return nil
}

func (r *ProtocolRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	log := logQuery(ctx, r.log, "UPDATE", "protocols",
		zap.String("protocol_id", id),
		zap.String("new_status", status))
	log.Debug("updating protocol status")
	_, err := r.db.ExecContext(ctx, `UPDATE protocols SET status = ? WHERE id = ?`, status, id)
	if err != nil {
		log.Error("status update failed", zap.Error(err))
		return err
	}
	log.Info("protocol status updated successfully",
		zap.String("protocol_id", id),
		zap.String("new_status", status))
	return nil
}
