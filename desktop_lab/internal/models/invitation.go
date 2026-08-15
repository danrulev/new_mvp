package models

import "time"

// InvitationStatus представляет статус приглашения
type InvitationStatus string

const (
	InvitationStatusPending   InvitationStatus = "pending"
	InvitationStatusAccepted  InvitationStatus = "accepted"
	InvitationStatusDeclined  InvitationStatus = "declined"
	InvitationStatusExpired   InvitationStatus = "expired"
)

// IsValid проверяет, является ли статус допустимым
func (s InvitationStatus) IsValid() bool {
	switch s {
	case InvitationStatusPending, InvitationStatusAccepted,
		InvitationStatusDeclined, InvitationStatusExpired:
		return true
	default:
		return false
	}
}

// OrganizationInvitation представляет приглашение в организацию
type OrganizationInvitation struct {
	ID             string           `json:"id" db:"id"`
	OrganizationID string           `json:"organization_id" db:"organization_id"`
	Email          string           `json:"email" db:"email"`
	Role           string           `json:"role" db:"role"` // org_admin, manager, engineer, technician, client
	InvitedBy      string           `json:"invited_by" db:"invited_by"`
	InviterName    string           `json:"inviter_name,omitempty" db:"inviter_name"`
	OrganizationName string         `json:"organization_name,omitempty" db:"organization_name"`
	Status         InvitationStatus `json:"status" db:"status"`
	Token          string           `json:"token,omitempty" db:"-"` // Не возвращаем токен в JSON
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
	ExpiresAt      time.Time        `json:"expires_at" db:"expires_at"`
	AcceptedAt     *time.Time       `json:"accepted_at,omitempty" db:"accepted_at"`
}

// CreateInvitationRequest представляет запрос на создание приглашения
type CreateInvitationRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
	Email          string `json:"email" validate:"required,email"`
	Role           string `json:"role" validate:"required,oneof=org_admin manager engineer technician client"`
}

// AcceptInvitationRequest представляет запрос на принятие приглашения
type AcceptInvitationRequest struct {
	Token string `json:"token" validate:"required"`
}

// InvitationListFilter представляет фильтры для списка приглашений
type InvitationListFilter struct {
	Paginated
	OrganizationID string           `form:"organization_id"`
	Email          string           `form:"email"`
	Status         InvitationStatus `form:"status"`
	InvitedBy      string           `form:"invited_by"`
}

// InvitationResponse представляет ответ с данными приглашения
type InvitationResponse struct {
	Invitation OrganizationInvitation `json:"invitation"`
}

// InvitationListResponse представляет ответ со списком приглашений
type InvitationListResponse struct {
	Invitations []OrganizationInvitation `json:"invitations"`
	Meta        PaginatedMetadata        `json:"meta"`
}

// IsExpired проверяет, истекло ли приглашение
func (i *OrganizationInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// CanBeAccepted проверяет, можно ли принять приглашение
func (i *OrganizationInvitation) CanBeAccepted() bool {
	return i.Status == InvitationStatusPending && !i.IsExpired()
}
