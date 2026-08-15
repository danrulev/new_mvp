package service

import (
"context"
"crypto/rand"
"encoding/hex"
"desktop_lab/internal/models"
"errors"
"fmt"
"time"

"github.com/google/uuid"
"go.uber.org/zap"
)

var (
ErrInvitationNotFound    = errors.New("приглашение не найдено")
ErrInvitationExpired     = errors.New("срок действия приглашения истек")
ErrInvitationAlreadyUsed = errors.New("приглашение уже было использовано")
ErrInvalidRole           = errors.New("недопустимая роль")
)

type InvitationRepo interface {
Create(ctx context.Context, invitation models.OrganizationInvitation) error
GetByID(ctx context.Context, id string) (models.OrganizationInvitation, error)
GetByToken(ctx context.Context, token string) (models.OrganizationInvitation, error)
List(ctx context.Context, filter models.InvitationListFilter) ([]models.OrganizationInvitation, int64, error)
UpdateStatus(ctx context.Context, id string, status models.InvitationStatus, acceptedAt *time.Time) error
Delete(ctx context.Context, id string) error
GetPendingByOrgAndEmail(ctx context.Context, orgID, email string) ([]models.OrganizationInvitation, error)
UpdateTokenAndExpires(ctx context.Context, id, token string, expiresAt time.Time) error
AddOrganizationUser(ctx context.Context, orgUser models.OrganizationUser) error
}



type InvitationService struct {
repo     InvitationRepo
orgRepo  OrganizationRepo
userRepo UserRepo
log      *zap.Logger
}

func NewInvitationService(repo InvitationRepo, orgRepo OrganizationRepo, userRepo UserRepo, log *zap.Logger) *InvitationService {
return &InvitationService{
repo:     repo,
orgRepo:  orgRepo,
userRepo: userRepo,
log:      log,
}
}

// generateToken генерирует случайный токен для приглашения
func (s *InvitationService) generateToken() (string, error) {
bytes := make([]byte, 32)
if _, err := rand.Read(bytes); err != nil {
return "", fmt.Errorf("failed to generate random bytes: %w", err)
}
return hex.EncodeToString(bytes), nil
}

// CreateInvitation создает новое приглашение в организацию
func (s *InvitationService) CreateInvitation(ctx context.Context, req models.CreateInvitationRequest, inviterID, inviterName string) (models.OrganizationInvitation, error) {
logger := loggerWith(ctx, s.log,
zap.String("operation", "CreateInvitation"),
zap.String("organization_id", req.OrganizationID),
zap.String("email", req.Email),
zap.String("role", req.Role))

logger.Info("creating new invitation")

// Проверяем существование организации
org, err := s.orgRepo.GetByID(ctx, req.OrganizationID)
if err != nil {
logger.Error("organization not found", zap.Error(err))
return models.OrganizationInvitation{}, fmt.Errorf("организация не найдена: %w", err)
}

// Проверяем допустимость роли
role := models.Role(req.Role)
if !role.IsValid() {
logger.Error("invalid role", zap.String("role", req.Role))
return models.OrganizationInvitation{}, ErrInvalidRole
}

// Проверяем, нет ли уже активных приглашений для этого email в этой организации
existingInvites, err := s.repo.GetPendingByOrgAndEmail(ctx, req.OrganizationID, req.Email)
if err != nil {
logger.Warn("failed to check existing invitations", zap.Error(err))
} else if len(existingInvites) > 0 {
logger.Warn("pending invitation already exists", zap.String("existing_id", existingInvites[0].ID))
return models.OrganizationInvitation{}, fmt.Errorf("уже существует активное приглашение для email %s", req.Email)
}

token, err := s.generateToken()
if err != nil {
logger.Error("failed to generate token", zap.Error(err))
return models.OrganizationInvitation{}, err
}

now := time.Now()
invitation := models.OrganizationInvitation{
ID:               uuid.New().String(),
OrganizationID:   req.OrganizationID,
Email:            req.Email,
Role:             req.Role,
InvitedBy:        inviterID,
InviterName:      inviterName,
OrganizationName: org.Name,
Status:           models.InvitationStatusPending,
Token:            token,
CreatedAt:        now,
ExpiresAt:        now.Add(7 * 24 * time.Hour), // 7 дней
}

if err := s.repo.Create(ctx, invitation); err != nil {
logger.Error("failed to create invitation", zap.Error(err))
return models.OrganizationInvitation{}, err
}

logger.Info("invitation created successfully", zap.String("invitation_id", invitation.ID))

// Возвращаем приглашение без токена (токен нужно отправить только на email)
invitation.Token = ""
return invitation, nil
}

