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

// CreateFull создает протокол и результаты в одной транзакции
func (r *ProtocolRepo) CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error {
	log := logQuery(ctx, r.log, "INSERT (TX)", "protocols + test_results",
		zap.String("protocol_id", protocol.ID),
		zap.String("protocol_number", protocol.ProtocolNumber),
		zap.String("sample_id", protocol.SampleID),
		zap.Int("results_count", len(results)),
		zap.String("status", protocol.Status),
	)
	log.Info("starting protocol creation transaction")

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

	for _, res := range results {
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

		_, err = tx.ExecContext(ctx,
			`INSERT INTO test_results 
			 (id, protocol_id, method_id, input_data, calculated_value, applied_limit_id, is_compliant, deviation_msg, note, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			res.ID, res.ProtocolID, res.MethodID, inputJSON, calcVal,
			res.AppliedLimitID, compliant, res.DeviationMsg, res.Note, nowStr,
		)
		if err != nil {
			return fmt.Errorf("failed to insert result for method %s: %w", res.MethodID, err)
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

	p.TestDate, err = helperParseTime(testDateStr)
	if err != nil {
		r.log.Warn("failed to parse test_date for protocol", zap.String("id", id), zap.Error(err))
		p.TestDate = time.Now()
	}

	p.CreatedAt, err = helperParseTime(createdAt)
	if err != nil {
		r.log.Warn("failed to parse created_at", zap.String("val", createdAt), zap.Error(err))
		p.CreatedAt = time.Now() // Fallback
	}

	p.UpdatedAt, err = helperParseTime(updatedAt)
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

		r.CreatedAt, err = helperParseTime(createdAt)
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

// GetByGroupID возвращает список протоколов группы
func (r *ProtocolRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
	log := logQuery(ctx, r.log, "SELECT", "protocols",
		zap.String("group_id", groupID))
	log.Debug("fetching protocols by group ID")

	query := `
		SELECT p.id, p.sample_id, p.protocol_number, p.status, p.created_at
		FROM protocols p
		JOIN samples s ON p.sample_id = s.id
		WHERE s.group_id = ?
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var protocols []models.Protocol
	for rows.Next() {
		var p models.Protocol
		var createdAt string
		if err := rows.Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.Status, &createdAt); err != nil {
			return nil, err
		}

		p.CreatedAt, err = helperParseTime(createdAt)
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

	return protocols, rows.Err()
}

func (r *ProtocolRepo) GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "protocols", zap.Int64("limit", limit),
		zap.Int64("offset", offset))
	log.Debug("fetching paginated protocols list")
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM protocols`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, sample_id, protocol_number, lab_name, operator_name, test_date, status, created_at, updated_at 
         FROM protocols 
         ORDER BY created_at DESC 
         LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var protocols []models.Protocol
	for rows.Next() {
		var p models.Protocol
		var testDateStr, createdAt, updatedAt string

		err := rows.Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &createdAt, &updatedAt)
		if err != nil {
			return nil, 0, err
		}

		p.CreatedAt, err = helperParseTime(createdAt)
		if err != nil {
			p.CreatedAt = time.Now()
		}
		p.UpdatedAt, err = helperParseTime(updatedAt)
		if err != nil {
			p.UpdatedAt = time.Now()
		}

		p.TestDate, err = helperParseTime(testDateStr)
		if err != nil {
			p.TestDate = time.Now()
		}

		protocols = append(protocols, p)
	}
	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, 0, err
	}

	log.Debug("protocols retrieved successfully",
		zap.Int("returned_count", len(protocols)),
		zap.Int64("total_count", total),
	)

	return protocols, total, nil
}

func (r *ProtocolRepo) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
	log := logQuery(ctx, r.log, "SELECT (FULL JOIN)", "protocols + samples + materials + test_results + test_methods",
		zap.String("protocol_id", id))
	log.Debug("fetching full protocol with joins")
	var full models.ProtocolFull

	query := `
		SELECT 
			p.id, p.sample_id, p.protocol_number, p.lab_name, p.operator_name, p.test_date, p.status, p.note, p.created_at, p.updated_at,
			s.id, s.group_id, s.material_id, s.sample_number, s.collection_place, s.collection_date, s.context_params, s.note, s.created_at,
			m.id, m.name, m.code, m.created_at
		FROM protocols p
		JOIN samples s ON p.sample_id = s.id
		JOIN materials m ON s.material_id = m.id
		WHERE p.id = ?
	`

	var rawContext sql.NullString
	var testDateStr, collDateStr, protCreatedAt, protUpdatedAt, sampCreatedAt, matCreatedAt string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&full.Protocol.ID, &full.Protocol.SampleID, &full.Protocol.ProtocolNumber, &full.Protocol.LabName,
		&full.Protocol.OperatorName, &testDateStr, &full.Protocol.Status, &full.Protocol.Note, &protCreatedAt, &protUpdatedAt,

		&full.Sample.ID, &full.Sample.GroupID, &full.Sample.MaterialID, &full.Sample.SampleNumber, &full.Sample.CollectionPlace, &collDateStr, &rawContext, &full.Sample.Note, &sampCreatedAt,

		&full.Material.ID, &full.Material.Name, &full.Material.Code, &matCreatedAt,
	)
	if err == sql.ErrNoRows {
		log.Debug("protocol not found")
		return models.ProtocolFull{}, nil
	}
	if err != nil {
		return models.ProtocolFull{}, err
	}

	full.Protocol.TestDate, _ = helperParseTime(testDateStr)
	full.Sample.CollectionDate, _ = helperParseTime(collDateStr)

	full.Protocol.CreatedAt, _ = helperParseTime(protCreatedAt)
	full.Protocol.UpdatedAt, _ = helperParseTime(protUpdatedAt)
	full.Sample.CreatedAt, _ = helperParseTime(sampCreatedAt)
	full.Material.CreatedAt, _ = helperParseTime(matCreatedAt)

	if rawContext.Valid {
		if err := full.Sample.FromJSON(rawContext.String); err != nil {
			r.log.Warn("failed to parse sample context", zap.Error(err))
			full.Sample.ContextParams = make(map[string]string)
		}
	} else {
		full.Sample.ContextParams = make(map[string]string)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT 
			tr.id, tr.method_id, tr.input_data,
			tr.calculated_value, tr.applied_limit_id, tr.is_compliant,
			tr.deviation_msg, tr.note, tr.created_at,
			m.name AS method_name,  
			m.unit AS method_unit,
			nl.limit_type, nl.min_value, nl.max_value  -- 🔥 Подтягиваем лимит
		FROM test_results tr
		JOIN test_methods m ON tr.method_id = m.id
		LEFT JOIN normative_limits nl ON tr.applied_limit_id = nl.id -- 🔥 JOIN
		WHERE tr.protocol_id = ?`,
		id,
	)
	if err != nil {
		return models.ProtocolFull{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var res models.TestResultResponse
		var rawJSON, createdAt string
		var compliant *int

		// 🔥 Переменные для сканирования лимита
		var limitType sql.NullString // <-- ИСПРАВЛЕНИЕ: используем sql.NullString вместо string
		var minVal, maxVal sql.NullFloat64

		if err := rows.Scan(
			&res.ID, &res.MethodID, &rawJSON,
			&res.CalculatedValue, &res.AppliedLimitID, &compliant,
			&res.DeviationMsg, &res.Note, &createdAt,
			&res.MethodName, &res.MethodUnit,
			&limitType, &minVal, &maxVal, // 🔥 Сканируем лимит
		); err != nil {
			return models.ProtocolFull{}, err
		}

		if rawJSON != "" && rawJSON != "{}" {
			json.Unmarshal([]byte(rawJSON), &res.InputData)
		} else {
			res.InputData = make(map[string]interface{})
		}

		res.CreatedAt, _ = helperParseTime(createdAt)

		if compliant != nil {
			v := *compliant == 1
			res.IsCompliant = &v
		}

		// 🔥 Сохраняем данные лимита в ответ
		if limitType.Valid { // <-- ИСПРАВЛЕНИЕ: проверяем валидность перед присваиванием
			res.LimitType = limitType.String
		} else {
			res.LimitType = "" // Если в БД NULL, оставляем пустую строку
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
