package repository

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type dimensionRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewDimensionRepo(db *sqlx.DB, log *zap.Logger) DimensionRepo {
	return &dimensionRepo{db: db, log: log}
}

// GetAvailableDimensions возвращает все доступные измерения из глобального справочника
func (r *dimensionRepo) GetAvailableDimensions(ctx context.Context) ([]models.ContextDimension, error) {
	query := `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions ORDER BY label`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dims []models.ContextDimension
	for rows.Next() {
		var d models.ContextDimension
		var possibleValuesRaw sql.NullString
		err := rows.Scan(&d.ID, &d.KeyName, &d.Label, &d.DataType, &possibleValuesRaw)
		if err != nil {
			return nil, err
		}
		if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
			if err := json.Unmarshal([]byte(possibleValuesRaw.String), &d.PossibleValues); err != nil {
				r.log.Warn("failed to unmarshal possible_values", zap.Error(err), zap.String("dim_id", d.ID))
				d.PossibleValues = []string{}
			}
		} else {
			d.PossibleValues = []string{}
		}
		dims = append(dims, d)
	}
	return dims, rows.Err()
}

// GetDimensionByID получает измерение по ID (полезно для проверки перед обновлением)
func (r *dimensionRepo) GetDimensionByID(ctx context.Context, id string) (models.ContextDimension, error) {
	query := `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions WHERE id = ?`
	var d models.ContextDimension
	var possibleValuesRaw sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(&d.ID, &d.KeyName, &d.Label, &d.DataType, &possibleValuesRaw)
	if err == sql.ErrNoRows {
		return models.ContextDimension{}, nil
	}
	if err != nil {
		return models.ContextDimension{}, err
	}

	if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
		if err := json.Unmarshal([]byte(possibleValuesRaw.String), &d.PossibleValues); err != nil {
			d.PossibleValues = []string{}
		}
	} else {
		d.PossibleValues = []string{}
	}

	return d, nil
}

// AddDimension добавляет новое измерение в глобальный справочник
func (r *dimensionRepo) AddDimension(ctx context.Context, dim models.ContextDimension) error {
	id := uuid.New().String()
	valuesJSON := "null"
	if len(dim.PossibleValues) > 0 {
		b, err := json.Marshal(dim.PossibleValues)
		if err != nil {
			return fmt.Errorf("failed to marshal possible_values: %w", err)
		}
		valuesJSON = string(b)
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO context_dimensions (id, key_name, label, data_type, possible_values) VALUES (?, ?, ?, ?, ?)`,
		id, dim.KeyName, dim.Label, dim.DataType, valuesJSON,
	)
	return err
}

// UpdatePossibleValues обновляет список возможных значений для существующего измерения
func (r *dimensionRepo) UpdatePossibleValues(ctx context.Context, id string, values []string) error {
	// Сериализуем слайс в JSON
	valuesJSON := "null"
	if len(values) > 0 {
		b, err := json.Marshal(values)
		if err != nil {
			return fmt.Errorf("failed to marshal possible_values: %w", err)
		}
		valuesJSON = string(b)
	}

	// Выполняем UPDATE
	res, err := r.db.ExecContext(ctx,
		`UPDATE context_dimensions SET possible_values = ? WHERE id = ?`,
		valuesJSON, id,
	)
	if err != nil {
		return err
	}

	// Проверяем, была ли затронута хотя бы одна строка
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("dimension with id %s not found", id)
	}

	return nil
}

// DeletePossibleValues очищает список возможных значений (устанавливает NULL или пустой массив)
// Это полезно, если тип измерения меняется с 'enum' на 'text' или значения больше не нужны
func (r *dimensionRepo) DeletePossibleValues(ctx context.Context, id string) error {
	// Вариант 1: Установить в NULL
	// _, err := r.db.ExecContext(ctx, `UPDATE context_dimensions SET possible_values = NULL WHERE id = ?`, id)

	// Вариант 2: Установить в пустой JSON массив (рекомендуется для консистентности)
	_, err := r.db.ExecContext(ctx, `UPDATE context_dimensions SET possible_values = '[]' WHERE id = ?`, id)

	return err
}

func (r *dimensionRepo) DeleteDimension(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM context_dimensions WHERE id = ?`, id)
	return err
}

func (r *dimensionRepo) GetDimensionByKey(ctx context.Context, key string) (models.ContextDimension, error) {
	var dim models.ContextDimension
	var possibleValuesRaw sql.NullString

	query := `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions WHERE key_name = ?`

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&dim.ID, &dim.KeyName, &dim.Label, &dim.DataType, &possibleValuesRaw,
	)
	if err == sql.ErrNoRows {
		return models.ContextDimension{}, nil
	}
	if err != nil {
		return models.ContextDimension{}, err
	}

	if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
		if err := json.Unmarshal([]byte(possibleValuesRaw.String), &dim.PossibleValues); err != nil {
			dim.PossibleValues = []string{}
		}
	} else {
		dim.PossibleValues = []string{}
	}

	return dim, nil
}
