package service

import (
	"context"
	"desktop_lab/internal/models"

	"go.uber.org/zap"
)

type DimensionService struct {
	repo DimensionRepo
	log  *zap.Logger
}

func NewDimensionService(repo DimensionRepo, log *zap.Logger) *DimensionService {
	return &DimensionService{repo: repo, log: log}
}

func (s *DimensionService) GetAvailableDimensions(ctx context.Context) ([]models.ContextDimension, error) {
	log := loggerWith(ctx, s.log, zap.String("service_name", "GetAvailableDimensions"))
	log.Debug("getting available dimensions")

	dimensions, err := s.repo.GetAvailableDimensions(ctx)
	if err != nil {
		log.Error("failed to get available dimensions from repo", zap.Error(err))
		return nil, err
	}

	if len(dimensions) == 0 {
		log.Warn("no dimensions found")
	} else {
		log.Debug("successfully retrieved dimensions", zap.Int("count", len(dimensions)))
	}

	return dimensions, nil
}

func (s *DimensionService) AddDimension(ctx context.Context, dim models.ContextDimension) error {
	log := loggerWith(ctx, s.log, zap.String("service_name", "AddDimension"))
	log.Debug("adding new dimension", zap.String("key_name", dim.KeyName), zap.String("label", dim.Label))

	err := s.repo.AddDimension(ctx, dim)
	if err != nil {
		log.Error("failed to add dimension to repo", zap.Error(err), zap.String("key_name", dim.KeyName))
		return err
	}

	log.Info("dimension successfully added", zap.String("key_name", dim.KeyName))
	return nil
}

func (s *DimensionService) UpdatePossibleValues(ctx context.Context, id string, values []string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "UpdatePossibleValues"),
		zap.String("dimension_id", id),
		zap.Int("values_count", len(values)),
	)
	log.Debug("updating possible values for dimension")

	err := s.repo.UpdatePossibleValues(ctx, id, values)
	if err != nil {
		log.Error("failed to update possible values", zap.Error(err))
		return err
	}

	log.Info("possible values successfully updated")
	return nil
}

func (s *DimensionService) DeletePossibleValues(ctx context.Context, id string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "DeletePossibleValues"),
		zap.String("dimension_id", id),
	)
	log.Debug("deleting possible values for dimension")

	err := s.repo.DeletePossibleValues(ctx, id)
	if err != nil {
		log.Error("failed to delete possible values", zap.Error(err))
		return err
	}

	log.Info("possible values successfully deleted")
	return nil
}

func (s *DimensionService) DeleteDimension(ctx context.Context, id string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "DeleteDimension"),
		zap.String("dimension_id", id),
	)
	log.Info("deleting dimension")

	err := s.repo.DeleteDimension(ctx, id)
	if err != nil {
		log.Error("failed to delete dimension from repo", zap.Error(err))
		return err
	}

	log.Info("dimension successfully deleted")
	return nil
}

func (s *DimensionService) GetDimensionByKey(ctx context.Context, key string) (models.ContextDimension, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetDimensionByKey"),
		zap.String("dimension_key", key),
	)
	log.Debug("getting dimension by key")

	dim, err := s.repo.GetDimensionByKey(ctx, key)
	if err != nil {
		log.Error("failed to get dimension by key from repo", zap.Error(err))
		return models.ContextDimension{}, err
	}

	log.Debug("dimension successfully retrieved", zap.String("key_name", dim.KeyName))
	return dim, nil
}
