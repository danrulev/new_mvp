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

	now := time.Now().Format("2006-01-02 15:04:05")

	// Обновляем временные метки
	protocol.CreatedAt = time.Now()
	protocol.UpdatedAt = time.Now()

	// 1. Создаем Протокол
	_, err = tx.ExecContext(ctx,
		`INSERT INTO protocols (id, sample_id, protocol_number, lab_name, operator_name, test_date, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		protocol.ID, protocol.SampleID, protocol.ProtocolNumber, protocol.LabName,
		protocol.OperatorName, protocol.TestDate, protocol.Status, now, now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert protocol: %w", err)
	}

	// 2. Создаем Результаты
	for _, res := range results {
		res.ID = uuid.New().String()
		res.ProtocolID = protocol.ID
		res.CreatedAt = time.Now()

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
			res.AppliedLimitID, compliant, res.DeviationMsg, res.Note, now,
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
		t, _ := time.Parse("2006-01-02", testDateStr.String)
		p.TestDate = &t
	}
	p.CreatedAt, err = time.Parse(time.DateTime, createdAt)
	if err != nil {
		return models.Protocol{}, err
	}
	p.UpdatedAt, err = time.Parse(time.DateTime, updatedAt)
	if err != nil {
		return models.Protocol{}, err
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

		// Парсим входные данные
		if rawJSON != "" && rawJSON != "{}" {
			if err := json.Unmarshal([]byte(rawJSON), &r.InputData); err != nil {
				// Логгируем ошибку, но не прерываем, чтобы показать хоть что-то
				r.InputData = make(map[string]interface{})
			}
		} else {
			r.InputData = make(map[string]interface{})
		}

		if compliant != nil {
			v := *compliant == 1
			r.IsCompliant = &v
		}
		r.CreatedAt, err = time.Parse(time.DateTime, createdAt)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
}

// GetByGroupID возвращает список протоколов группы (краткий)
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
		p.CreatedAt, err = time.Parse(time.DateTime, createdAt)
		if err != nil {
			return nil, err
		}
		protocols = append(protocols, p)
	}

	return protocols, rows.Err()
}

func (r *protocolRepo) GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error) {
	// 1. Считаем общее количество
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM protocols`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 2. Получаем данные с пагинацией
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
		p.CreatedAt, err = time.Parse(time.DateTime, createdAt)
		if err != nil {
			return nil, 0, err
		}
		p.UpdatedAt, err = time.Parse(time.DateTime, updatedAt)
		if err != nil {
			return nil, 0, err
		}
		protocols = append(protocols, p)
	}
	return protocols, total, nil
}

func (r *protocolRepo) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
	var full models.ProtocolFull

	// 1. Загружаем Протокол + Пробу + Материал через JOIN
	// Это убирает 2 отдельных запроса
	query := `
		SELECT 
			p.id, p.sample_id, p.protocol_number, p.lab_name, p.operator_name, p.test_date, p.status, p.created_at, p.updated_at,
			s.id, s.group_id, s.material_id, s.sample_number, s.collection_date, s.context_params, s.note, s.created_at,
			m.id, m.name, m.code, m.created_at
		FROM protocols p
		JOIN samples s ON p.sample_id = s.id
		JOIN materials m ON s.material_id = m.id
		WHERE p.id = ?
	`

	var testDateStr, collDateStr sql.NullString
	var rawContext sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&full.Protocol.ID, &full.Protocol.SampleID, &full.Protocol.ProtocolNumber, &full.Protocol.LabName,
		&full.Protocol.OperatorName, &testDateStr, &full.Protocol.Status, &full.Protocol.CreatedAt, &full.Protocol.UpdatedAt,

		&full.Sample.ID, &full.Sample.GroupID, &full.Sample.MaterialID, &full.Sample.SampleNumber, &collDateStr, &rawContext, &full.Sample.Note, &full.Sample.CreatedAt,

		&full.Material.ID, &full.Material.Name, &full.Material.Code, &full.Material.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return models.ProtocolFull{}, nil
	}
	if err != nil {
		return models.ProtocolFull{}, err
	}

	// Парсинг дат
	if testDateStr.Valid {
		t, _ := time.Parse("2006-01-02", testDateStr.String) // Или полный формат, зависит от хранения
		full.Protocol.TestDate = &t
	}
	if collDateStr.Valid {
		t, _ := time.Parse("2006-01-02", collDateStr.String)
		full.Sample.CollectionDate = &t
	}

	// Парсинг контекста пробы
	if rawContext.Valid {
		if err := full.Sample.FromJSON(rawContext.String); err != nil {
			r.log.Warn("failed to parse sample context", zap.Error(err))
			full.Sample.ContextParams = make(map[string]string)
		}
	} else {
		full.Sample.ContextParams = make(map[string]string)
	}

	// 2. Загружаем Результаты (отдельный запрос, так как их много)
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
		var rawJSON string
		var compliant *int

		if err := rows.Scan(&res.ID, &res.MethodID, &rawJSON, &res.CalculatedValue, &res.AppliedLimitID, &compliant, &res.DeviationMsg, &res.Note, &res.CreatedAt); err != nil {
			return models.ProtocolFull{}, err
		}

		if rawJSON != "" && rawJSON != "{}" {
			json.Unmarshal([]byte(rawJSON), &res.InputData)
		} else {
			res.InputData = make(map[string]interface{})
		}

		if compliant != nil {
			v := *compliant == 1
			res.IsCompliant = &v
		}

		full.Results = append(full.Results, res)
	}

	return full, rows.Err()
}
