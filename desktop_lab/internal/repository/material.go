package repository

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type materialRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewMaterialRepo(db *sqlx.DB, log *zap.Logger) MaterialRepo {
	return &materialRepo{db: db, log: log}
}

// helperParseTimeMaterial безопасно парсит время из строки (ожидается UTC в БД) и возвращает локальное время
func helperParseTimeMaterial(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}
	// Парсим как UTC
	t, err := time.ParseInLocation(timeLayout, timeStr, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	// Конвертируем в локальную зону
	return t.Local(), nil
}

func (r *materialRepo) Create(ctx context.Context, m models.Material) error {
	if m.ID == "" {
		return fmt.Errorf("material ID cannot be empty")
	}

	// 🔥 ИСПРАВЛЕНИЕ: Используем UTC для записи
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

func (r *materialRepo) GetAll(ctx context.Context) ([]models.Material, error) {
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

		// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
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

func (r *materialRepo) GetByID(ctx context.Context, id string) (models.Material, error) {
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

	// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
	if createdAtRaw != "" {
		t, err := helperParseTimeMaterial(createdAtRaw)
		if err == nil {
			m.CreatedAt = t
		}
	}

	return m, nil
}

func (r *materialRepo) GetByName(ctx context.Context, name string) (models.Material, error) {
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

	// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
	if createdAtRaw != "" {
		t, err := helperParseTimeMaterial(createdAtRaw)
		if err == nil {
			m.CreatedAt = t
		}
	}

	return m, nil
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

	// SQLite возвращает формат "2006-01-02 15:04:05"
	return time.Parse("2006-01-02 15:04:05", str)
}
