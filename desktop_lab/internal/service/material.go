package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type MaterialService struct {
	repo repository.MaterialRepo
	log  *zap.Logger
}

func NewMaterialService(repo repository.MaterialRepo, log *zap.Logger) *MaterialService {
	return &MaterialService{repo: repo, log: log}
}

func (s *MaterialService) Create(ctx context.Context, name, code string) (models.Material, error) {
	if name == "" {
		return models.Material{}, fmt.Errorf("name is required")
	}

	mat := models.Material{
		ID:   uuid.New().String(),
		Name: name,
		Code: code,
	}

	if err := s.repo.Create(ctx, mat); err != nil {
		s.log.Error("failed to create material", zap.Error(err))
		return models.Material{}, err
	}

	s.log.Info("material created", zap.String("id", mat.ID), zap.String("name", mat.Name))
	return mat, nil
}

func (s *MaterialService) GetAll(ctx context.Context) ([]models.Material, error) {
	mats, err := s.repo.GetAll(ctx)
	if err != nil {
		s.log.Error("failed to get materials", zap.Error(err))
		return nil, err
	}
	return mats, nil
}

func (s *MaterialService) GetByID(ctx context.Context, id string) (models.Material, error) {
	mat, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return models.Material{}, err
	}
	if mat.ID == "" {
		return models.Material{}, fmt.Errorf("material not found")
	}
	return mat, nil
}

func (s *MaterialService) GetOrCreate(ctx context.Context, name, code string) (string, error) {
	// Пробуем найти
	mat, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return "", err
	}

	// Если нашли, возвращаем ID
	if mat.ID != "" {
		return mat.ID, nil
	}

	// Если не нашли, создаем
	newMat, err := s.Create(ctx, name, code)
	if err != nil {
		return "", err
	}

	return newMat.ID, nil
}
