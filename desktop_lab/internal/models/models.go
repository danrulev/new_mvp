package models

import (
	"encoding/json"
	"time"
)

// ============================================================================
// СПРАВОЧНИКИ (Базовые сущности)
// ============================================================================

// Material соответствует таблице materials
type Material struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Standard соответствует таблице standards (бывший GOST)
type Standard struct {
	ID          string     `json:"id"`
	MaterialID  string     `json:"material_id"`
	Name        string     `json:"name"` // Например "ГОСТ 9128-2013"
	Description string     `json:"description,omitempty"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidTo     *time.Time `json:"valid_to,omitempty"`

	// Поля для удобства (не хранятся в БД напрямую, заполняются сервисом)
	Dimensions []ContextDimension `json:"dimensions,omitempty"`
	Methods    []TestMethod       `json:"methods,omitempty"`
}

// ContextDimension соответствует таблице context_dimensions
// Описывает, от чего зависят нормы (Климатическая зона, Тип слоя)
type ContextDimension struct {
	ID             string   `json:"id"`
	StandardID     string   `json:"standard_id"`
	KeyName        string   `json:"key_name"` // 'climate_zone'
	Label          string   `json:"label"`    // 'Климатическая зона'
	DataType       string   `json:"data_type"`
	PossibleValues []string `json:"possible_values"` // Распарсенный JSON
}

// ============================================================================
// МЕТОДЫ И НОРМАТИВЫ
// ============================================================================

// TestMethod соответствует таблице test_methods (бывший Method)
type TestMethod struct {
	ID          string `json:"id"`
	StandardID  string `json:"standard_id"`
	Code        string `json:"code,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	FormulaExpr string `json:"formula_expr,omitempty"` // Формула для govaluate
	Unit        string `json:"unit"`
	ResultType  string `json:"result_type"` // 'scalar', 'complex'
	IsMandatory bool   `json:"is_mandatory"`

	// Связанные данные (загружаются отдельно)
	Inputs []MethodInput    `json:"inputs,omitempty"`
	Limits []NormativeLimit `json:"limits,omitempty"`
}

// MethodInput соответствует таблице method_inputs
type MethodInput struct {
	ID         string `json:"id"`
	MethodID   string `json:"method_id"`
	ParamKey   string `json:"param_key"` // Переменная в формуле (F, A)
	Label      string `json:"label"`
	Unit       string `json:"unit,omitempty"`
	InputType  string `json:"input_type"` // 'number', 'select'
	IsRequired bool   `json:"is_required"`
}

// NormativeLimit соответствует таблице normative_limits
// Заменяет простые MinValue/MaxValue из старой модели
type NormativeLimit struct {
	ID             string   `json:"id"`
	MethodID       string   `json:"method_id"`
	LimitType      string   `json:"limit_type"` // 'min', 'max', 'range'
	MinValue       *float64 `json:"min_value,omitempty"`
	MaxValue       *float64 `json:"max_value,omitempty"`
	DiscreteValues []string `json:"discrete_values,omitempty"` // Распарсенный JSON
	Note           string   `json:"note,omitempty"`
	Priority       int      `json:"priority"`

	// Условия применения этого лимита (фильтры)
	Conditions []LimitCondition `json:"conditions,omitempty"`
}

// LimitCondition соответствует таблице limit_conditions
type LimitCondition struct {
	ID                string `json:"id"`
	LimitID           string `json:"limit_id"`
	DimensionKey      string `json:"dimension_key"`
	ConditionOperator string `json:"condition_operator"` // '=', 'IN'
	ExpectedValue     string `json:"expected_value"`
}

// ============================================================================
// ЭКСПЕРИМЕНТАЛЬНЫЕ ДАННЫЕ
// ============================================================================

// ExperimentGroup соответствует таблице experiment_groups
type ExperimentGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	MaterialID  string    `json:"material_id"`
	ProjectName string    `json:"project_name"`
	Location    string    `json:"location,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	// Для отображения
	MaterialName string `json:"material_name,omitempty"`
	SampleCount  int    `json:"sample_count,omitempty"`
}

// Sample соответствует таблице samples
type Sample struct {
	ID             string            `json:"id"`
	GroupID        string            `json:"group_id"`
	MaterialID     string            `json:"material_id"`
	SampleNumber   string            `json:"sample_number"`
	CollectionDate *time.Time        `json:"collection_date,omitempty"`
	ContextParams  map[string]string `json:"context_params"` // Распарсенный JSON из БД
	RawContext     string            `json:"-"`              // Сырой JSON для сохранения в БД
	Note           string            `json:"note,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
}

// Protocol соответствует таблице protocols
type Protocol struct {
	ID             string     `json:"id"`
	SampleID       string     `json:"sample_id"`
	ProtocolNumber string     `json:"protocol_number,omitempty"`
	LabName        string     `json:"lab_name,omitempty"`
	OperatorName   string     `json:"operator_name,omitempty"`
	TestDate       *time.Time `json:"test_date,omitempty"`
	Status         string     `json:"status"` // draft, completed, approved
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Вложенные данные для ответов API
	Sample  *Sample      `json:"sample,omitempty"`
	Results []TestResult `json:"results,omitempty"`
}

