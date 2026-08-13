package models

import "time"

// ExperimentGroup соответствует таблице experiment_groups
type ExperimentGroup struct {
	ID                  string    `json:"id" db:"id"`
	Name                string    `json:"name" db:"name"`
	MaterialID          string    `json:"material_id" db:"material_id"`
	ProjectName         string    `json:"project_name" db:"project_name"`
	ObjectType          string    `json:"object_type,omitempty" db:"object_type"`
	Customer            string    `json:"customer,omitempty" db:"customer"`
	ContractNumber      string    `json:"contract_number,omitempty" db:"contract_number"`
	Status              string    `json:"status" db:"status"`
	ResponsiblePersonID string    `json:"responsible_person_id,omitempty" db:"responsible_person_id"`
	Location            string    `json:"location,omitempty" db:"location"`
	Description         string    `json:"description,omitempty" db:"description"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

type UpdateExperimentGroup struct {
	Name                *string `json:"name" db:"name"`
	ProjectName         *string `json:"project_name" db:"project_name"`
	ObjectType          *string `json:"object_type,omitempty" db:"object_type"`
	Customer            *string `json:"customer,omitempty" db:"customer"`
	ContractNumber      *string `json:"contract_number,omitempty" db:"contract_number"`
	Status              *string `json:"status" db:"status"`
	ResponsiblePersonID *string `json:"responsible_person_id,omitempty" db:"responsible_person_id"`
	Location            *string `json:"location,omitempty" db:"location"`
	Description         *string `json:"description,omitempty" db:"description"`
}

type FullProtocolGroupResponse struct {
	Protocols []Protocol `json:"protocols" db:"-"`
	Samples   []Sample   `json:"samples" db:"-"`
}

type GroupSummary struct {
	GroupID       string                `json:"group_id"`
	GroupName     string                `json:"group_name"`
	MaterialID    string                `json:"material_id"`
	TotalSamples  int                   `json:"total_samples"`
	CompliantRate float64               `json:"compliant_rate"`
	Results       []MethodResultSummary `json:"results"`
}

type GroupListFilter struct {
	Name        *string `form:"name"`
	Material    *string `form:"material"`
	ProjectName *string `form:"project_name"`
	ObjectType  *string `form:"object_type"`
	Customer    *string `form:"customer"`
	Status      *string `form:"status"`
	Location    *string `form:"location"`
	Paginated
}

type GroupListResponse struct {
	Items []ExperimentGroup `json:"items"`
	Meta  PaginatedMetadata `json:"meta"`
}
