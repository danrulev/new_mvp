package models

// LimitCondition соответствует таблице limit_conditions
type LimitCondition struct {
	ID                string `json:"id" db:"id"`
	LimitID           string `json:"limit_id" db:"limit_id"`
	DimensionKey      string `json:"dimension_key" db:"dimension_key"`
	ConditionOperator string `json:"condition_operator" db:"condition_operator"`
	ExpectedValue     string `json:"expected_value" db:"expected_value"`
}

type CreateLimitDTO struct {
	LimitType  string         `json:"limit_type"`
	MinValue   *float64       `json:"min_value,omitempty"`
	MaxValue   *float64       `json:"max_value,omitempty"`
	Conditions []ConditionDTO `json:"conditions,omitempty"`
}

type ConditionDTO struct {
	DimensionKey  string `json:"dimension_key"`
	Operator      string `json:"operator"`
	ExpectedValue string `json:"expected_value"`
}
