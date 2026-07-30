package mysql_repo

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

type DimensionRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewDimensionRepo(db *sqlx.DB, log *zap.Logger) *DimensionRepo {
	return &DimensionRepo{db: db, log: log}
}

// GetAvailableDimensions возвращает все доступные измерения из глобального справочника
func (r *DimensionRepo) GetAvailableDimensions(ctx context.Context) ([]models.ContextDimension, error) {
	const query = `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions ORDER BY label`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dims []models.ContextDimension
	for rows.Next() {
		var d models.ContextDimension
		var possibleValuesRaw sql.NullString
		if err := rows.Scan(&d.ID, &d.KeyName, &d.Label, &d.DataType, &possibleValuesRaw); err != nil {
			return nil, err
		}
		
		if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
			if err := json.Unmarshal([]byte(possibleValuesRaw.String), &d.PossibleValues); err != nil {
				d.PossibleValues = []string{}
			}
		} else {
			d.PossibleValues = []string{}
		}
		dims = append(dims, d)
	}
	
	return dims, rows.Err()
}

// GetDimensionByID получает измерение по ID
func (r *DimensionRepo) GetDimensionByID(ctx context.Context, id string) (models.ContextDimension, error) {
	const query = `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions WHERE id = ?`
	
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
func (r *DimensionRepo) AddDimension(ctx context.Context, dim models.ContextDimension) error {
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
func (r *DimensionRepo) UpdatePossibleValues(ctx context.Context, id string, values []string) error {
	valuesJSON := "null"
	if len(values) > 0 {
		b, err := json.Marshal(values)
		if err != nil {
			return fmt.Errorf("failed to marshal possible_values: %w", err)
		}
		valuesJSON = string(b)
	}

	res, err := r.db.ExecContext(ctx,
		`UPDATE context_dimensions SET possible_values = ? WHERE id = ?`,
		valuesJSON, id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("dimension with id %s not found", id)
	}
	return nil
}

// DeletePossibleValues очищает список возможных значений
func (r *DimensionRepo) DeletePossibleValues(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE context_dimensions SET possible_values = '[]' WHERE id = ?`, id)
	return err
}

func (r *DimensionRepo) DeleteDimension(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM context_dimensions WHERE id = ?`, id)
	return err
}

func (r *DimensionRepo) GetDimensionByKey(ctx context.Context, key string) (models.ContextDimension, error) {
	var dim models.ContextDimension
	var possibleValuesRaw sql.NullString

	const query = `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions WHERE key_name = ?`
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
