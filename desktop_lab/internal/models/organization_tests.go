package models

import "time"

type OrganizationTest struct {
	ID             string    `json:"id" db:"id"`
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	TestMethodID   string    `json:"test_method_id" db:"test_method_id"`
	Price          float64   `json:"price" db:"price"`
	Description    string    `json:"description" db:"description"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type CreateOrganizationTestRequest struct {
	OrganizationID string  `json:"organization_id" validate:"required"`
	TestMethodID   string  `json:"test_method_id" validate:"required"`
	Price          float64 `json:"price" validate:"required"`
	Description    string  `json:"description"`
}

type UpdateOrganizationTestRequest struct {
	Price       *float64 `json:"price" validate:"required"`
	Description *string  `json:"description"`
}

type ListOrganizationTests struct {
	OrganizationTests []OrganizationTest `json:"organization_tests"`
	Meta              PaginatedMetadata  `json:"meta"`
}
