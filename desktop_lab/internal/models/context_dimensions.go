package models

// ContextDimension соответствует таблице context_dimensions
type ContextDimension struct {
	ID             string   `json:"id" db:"id"`
	KeyName        string   `json:"key_name" db:"key_name"`
	Label          string   `json:"label" db:"label"`
	DataType       string   `json:"data_type" db:"data_type"`
	PossibleValues []string `json:"possible_values" db:"possible_values"`
}

type ContextDimensionDTO struct {
	KeyName        string   `json:"key_name"`
	Label          string   `json:"label"`
	DataType       string   `json:"data_type"`
	PossibleValues []string `json:"possible_values"`
}
