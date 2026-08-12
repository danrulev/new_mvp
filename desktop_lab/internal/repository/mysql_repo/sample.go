package mysql_repo

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

func (r *SampleRepo) Create(ctx context.Context, s models.Sample) error {
	log := logQuery(ctx, r.log, "INSERT", "samples",
		zap.String("sample_id", s.ID),
		zap.String("sample_number", s.SampleNumber),
		zap.String("group_id", s.GroupID),
		zap.String("material_id", s.MaterialID),
		zap.String("collection_place", s.CollectionPlace),
		zap.Int("context_params_count", len(s.ContextParams)),
	)
	log.Debug("creating new sample")

	jsonData, err := s.ToJSON()
	if err != nil {
		log.Error("failed to marshal context params", zap.Error(err))
		return fmt.Errorf("failed to marshal context params: %w", err)
	}

	collDateStr := s.CollectionDate.Format(timeLayout)
	nowUTC := time.Now().UTC()
	nowStr := nowUTC.Format(timeLayout)

	_, err = r.db.ExecContext(ctx,
		`INSERT INTO samples (id, group_id, material_id, sample_number, collection_place, collection_date, 
		                      context_params, note, photo_url, length_mm, width_mm, height_mm, shape, 
		                      weight_grams, color, batch_number, manufacturer, created_at, updated_at) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.GroupID, s.MaterialID, s.SampleNumber, s.CollectionPlace, collDateStr, 
		jsonData, s.Note, 
		nullString(s.PhotoURL),
		nullFloat64(s.LengthMM),
		nullFloat64(s.WidthMM),
		nullFloat64(s.HeightMM),
		nullString(s.Shape),
		nullFloat64(s.WeightGrams),
		nullString(s.Color),
		nullString(s.BatchNumber),
		nullString(s.Manufacturer),
		nowStr,
		nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to create sample: %w", err)
	}

	r.log.Info("Sample created", zap.String("id", s.ID), zap.String("sample_number", s.SampleNumber))
	return nil
}

func (r *SampleRepo) GetByID(ctx context.Context, id string) (models.Sample, error) {
	log := logQuery(ctx, r.log, "SELECT", "samples",
		zap.String("sample_id", id))
	log.Debug("fetching sample by ID")
	s := models.Sample{}
	var collDateStr, rawJSON, createdAt, updatedAt string
	var photoURL, shape, color, batchNum, manufacturer sql.NullString
	var lengthMM, widthMM, heightMM, weightGrams sql.NullFloat64

	err := r.db.QueryRowContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_place, collection_date, 
		        context_params, note, photo_url, length_mm, width_mm, height_mm, shape, 
		        weight_grams, color, batch_number, manufacturer, created_at, updated_at 
		 FROM samples WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.GroupID, &s.MaterialID, &s.SampleNumber, &s.CollectionPlace, &collDateStr, 
	       &rawJSON, &s.Note, &photoURL, &lengthMM, &widthMM, &heightMM, &shape, 
	       &weightGrams, &color, &batchNum, &manufacturer, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		log.Debug("sample not found")
		return models.Sample{}, nil
	}
	if err != nil {
		return models.Sample{}, err
	}

	s.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		r.log.Warn("failed parse created at date", zap.Error(err), zap.String("val", createdAt))
		s.CreatedAt = time.Now()
	}

	if updatedAt != "" {
		s.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			r.log.Warn("failed parse updated at date", zap.Error(err), zap.String("val", updatedAt))
		}
	}

	s.CollectionDate, err = parseTime(collDateStr)
	if err != nil {
		r.log.Warn("failed parse collection date", zap.Error(err), zap.String("val", collDateStr))
		s.CollectionDate = time.Now()
	}

	if photoURL.Valid {
		s.PhotoURL = photoURL.String
	}
	if lengthMM.Valid {
		s.LengthMM = &lengthMM.Float64
	}
	if widthMM.Valid {
		s.WidthMM = &widthMM.Float64
	}
	if heightMM.Valid {
		s.HeightMM = &heightMM.Float64
	}
	if shape.Valid {
		s.Shape = shape.String
	}
	if weightGrams.Valid {
		s.WeightGrams = &weightGrams.Float64
	}
	if color.Valid {
		s.Color = color.String
	}
	if batchNum.Valid {
		s.BatchNumber = batchNum.String
	}
	if manufacturer.Valid {
		s.Manufacturer = manufacturer.String
	}

	if err := s.FromJSON(rawJSON); err != nil {
		r.log.Warn("Failed to unmarshal sample context", zap.Error(err), zap.String("id", id))
		s.ContextParams = make(map[string]string)
	}
	log.Debug("sample retrieved successfully",
		zap.String("sample_number", s.SampleNumber),
		zap.String("material_id", s.MaterialID),
		zap.Int("context_params_count", len(s.ContextParams)))
	return s, nil
}

func (r *SampleRepo) GetByGroupID(ctx context.Context, groupID string) ([]models.Sample, error) {
	log := logQuery(ctx, r.log, "SELECT", "samples",
		zap.String("group_id", groupID))
	log.Debug("fetching samples for group")

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, group_id, material_id, sample_number, collection_place, collection_date, 
		        context_params, note, photo_url, length_mm, width_mm, height_mm, shape, 
		        weight_grams, color, batch_number, manufacturer, created_at, updated_at 
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
		var collDateStr, rawJSON, createdAt, updatedAt string
		var photoURL, shape, color, batchNum, manufacturer sql.NullString
		var lengthMM, widthMM, heightMM, weightGrams sql.NullFloat64

		err := rows.Scan(
			&s.ID,
			&s.GroupID,
			&s.MaterialID,
			&s.SampleNumber,
			&s.CollectionPlace,
			&collDateStr,
			&rawJSON,
			&s.Note,
			&photoURL,
			&lengthMM,
			&widthMM,
			&heightMM,
			&shape,
			&weightGrams,
			&color,
			&batchNum,
			&manufacturer,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sample: %w", err)
		}

		s.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			r.log.Warn("failed parse created at date", zap.Error(err))
			s.CreatedAt = time.Now()
		}

		if updatedAt != "" {
			s.UpdatedAt, err = parseTime(updatedAt)
			if err != nil {
				r.log.Warn("failed parse updated at date", zap.Error(err))
			}
		}

		s.CollectionDate, err = parseTime(collDateStr)
		if err != nil {
			r.log.Warn("failed parse collection date", zap.Error(err))
			s.CollectionDate = time.Now()
		}

		if photoURL.Valid {
			s.PhotoURL = photoURL.String
		}
		if lengthMM.Valid {
			s.LengthMM = &lengthMM.Float64
		}
		if widthMM.Valid {
			s.WidthMM = &widthMM.Float64
		}
		if heightMM.Valid {
			s.HeightMM = &heightMM.Float64
		}
		if shape.Valid {
			s.Shape = shape.String
		}
		if weightGrams.Valid {
			s.WeightGrams = &weightGrams.Float64
		}
		if color.Valid {
			s.Color = color.String
		}
		if batchNum.Valid {
			s.BatchNumber = batchNum.String
		}
		if manufacturer.Valid {
			s.Manufacturer = manufacturer.String
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
		log.Error("rows iteration error", zap.Error(err))
		return nil, fmt.Errorf("error iterating samples: %w", err)
	}

	log.Debug("samples retrieved successfully",
		zap.String("group_id", groupID))
	if samples == nil {
		return []models.Sample{}, nil
	}

	return samples, nil
}
