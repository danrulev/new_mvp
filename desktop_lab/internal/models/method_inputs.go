package models

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

type MethodInputDTO struct {
	ParamKey   string `json:"param_key"`
	Label      string `json:"label"`
	Unit       string `json:"unit,omitempty"`
	InputType  string `json:"input_type"`
	IsRequired bool   `json:"is_required"`
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
