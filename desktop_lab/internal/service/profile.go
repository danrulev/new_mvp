package service

import (
	"context"
	"desktop_lab/internal/models"

	"go.uber.org/zap"
)

type ProfileService struct {
	userRepo UserRepo
}

func NewProfileService(userRepo UserRepo, log *zap.Logger) *ProfileService {
	return &ProfileService{
		userRepo: userRepo,
	}
}

func (s *ProfileService) GetProfile(ctx context.Context, userID string) (models.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *ProfileService) UpdateProfile(ctx context.Context, userID string, req models.UpdateUserRequest) (models.User, error) {
	return s.userRepo.Update(ctx, userID, req)
}

func (s *ProfileService) DeleteProfile(ctx context.Context, userID string) error {
	return s.userRepo.Delete(ctx, userID)
}

// GetEmployeesList возвращает список всех активных сотрудников
func (s *ProfileService) GetEmployeesList(ctx context.Context) ([]models.User, error) {
	return s.userRepo.ListActive(ctx)
}
