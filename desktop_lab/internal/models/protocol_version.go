package models

import (
	"time"
)

// ProtocolVersion версия протокола испытаний
type ProtocolVersion struct {
	ID              int64     `json:"id" db:"id"`
	ProtocolID      int64     `json:"protocol_id" db:"protocol_id"`
	VersionNumber   int       `json:"version_number" db:"version_number"` // 1, 2, 3...
	ContentSnapshot []byte    `json:"content_snapshot" db:"content_snapshot"` // JSONB полное состояние протокола
	PDFSnapshot     []byte    `json:"pdf_snapshot,omitempty" db:"pdf_snapshot"` // Бинарный PDF срез
	ChangedBy       int64     `json:"changed_by" db:"changed_by"`
	ChangedByName   string    `json:"changed_by_name,omitempty" db:"changed_by_name"`
	ChangedAt       time.Time `json:"changed_at" db:"changed_at"`
	Comment         string    `json:"comment,omitempty" db:"comment"` // Причина изменений
	IsCurrent       bool      `json:"is_current" db:"is_current"`     // Флаг текущей версии
}

// ProtocolVersionCreate запрос на создание версии
type ProtocolVersionCreate struct {
	ProtocolID    int64  `json:"protocol_id" validate:"required"`
	Comment       string `json:"comment"`
	ContentJSON   []byte `json:"content_json" validate:"required"` // Текущее состояние протокола
	PDFFile       []byte `json:"pdf_file,omitempty"`               // Опционально PDF
	ChangedBy     int64  `json:"changed_by" validate:"required"`
	ChangedByName string `json:"changed_by_name"`
}

// ProtocolVersionResponse ответ с версией
type ProtocolVersionResponse struct {
	ID              int64      `json:"id"`
	ProtocolID      int64      `json:"protocol_id"`
	VersionNumber   int        `json:"version_number"`
	ContentSnapshot interface{} `json:"content_snapshot"` // Распарсенный JSON
	PDFAvailable    bool       `json:"pdf_available"`
	ChangedBy       int64      `json:"changed_by"`
	ChangedByName   string     `json:"changed_by_name"`
	ChangedAt       time.Time  `json:"changed_at"`
	Comment         string     `json:"comment"`
	IsCurrent       bool       `json:"is_current"`
}

// ProtocolTemplate шаблон протокола
type ProtocolTemplate struct {
	ID             int64     `json:"id" db:"id"`
	OrganizationID int64     `json:"organization_id" db:"organization_id"`
	TestMethodID   int64     `json:"test_method_id" db:"test_method_id"` // Привязка к методу
	Name           string    `json:"name" db:"name"`                      // Название шаблона
	Description    string    `json:"description,omitempty" db:"description"`
	TemplateHTML   string    `json:"template_html" db:"template_html"`    // HTML шаблон с переменными {{.Variable}}
	TemplateCSS    string    `json:"template_css,omitempty" db:"template_css"`
	IsActive       bool      `json:"is_active" db:"is_active"`
	CreatedBy      int64     `json:"created_by" db:"created_by"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// ProtocolTemplateCreate запрос на создание шаблона
type ProtocolTemplateCreate struct {
	OrganizationID int64  `json:"organization_id" validate:"required"`
	TestMethodID   int64  `json:"test_method_id" validate:"required"`
	Name           string `json:"name" validate:"required,min=3,max=200"`
	Description    string `json:"description"`
	TemplateHTML   string `json:"template_html" validate:"required"`
	TemplateCSS    string `json:"template_css"`
	IsActive       bool   `json:"is_active"`
	CreatedBy      int64  `json:"created_by" validate:"required"`
}

// ProtocolTemplateUpdate запрос на обновление шаблона
type ProtocolTemplateUpdate struct {
	Name         string `json:"name" validate:"omitempty,min=3,max=200"`
	Description  string `json:"description"`
	TemplateHTML string `json:"template_html"`
	TemplateCSS  string `json:"template_css"`
	IsActive     *bool  `json:"is_active"`
}

// ProtocolTemplateResponse ответ с шаблоном
type ProtocolTemplateResponse struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	TestMethodID   int64     `json:"test_method_id"`
	MethodName     string    `json:"method_name,omitempty"` // Из JOIN
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	IsActive       bool      `json:"is_active"`
	CreatedBy      int64     `json:"created_by"`
	CreatedByName  string    `json:"created_by_name,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TemplateVariable переменная шаблона
type TemplateVariable struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	DataType    string      `json:"data_type"` // string, number, date, array
	Required    bool        `json:"required"`
	DefaultValue interface{} `json:"default_value,omitempty"`
}
