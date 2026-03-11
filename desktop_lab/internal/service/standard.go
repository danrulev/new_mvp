package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
	"fmt"

	"go.uber.org/zap"
)

type StandardService struct {
	repo repository.StandardRepo
	log  *zap.Logger
}

func NewStandardService(repo repository.StandardRepo, log *zap.Logger) *StandardService {
	return &StandardService{repo: repo, log: log}
}

// CreateStandard заменяет старый CreateGOST
// Принимает DTO, который содержит всю иерархию: стандарт -> размеры -> методы -> инпуты -> лимиты -> условия
func (s *StandardService) CreateStandard(ctx context.Context, req models.CreateStandardRequest) (string, error) {
	if req.MaterialID == "" || req.Name == "" {
		return "", fmt.Errorf("material_id and name are required")
	}

	// Валидация методов перед сохранением
	for i, m := range req.Methods {
		if m.Name == "" {
			return "", fmt.Errorf("method[%d] name is required", i)
		}
		if m.FormulaExpr != "" {
			// Тут можно добавить проверку синтаксиса формулы через govaluate заранее
			// Но оставим это на момент расчета или доверим репозиторию
		}
	}

	id, err := s.repo.CreateFull(ctx, req)
	if err != nil {
		s.log.Error("failed to create standard", zap.Error(err))
		return "", fmt.Errorf("ошибка создания стандарта: %w", err)
	}

	s.log.Info("standard created successfully", zap.String("id", id), zap.String("name", req.Name))
	return id, nil
}

// GetByMaterialID загружает стандарты для материала
func (s *StandardService) GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error) {
	stds, err := s.repo.GetByMaterialID(ctx, materialID)
	if err != nil {
		return nil, err
	}

	// Дополнительно можно загрузить Dimensions для каждого стандарта, если нужно для UI
	// В текущей реализации репозитория они не грузятся глубоко, можно доработать при необходимости

	s.log.Info("standard loaded successfully", zap.Int("count", len(stds)))
	return stds, nil
}

// GetMethodDetails загружает метод со всеми входными параметрами (для формы ввода)
func (s *StandardService) GetMethodDetails(ctx context.Context, methodID string) (models.TestMethod, error) {
	method, err := s.repo.GetMethodWithInputs(ctx, methodID)
	if err != nil {
		return models.TestMethod{}, err
	}
	if method.ID == "" {
		return models.TestMethod{}, fmt.Errorf("method not found")
	}
	s.log.Info("method loaded successfully", zap.String("id", method.ID), zap.String("name", method.Name))
	return method, nil
}

// GetMethodsByStandardID возвращает методы стандарта с входными параметрами
func (s *StandardService) GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error) {
	// Вызываем метод репозитория
	methods, err := s.repo.GetMethodsByStandardID(ctx, standardID)
	if err != nil {
		s.log.Error("failed to get methods by standard id", zap.Error(err))
		return nil, err
	}

	s.log.Info("methods loaded successfully", zap.Int("count", len(methods)))
	return methods, nil
}
