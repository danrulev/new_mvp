package models

// NormativeLimit соответствует таблице normative_limits
type NormativeLimit struct {
	ID             string   `json:"id" db:"id"`
	MethodID       string   `json:"method_id" db:"method_id"`
	LimitType      string   `json:"limit_type" db:"limit_type"`
	MinValue       *float64 `json:"min_value,omitempty" db:"min_value"`
	MaxValue       *float64 `json:"max_value,omitempty" db:"max_value"`
	DiscreteValues []string `json:"discrete_values,omitempty" db:"discrete_values"`
	Note           *string  `json:"note,omitempty" db:"note"`
	Priority       int      `json:"priority" db:"priority"`
}
