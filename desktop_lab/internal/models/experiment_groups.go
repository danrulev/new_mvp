package models

import "time"

// ExperimentGroup соответствует таблице experiment_groups
type ExperimentGroup struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	MaterialID  string    `json:"material_id" db:"material_id"`
	ProjectName string    `json:"project_name" db:"project_name"`
	Location    string    `json:"location,omitempty" db:"location"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type UpdateExperimentGroup struct {
	Name        *string `json:"name" db:"name"`
	ProjectName *string `json:"project_name" db:"project_name"`
	Location    *string `json:"location,omitempty" db:"location"`
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

type GroupListResponse struct {
	Items []ExperimentGroup `json:"items"`
	Meta  PaginatedMetadata `json:"meta"`
}