// GetInvitation получает приглашение по ID
func (s *InvitationService) GetInvitation(ctx context.Context, id string) (models.OrganizationInvitation, error) {
logger := loggerWith(ctx, s.log,
zap.String("invitation_id", id),
zap.String("operation", "GetInvitation"))

logger.Debug("fetching invitation")

invitation, err := s.repo.GetByID(ctx, id)
if err != nil {
logger.Error("failed to get invitation", zap.Error(err))
return models.OrganizationInvitation{}, ErrInvitationNotFound
}

logger.Debug("invitation retrieved successfully")
return invitation, nil
}

// GetInvitationByToken получает приглашение по токену
func (s *InvitationService) GetInvitationByToken(ctx context.Context, token string) (models.OrganizationInvitation, error) {
logger := loggerWith(ctx, s.log,
zap.String("operation", "GetInvitationByToken"))

logger.Debug("fetching invitation by token")

invitation, err := s.repo.GetByToken(ctx, token)
if err != nil {
logger.Error("failed to get invitation by token", zap.Error(err))
return models.OrganizationInvitation{}, ErrInvitationNotFound
}

logger.Debug("invitation retrieved successfully")
return invitation, nil
}

// ListInvitations получает список приглашений с фильтрацией
func (s *InvitationService) ListInvitations(ctx context.Context, filter models.InvitationListFilter) ([]models.OrganizationInvitation, int64, error) {
logger := loggerWith(ctx, s.log,
zap.Any("filter", filter),
zap.String("operation", "ListInvitations"))

logger.Debug("fetching invitations list")

return s.repo.List(ctx, filter)
}

// AcceptInvitation принимает приглашение и добавляет пользователя в организацию
func (s *InvitationService) AcceptInvitation(ctx context.Context, token string, userID string) error {
logger := loggerWith(ctx, s.log,
zap.String("operation", "AcceptInvitation"),
zap.String("user_id", userID))

logger.Info("accepting invitation")

// Получаем приглашение по токену
invitation, err := s.repo.GetByToken(ctx, token)
if err != nil {
logger.Error("invitation not found", zap.Error(err))
return ErrInvitationNotFound
}

// Проверяем, не истекло ли приглашение
if invitation.IsExpired() {
logger.Error("invitation expired", zap.Time("expires_at", invitation.ExpiresAt))
return ErrInvitationExpired
}

// Проверяем статус приглашения
if !invitation.CanBeAccepted() {
logger.Error("invitation cannot be accepted", zap.String("status", string(invitation.Status)))
if invitation.Status == models.InvitationStatusAccepted {
return ErrInvitationAlreadyUsed
}
if invitation.Status == models.InvitationStatusDeclined {
return errors.New("приглашение было отклонено")
}
return fmt.Errorf("приглашение не может быть принято в текущем статусе: %s", invitation.Status)
}

// Проверяем существование пользователя
user, err := s.userRepo.GetByID(ctx, userID)
if err != nil {
logger.Error("user not found", zap.Error(err))
return fmt.Errorf("пользователь не найден: %w", err)
}

// Проверяем, что email пользователя совпадает с email в приглашении
if user.Email != invitation.Email {
logger.Error("email mismatch", zap.String("user_email", user.Email), zap.String("invite_email", invitation.Email))
return errors.New("email пользователя не совпадает с email в приглашении")
}

// Добавляем пользователя в организацию с указанной ролью
orgUser := models.OrganizationUser{
ID:             uuid.New().String(),
OrganizationID: invitation.OrganizationID,
UserID:         userID,
Role:           invitation.Role,
CreatedAt:      time.Now(),
UpdatedAt:      time.Now(),
}

if err := s.repo.AddOrganizationUser(ctx, orgUser); err != nil {
logger.Error("failed to add user to organization", zap.Error(err))
return fmt.Errorf("не удалось добавить пользователя в организацию: %w", err)
}

// Обновляем статус приглашения
now := time.Now()
if err := s.repo.UpdateStatus(ctx, invitation.ID, models.InvitationStatusAccepted, &now); err != nil {
logger.Error("failed to update invitation status", zap.Error(err))
return fmt.Errorf("не удалось обновить статус приглашения: %w", err)
}

logger.Info("invitation accepted successfully",
zap.String("invitation_id", invitation.ID),
zap.String("organization_id", invitation.OrganizationID),
zap.String("user_id", userID))

return nil
}