// TestResult соответствует таблице test_results
type TestResult struct {
	ID              string                 `json:"id"`
	ProtocolID      string                 `json:"protocol_id"`
	MethodID        string                 `json:"method_id"`
	InputData       map[string]interface{} `json:"input_data"` // Распарсенный JSON
	RawInputData    string                 `json:"-"`          // Сырой JSON для БД
	CalculatedValue *float64               `json:"calculated_value,omitempty"`
	AppliedLimitID  *string                `json:"applied_limit_id,omitempty"`
	IsCompliant     *bool                  `json:"is_compliant,omitempty"`
	DeviationMsg    string                 `json:"deviation_msg,omitempty"`
	Note            string                 `json:"note,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`

	// Для отображения в UI/PDF (денормализация)
	MethodName string   `json:"method_name,omitempty"`
	MethodUnit string   `json:"method_unit,omitempty"`
	MinNorm    *float64 `json:"min_norm,omitempty"`
	MaxNorm    *float64 `json:"max_norm,omitempty"`
}

// ============================================================================
// DTO ДЛЯ ЗАПРОСОВ (INPUTS)
// ============================================================================

// CreateStandardRequest - запрос на создание стандарта с методами
type CreateStandardRequest struct {
	MaterialID  string                `json:"material_id"`
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Dimensions  []ContextDimensionDTO `json:"dimensions,omitempty"`
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

// CreateProtocolRequest - запрос на создание протокола (аналог старого)
type CreateProtocolRequest struct {
	GroupID      string            `json:"group_id"`
	Sample       CreateSampleDTO   `json:"sample"`
	LabName      string            `json:"lab_name"`
	OperatorName string            `json:"operator_name"`
	Results      []CreateResultDTO `json:"results"`
}

type CreateSampleDTO struct {
	SampleNumber   string            `json:"sample_number"`
	MaterialID     string            `json:"material_id"`
	CollectionDate *time.Time        `json:"collection_date,omitempty"`
	ContextParams  map[string]string `json:"context_params"` // Важно: контекст пробы
	Note           string            `json:"note,omitempty"`
}

type CreateResultDTO struct {
	MethodID  string            `json:"method_id"`
	RawInputs map[string]string `json:"raw_inputs"` // Строки для парсинга
	Note      string            `json:"note,omitempty"`
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ============================================================================

// ToJSON преобразует мапку контекста в JSON строку для БД
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

// FromJSON заполняет мапку контекста из строки БД
func (s *Sample) FromJSON(raw string) error {
	s.RawContext = raw
	if raw == "" || raw == "{}" {
		s.ContextParams = make(map[string]string)
		return nil
	}
	return json.Unmarshal([]byte(raw), &s.ContextParams)
}

// ToJSON для результатов
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
// DTO ДЛЯ ОТВЕТОВ (Агрегированные данные для UI)
// ============================================================================

// GroupSummary — сводная информация по группе испытаний (для модального окна и PDF)
type GroupSummary struct {
	GroupID       string                `json:"group_id"`
	GroupName     string                `json:"group_name"`
	MaterialID    string                `json:"material_id"`
	TotalSamples  int                   `json:"total_samples"`
	CompliantRate float64               `json:"compliant_rate"` // Процент соответствия
	Results       []MethodResultSummary `json:"results"`
}

// MethodResultSummary — сводка по одному методу в рамках группы
type MethodResultSummary struct {
	MethodID    string        `json:"method_id"`
	MethodName  string        `json:"method_name"`
	Unit        string        `json:"unit"`
	MinValue    *float64      `json:"min_value,omitempty"`
	MaxValue    *float64      `json:"max_value,omitempty"`
	IsCompliant bool          `json:"is_compliant"`
	Trials      []MethodTrial `json:"trials"`
}

// MethodTrial — отдельное испытание (проба) внутри сводки
type MethodTrial struct {
	ProtocolID   string  `json:"protocol_id"`
	SampleNumber string  `json:"sample_number"`
	Value        float64 `json:"value"`
	IsCompliant  bool    `json:"is_compliant"`
	Deviation    *string `json:"deviation,omitempty"`
}

// ProtocolListResponse — ответ списка протоколов с пагинацией
type ProtocolListResponse struct {
	Items []ProtocolListItem `json:"items"`
	Meta  PaginatedMetadata  `json:"meta"`
}

// ProtocolListItem — протокол с дополнительными полями для списка (имя материала)
type ProtocolListItem struct {
	Protocol
	MaterialName string `json:"material_name"`
}

// PaginatedMetadata — мета-данные пагинации
type PaginatedMetadata struct {
	Total      int64 `json:"total"`
	Page       int64 `json:"page"`
	PageSize   int64 `json:"page_size"`
	TotalPages int64 `json:"total_pages"`
}
