package models

import "time"

// Material соответствует таблице materials
type Material struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code,omitempty" db:"code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
