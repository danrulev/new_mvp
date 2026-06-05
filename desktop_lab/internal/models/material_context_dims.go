package models

type MaterialContext struct {
	MaterialID string
	Dimensions []ContextDimension `json:"dimensions" db:"-"`
}
