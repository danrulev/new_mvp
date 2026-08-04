package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrganizationService struct {
	org   OrganizationRepo
	user  OrganizationUserRepo
	tests OrganizationTestsRepo

	log *zap.Logger
}

func NewOrganizationService(org OrganizationRepo, user OrganizationUserRepo, tests OrganizationTestsRepo, log *zap.Logger) *OrganizationService {
	return &OrganizationService{
		org:   org,
		user:  user,
		tests: tests,
		log:   log,
	}
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, req models.CreateOrganizationRequest) (string, error) {
	id := uuid.New().String()
	err := s.org.Create(ctx, id, req)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (s *OrganizationService) GetOrganizationByID(ctx context.Context, id string) (models.Organization, error) {
	return s.org.GetByID(ctx, id)
}

func (s *OrganizationService) GetOrganizationByName(ctx context.Context, name string) (models.Organization, error) {
	return s.org.GetOrganizationByName(ctx, name)
}

func (s *OrganizationService) ListOrganizations(ctx context.Context, limit, offset int64) (models.OrganizationListResponse, error) {
	organizations, total, err := s.org.List(ctx, limit, offset)
	if err != nil {
		return models.OrganizationListResponse{}, err
	}

	return models.OrganizationListResponse{Organizations: organizations, Meta: models.MakePaginatedMetadata(limit, offset, total)}, nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, userID, organizationID string, req models.UpdateOrganizationRequest) (models.Organization, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return models.Organization{}, err
	}

	if user.Role != "super_admin" && user.OrganizationID != organizationID {
		return models.Organization{}, fmt.Errorf("permission denied")
	}
	return s.org.Update(ctx, organizationID, req)
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, userID, organizationID string) error {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Role != "super_admin" && user.OrganizationID != organizationID {
		return fmt.Errorf("permission denied")
	}
	return s.org.Delete(ctx, organizationID)
}

func (s *OrganizationService) CreateOrganizationUser(ctx context.Context, userID, organizationID string, req models.CreateOrganizationUserRequest) (string, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if ((user.Role != "super_admin" && user.Role != "admin") && user.OrganizationID != organizationID) || user.OrganizationID != req.OrganizationID {
		return "", fmt.Errorf("permission denied")
	}
	id := uuid.New().String()
	if err := s.user.Create(ctx, id, req); err != nil {
		return "", err
	}

	return id, nil
}

func (s *OrganizationService) GetOrganizationUserByID(ctx context.Context, userID, organizationID string, id string) (models.OrganizationUser, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return models.OrganizationUser{}, err
	}

	profile, err := s.user.GetByID(ctx, id)
	if err != nil {
		return models.OrganizationUser{}, err
	}

	if (user.OrganizationID != organizationID) && (profile.OrganizationID != organizationID) {
		return models.OrganizationUser{}, fmt.Errorf("permission denied")
	}

	return profile, nil
}

func (s *OrganizationService) GetOrganizationUserByRole(ctx context.Context, userID, organizationID, role string, limit, offset int64) (models.OrganizationUserListResponse, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return models.OrganizationUserListResponse{}, err
	}

	if user.OrganizationID != organizationID {
		return models.OrganizationUserListResponse{}, fmt.Errorf("permission denied")
	}
	users, total, err := s.user.GetByRole(ctx, organizationID, role, limit, offset)
	if err != nil {
		return models.OrganizationUserListResponse{}, err
	}

	return models.OrganizationUserListResponse{OrganizationUsers: users, Meta: models.MakePaginatedMetadata(limit, offset, total)}, nil
}

func (s *OrganizationService) ListOrganizationUsers(ctx context.Context, userID, organizationID string, limit, offset int64) (models.OrganizationUserListResponse, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return models.OrganizationUserListResponse{}, err
	}

	if user.OrganizationID != organizationID {
		return models.OrganizationUserListResponse{}, fmt.Errorf("permission denied")
	}
	users, total, err := s.user.List(ctx, organizationID, limit, offset)
	if err != nil {
		return models.OrganizationUserListResponse{}, err
	}

	return models.OrganizationUserListResponse{OrganizationUsers: users, Meta: models.MakePaginatedMetadata(limit, offset, total)}, nil
}

func (s *OrganizationService) UpdateOrganizationUser(ctx context.Context, userID, organizationID, id string, role *string) (models.OrganizationUser, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return models.OrganizationUser{}, err
	}

	if user.Role != "super_admin" && user.Role != "admin" && user.OrganizationID != organizationID {
		return models.OrganizationUser{}, fmt.Errorf("permission denied")
	}

	user, err = s.user.UpdateUser(ctx, id, role)
	if err != nil {
		return models.OrganizationUser{}, err
	}

	return user, nil
}

func (s *OrganizationService) DeleteOrganizationUser(ctx context.Context, userID, organizationID, id string) error {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Role != "super_admin" && user.Role != "admin" && user.OrganizationID != organizationID {
		return fmt.Errorf("permission denied")
	}

	err = s.user.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *OrganizationService) CreateOrganizationTest(ctx context.Context, userID, organizationID string, req models.CreateOrganizationTestRequest) (string, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if user.Role != "super_admin" && user.Role != "admin" && user.OrganizationID != organizationID {
		return "", fmt.Errorf("permission denied")
	}

	id := uuid.New().String()
	err = s.tests.Create(ctx, id, req)
	if err != nil {
		return "", err
	}

	return id, nil
}

func (s *OrganizationService) GetOrganizationTest(ctx context.Context, id string) (models.OrganizationTest, error) {
	return s.tests.GetOrganizationTest(ctx, id)
}

func (s *OrganizationService) ListOrganizationTests(ctx context.Context, limit, offset int64) (models.ListOrganizationTests, error) {
	tests, total, err := s.tests.ListOrganizationTests(ctx, limit, offset)
	if err != nil {
		return models.ListOrganizationTests{}, err
	}

	return models.ListOrganizationTests{OrganizationTests: tests, Meta: models.MakePaginatedMetadata(limit, offset, total)}, nil
}

func (s *OrganizationService) UpdateOrganizationTest(ctx context.Context, userID, organizationID, id string, req models.UpdateOrganizationTestRequest) (models.OrganizationTest, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return models.OrganizationTest{}, err
	}

	if user.Role != "super_admin" && user.Role != "admin" && user.OrganizationID != organizationID {
		return models.OrganizationTest{}, fmt.Errorf("permission denied")
	}

	return s.tests.Update(ctx, id, req)
}

func (s *OrganizationService) DeleteOrganizationTest(ctx context.Context, userID, organizationID, id string) error {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.Role != "super_admin" && user.Role != "admin" && user.OrganizationID != organizationID {
		return fmt.Errorf("permission denied")
	}

	return s.tests.DeleteOrganizationTest(ctx, id)
}
