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
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
}

type CreateSampleDTO struct {
	SampleNumber    string            `json:"sample_number"`
	MaterialID      string            `json:"material_id"`
	CollectionDate  *time.Time        `json:"collection_date,omitempty"`
	CollectionPlace string            `json:"collection_place,omitempty"`
	ContextParams   map[string]string `json:"context_params"`
	Note            string            `json:"note,omitempty"`
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
