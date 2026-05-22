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

type MaterialRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewMaterialRepo(db *sqlx.DB, log *zap.Logger) *MaterialRepo {
	return &MaterialRepo{db: db, log: log}
}

// helperParseTimeMaterial безопасно парсит время из строки (ожидается UTC в БД) и возвращает локальное время
func helperParseTimeMaterial(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	t, err := time.ParseInLocation(timeLayout, timeStr, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return t.Local(), nil
}

func (r *MaterialRepo) Create(ctx context.Context, m models.Material) error {
	if m.ID == "" {
		return fmt.Errorf("material ID cannot be empty")
	}

	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO materials (id, name, code, created_at) VALUES (?, ?, ?, ?)`,
		m.ID, m.Name, m.Code, nowStr,
	)
	if err != nil {
		if m.Name != "" {
			return fmt.Errorf("failed to create material (name might be duplicate): %w", err)
		}
		return fmt.Errorf("failed to create material: %w", err)
	}

	r.log.Debug("Material created", zap.String("id", m.ID), zap.String("name", m.Name))
	return nil
}

func (r *MaterialRepo) GetAll(ctx context.Context) ([]models.Material, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, code, created_at FROM materials ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []models.Material
	for rows.Next() {
		var m models.Material
		var createdAtRaw string

		err := rows.Scan(&m.ID, &m.Name, &m.Code, &createdAtRaw)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}

		if createdAtRaw != "" {
			t, err := helperParseTimeMaterial(createdAtRaw)
			if err == nil {
				m.CreatedAt = t
			} else {
				r.log.Warn("failed to parse material created_at", zap.String("raw", createdAtRaw), zap.Error(err))
			}
		}

		materials = append(materials, m)
	}
	return materials, nil
}

func (r *MaterialRepo) GetByID(ctx context.Context, id string) (models.Material, error) {
	var m models.Material
	var createdAtRaw string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, code, created_at FROM materials WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.Code, &createdAtRaw)

	if err == sql.ErrNoRows {
		return models.Material{}, nil
	}
	if err != nil {
		return models.Material{}, err
	}

	if createdAtRaw != "" {
		t, err := helperParseTimeMaterial(createdAtRaw)
		if err == nil {
			m.CreatedAt = t
		}
	}

	return m, nil
}

func (r *MaterialRepo) GetByName(ctx context.Context, name string) (models.Material, error) {
	var m models.Material
	var createdAtRaw string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, code, created_at FROM materials WHERE name = ?`, name,
	).Scan(&m.ID, &m.Name, &m.Code, &createdAtRaw)

	if err == sql.ErrNoRows {
		return models.Material{}, nil
	}
	if err != nil {
		return models.Material{}, err
	}

	if createdAtRaw != "" {
		t, err := helperParseTimeMaterial(createdAtRaw)
		if err == nil {
			m.CreatedAt = t
		}
	}

	return m, nil
}

func (r *MaterialRepo) GetContextDimensionsByMaterialID(ctx context.Context, materialID string) ([]models.ContextDimension, error) {
	query := `
		SELECT cd.id, cd.key_name, cd.label, cd.data_type, cd.possible_values, cd.description
		FROM context_dimensions cd
		JOIN material_context_dims mcd ON cd.id = mcd.dimension_id
		WHERE mcd.material_id = ?
		ORDER BY cd.label
	`
	rows, err := r.db.QueryContext(ctx, query, materialID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dims []models.ContextDimension
	for rows.Next() {
		var d models.ContextDimension
		var possibleValuesRaw sql.NullString
		var description sql.NullString

		err := rows.Scan(&d.ID, &d.KeyName, &d.Label, &d.DataType, &possibleValuesRaw, &description)
		if err != nil {
			return nil, err
		}

		if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
			json.Unmarshal([]byte(possibleValuesRaw.String), &d.PossibleValues)
		} else {
			d.PossibleValues = []string{}
		}

		dims = append(dims, d)
	}
	return dims, rows.Err()
}

func (r *MaterialRepo) AddContextDimensionToMaterial(ctx context.Context, materialID, dimensionID string, isRequired bool) error {
	id := uuid.New().String()
	isReqInt := 0
	if isRequired {
		isReqInt = 1
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO material_context_dims (id, material_id, dimension_id, is_required) VALUES (?, ?, ?, ?)`,
		id, materialID, dimensionID, isReqInt,
	)
	return err
}

func (r *MaterialRepo) DeleteContextDimensionFromMaterial(ctx context.Context, materialID, dimensionID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM material_context_dims WHERE material_id = ? AND dimension_id = ?`,
		materialID, dimensionID,
	)
	return err
}

// helperParseTime вспомогательная функция для парсинга даты из SQLite
func helperParseTime(raw interface{}) (time.Time, error) {
	if raw == nil {
		return time.Time{}, nil
	}

	str, ok := raw.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("expected string for time, got %T", raw)
	}

	return time.Parse("2006-01-02 15:04:05", str)
}
