package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ============================================================================
// СПРАВОЧНИКИ (Базовые сущности)
// ============================================================================

// Material соответствует таблице materials
type Material struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code,omitempty" db:"code"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Standard соответствует таблице standards
type Standard struct {
	ID          string     `json:"id" db:"id"`
	MaterialID  string     `json:"material_id" db:"material_id"`
	Name        string     `json:"name" db:"name"`
	Description *string    `json:"description,omitempty" db:"description"`
	ValidFrom   *time.Time `json:"valid_from,omitempty" db:"valid_from"`
	ValidTo     *time.Time `json:"valid_to,omitempty" db:"valid_to"`
}

// ContextDimension соответствует таблице context_dimensions
type ContextDimension struct {
	ID             string   `json:"id" db:"id"`
	KeyName        string   `json:"key_name" db:"key_name"`
	Label          string   `json:"label" db:"label"`
	DataType       string   `json:"data_type" db:"data_type"`
	PossibleValues []string `json:"possible_values" db:"possible_values"`
}

// ProtocolFull — агрегированный ответ (не маппится напрямую в БД)
type ProtocolFull struct {
	Protocol Protocol     `json:"protocol" db:"-"`
	Sample   Sample       `json:"sample" db:"-"`
	Material Material     `json:"material" db:"-"`
	Results  []TestResult `json:"results" db:"-"`
}

// TestMethodFull — агрегированный ответ (не маппится напрямую в БД)
type TestMethodFull struct {
	Method          TestMethod                  `json:"method" db:"-"`
	Inputs          []MethodInput               `json:"inputs" db:"-"`
	Limits          []NormativeLimit            `json:"limits" db:"-"`
	LimitConditions map[string][]LimitCondition `json:"-" db:"-"`
}

// GetProtocolByIDRequest — агрегированный ответ
type GetProtocolByIDRequest struct {
	Protocol Protocol     `json:"Protocol" db:"-"`
	Results  []TestResult `json:"Results" db:"-"`
}

// StandardContext — служебная структура для кэширования
type StandardContext struct {
	StandardID string                    `json:"standard_id" db:"-"`
	Dimensions []ContextDimension        `json:"dimensions" db:"-"`
	Methods    map[string]TestMethodFull `json:"methods" db:"-"`
}

type MaterialContext struct {
	MaterialID string
	Dimensions []ContextDimension `json:"dimensions" db:"-"`
}

func (p *ProtocolFull) IsEmpty() bool {
	return p.Protocol.ID == ""
}

// ============================================================================
// МЕТОДЫ И НОРМАТИВЫ
// ============================================================================

