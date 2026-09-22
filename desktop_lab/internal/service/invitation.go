package service

import (
	"context"
	"desktop_lab/internal/models"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInvitationNotFound = errors.New("заявка не найдена")
	ErrInvitationExists   = errors.New("заявка уже существует")
	ErrInvalidRole        = errors.New("недопустимая роль")
	ErrAlreadyReviewed    = errors.New("заявка уже рассмотрена")
)

type InvitationRepo interface {
	Create(ctx context.Context, invitation models.RegistrationInvitation) error
	GetByID(ctx context.Context, id string) (models.RegistrationInvitation, error)
	GetByEmail(ctx context.Context, email string) (models.RegistrationInvitation, error)
	List(ctx context.Context, filter models.InvitationListFilter) ([]models.RegistrationInvitation, int64, error)
	UpdateStatus(ctx context.Context, id string, status models.InvitationStatus, reviewedBy *string, reviewedAt *time.Time, message string) error
	Delete(ctx context.Context, id string) error
}

type InvitationService struct {
	repo     InvitationRepo
	userRepo UserRepo
	log      *zap.Logger
}

func NewInvitationService(repo InvitationRepo, userRepo UserRepo, log *zap.Logger) *InvitationService {
	return &InvitationService{
		repo:     repo,
		userRepo: userRepo,
		log:      log,
	}
}

// CreateInvitation создает новую заявку на регистрацию
func (s *InvitationService) CreateInvitation(ctx context.Context, req models.CreateInvitationRequest) (models.RegistrationInvitation, error) {
	logger := loggerWith(ctx, s.log,
		zap.String("operation", "CreateInvitation"),
		zap.String("email", req.Email),
		zap.String("role", req.Role))

	logger.Info("creating new registration invitation")

	// Проверяем допустимость роли
	role := models.Role(req.Role)
	if !role.IsValid() {
		logger.Error("invalid role", zap.String("role", req.Role))
		return models.RegistrationInvitation{}, ErrInvalidRole
	}

	// Проверяем, не существует ли уже заявка с этим email
	existingInvite, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existingInvite.ID != "" {
		if existingInvite.Status == models.InvitationStatusPending {
			logger.Warn("pending invitation already exists", zap.String("existing_id", existingInvite.ID))
			return models.RegistrationInvitation{}, ErrInvitationExists
		}
		// Если заявка уже рассмотрена, можно создать новую
	}

	now := time.Now()
	invitation := models.RegistrationInvitation{
		ID:        uuid.New().String(),
		Email:     req.Email,
		Name:      req.Name,
		Role:      req.Role,
		Status:    models.InvitationStatusPending,
		Message:   req.Message,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, invitation); err != nil {
		logger.Error("failed to create invitation", zap.Error(err))
		return models.RegistrationInvitation{}, err
	}

	logger.Info("invitation created successfully", zap.String("invitation_id", invitation.ID))

	return invitation, nil
}

// GetInvitation получает заявку по ID
func (s *InvitationService) GetInvitation(ctx context.Context, id string) (models.RegistrationInvitation, error) {
	logger := loggerWith(ctx, s.log,
		zap.String("invitation_id", id),
		zap.String("operation", "GetInvitation"))

	logger.Debug("fetching invitation")

	invitation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("failed to get invitation", zap.Error(err))
		return models.RegistrationInvitation{}, ErrInvitationNotFound
	}

	logger.Debug("invitation retrieved successfully")
	return invitation, nil
}

// ListInvitations получает список заявок с фильтрацией
func (s *InvitationService) ListInvitations(ctx context.Context, filter models.InvitationListFilter) ([]models.RegistrationInvitation, int64, error) {
	logger := loggerWith(ctx, s.log,
		zap.Any("filter", filter),
		zap.String("operation", "ListInvitations"))

	logger.Debug("fetching invitations list")

	return s.repo.List(ctx, filter)
}

// ReviewInvitation рассматривает заявку (подтверждает или отказывает)
func (s *InvitationService) ReviewInvitation(ctx context.Context, id string, adminID string, req models.ReviewInvitationRequest) error {
	logger := loggerWith(ctx, s.log,
		zap.String("operation", "ReviewInvitation"),
		zap.String("invitation_id", id),
		zap.String("admin_id", adminID))

	logger.Info("reviewing invitation")

	// Получаем заявку
	invitation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("invitation not found", zap.Error(err))
		return ErrInvitationNotFound
	}

	// Проверяем, не была ли заявка уже рассмотрена
	if invitation.Status != models.InvitationStatusPending {
		logger.Error("invitation already reviewed", zap.String("status", string(invitation.Status)))
		return ErrAlreadyReviewed
	}

	// Определяем новый статус
	var newStatus models.InvitationStatus
	if req.Approved {
		newStatus = models.InvitationStatusAccepted
	} else {
		newStatus = models.InvitationStatusDeclined
	}

	// Обновляем статус заявки
	now := time.Now()
	if err := s.repo.UpdateStatus(ctx, id, newStatus, &adminID, &now, req.Message); err != nil {
		logger.Error("failed to update invitation status", zap.Error(err))
		return fmt.Errorf("не удалось обновить статус заявки: %w", err)
	}

	// Если заявка одобрена, создаем пользователя
	if req.Approved {
		logger.Info("creating user for approved invitation",
			zap.String("email", invitation.Email),
			zap.String("name", invitation.Name),
			zap.String("role", invitation.Role))

		user := models.User{
			ID:       uuid.New().String(),
			Email:    invitation.Email,
			Name:     invitation.Name,
			Role:     models.Role(invitation.Role),
			Password: "", // Пользователь должен будет установить пароль при первом входе
		}

		if err := s.userRepo.Create(ctx, user); err != nil {
			logger.Error("failed to create user after approval", zap.Error(err))
			// Откатываем статус заявки
			s.repo.UpdateStatus(ctx, id, models.InvitationStatusPending, nil, nil, "")
			return fmt.Errorf("не удалось создать пользователя: %w", err)
		}

		logger.Info("user created successfully", zap.String("user_id", user.ID))
	}

	logger.Info("invitation reviewed successfully",
		zap.String("invitation_id", id),
		zap.String("new_status", string(newStatus)))

	return nil
}

// DeleteInvitation удаляет заявку (только для администраторов)
func (s *InvitationService) DeleteInvitation(ctx context.Context, id string) error {
	logger := loggerWith(ctx, s.log,
		zap.String("invitation_id", id),
		zap.String("operation", "DeleteInvitation"))

	logger.Info("deleting invitation")

	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		logger.Error("invitation not found", zap.Error(err))
		return ErrInvitationNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		logger.Error("failed to delete invitation", zap.Error(err))
		return err
	}

	logger.Info("invitation deleted successfully")
	return nil
}

// GetAvailableRoles возвращает доступные роли для регистрации
func (s *InvitationService) GetAvailableRoles() []map[string]string {
	return []map[string]string{
		{"role": "admin", "description": "Администратор системы - полный доступ"},
		{"role": "manager", "description": "Менеджер - управление заявками"},
		{"role": "engineer", "description": "Инженер - выполнение исследований"},
		{"role": "technician", "description": "Техник - проведение тестов"},
		{"role": "client", "description": "Клиент - просмотр своих заявок"},
	}
}
