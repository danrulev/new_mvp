package models

type OrganizationUser struct {
	ID             string `json:"id" db:"id"`
	OrganizationID string `json:"organization_id" db:"organization_id"`
	UserID         string `json:"user_id" db:"user_id"`
	Role           string `json:"role" db:"role"`
	CreatedAt      string `json:"created_at" db:"created_at"`
	UpdatedAt      string `json:"updated_at" db:"updated_at"`
}

type CreateOrganizationUserRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
	UserID         string `json:"user_id" validate:"required"`
	Role           string `json:"role" validate:"required"`
}