// TestMethod соответствует таблице test_methods
type TestMethod struct {
	ID          string  `json:"id" db:"id"`
	StandardID  string  `json:"standard_id" db:"standard_id"`
	Code        string  `json:"code,omitempty" db:"code"`
	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`
	FormulaExpr string  `json:"formula_expr,omitempty" db:"formula_expr"`
	Unit        string  `json:"unit" db:"unit"`
	ResultType  string  `json:"result_type" db:"result_type"`
	IsMandatory bool    `json:"is_mandatory" db:"is_mandatory"`
}

// MethodInput соответствует таблице method_inputs
type MethodInput struct {
	ID         string `json:"id" db:"id"`
	MethodID   string `json:"method_id" db:"method_id"`
	ParamKey   string `json:"param_key" db:"param_key"`
	Label      string `json:"label" db:"label"`
	Unit       string `json:"unit,omitempty" db:"unit"`
	InputType  string `json:"input_type" db:"input_type"`
	IsRequired bool   `json:"is_required" db:"is_required"`
}

// NormativeLimit соответствует таблице normative_limits
type NormativeLimit struct {
	ID             string   `json:"id" db:"id"`
	MethodID       string   `json:"method_id" db:"method_id"`
	LimitType      string   `json:"limit_type" db:"limit_type"`
	MinValue       *float64 `json:"min_value,omitempty" db:"min_value"`
	MaxValue       *float64 `json:"max_value,omitempty" db:"max_value"`
	DiscreteValues []string `json:"discrete_values,omitempty" db:"discrete_values"`
	Note           *string  `json:"note,omitempty" db:"note"`
	Priority       int      `json:"priority" db:"priority"`
}

// LimitCondition соответствует таблице limit_conditions
type LimitCondition struct {
	ID                string `json:"id" db:"id"`
	LimitID           string `json:"limit_id" db:"limit_id"`
	DimensionKey      string `json:"dimension_key" db:"dimension_key"`
	ConditionOperator string `json:"condition_operator" db:"condition_operator"`
	ExpectedValue     string `json:"expected_value" db:"expected_value"`
}

// ============================================================================
// ЭКСПЕРИМЕНТАЛЬНЫЕ ДАННЫЕ
// ============================================================================

// ExperimentGroup соответствует таблице experiment_groups
type ExperimentGroup struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	MaterialID  string    `json:"material_id" db:"material_id"`
	ProjectName string    `json:"project_name" db:"project_name"`
	Location    string    `json:"location,omitempty" db:"location"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// Sample соответствует таблице samples
type Sample struct {
	ID              string            `json:"id" db:"id"`
	GroupID         string            `json:"group_id" db:"group_id"`
	MaterialID      string            `json:"material_id" db:"material_id"`
	SampleNumber    string            `json:"sample_number" db:"sample_number"`
	CollectionPlace string            `json:"collection_place" db:"collection_place"`
	CollectionDate  *time.Time        `json:"collection_date,omitempty" db:"collection_date"`
	ContextParams   map[string]string `json:"context_params" db:"context_params"`
	RawContext      string            `json:"-" db:"-"`
	Note            string            `json:"note,omitempty" db:"note"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
}

type UpdateSampleRequest struct {
	GroupID         string            `json:"group_id" db:"group_id"`
	SampleNumber    string            `json:"sample_number" db:"sample_number"`
	CollectionPlace string            `json:"collection_place" db:"collection_place"`
	CollectionDate  *time.Time        `json:"collection_date,omitempty" db:"collection_date"`
	ContextParams   map[string]string `json:"context_params" db:"context_params"`
	RawContext      string            `json:"-" db:"-"`
	Note            string            `json:"note,omitempty" db:"note"`
}

// Protocol соответствует таблице protocols
type Protocol struct {
	ID             string     `json:"id" db:"id"`
	SampleID       string     `json:"sample_id" db:"sample_id"`
	ProtocolNumber string     `json:"protocol_number,omitempty" db:"protocol_number"`
	LabName        string     `json:"lab_name,omitempty" db:"lab_name"`
	OperatorName   string     `json:"operator_name,omitempty" db:"operator_name"`
	TestDate       *time.Time `json:"test_date,omitempty" db:"test_date"`
	Status         string     `json:"status" db:"status"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
}

// TestResult соответствует таблице test_results
type TestResult struct {
	ID              string                 `json:"id" db:"id"`
	ProtocolID      string                 `json:"protocol_id" db:"protocol_id"`
	MethodID        string                 `json:"method_id" db:"method_id"`
	InputData       map[string]interface{} `json:"input_data" db:"input_data"`
	RawInputData    string                 `json:"-" db:"-"`
	CalculatedValue *float64               `json:"calculated_value,omitempty" db:"calculated_value"`
	AppliedLimitID  *string                `json:"applied_limit_id,omitempty" db:"applied_limit_id"`
	IsCompliant     *bool                  `json:"is_compliant,omitempty" db:"is_compliant"`
	DeviationMsg    string                 `json:"deviation_msg,omitempty" db:"deviation_msg"`
	Note            string                 `json:"note,omitempty" db:"note"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// ============================================================================
// DTO ДЛЯ ЗАПРОСОВ (не требуют db-тегов — только JSON)
// ============================================================================

type CreateStandardRequest struct {
	MaterialID  string                `json:"material_id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Dimensions  []ContextDimensionDTO `json:"dimension_ids,omitempty"`
	Methods     []CreateMethodDTO     `json:"methods,omitempty"`
}

type ContextDimensionDTO struct {
	KeyName        string   `json:"key_name"`
	Label          string   `json:"label"`
	DataType       string   `json:"data_type"`
	PossibleValues []string `json:"possible_values"`
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

type MethodInputDTO struct {
	ParamKey   string `json:"param_key"`
	Label      string `json:"label"`
	Unit       string `json:"unit,omitempty"`
	InputType  string `json:"input_type"`
	IsRequired bool   `json:"is_required"`
}

type CreateLimitDTO struct {
	LimitType  string         `json:"limit_type"`
	MinValue   *float64       `json:"min_value,omitempty"`
	MaxValue   *float64       `json:"max_value,omitempty"`
	Conditions []ConditionDTO `json:"conditions,omitempty"`
}

type ConditionDTO struct {
	DimensionKey  string `json:"dimension_key"`
	Operator      string `json:"operator"`
	ExpectedValue string `json:"expected_value"`
}

type CreateProtocolRequest struct {
	GroupID      string            `json:"group_id"`
	Sample       CreateSampleDTO   `json:"sample"`
	LabName      string            `json:"lab_name"`
	OperatorName string            `json:"operator_name"`
	Results      []CreateResultDTO `json:"results"`
}

type CreateSampleDTO struct {
	SampleNumber    string            `json:"sample_number"`
	MaterialID      string            `json:"material_id"`
	CollectionDate  *time.Time        `json:"collection_date,omitempty"`
	CollectionPlace string            `json:"collection_place,omitempty"`
	ContextParams   map[string]string `json:"context_params"`
	Note            string            `json:"note,omitempty"`
}

type CreateResultDTO struct {
	MethodID  string            `json:"method_id"`
	RawInputs map[string]string `json:"raw_inputs"`
	Note      string            `json:"note,omitempty"`
}

type FullProtocolGroupResponse struct {
	Protocols []Protocol `json:"protocols" db:"-"`
	Samples   []Sample   `json:"samples" db:"-"`
}

type FullProtocolResponse struct {
	Protocol Protocol     `json:"protocol" db:"-"`
	Results  []TestResult `json:"results" db:"-"`
	Samples  []Sample     `json:"samples" db:"-"`
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ============================================================================

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

func (r *TestResult) InputsToJSON() (string, error) {
	if r.InputData == nil {
		return "{}", nil
	}
	b, err := json.Marshal(r.InputData)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ============================================================================
// DTO ДЛЯ ОТВЕТОВ (агрегированные данные — не требуют db-тегов)
// ============================================================================

type GroupSummary struct {
	GroupID       string                `json:"group_id"`
	GroupName     string                `json:"group_name"`
	MaterialID    string                `json:"material_id"`
	TotalSamples  int                   `json:"total_samples"`
	CompliantRate float64               `json:"compliant_rate"`
	Results       []MethodResultSummary `json:"results"`
}

type MethodResultSummary struct {
	MethodID    string        `json:"method_id"`
	MethodName  string        `json:"method_name"`
	Unit        string        `json:"unit"`
	IsCompliant bool          `json:"is_compliant"`
	Trials      []MethodTrial `json:"trials"`
}

type MethodTrial struct {
	ProtocolID   string  `json:"protocol_id"`
	SampleNumber string  `json:"sample_number"`
	Value        float64 `json:"value"`
	IsCompliant  bool    `json:"is_compliant"`
	Deviation    *string `json:"deviation,omitempty"`
}

type ProtocolListResponse struct {
	Items []Protocol        `json:"items"`
	Meta  PaginatedMetadata `json:"meta"`
}

type PaginatedMetadata struct {
	Page        int64 `json:"page"`
	Total       int64 `json:"total"`
	TotalPages  int64 `json:"total_pages"`
	HasNextPage bool  `json:"has_next_page"`
	HasPrevPage bool  `json:"has_prev_page"`
}

type GroupListResponse struct {
	Items []ExperimentGroup `json:"items"`
	Meta  PaginatedMetadata `json:"meta"`
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
