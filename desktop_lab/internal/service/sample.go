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
func (s *SampleService) CreateSample(ctx context.Context, groupID, number string, context map[string]string, note string) (models.Sample, error) {
	sample := models.Sample{
		ID:            uuid.New().String(),
		GroupID:       groupID,
		SampleNumber:  number,
		ContextParams: context,
		Note:          note,
	}
	if err := s.repo.Create(ctx, sample); err != nil {
		return models.Sample{}, err
	}
	return sample, nil
}
