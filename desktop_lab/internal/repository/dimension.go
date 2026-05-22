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

type DimensionRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewDimensionRepo(db *sqlx.DB, log *zap.Logger) *DimensionRepo {
	return &DimensionRepo{db: db, log: log}
}

// GetAvailableDimensions возвращает все доступные измерения из глобального справочника
func (r *DimensionRepo) GetAvailableDimensions(ctx context.Context) ([]models.ContextDimension, error) {
	log := logQuery(ctx, r.log, "SELECT", "context_dimensions")
	log.Debug("executing query to fetch available dimensions")

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
			log.Error("failed to scan row", zap.Error(err))
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
	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	log.Debug("query completed successfully",
		zap.Int("rows_returned", len(dims)))

	return dims, rows.Err()
}

// GetDimensionByID получает измерение по ID (полезно для проверки перед обновлением)
func (r *DimensionRepo) GetDimensionByID(ctx context.Context, id string) (models.ContextDimension, error) {
	log := logQuery(ctx, r.log, "SELECT", "context_dimensions", zap.String("dimension_id", id))
	log.Debug("fetching dimension by ID")

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

	log.Debug("dimension retrieved successfully",
		zap.String("key_name", d.KeyName))
	return d, nil
}

// AddDimension добавляет новое измерение в глобальный справочник
func (r *DimensionRepo) AddDimension(ctx context.Context, dim models.ContextDimension) error {
	log := logQuery(ctx, r.log, "INSERT", "context_dimensions",
		zap.String("key_name", dim.KeyName),
		zap.String("label", dim.Label),
		zap.String("data_type", dim.DataType),
		zap.Int("possible_values_count", len(dim.PossibleValues)))
	log.Debug("preparing to insert new dimension")

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
	log.Info("dimension inserted successfully",
		zap.String("dimension_id", id))
	return err
}

// UpdatePossibleValues обновляет список возможных значений для существующего измерения
func (r *DimensionRepo) UpdatePossibleValues(ctx context.Context, id string, values []string) error {
	log := logQuery(ctx, r.log, "UPDATE", "context_dimensions",
		zap.String("dimension_id", id),
		zap.Int("new_values_count", len(values)))
	log.Debug("updating possible values for dimension")
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
	log.Info("possible values updated successfully",
		zap.Int64("rows_affected", rowsAffected))
	return nil
}

// DeletePossibleValues очищает список возможных значений
func (r *DimensionRepo) DeletePossibleValues(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "UPDATE", "context_dimensions",
		zap.String("dimension_id", id))
	log.Debug("clearing possible values for dimension")
	_, err := r.db.ExecContext(ctx, `UPDATE context_dimensions SET possible_values = '[]' WHERE id = ?`, id)

	log.Info("possible values cleared")
	return err
}

func (r *DimensionRepo) DeleteDimension(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "context_dimensions",
		zap.String("dimension_id", id))
	log.Info("deleting dimension")

	_, err := r.db.ExecContext(ctx, `DELETE FROM context_dimensions WHERE id = ?`, id)

	log.Info("dimension deleted successfully")
	return err
}

func (r *DimensionRepo) GetDimensionByKey(ctx context.Context, key string) (models.ContextDimension, error) {
	log := logQuery(ctx, r.log, "SELECT", "context_dimensions",
		zap.String("key_name", key))
	log.Debug("fetching dimension by key")

	var dim models.ContextDimension
	var possibleValuesRaw sql.NullString

	query := `SELECT id, key_name, label, data_type, possible_values FROM context_dimensions WHERE key_name = ?`

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&dim.ID, &dim.KeyName, &dim.Label, &dim.DataType, &possibleValuesRaw,
	)
	if err == sql.ErrNoRows {
		log.Debug("dimension not found by key")
		return models.ContextDimension{}, nil
	}
	if err != nil {
		log.Error("query execution failed", zap.Error(err))
		return models.ContextDimension{}, err
	}

	if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
		if err := json.Unmarshal([]byte(possibleValuesRaw.String), &dim.PossibleValues); err != nil {
			log.Warn("failed to unmarshal possible_values",
				zap.Error(err),
				zap.String("dim_id", dim.ID))
			dim.PossibleValues = []string{}
		}
	} else {
		dim.PossibleValues = []string{}
	}
	log.Debug("dimension retrieved successfully",
		zap.String("dimension_id", dim.ID))
	return dim, nil
}
