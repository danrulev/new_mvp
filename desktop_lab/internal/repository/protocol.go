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
func (r *protocolRepo) GetByID(ctx context.Context, id string) (*models.Protocol, error) {
	p := &models.Protocol{}

	var testDateStr sql.NullString
	err := r.db.QueryRowContext(ctx,
		`SELECT id, sample_id, protocol_number, lab_name, operator_name, test_date, status, created_at, updated_at 
		 FROM protocols WHERE id = ?`,
		id,
	).Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if testDateStr.Valid {
		t, _ := time.Parse("2006-01-02", testDateStr.String)
		p.TestDate = &t
	}

	// Загружаем данные пробы
	sample, err := r.getSample(ctx, p.SampleID)
	if err != nil {
		return nil, err
	}
	p.Sample = sample

	// Загружаем результаты (без полной денормализации имен методов, это сделает сервис)
	results, err := r.GetResultsByProtocolID(ctx, id)
	if err != nil {
		return nil, err
	}
	p.Results = results

	return p, nil
}

// getSample вспомогательный метод
func (r *protocolRepo) getSample(ctx context.Context, id string) (*models.Sample, error) {
	s := &models.Sample{}
	var collDateStr sql.NullString
	var rawJSON string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_date, context_params, note, created_at 
		 FROM samples WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.GroupID, &s.MaterialID, &s.SampleNumber, &collDateStr, &rawJSON, &s.Note, &s.CreatedAt)

	if err != nil {
		return nil, err
	}

	if collDateStr.Valid {
		t, _ := time.Parse("2006-01-02", collDateStr.String)
		s.CollectionDate = &t
	}

	// Парсим JSON контекста
	if err := s.FromJSON(rawJSON); err != nil {
		return nil, err
	}

	return s, nil
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
		var rawJSON string
		var compliant *int

		err := rows.Scan(&r.ID, &r.MethodID, &rawJSON, &r.CalculatedValue, &r.AppliedLimitID, &compliant, &r.DeviationMsg, &r.Note, &r.CreatedAt)
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

		results = append(results, r)
	}

	return results, nil
}

// GetByGroupID возвращает список протоколов группы (краткий)
func (r *protocolRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
	// Сначала найдем все пробы этой группы
	sampleRows, err := r.db.QueryContext(ctx, `SELECT id FROM samples WHERE group_id = ?`, groupID)
	if err != nil {
		return nil, err
	}

	var sampleIDs []string
	for sampleRows.Next() {
		var sid string
		sampleRows.Scan(&sid)
		sampleIDs = append(sampleIDs, sid)
	}
	sampleRows.Close()

	if len(sampleIDs) == 0 {
		return []models.Protocol{}, nil
	}

	// Теперь протоколы для этих проб
	placeholders := strings.Repeat("?,", len(sampleIDs))
	query := fmt.Sprintf(`SELECT id, sample_id, protocol_number, status, created_at FROM protocols WHERE sample_id IN (%s)`, placeholders[:len(placeholders)-1])

	args := make([]interface{}, len(sampleIDs))
	for i, v := range sampleIDs {
		args[i] = v
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var protocols []models.Protocol
	for rows.Next() {
		var p models.Protocol
		err := rows.Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.Status, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		protocols = append(protocols, p)
	}

	return protocols, nil
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
		err := rows.Scan(&p.ID, &p.SampleID, &p.ProtocolNumber, &p.LabName, &p.OperatorName, &testDateStr, &p.Status, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		// Парсинг даты и загрузка Sample (упрощенно)
		// ...
		protocols = append(protocols, p)
	}
	return protocols, total, nil
}
