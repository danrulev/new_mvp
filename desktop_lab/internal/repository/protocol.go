package repository

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

// Константа формата времени
const timeLayout = "2006-01-02 15:04:05"

// CreateFull создает протокол и результаты в одной транзакции
func (r *ProtocolRepo) CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 🔥 ИСПРАВЛЕНИЕ: Используем UTC для хранения в БД
	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	testDate := protocol.TestDate.Format(timeLayout)

	// Обновляем временные метки в структуре (тоже в UTC для консистентности внутри сессии)
	protocol.CreatedAt = nowUTC
	protocol.UpdatedAt = nowUTC

	// 1. Создаем Протокол
	_, err = tx.ExecContext(ctx,
		`INSERT INTO protocols (id, sample_id, protocol_number, lab_name, operator_name, test_date, status, note, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		protocol.ID, protocol.SampleID, protocol.ProtocolNumber, protocol.LabName,
		protocol.OperatorName, testDate, protocol.Status, protocol.Note, nowStr, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert protocol: %w", err)
	}

	// 2. Создаем Результаты
	for _, res := range results {
		res.ID = uuid.New().String()
		res.ProtocolID = protocol.ID
		res.CreatedAt = nowUTC

		// Сериализуем InputData в JSON
		inputJSON := "{}"
		if len(res.InputData) > 0 {
			b, marshalErr := json.Marshal(res.InputData)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal input data: %w", marshalErr)
			}
			inputJSON = string(b)
		}

		// Обработка nullable полей
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

	return tx.Commit()
}

// GetByID загружает протокол с данными пробы
func (r *ProtocolRepo) GetByID(ctx context.Context, id string) (models.Protocol, error) {
	p := models.Protocol{}
	var testDateStr, createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, sample_id, protocol_number, lab_name, operator_name, test_date, status, note, created_at, updated_at 
		 FROM protocols WHERE id = ?`, id,
	).Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &p.Note, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
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

	return p, nil
}

// GetResultsByProtocolID загружает результаты
func (r *ProtocolRepo) GetResultsByProtocolID(ctx context.Context, protocolID string) ([]models.TestResult, error) {
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

	return results, nil
}

// GetByGroupID возвращает список протоколов группы
func (r *ProtocolRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
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

	return protocols, rows.Err()
}

func (r *ProtocolRepo) GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error) {
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
	return protocols, total, nil
}

func (r *ProtocolRepo) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
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
		return models.ProtocolFull{}, nil
	}
	if err != nil {
		return models.ProtocolFull{}, err
	}

	// Парсинг дат с использованием хелпера
	full.Protocol.TestDate, _ = helperParseTime(testDateStr)
	full.Sample.CollectionDate, _ = helperParseTime(collDateStr)

	full.Protocol.CreatedAt, _ = helperParseTime(protCreatedAt)
	full.Protocol.UpdatedAt, _ = helperParseTime(protUpdatedAt)
	full.Sample.CreatedAt, _ = helperParseTime(sampCreatedAt)
	full.Material.CreatedAt, _ = helperParseTime(matCreatedAt)

	// Парсинг контекста пробы
	if rawContext.Valid {
		if err := full.Sample.FromJSON(rawContext.String); err != nil {
			r.log.Warn("failed to parse sample context", zap.Error(err))
			full.Sample.ContextParams = make(map[string]string)
		}
	} else {
		full.Sample.ContextParams = make(map[string]string)
	}

	// Загрузка результатов
	rows, err := r.db.QueryContext(ctx,
		`SELECT 
			tr.id, tr.method_id, tr.input_data,
			tr.calculated_value, tr.applied_limit_id, tr.is_compliant,
			tr.deviation_msg, tr.note, tr.created_at,
			m.name AS method_name,  
			m.unit AS method_unit
		FROM test_results tr
		JOIN test_methods m ON tr.method_id = m.id
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

		if err := rows.Scan(&res.ID, &res.MethodID, &rawJSON, &res.CalculatedValue, &res.AppliedLimitID, &compliant, &res.DeviationMsg, &res.Note, &createdAt, &res.MethodName, &res.MethodUnit); err != nil {
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

		full.Results = append(full.Results, res)
	}

	return full, rows.Err()
}

func (r *ProtocolRepo) UpdateProtocol(ctx context.Context, id string, req models.UpdateProtocolRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var sampleID string
	err = tx.QueryRowContext(ctx, `SELECT sample_id FROM protocols WHERE id = ?`, id).Scan(&sampleID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("protocol not found")
	}

	var (
		protocolUpdateFields []string
		protocolUpdateValues []interface{}
	)
	if req.LabName != nil {
		protocolUpdateFields = append(protocolUpdateFields, "lab_name = ?")
		protocolUpdateValues = append(protocolUpdateValues, *req.LabName)
	}
	if req.OperatorName != nil {
		protocolUpdateFields = append(protocolUpdateFields, "operator_name = ?")
		protocolUpdateValues = append(protocolUpdateValues, *req.OperatorName)
	}
	if req.TestDate != nil && !req.TestDate.IsZero() {
		protocolUpdateFields = append(protocolUpdateFields, "test_date = ?")
		testDate := req.TestDate.Format(timeLayout)
		protocolUpdateValues = append(protocolUpdateValues, testDate)
	}

	_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE protocols SET %v WHERE id = ? AND status = 'draft'`, strings.Join(protocolUpdateFields, ", ")), append(protocolUpdateValues, id)...)
	if err != nil {
		return fmt.Errorf("failed to update protocol: %w", err)
	}

	var (
		sampleUpdateFields []string
		sampleUpdateValues []interface{}
	)
	if req.GroupID != nil {
		sampleUpdateFields = append(sampleUpdateFields, "group_id = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.GroupID)
	}
	if req.SampleNumber != nil {
		sampleUpdateFields = append(sampleUpdateFields, "sample_number = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.SampleNumber)
	}
	if req.CollectionPlace != nil {
		sampleUpdateFields = append(sampleUpdateFields, "collection_place = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.CollectionPlace)
	}
	if req.CollectionDate != nil {
		sampleUpdateFields = append(sampleUpdateFields, "collection_date = ?")
		collectionDate := req.CollectionDate.Format(timeLayout)
		sampleUpdateValues = append(sampleUpdateValues, collectionDate)
	}
	if req.Note != nil {
		sampleUpdateFields = append(sampleUpdateFields, "note = ?")
		sampleUpdateValues = append(sampleUpdateValues, *req.Note)
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
	}
	_, err = tx.ExecContext(ctx, fmt.Sprintf(`UPDATE samples SET %v WHERE id = ?`, strings.Join(sampleUpdateFields, ", ")), append(sampleUpdateValues, sampleID)...)
	if err != nil {
		return fmt.Errorf("failed to update sample: %w", err)
	}

	return tx.Commit()
}

func (r *ProtocolRepo) DeleteProtocol(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM protocols WHERE id = ?`, id)
	return err
}

func (r *ProtocolRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE protocols SET status = ? WHERE id = ?`, status, id)
	return err
}
