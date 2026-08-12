// internal/service/sample_service.go
package service

import (
	"context"
	"desktop_lab/internal/models"

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

	sample := models.Sample{
		ID:           uuid.New().String(),
		GroupID:      groupID,
		MaterialID:   dto.MaterialID,
		SampleNumber: dto.SampleNumber,
		ContextParams: dto.ContextParams,
		Note:         dto.Note,
		PhotoURL:     dto.PhotoURL,
		LengthMM:     dto.LengthMM,
		WidthMM:      dto.WidthMM,
		HeightMM:     dto.HeightMM,
		Shape:        dto.Shape,
		WeightGrams:  dto.WeightGrams,
		Color:        dto.Color,
		BatchNumber:  dto.BatchNumber,
		Manufacturer: dto.Manufacturer,
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
