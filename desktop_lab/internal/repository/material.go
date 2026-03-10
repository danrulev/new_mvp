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

func (r *materialRepo) Create(ctx context.Context, m models.Material) error {
	// Если ID пустой, генерируем его (хотя обычно это делает сервис)
	if m.ID == "" {
		// В реальной практике лучше требовать ID от сервиса, но для гибкости:
		// Здесь мы полагаемся, что сервис передал ID или БД имеет дефолт (но у нас TEXT PRIMARY KEY без AUTO)
		// Поэтому сервис должен гарантировать наличие ID.
		if m.ID == "" {
			return fmt.Errorf("material ID cannot be empty")
		}
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO materials (id, name, code, created_at) VALUES (?, ?, ?, ?)`,
		m.ID, m.Name, m.Code, time.Now().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		// Проверка на уникальность имени
		if m.Name != "" {
			// SQLite вернет ошибку ограничения UNIQUE
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
		err := rows.Scan(&m.ID, &m.Name, &m.Code, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		materials = append(materials, m)
	}
	return materials, nil
}

func (r *materialRepo) GetByID(ctx context.Context, id string) (models.Material, error) {
	var m models.Material
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, code, created_at FROM materials WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.Code, &m.CreatedAt)

	if err == sql.ErrNoRows {
		return models.Material{}, nil // Возвращаем пустую структуру, сервис решит, что делать
	}
	if err != nil {
		return models.Material{}, err
	}
	return m, nil
}

func (r *materialRepo) GetByName(ctx context.Context, name string) (models.Material, error) {
	var m models.Material
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, code, created_at FROM materials WHERE name = ?`, name,
	).Scan(&m.ID, &m.Name, &m.Code, &m.CreatedAt)

	if err == sql.ErrNoRows {
		return models.Material{}, nil // Возвращаем пустой, не ошибку
	}
	if err != nil {
		return models.Material{}, err
	}
	return m, nil
}
