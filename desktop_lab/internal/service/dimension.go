package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"

	"go.uber.org/zap"
)

type DimensionService struct {
	repo repository.DimensionRepo
	log  *zap.Logger
}

func NewDimensionService(repo repository.DimensionRepo, log *zap.Logger) *DimensionService {
	return &DimensionService{repo: repo, log: log}
}

func (s *DimensionService) GetAvailableDimensions(ctx context.Context) ([]models.ContextDimension, error) {
	return s.repo.GetAvailableDimensions(ctx)
}

func (s *DimensionService) AddDimension(ctx context.Context, dim models.ContextDimension) error {
	return s.repo.AddDimension(ctx, dim)
}

func (s *DimensionService) UpdatePossibleValues(ctx context.Context, id string, values []string) error {
	return s.repo.UpdatePossibleValues(ctx, id, values)
}

func (s *DimensionService) DeletePossibleValues(ctx context.Context, id string) error {
	return s.repo.DeletePossibleValues(ctx, id)
}

func (s *DimensionService) DeleteDimension(ctx context.Context, id string) error {
	return s.repo.DeleteDimension(ctx, id)
}

func (s *DimensionService) GetDimensionByKey(ctx context.Context, key string) (models.ContextDimension, error) {
	return s.repo.GetDimensionByKey(ctx, key)
}
