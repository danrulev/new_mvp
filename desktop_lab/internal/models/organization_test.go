package models

type OrganizationTest struct {
	ID             string `json:"id" db:"id"`
	OrganizationID string `json:"organization_id" db:"organization_id"`
	TestMethodID   string `json:"test_method_id" db:"test_method_id"`
	Price          string `json:"price" db:"price"`
	CreatedAt      string `json:"created_at" db:"created_at"`
	UpdatedAt      string `json:"updated_at" db:"updated_at"`
}

type CreateOrganizationTestRequest struct {
	OrganizationID string `json:"organization_id" validate:"required"`
	TestMethodID   string `json:"test_method_id" validate:"required"`
	Price          string `json:"price" validate:"required"`
}
