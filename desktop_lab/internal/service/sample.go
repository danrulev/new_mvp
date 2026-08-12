// internal/service/sample_service.go
package service

import (
	"context"
	"desktop_lab/internal/models"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type SampleService struct {
	repo SampleRepo
	log  *zap.Logger
}

func NewSampleService(repo SampleRepo, log *zap.Logger) *SampleService {
	return &SampleService{repo: repo, log: log}
}

// CreateSample создает пробу отдельно (если нужно без протокола сразу)
func (s *SampleService) CreateSample(ctx context.Context, dto models.CreateSampleDTO, groupID string) (models.Sample, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "CreateSample"),
		zap.String("group_id", groupID),
		zap.String("sample_number", dto.SampleNumber),
		zap.Int("context_params_count", len(dto.ContextParams)),
	)
	log.Debug("creating new sample")

	// Валидация обязательных полей
	if dto.SampleNumber == "" {
		return models.Sample{}, errors.New("sample_number is required")
	}
	if dto.MaterialID == "" {
		return models.Sample{}, errors.New("material_id is required")
	}
	if groupID == "" {
		return models.Sample{}, errors.New("group_id is required")
	}

	nowUTC := time.Now().UTC()
	sample := models.Sample{
		ID:            uuid.New().String(),
		GroupID:       groupID,
		MaterialID:    dto.MaterialID,
		SampleNumber:  dto.SampleNumber,
		ContextParams: dto.ContextParams,
		Note:          dto.Note,
		PhotoURL:      dto.PhotoURL,
		LengthMM:      dto.LengthMM,
		WidthMM:       dto.WidthMM,
		HeightMM:      dto.HeightMM,
		Shape:         dto.Shape,
		WeightGrams:   dto.WeightGrams,
		Color:         dto.Color,
		BatchNumber:   dto.BatchNumber,
		Manufacturer:  dto.Manufacturer,
		CreatedAt:     nowUTC,
		UpdatedAt:     nowUTC,
	}

	if dto.CollectionDate != nil {
		sample.CollectionDate = *dto.CollectionDate
	}
	sample.CollectionPlace = dto.CollectionPlace

	log.Debug("saving sample to repository", zap.String("sample_id", sample.ID))
	if err := s.repo.Create(ctx, sample); err != nil {
		log.Error("failed to create sample in repository",
			zap.Error(err),
			zap.String("sample_id", sample.ID),
			zap.String("group_id", groupID))
		return models.Sample{}, err
	}

	log.Info("sample successfully created",
		zap.String("sample_id", sample.ID),
		zap.String("sample_number", sample.SampleNumber))
	return sample, nil
}

// UpdateSample обновляет данные пробы с валидацией и логированием
func (s *SampleService) UpdateSample(ctx context.Context, id string, dto models.UpdateSampleDTO) (models.Sample, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "UpdateSample"),
		zap.String("sample_id", id),
	)
	log.Debug("updating sample")

	// Получаем текущий образец
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get existing sample", zap.Error(err))
		return models.Sample{}, fmt.Errorf("failed to get sample: %w", err)
	}
	if existing.ID == "" {
		return models.Sample{}, errors.New("sample not found")
	}

	// Валидация обязательных полей
	if dto.SampleNumber != "" {
		existing.SampleNumber = dto.SampleNumber
	} else {
		return models.Sample{}, errors.New("sample_number is required")
	}
	if dto.MaterialID != "" {
		existing.MaterialID = dto.MaterialID
	} else {
		return models.Sample{}, errors.New("material_id is required")
	}

	// Обновляем остальные поля из DTO
	if dto.CollectionPlace != "" {
		existing.CollectionPlace = dto.CollectionPlace
	}
	if dto.CollectionDate != nil {
		existing.CollectionDate = *dto.CollectionDate
	}
	if dto.ContextParams != nil {
		existing.ContextParams = dto.ContextParams
	}
	
	// Обновляем расширенные поля
	existing.Note = dto.Note
	existing.PhotoURL = dto.PhotoURL
	existing.LengthMM = dto.LengthMM
	existing.WidthMM = dto.WidthMM
	existing.HeightMM = dto.HeightMM
	existing.Shape = dto.Shape
	existing.WeightGrams = dto.WeightGrams
	existing.Color = dto.Color
	existing.BatchNumber = dto.BatchNumber
	existing.Manufacturer = dto.Manufacturer
	existing.UpdatedAt = time.Now().UTC()

	log.Debug("updating sample in repository", zap.String("sample_id", existing.ID))
	if err := s.repo.Update(ctx, existing); err != nil {
		log.Error("failed to update sample in repository",
			zap.Error(err),
			zap.String("sample_id", existing.ID))
		return models.Sample{}, fmt.Errorf("failed to update sample: %w", err)
	}

	log.Info("sample successfully updated",
		zap.String("sample_id", existing.ID),
		zap.String("sample_number", existing.SampleNumber))
	return existing, nil
}

// DeleteSample удаляет пробу по ID
func (s *SampleService) DeleteSample(ctx context.Context, id string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "DeleteSample"),
		zap.String("sample_id", id),
	)
	log.Debug("deleting sample")

	if id == "" {
		return errors.New("sample id is required")
	}

	// Проверяем существование образца
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get sample before deletion", zap.Error(err))
		return fmt.Errorf("failed to get sample: %w", err)
	}
	if existing.ID == "" {
		return errors.New("sample not found")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		log.Error("failed to delete sample in repository",
			zap.Error(err),
			zap.String("sample_id", id))
		return fmt.Errorf("failed to delete sample: %w", err)
	}

	log.Info("sample successfully deleted",
		zap.String("sample_id", id),
		zap.String("sample_number", existing.SampleNumber))
	return nil
}

// GetSampleByID возвращает пробу по ID
func (s *SampleService) GetSampleByID(ctx context.Context, id string) (models.Sample, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetSampleByID"),
		zap.String("sample_id", id),
	)
	log.Debug("getting sample by ID")

	if id == "" {
		return models.Sample{}, errors.New("sample id is required")
	}

	sample, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get sample from repository",
			zap.Error(err),
			zap.String("sample_id", id))
		return models.Sample{}, fmt.Errorf("failed to get sample: %w", err)
	}
	if sample.ID == "" {
		return models.Sample{}, errors.New("sample not found")
	}

	log.Debug("sample retrieved successfully",
		zap.String("sample_id", sample.ID),
		zap.String("sample_number", sample.SampleNumber))
	return sample, nil
}

// GetSamplesByGroupID возвращает все пробы группы
func (s *SampleService) GetSamplesByGroupID(ctx context.Context, groupID string) ([]models.Sample, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetSamplesByGroupID"),
		zap.String("group_id", groupID),
	)
	log.Debug("getting samples by group ID")

	if groupID == "" {
		return []models.Sample{}, nil
	}

	samples, err := s.repo.GetByGroupID(ctx, groupID)
	if err != nil {
		log.Error("failed to get samples from repository",
			zap.Error(err),
			zap.String("group_id", groupID))
		return nil, fmt.Errorf("failed to get samples: %w", err)
	}

	log.Debug("samples retrieved successfully",
		zap.String("group_id", groupID),
		zap.Int("count", len(samples)))
	return samples, nil
}
