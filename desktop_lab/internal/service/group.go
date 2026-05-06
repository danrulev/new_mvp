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

func (s *ExperimentGroupService) GetList(ctx context.Context, limit, offset int64) (models.GroupListResponse, error) {
	groups, total, err := s.repo.GetList(ctx, limit, offset)
	if err != nil {
		return models.GroupListResponse{}, err
	}

	meta := models.MakePaginatedMetadata(limit, offset, total)
	return models.GroupListResponse{
		Items: groups,
		Meta:  meta,
	}, nil
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

func (s *ExperimentGroupService) DeleteGroupByID(ctx context.Context, id string) error {
	err := s.repo.DeleteGroup(ctx, id)
	if err != nil {
		s.log.Error("failed to delete group", zap.Error(err))
		return err
	}

	return nil
}
