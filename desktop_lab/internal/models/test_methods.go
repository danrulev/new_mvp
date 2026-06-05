package models

// TestMethodFull — агрегированный ответ (не маппится напрямую в БД)
type TestMethodFull struct {
	Method          TestMethod                  `json:"method" db:"-"`
	Inputs          []MethodInput               `json:"inputs" db:"-"`
	Limits          []NormativeLimit            `json:"limits" db:"-"`
	LimitConditions map[string][]LimitCondition `json:"-" db:"-"`
}

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
