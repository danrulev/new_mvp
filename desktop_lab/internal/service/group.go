// internal/service/group_service.go
package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ExperimentGroupService struct {
	repo repository.ExperimentGroupRepo
	log  *zap.Logger
}

func NewExperimentGroupService(repo repository.ExperimentGroupRepo, log *zap.Logger) *ExperimentGroupService {
	return &ExperimentGroupService{repo: repo, log: log}
}

func (s *ExperimentGroupService) Create(ctx context.Context, name, projectName, location, materialID string) (models.ExperimentGroup, error) {
	g := models.ExperimentGroup{
		ID:          uuid.New().String(),
		Name:        name,
		ProjectName: projectName,
		Location:    location,
		MaterialID:  materialID,
	}
	if err := s.repo.Create(ctx, g); err != nil {
		return models.ExperimentGroup{}, err
	}
	return g, nil
}

func (s *ExperimentGroupService) GetList(ctx context.Context, limit, offset int64) ([]models.ExperimentGroup, int64, error) {
	return s.repo.GetList(ctx, limit, offset)
}

func (s *ExperimentGroupService) GetByID(ctx context.Context, id string) (models.ExperimentGroup, error) {
	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return models.ExperimentGroup{}, err
	}
	if g.ID == "" {
		return models.ExperimentGroup{}, fmt.Errorf("group not found")
	}
	return g, nil
}
