package models

// StandardContext — служебная структура для кэширования
type StandardContext struct {
	StandardID string                    `json:"standard_id" db:"-"`
	Dimensions []ContextDimension        `json:"dimensions" db:"-"`
	Methods    map[string]TestMethodFull `json:"methods" db:"-"`
}
