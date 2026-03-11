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

type sampleRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewSampleRepo(db *sqlx.DB, log *zap.Logger) SampleRepo {
	return &sampleRepo{db: db, log: log}
}

func (r *sampleRepo) Create(ctx context.Context, s models.Sample) error {
	// Сериализуем ContextParams в JSON строку
	jsonData, err := s.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal context params: %w", err)
	}

	var collDateStr interface{}
	if s.CollectionDate != nil {
		collDateStr = s.CollectionDate.Format("2006-01-02")
	} else {
		collDateStr = nil
	}

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO samples (id, group_id, material_id, sample_number, collection_date, context_params, note, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.GroupID, s.MaterialID, s.SampleNumber, collDateStr, jsonData, s.Note, time.Now().Format("2006-01-02 15:04:05"),
	)

	if err != nil {
		return fmt.Errorf("failed to create sample: %w", err)
	}

	r.log.Info("Sample created", zap.String("id", s.ID), zap.String("number", s.SampleNumber))
	return nil
}

func (r *sampleRepo) GetByID(ctx context.Context, id string) (models.Sample, error) {
	s := models.Sample{}
	var collDateStr sql.NullString
	var rawJSON string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_date, context_params, note, created_at 
		 FROM samples WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.GroupID, &s.MaterialID, &s.SampleNumber, &collDateStr, &rawJSON, &s.Note, &s.CreatedAt)

	if err == sql.ErrNoRows {
		return models.Sample{}, nil
	}
	if err != nil {
		return models.Sample{}, err
	}

	if collDateStr.Valid {
		t, err := time.Parse("2006-01-02", collDateStr.String)
		if err == nil {
			s.CollectionDate = &t
		}
	}

	// Парсим JSON обратно в мапу
	if err := s.FromJSON(rawJSON); err != nil {
		r.log.Warn("Failed to unmarshal sample context", zap.Error(err), zap.String("id", id))
		// Не прерываем работу, просто контекст будет пустым
		s.ContextParams = make(map[string]string)
	}

	return s, nil
}

func (r *sampleRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Sample, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_date, context_params, note, created_at 
		 FROM samples WHERE group_id = ?`,
		groupID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get samples by group id: %w", err)
	}
	defer rows.Close()

	var samples []models.Sample
	for rows.Next() {
		s := models.Sample{
			ContextParams: make(map[string]string),
		}
		var collDateStr sql.NullString
		var rawJSON string

		err := rows.Scan(
			&s.ID,
			&s.GroupID,
			&s.MaterialID,
			&s.SampleNumber,
			&collDateStr,
			&rawJSON,
			&s.Note,
			&s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sample: %w", err)
		}

		// 📅 Парсим дату отбора
		if collDateStr.Valid {
			t, err := time.Parse("2006-01-02", collDateStr.String)
			if err == nil {
				s.CollectionDate = &t
			}
		}

		// 🗂️ Парсим JSON контекста в мапу
		if err := s.FromJSON(rawJSON); err != nil {
			r.log.Warn("Failed to unmarshal sample context",
				zap.Error(err),
				zap.String("sample_id", s.ID))
			// Не прерываем работу — просто контекст будет пустым
			s.ContextParams = make(map[string]string)
		}

		samples = append(samples, s)
	}

	// 🔍 Проверяем ошибки итерации
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating samples: %w", err)
	}

	// 🛡️ Возвращаем пустой слайс вместо nil для удобства на фронтенде
	if samples == nil {
		return []models.Sample{}, nil
	}

	return samples, nil
}
