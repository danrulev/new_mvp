package models

import (
	"encoding/json"
	"time"
)

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

type TestResultResponse struct {
	TestResult        // встраиваем все поля оригинала
	MethodName string `json:"method_name" db:"method_name"`           // ← новое поле
	MethodUnit string `json:"method_unit,omitempty" db:"method_unit"` // опционально
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
