package repository

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type protocolRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewProtocolRepo(db *sqlx.DB, log *zap.Logger) ProtocolRepo {
	return &protocolRepo{db: db, log: log}
}

// Константа формата времени
const timeLayout = "2006-01-02 15:04:05"

// CreateFull создает протокол и результаты в одной транзакции
func (r *protocolRepo) CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error {
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

	// Обновляем временные метки в структуре (тоже в UTC для консистентности внутри сессии)
	protocol.CreatedAt = nowUTC
	protocol.UpdatedAt = nowUTC

	// 1. Создаем Протокол
	_, err = tx.ExecContext(ctx,
		`INSERT INTO protocols (id, sample_id, protocol_number, lab_name, operator_name, test_date, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		protocol.ID, protocol.SampleID, protocol.ProtocolNumber, protocol.LabName,
		protocol.OperatorName, protocol.TestDate, protocol.Status, nowStr, nowStr,
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
func (r *protocolRepo) GetByID(ctx context.Context, id string) (models.Protocol, error) {
	p := models.Protocol{}
	var testDateStr sql.NullString
	var createdAt, updatedAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, sample_id, protocol_number, lab_name, operator_name, test_date, status, created_at, updated_at 
		 FROM protocols WHERE id = ?`, id,
	).Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return models.Protocol{}, nil
	}
	if err != nil {
		return models.Protocol{}, err
	}

	if testDateStr.Valid {
		// Для test_date формат может быть просто датой, проверим длину или попробуем полный парсинг
		if len(testDateStr.String) > 10 {
			t, err := helperParseTime(testDateStr.String)
			if err == nil {
				p.TestDate = &t
			}
		} else {
			t, err := time.ParseInLocation("2006-01-02", testDateStr.String, time.UTC)
			if err == nil {
				p.TestDate = &t
			}
		}
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
func (r *protocolRepo) GetResultsByProtocolID(ctx context.Context, protocolID string) ([]models.TestResult, error) {
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
func (r *protocolRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
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

func (r *protocolRepo) GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error) {
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
		var testDateStr sql.NullString
		var createdAt, updatedAt string

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

		// Обработка test_date аналогично GetByID
		if testDateStr.Valid {
			if len(testDateStr.String) > 10 {
				t, err := helperParseTime(testDateStr.String)
				if err == nil {
					p.TestDate = &t
				}
			} else {
				t, err := time.ParseInLocation("2006-01-02", testDateStr.String, time.UTC)
				if err == nil {
					p.TestDate = &t
				}
			}
		}

		protocols = append(protocols, p)
	}
	return protocols, total, nil
}

func (r *protocolRepo) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
	var full models.ProtocolFull

	query := `
		SELECT 
			p.id, p.sample_id, p.protocol_number, p.lab_name, p.operator_name, p.test_date, p.status, p.created_at, p.updated_at,
			s.id, s.group_id, s.material_id, s.sample_number, s.collection_place, s.collection_date, s.context_params, s.note, s.created_at,
			m.id, m.name, m.code, m.created_at
		FROM protocols p
		JOIN samples s ON p.sample_id = s.id
		JOIN materials m ON s.material_id = m.id
		WHERE p.id = ?
	`

	var testDateStr, collDateStr sql.NullString
	var rawContext sql.NullString
	var protCreatedAt, protUpdatedAt, sampCreatedAt, matCreatedAt string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&full.Protocol.ID, &full.Protocol.SampleID, &full.Protocol.ProtocolNumber, &full.Protocol.LabName,
		&full.Protocol.OperatorName, &testDateStr, &full.Protocol.Status, &protCreatedAt, &protUpdatedAt,

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
	if testDateStr.Valid {
		if len(testDateStr.String) > 10 {
			t, err := helperParseTime(testDateStr.String)
			if err == nil {
				full.Protocol.TestDate = &t
			}
		} else {
			t, err := time.ParseInLocation("2006-01-02", testDateStr.String, time.UTC)
			if err == nil {
				localT := t.Local() // 🔥 Конвертация в локальное время
				full.Protocol.TestDate = &localT
			}
		}
	}

	if collDateStr.Valid {
		if len(collDateStr.String) > 10 {
			t, _ := helperParseTime(collDateStr.String)
			full.Sample.CollectionDate = &t
		} else {
			t, _ := time.ParseInLocation("2006-01-02", collDateStr.String, time.UTC)
			full.Sample.CollectionDate = &t
		}
	}

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
		`SELECT id, method_id, input_data, calculated_value, applied_limit_id, is_compliant, deviation_msg, note, created_at 
		 FROM test_results WHERE protocol_id = ?`,
		id,
	)
	if err != nil {
		return models.ProtocolFull{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var res models.TestResult
		var rawJSON, createdAt string
		var compliant *int

		if err := rows.Scan(&res.ID, &res.MethodID, &rawJSON, &res.CalculatedValue, &res.AppliedLimitID, &compliant, &res.DeviationMsg, &res.Note, &createdAt); err != nil {
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

func (r *protocolRepo) DeleteProtocol(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM protocols WHERE id = ?`, id)
	return err
}

func (r *protocolRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE protocols SET status = ? WHERE id = ?`, status, id)
	return err
}
