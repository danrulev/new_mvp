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

type experimentGroupRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewExperimentGroupRepo(db *sqlx.DB, log *zap.Logger) ExperimentGroupRepo {
	return &experimentGroupRepo{db: db, log: log}
}

func (r *experimentGroupRepo) Create(ctx context.Context, g models.ExperimentGroup) error {
	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO experiment_groups (id, name, material_id, project_name, location, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		g.ID, g.Name, g.MaterialID, g.ProjectName, g.Location, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create experiment group: %w", err)
	}
	r.log.Info("Experiment group created", zap.String("id", g.ID))
	return nil
}

func (r *experimentGroupRepo) GetByID(ctx context.Context, id string) (models.ExperimentGroup, error) {
	var createdAt string
	var g models.ExperimentGroup
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, material_id, project_name, location, created_at 
		 FROM experiment_groups WHERE id = ?`, id,
	).Scan(&g.ID, &g.Name, &g.MaterialID, &g.ProjectName, &g.Location, &createdAt)

	if err == sql.ErrNoRows {
		return models.ExperimentGroup{}, nil
	}
	if err != nil {
		return models.ExperimentGroup{}, err
	}

	// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
	g.CreatedAt, err = helperParseTime(createdAt)
	if err != nil {
		r.log.Warn("failed to parse created_at for group", zap.String("id", id), zap.Error(err))
		g.CreatedAt = time.Now() // Fallback
	}

	return g, nil
}

func (r *experimentGroupRepo) GetList(ctx context.Context, limit, offset int64) ([]models.ExperimentGroup, int64, error) {
	// 1. Считаем общее количество
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM experiment_groups`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 2. Получаем данные с пагинацией
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, material_id, project_name, location, created_at 
		 FROM experiment_groups 
		 ORDER BY created_at DESC 
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var groups []models.ExperimentGroup
	for rows.Next() {
		var g models.ExperimentGroup
		var createdAt string
		err := rows.Scan(&g.ID, &g.Name, &g.MaterialID, &g.ProjectName, &g.Location, &createdAt)
		if err != nil {
			return nil, 0, err
		}

		// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
		parsedTime, err := helperParseTime(createdAt)
		if err != nil {
			r.log.Warn("failed to parse created_at in list", zap.Error(err))
			parsedTime = time.Now()
		}
		g.CreatedAt = parsedTime

		groups = append(groups, g)
	}

	return groups, total, nil
}

// AddSampleToGroup обновляет группу у пробы
func (r *experimentGroupRepo) AddSampleToGroup(ctx context.Context, sampleID, groupID string) error {
	// Сначала проверим существование группы
	var exists int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM experiment_groups WHERE id = ?`, groupID).Scan(&exists)
	if err == sql.ErrNoRows {
		return fmt.Errorf("group not found")
	}
	if err != nil {
		return err
	}

	// Обновляем пробу
	res, err := r.db.ExecContext(ctx,
		`UPDATE samples SET group_id = ? WHERE id = ?`,
		groupID, sampleID,
	)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("sample not found")
	}

	return nil
}

func (r *experimentGroupRepo) DeleteGroup(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, "DELETE FROM samples WHERE group_id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete samples: %w", err)
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM experiment_groups WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}

	return tx.Commit()
}
