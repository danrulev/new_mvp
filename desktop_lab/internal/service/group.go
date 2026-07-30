package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ExperimentGroupService struct {
	repo ExperimentGroupRepo
	log  *zap.Logger
}

func NewExperimentGroupService(repo ExperimentGroupRepo, log *zap.Logger) *ExperimentGroupService {
	return &ExperimentGroupService{repo: repo, log: log}
}

func (s *ExperimentGroupService) Create(ctx context.Context, name, projectName, location, materialID string) (models.ExperimentGroup, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "Create"),
		zap.String("group_name", name),
		zap.String("project_name", projectName),
		zap.String("location", location),
		zap.String("material_id", materialID),
	)
	log.Debug("creating new experiment group")

	g := models.ExperimentGroup{
		ID:          uuid.New().String(),
		Name:        name,
		ProjectName: projectName,
		Location:    location,
		MaterialID:  materialID,
	}

	if err := s.repo.Create(ctx, g); err != nil {
		log.Error("failed to create experiment group in repo", zap.Error(err))
		return models.ExperimentGroup{}, err
	}

	log.Info("experiment group successfully created", zap.String("group_id", g.ID))
	return g, nil
}

func (s *ExperimentGroupService) GetList(ctx context.Context, limit, offset int64) (models.GroupListResponse, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetList"),
		zap.Int64("limit", limit),
		zap.Int64("offset", offset),
	)
	log.Debug("fetching experiment groups list")

	groups, total, err := s.repo.GetList(ctx, limit, offset)
	if err != nil {
		log.Error("failed to get groups list from repo", zap.Error(err))
		return models.GroupListResponse{}, err
	}

	meta := models.MakePaginatedMetadata(limit, offset, total)

	log.Debug("successfully retrieved groups list",
		zap.Int("returned_count", len(groups)),
		zap.Int64("total_count", total),
	)

	return models.GroupListResponse{
		Items: groups,
		Meta:  meta,
	}, nil
}

func (s *ExperimentGroupService) GetByID(ctx context.Context, id string) (models.ExperimentGroup, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetByID"),
		zap.String("group_id", id),
	)
	log.Debug("fetching experiment group by ID")

	g, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to get group from repo", zap.Error(err))
		return models.ExperimentGroup{}, err
	}

	if g.ID == "" {
		log.Warn("group not found in repo", zap.String("searched_id", id))
		return models.ExperimentGroup{}, fmt.Errorf("group not found")
	}

	log.Debug("group successfully retrieved", zap.String("group_name", g.Name))
	return g, nil
}

func (s *ExperimentGroupService) UpdateGroupByID(ctx context.Context, id string, g models.UpdateExperimentGroup) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "UpdateGroupByID"),
		zap.String("group_id", id),
	)
	log.Debug("updating experiment group", zap.Any("update_payload", g))

	err := s.repo.UpdateGroup(ctx, id, g)
	if err != nil {
		log.Error("failed to update group in repo", zap.Error(err))
		return err
	}

	log.Info("experiment group successfully updated")
	return nil
}

func (s *ExperimentGroupService) DeleteGroupByID(ctx context.Context, id string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "DeleteGroupByID"),
		zap.String("group_id", id),
	)
	log.Info("deleting experiment group")

	err := s.repo.DeleteGroup(ctx, id)
	if err != nil {
		log.Error("failed to delete group from repo", zap.Error(err))
		return err
	}

	log.Info("experiment group successfully deleted")
	return nil
}
