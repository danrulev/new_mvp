package models

import "time"

// Protocol соответствует таблице protocols
type Protocol struct {
	ID             string    `json:"id" db:"id"`
	SampleID       string    `json:"sample_id" db:"sample_id"`
	ProtocolNumber string    `json:"protocol_number,omitempty" db:"protocol_number"`
	LabName        string    `json:"lab_name,omitempty" db:"lab_name"`
	OperatorName   string    `json:"operator_name,omitempty" db:"operator_name"`
	TestDate       time.Time `json:"test_date,omitempty" db:"test_date"`
	Status         string    `json:"status" db:"status"`
	Note           string    `json:"note,omitempty" db:"note"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// ProtocolFull — агрегированный ответ (не маппится напрямую в БД)
type ProtocolFull struct {
	Protocol Protocol             `json:"protocol" db:"-"`
	Sample   Sample               `json:"sample" db:"-"`
	Material Material             `json:"material" db:"-"`
	Results  []TestResultResponse `json:"results" db:"-"`
}

// GetProtocolByIDRequest — агрегированный ответ
type GetProtocolByIDRequest struct {
	Protocol Protocol     `json:"Protocol" db:"-"`
	Results  []TestResult `json:"Results" db:"-"`
}

type UpdateProtocolRequest struct {
	GroupID         *string           `json:"group_id" db:"group_id"`
	LabName         *string           `json:"lab_name" db:"lab_name"`
	OperatorName    *string           `json:"operator_name" db:"operator_name"`
	TestDate        *time.Time        `json:"test_date,omitempty" db:"test_date"`
	SampleNumber    *string           `json:"sample_number" db:"sample_number"`
	CollectionPlace *string           `json:"collection_place" db:"collection_place"`
	CollectionDate  *time.Time        `json:"collection_date,omitempty" db:"collection_date"`
	ContextParams   map[string]string `json:"context_params" db:"context_params"`
	Note            *string           `json:"note,omitempty" db:"note"`
}

type CreateProtocolRequest struct {
	GroupID      string            `json:"group_id"`
	Sample       CreateSampleDTO   `json:"sample"`
	LabName      string            `json:"lab_name"`
	OperatorName string            `json:"operator_name"`
	Note         string            `json:"note"`
	Results      []CreateResultDTO `json:"results"`
}

type CreateResultDTO struct {
	MethodID  string            `json:"method_id"`
	RawInputs map[string]string `json:"raw_inputs"`
	Note      string            `json:"note,omitempty"`
}

type FullProtocolResponse struct {
	Protocol Protocol     `json:"protocol" db:"-"`
	Results  []TestResult `json:"results" db:"-"`
	Samples  []Sample     `json:"samples" db:"-"`
}

type ProtocolListResponse struct {
	Items []Protocol        `json:"items"`
	Meta  PaginatedMetadata `json:"meta"`
}

func (p *ProtocolFull) IsEmpty() bool {
	return p.Protocol.ID == ""
}
