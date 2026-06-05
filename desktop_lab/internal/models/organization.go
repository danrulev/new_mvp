package models

import (
	"time"
)

type Organization struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Address   string    `json:"address" db:"address"`
	Phone     string    `json:"phone" db:"phone"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	DeletedAt time.Time `json:"deleted_at" db:"deleted_at"`
}

type CreateOrganizationRequest struct {
	Name    string `json:"name" validate:"required"`
	Address string `json:"address" validate:"required"`
	Phone   string `json:"phone" validate:"required"`
	Email   string `json:"email" validate:"required"`
}

type UpdateOrganizationRequest struct {
	Name    *string `json:"name" db:"name"`
	Address *string `json:"address" db:"address"`
	Phone   *string `json:"phone" db:"phone"`
	Email   *string `json:"email" db:"email"`
}

type OrganizationListResponse struct {
	Organizations []Organization    `json:"organizations"`
	Meta          PaginatedMetadata `json:"meta"`
}
