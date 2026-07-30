package models

import "time"

// Standard соответствует таблице standards
type Standard struct {
	ID          string     `json:"id" db:"id"`
	MaterialID  string     `json:"material_id" db:"material_id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	ValidFrom   *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidTo     *time.Time `json:"valid_to,omitempty" db:"valid_to"`
}

type CreateStandardRequest struct {
	MaterialID  string                `json:"material_id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Dimensions  []ContextDimensionDTO `json:"dimension_ids,omitempty"`
	Methods     []CreateMethodDTO     `json:"methods,omitempty"`
}

type CreateMethodDTO struct {
	Code        string           `json:"code"`
	Name        string           `json:"name"`
	IsMandatory bool             `json:"is_mandatory"`
	FormulaExpr string           `json:"formula_expr,omitempty"`
	Unit        string           `json:"unit"`
	Inputs      []MethodInputDTO `json:"inputs,omitempty"`
	Limits      []CreateLimitDTO `json:"limits,omitempty"`
}