// DeclineInvitation отклоняет приглашение
func (s *InvitationService) DeclineInvitation(ctx context.Context, token string) error {
logger := loggerWith(ctx, s.log,
zap.String("operation", "DeclineInvitation"))

logger.Info("declining invitation")

invitation, err := s.repo.GetByToken(ctx, token)
if err != nil {
logger.Error("invitation not found", zap.Error(err))
return ErrInvitationNotFound
}

if invitation.Status != models.InvitationStatusPending {
logger.Error("invitation is not pending", zap.String("status", string(invitation.Status)))
return fmt.Errorf("нельзя отклонить приглашение в статусе %s", invitation.Status)
}

if err := s.repo.UpdateStatus(ctx, invitation.ID, models.InvitationStatusDeclined, nil); err != nil {
logger.Error("failed to update invitation status", zap.Error(err))
return err
}

logger.Info("invitation declined successfully", zap.String("invitation_id", invitation.ID))
return nil
}

// RevokeInvitation отзывает приглашение (только для администраторов организации)
func (s *InvitationService) RevokeInvitation(ctx context.Context, id string) error {
logger := loggerWith(ctx, s.log,
zap.String("invitation_id", id),
zap.String("operation", "RevokeInvitation"))

logger.Info("revoking invitation")

invitation, err := s.repo.GetByID(ctx, id)
if err != nil {
logger.Error("invitation not found", zap.Error(err))
return ErrInvitationNotFound
}

if invitation.Status != models.InvitationStatusPending {
logger.Error("cannot revoke non-pending invitation", zap.String("status", string(invitation.Status)))
return fmt.Errorf("нельзя отозвать приглашение в статусе %s", invitation.Status)
}

if err := s.repo.Delete(ctx, id); err != nil {
logger.Error("failed to delete invitation", zap.Error(err))
return err
}

logger.Info("invitation revoked successfully")
return nil
}

// ResendInvitation перевыпускает приглашение с новым токеном и сроком действия
func (s *InvitationService) ResendInvitation(ctx context.Context, id string) (models.OrganizationInvitation, error) {
logger := loggerWith(ctx, s.log,
zap.String("invitation_id", id),
zap.String("operation", "ResendInvitation"))

logger.Info("resending invitation")

invitation, err := s.repo.GetByID(ctx, id)
if err != nil {
logger.Error("invitation not found", zap.Error(err))
return models.OrganizationInvitation{}, ErrInvitationNotFound
}

if invitation.Status != models.InvitationStatusPending && invitation.Status != models.InvitationStatusExpired {
logger.Error("cannot resend non-pending/expired invitation", zap.String("status", string(invitation.Status)))
return models.OrganizationInvitation{}, fmt.Errorf("нельзя перевыпустить приглашение в статусе %s", invitation.Status)
}

// Генерируем новый токен
token, err := s.generateToken()
if err != nil {
logger.Error("failed to generate new token", zap.Error(err))
return models.OrganizationInvitation{}, err
}

// Обновляем токен и срок действия
now := time.Now()
if err := s.repo.UpdateTokenAndExpires(ctx, id, token, now.Add(7*24*time.Hour)); err != nil {
logger.Error("failed to update invitation token and expiry", zap.Error(err))
return models.OrganizationInvitation{}, err
}

logger.Info("invitation resent successfully", zap.String("invitation_id", invitation.ID))

// Возвращаем обновленное приглашение без токена
invitation.Token = ""
invitation.ExpiresAt = now.Add(7 * 24 * time.Hour)
invitation.Status = models.InvitationStatusPending
return invitation, nil
}

// GetOrganizationRoles возвращает доступные роли для организации
func (s *InvitationService) GetOrganizationRoles() []map[string]string {
return []map[string]string{
{"role": "org_admin", "description": "Администратор организации - полный доступ"},
{"role": "manager", "description": "Менеджер - управление заявками"},
{"role": "engineer", "description": "Инженер - выполнение исследований"},
{"role": "technician", "description": "Техник - проведение тестов"},
{"role": "client", "description": "Клиент - просмотр своих заявок"},
}
}
