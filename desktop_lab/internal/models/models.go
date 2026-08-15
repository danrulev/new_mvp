package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type PaginatedMetadata struct {
	Page        int64 `json:"page"`
	Total       int64 `json:"total"`
	TotalPages  int64 `json:"total_pages"`
	HasNextPage bool  `json:"has_next_page"`
	HasPrevPage bool  `json:"has_prev_page"`
	Limit       int64 `json:"limit,omitempty"`
	Offset      int64 `json:"offset,omitempty"`
}

// JSONStringSlice - кастомный тип для хранения []string в колонке TEXT (JSON)
type JSONStringSlice []string

// Scan реализует интерфейс sql.Scanner для чтения из БД
func (s *JSONStringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = []string{}
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("failed to scan JSONStringSlice: unsupported type %T", value)
	}

	// Если строка пустая, возвращаем пустой слайс
	if len(bytes) == 0 {
		*s = []string{}
		return nil
	}

	// Парсим JSON
	return json.Unmarshal(bytes, s)
}

// Value реализует интерфейс driver.Valuer для записи в БД
func (s JSONStringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil // Или return "[]", nil если хотите хранить пустой массив явно
	}
	return json.Marshal(s)
}

func MakePaginatedMetadata(limit, offset, total int64) PaginatedMetadata {
	if limit <= 0 {
		limit = 1
	}

	totalPages := int64(0)
	if total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return PaginatedMetadata{
		Page:        offset/limit + 1,
		Total:       total,
		TotalPages:  totalPages,
		HasNextPage: total > offset+limit,
		HasPrevPage: offset > 0,
	}
}

type Paginated struct {
	Limit  int64 `form:"limit" validate:"gte=10,lte=100"`
	Offset int64 `form:"offset" validate:"gte=0"`
}
