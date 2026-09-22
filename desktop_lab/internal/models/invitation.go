package models

import "time"

// InvitationStatus представляет статус приглашения/заявки на регистрацию
type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"  // На рассмотрении
	InvitationStatusAccepted InvitationStatus = "accepted" // Подтверждена
	InvitationStatusDeclined InvitationStatus = "declined" // Отказано
)

// IsValid проверяет, является ли статус допустимым
func (s InvitationStatus) IsValid() bool {
	switch s {
	case InvitationStatusPending, InvitationStatusAccepted, InvitationStatusDeclined:
		return true
	default:
		return false
	}
}

// RegistrationInvitation представляет заявку на регистрацию в системе
type RegistrationInvitation struct {
	ID        string           `json:"id" db:"id"`
	Email     string           `json:"email" db:"email"`
	Name      string           `json:"name" db:"name"`
	Role      string           `json:"role" db:"role"` // admin, manager, engineer, technician, client
	Status    InvitationStatus `json:"status" db:"status"`
	Message   string           `json:"message,omitempty" db:"message"` // Сообщение от пользователя
	ReviewedBy *string         `json:"reviewed_by,omitempty" db:"reviewed_by"` // Кто рассмотрел (ID админа)
	ReviewedAt *time.Time      `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedAt time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" db:"updated_at"`
}

// CreateInvitationRequest представляет запрос на создание заявки на регистрацию
type CreateInvitationRequest struct {
	Email   string `json:"email" validate:"required,email"`
	Name    string `json:"name" validate:"required"`
	Role    string `json:"role" validate:"required,oneof=admin manager engineer technician client"`
	Message string `json:"message"`
}

// ReviewInvitationRequest представляет запрос на рассмотрение заявки
type ReviewInvitationRequest struct {
	Approved bool   `json:"approved" validate:"required"`
	Message  string `json:"message"` // Комментарий админа (особенно важен при отказе)
}

// InvitationListFilter представляет фильтры для списка заявок
type InvitationListFilter struct {
	Paginated
	Email  string           `form:"email"`
	Status InvitationStatus `form:"status"`
}

// InvitationResponse представляет ответ с данными заявки
type InvitationResponse struct {
	Invitation RegistrationInvitation `json:"invitation"`
}

// InvitationListResponse представляет ответ со списком заявок
type InvitationListResponse struct {
	Invitations []RegistrationInvitation `json:"invitations"`
	Meta        PaginatedMetadata        `json:"meta"`
}
