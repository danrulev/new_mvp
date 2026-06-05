package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MaterialService struct {
	repo MaterialRepo
	log  *zap.Logger
}

func NewMaterialService(repo MaterialRepo, log *zap.Logger) *MaterialService {
	return &MaterialService{repo: repo, log: log}
}

func (s *MaterialService) Create(ctx context.Context, name, code string) (models.Material, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "Create"),
		zap.String("material_name", name),
		zap.String("material_code", code),
	)
	log.Debug("creating new material")

	if name == "" {
		log.Warn("validation failed: name is required")
		return models.Material{}, fmt.Errorf("name is required")
	}

	mat := models.Material{
		ID:   uuid.New().String(),
		Name: name,
		Code: code,
	}

	if err := s.repo.Create(ctx, mat); err != nil {
		log.Error("failed to create material in repo", zap.Error(err))
		return models.Material{}, err
	}

	log.Info("material successfully created", zap.String("material_id", mat.ID))
	return mat, nil
}

func (s *MaterialService) GetAll(ctx context.Context) ([]models.Material, error) {
	log := loggerWith(ctx, s.log, zap.String("service_name", "GetAll"))
	log.Debug("fetching all materials")

	mats, err := s.repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get materials from repo", zap.Error(err))
		return nil, err
	}

	log.Debug("successfully retrieved materials", zap.Int("count", len(mats)))
	return mats, nil
}

func (s *MaterialService) GetByID(ctx context.Context, id string) (models.Material, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetByID"),
		zap.String("material_id", id),
	)
	log.Debug("fetching material by ID")

	mat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get material from repo", zap.Error(err))
		return models.Material{}, err
	}

	if mat.ID == "" {
		log.Warn("material not found in repo", zap.String("searched_id", id))
		return models.Material{}, fmt.Errorf("material not found")
	}

	log.Debug("material successfully retrieved", zap.String("material_name", mat.Name))
	return mat, nil
}

func (s *MaterialService) GetOrCreate(ctx context.Context, name, code string) (string, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetOrCreate"),
		zap.String("material_name", name),
		zap.String("material_code", code),
	)
	log.Debug("getting or creating material")

	// Пробуем найти
	mat, err := s.repo.GetByName(ctx, name)
	if err != nil {
		log.Error("failed to search material by name in repo", zap.Error(err))
		return "", err
	}

	// Если нашли, возвращаем ID
	if mat.ID != "" {
		log.Debug("material found, returning existing ID", zap.String("material_id", mat.ID))
		return mat.ID, nil
	}

	// Если не нашли, создаем
	log.Info("material not found, creating new one")
	newMat, err := s.Create(ctx, name, code)
	if err != nil {
		// Create уже залогировал ошибку, но добавим контекст
		log.Error("failed to create new material in GetOrCreate", zap.Error(err))
		return "", err
	}

	log.Info("new material created via GetOrCreate", zap.String("material_id", newMat.ID))
	return newMat.ID, nil
}

func (s *MaterialService) GetContextDimensionsByMaterialID(ctx context.Context, materialID string) ([]models.ContextDimension, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetContextDimensionsByMaterialID"),
		zap.String("material_id", materialID),
	)
	log.Debug("fetching context dimensions for material")

	dimensions, err := s.repo.GetContextDimensionsByMaterialID(ctx, materialID)
	if err != nil {
		log.Error("failed to get context dimensions from repo", zap.Error(err))
		return nil, err
	}

	log.Debug("successfully retrieved context dimensions", zap.Int("count", len(dimensions)))
	return dimensions, nil
}

func (s *MaterialService) AddContextDimensionToMaterial(ctx context.Context, materialID, dimensionID string, isRequired bool) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "AddContextDimensionToMaterial"),
		zap.String("material_id", materialID),
		zap.String("dimension_id", dimensionID),
		zap.Bool("is_required", isRequired),
	)
	log.Debug("adding context dimension to material")

	err := s.repo.AddContextDimensionToMaterial(ctx, materialID, dimensionID, isRequired)
	if err != nil {
		log.Error("failed to add context dimension to material in repo", zap.Error(err))
		return err
	}

	log.Info("context dimension successfully added to material")
	return nil
}

func (s *MaterialService) DeleteContextDimensionFromMaterial(ctx context.Context, materialID, dimensionID string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "DeleteContextDimensionFromMaterial"),
		zap.String("material_id", materialID),
		zap.String("dimension_id", dimensionID),
	)
	log.Info("deleting context dimension from material")

	err := s.repo.DeleteContextDimensionFromMaterial(ctx, materialID, dimensionID)
	if err != nil {
		log.Error("failed to delete context dimension from material in repo", zap.Error(err))
		return err
	}

	log.Info("context dimension successfully deleted from material")
	return nil
}
