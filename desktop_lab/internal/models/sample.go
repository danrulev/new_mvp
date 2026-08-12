package models

import (
	"encoding/json"
	"time"
)

// Sample соответствует таблице samples
type Sample struct {
	ID              string            `json:"id" db:"id"`
	GroupID         string            `json:"group_id" db:"group_id"`
	MaterialID      string            `json:"material_id" db:"material_id"`
	SampleNumber    string            `json:"sample_number" db:"sample_number"`
	CollectionPlace string            `json:"collection_place" db:"collection_place"`
	CollectionDate  time.Time         `json:"collection_date,omitempty" db:"collection_date"`
	ContextParams   map[string]string `json:"context_params" db:"context_params"`
	RawContext      string            `json:"-" db:"-"`
	Note            string            `json:"note,omitempty" db:"note"`
	
	// Расширенные поля для образца
	PhotoURL      string   `json:"photo_url,omitempty" db:"photo_url"`       // URL или путь к фотографии
	LengthMM      *float64 `json:"length_mm,omitempty" db:"length_mm"`       // Длина в мм
	WidthMM       *float64 `json:"width_mm,omitempty" db:"width_mm"`         // Ширина в мм
	HeightMM      *float64 `json:"height_mm,omitempty" db:"height_mm"`       // Высота в мм
	Shape         string   `json:"shape,omitempty" db:"shape"`               // Форма (куб, цилиндр, призма и т.д.)
	WeightGrams   *float64 `json:"weight_grams,omitempty" db:"weight_grams"` // Вес в граммах
	Color         string   `json:"color,omitempty" db:"color"`               // Цвет
	BatchNumber   string   `json:"batch_number,omitempty" db:"batch_number"` // Номер партии
	Manufacturer  string   `json:"manufacturer,omitempty" db:"manufacturer"` // Производитель
	
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at,omitempty" db:"updated_at"`
}

type CreateSampleDTO struct {
	SampleNumber    string            `json:"sample_number"`
	MaterialID      string            `json:"material_id"`
	CollectionDate  *time.Time        `json:"collection_date,omitempty"`
	CollectionPlace string            `json:"collection_place,omitempty"`
	ContextParams   map[string]string `json:"context_params"`
	Note            string            `json:"note,omitempty"`
	
	// Расширенные поля для образца
	PhotoURL     string            `json:"photo_url,omitempty"`
	LengthMM     *float64          `json:"length_mm,omitempty"`
	WidthMM      *float64          `json:"width_mm,omitempty"`
	HeightMM     *float64          `json:"height_mm,omitempty"`
	Shape        string            `json:"shape,omitempty"`
	WeightGrams  *float64          `json:"weight_grams,omitempty"`
	Color        string            `json:"color,omitempty"`
	BatchNumber  string            `json:"batch_number,omitempty"`
	Manufacturer string            `json:"manufacturer,omitempty"`
}

func (s *Sample) ToJSON() (string, error) {
	if s.ContextParams == nil {
		return "{}", nil
	}
	b, err := json.Marshal(s.ContextParams)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (s *Sample) FromJSON(raw string) error {
	s.RawContext = raw
	if raw == "" || raw == "{}" {
		s.ContextParams = make(map[string]string)
		return nil
	}
	return json.Unmarshal([]byte(raw), &s.ContextParams)
}
