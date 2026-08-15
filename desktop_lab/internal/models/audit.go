package models

import (
	"time"
)

// ActionType определяет тип действия в аудите
type ActionType string

const (
	ActionCreate   ActionType = "create"
	ActionUpdate   ActionType = "update"
	ActionDelete   ActionType = "delete"
	ActionView     ActionType = "view"
	ActionPrint    ActionType = "print"
	ActionExport   ActionType = "export"
	ActionStatusChange ActionType = "status_change"
)

// AuditLog запись о действии пользователя
type AuditLog struct {
	ID           int64      `json:"id" db:"id"`
	UserID       int64      `json:"user_id" db:"user_id"`
	UserName     string     `json:"user_name,omitempty" db:"user_name"` // Денормализация для истории
	Action       ActionType `json:"action" db:"action"`
	ResourceType string     `json:"resource_type" db:"resource_type"` // e.g., "protocol", "order", "sample"
	ResourceID   int64      `json:"resource_id" db:"resource_id"`
	OldValues    []byte     `json:"old_values,omitempty" db:"old_values"` // JSONB
	NewValues    []byte     `json:"new_values,omitempty" db:"new_values"` // JSONB
	IPAddress    string     `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    string     `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// AuditLogFilter фильтры для поиска по логам
type AuditLogFilter struct {
	UserID       *int64     `json:"user_id,omitempty"`
	ResourceType *string    `json:"resource_type,omitempty"`
	ResourceID   *int64     `json:"resource_id,omitempty"`
	Action       *ActionType `json:"action,omitempty"`
	DateFrom     *time.Time `json:"date_from,omitempty"`
	DateTo       *time.Time `json:"date_to,omitempty"`
	Limit        int        `json:"limit"`
	Offset       int        `json:"offset"`
}
