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

type SampleRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewSampleRepo(db *sqlx.DB, log *zap.Logger) *SampleRepo {
	return &SampleRepo{db: db, log: log}
}

// Константа формата времени для БД
const (
	dateLayout = "2006-01-02"
)

// helperParseDate парсит дату (без времени)
func helperParseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}
	t, err := time.ParseInLocation(dateLayout, dateStr, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	return t.Local(), nil
}

func (r *SampleRepo) Create(ctx context.Context, s models.Sample) error {
	jsonData, err := s.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal context params: %w", err)
	}

	// 🔥 ИСПРАВЛЕНИЕ: Используем UTC для created_at
	collDateStr := s.CollectionDate.Format(timeLayout)
	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO samples (id, group_id, material_id, sample_number, collection_place, collection_date, context_params, note, created_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.GroupID, s.MaterialID, s.SampleNumber, s.CollectionPlace, collDateStr, jsonData, s.Note, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create sample: %w", err)
	}

	r.log.Info("Sample created", zap.String("id", s.ID), zap.String("number", s.SampleNumber))
	return nil
}

func (r *SampleRepo) GetByID(ctx context.Context, id string) (models.Sample, error) {
	s := models.Sample{}
	var collDateStr, rawJSON, createdAt string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_place, collection_date, context_params, note, created_at 
		 FROM samples WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.GroupID, &s.MaterialID, &s.SampleNumber, &s.CollectionPlace, &collDateStr, &rawJSON, &s.Note, &createdAt)

	if err == sql.ErrNoRows {
		return models.Sample{}, nil
	}
	if err != nil {
		return models.Sample{}, err
	}

	// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
	s.CreatedAt, err = helperParseTime(createdAt)
	if err != nil {
		r.log.Warn("failed parse created at date", zap.Error(err), zap.String("val", createdAt))
		s.CreatedAt = time.Now()
	}

	s.CollectionDate, err = helperParseTime(collDateStr)
	if err != nil {
		r.log.Warn("failed parse collection date", zap.Error(err), zap.String("val", collDateStr))
		s.CollectionDate = time.Now()
	}

	if err := s.FromJSON(rawJSON); err != nil {
		r.log.Warn("Failed to unmarshal sample context", zap.Error(err), zap.String("id", id))
		s.ContextParams = make(map[string]string)
	}

	return s, nil
}

func (r *SampleRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Sample, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_place, collection_date, context_params, note, created_at 
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
		var collDateStr, rawJSON, createdAt string

		err := rows.Scan(
			&s.ID,
			&s.GroupID,
			&s.MaterialID,
			&s.SampleNumber,
			&s.CollectionPlace,
			&collDateStr,
			&rawJSON,
			&s.Note,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sample: %w", err)
		}

		// 🔥 ИСПРАВЛЕНИЕ: Парсинг с учетом часовых поясов
		s.CreatedAt, err = helperParseTime(createdAt)
		if err != nil {
			r.log.Warn("failed parse created at date", zap.Error(err))
			s.CreatedAt = time.Now()
		}

		s.CollectionDate, err = helperParseTime(collDateStr)
		if err != nil {
			r.log.Warn("failed parse collection date", zap.Error(err))
			s.CollectionDate = time.Now()
		}

		if err := s.FromJSON(rawJSON); err != nil {
			r.log.Warn("Failed to unmarshal sample context",
				zap.Error(err),
				zap.String("sample_id", s.ID))
			s.ContextParams = make(map[string]string)
		}

		samples = append(samples, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating samples: %w", err)
	}

	if samples == nil {
		return []models.Sample{}, nil
	}

	return samples, nil
}
