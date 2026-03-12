package repository

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type standardRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewStandardRepo(db *sqlx.DB, log *zap.Logger) StandardRepo {
	return &standardRepo{db: db, log: log}
}

// CreateFull - ОПТИМИЗИРОВАННАЯ ВЕРСИЯ
// Использует подготовленные statements для ускорения массовых вставок
func (r *standardRepo) CreateFull(ctx context.Context, req models.CreateStandardRequest) (string, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stdID := uuid.New().String()

	// 1. Создаем Стандарт
	_, err = tx.ExecContext(ctx,
		`INSERT INTO standards (id, material_id, name, description) VALUES (?, ?, ?, ?)`,
		stdID, req.MaterialID, req.Name, req.Description,
	)
	if err != nil {
		return "", fmt.Errorf("failed to insert standard: %w", err)
	}

	// 2. Создаем Dimensions (пакетная вставка)
	if len(req.Dimensions) > 0 {
		// Подготавливаем statement один раз
		stmt, err := tx.PreparexContext(ctx,
			`INSERT INTO context_dimensions (id, standard_id, key_name, label, data_type, possible_values) 
			 VALUES (?, ?, ?, ?, ?, ?)`,
		)
		if err != nil {
			return "", fmt.Errorf("failed to prepare dimension statement: %w", err)
		}

		for _, dim := range req.Dimensions {
			dimID := uuid.New().String()
			valuesJSON := "null"
			if len(dim.PossibleValues) > 0 {
				b, _ := json.Marshal(dim.PossibleValues)
				valuesJSON = string(b)
			}

			_, err = stmt.ExecContext(ctx, dimID, stdID, dim.KeyName, dim.Label, dim.DataType, valuesJSON)
			if err != nil {
				_ = stmt.Close()
				return "", fmt.Errorf("failed to insert dimension %s: %w", dim.KeyName, err)
			}
		}
		_ = stmt.Close()
	}

	// 3. Создаем Методы и вложенные сущности
	// Подготавливаем statements заранее
	stmtMethod, err := tx.PreparexContext(ctx,
		`INSERT INTO test_methods (id, standard_id, code, name, formula_expr, unit, result_type, is_mandatory)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return "", fmt.Errorf("failed to prepare method statement: %w", err)
	}
	defer stmtMethod.Close()

	stmtInput, err := tx.PreparexContext(ctx,
		`INSERT INTO method_inputs (id, method_id, param_key, label, unit, input_type, is_required)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return "", fmt.Errorf("failed to prepare input statement: %w", err)
	}
	defer stmtInput.Close()

	stmtLimit, err := tx.PreparexContext(ctx,
		`INSERT INTO normative_limits (id, method_id, limit_type, min_value, max_value, discrete_values, priority)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return "", fmt.Errorf("failed to prepare limit statement: %w", err)
	}
	defer stmtLimit.Close()

	stmtCondition, err := tx.PreparexContext(ctx,
		`INSERT INTO limit_conditions (id, limit_id, dimension_key, condition_operator, expected_value)
		 VALUES (?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return "", fmt.Errorf("failed to prepare condition statement: %w", err)
	}
	defer stmtCondition.Close()

	for _, mReq := range req.Methods {
		methodID := uuid.New().String()
		isMandatory := 1

		_, err = stmtMethod.ExecContext(ctx,
			methodID, stdID, mReq.Code, mReq.Name, mReq.FormulaExpr, mReq.Unit, "scalar", isMandatory,
		)
		if err != nil {
			return "", fmt.Errorf("failed to insert method %s: %w", mReq.Name, err)
		}

		// Inputs
		for _, inp := range mReq.Inputs {
			inpID := uuid.New().String()
			isReq := 0
			if inp.IsRequired {
				isReq = 1
			}
			_, err = stmtInput.ExecContext(ctx,
				inpID, methodID, inp.ParamKey, inp.Label, inp.Unit, inp.InputType, isReq,
			)
			if err != nil {
				return "", fmt.Errorf("failed to insert input %s: %w", inp.ParamKey, err)
			}
		}

		// Limits + Conditions
		for _, lim := range mReq.Limits {
			limitID := uuid.New().String()
			discreteJSON := "null"

			_, err = stmtLimit.ExecContext(ctx,
				limitID, methodID, lim.LimitType, lim.MinValue, lim.MaxValue, discreteJSON, 0,
			)
			if err != nil {
				return "", fmt.Errorf("failed to insert limit: %w", err)
			}

			for _, cond := range lim.Conditions {
				condID := uuid.New().String()
				_, err = stmtCondition.ExecContext(ctx,
					condID, limitID, cond.DimensionKey, cond.Operator, cond.ExpectedValue,
				)
				if err != nil {
					return "", fmt.Errorf("failed to insert condition: %w", err)
				}
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("Standard created successfully", zap.String("id", stdID))
	return stdID, nil
}

// GetApplicableLimit - ОПТИМИЗИРОВАННАЯ ВЕРСИЯ
// Один JOIN-запрос вместо N+1, фильтрация в памяти
func (r *standardRepo) GetApplicableLimit(ctx context.Context, methodID string, contextParams map[string]string) (models.NormativeLimit, error) {
	// Один запрос с получением лимитов и их условий
	// Используем LEFT JOIN, чтобы получить даже лимиты без условий
	query := `
		SELECT 
			nl.id, nl.method_id, nl.limit_type, nl.min_value, nl.max_value, nl.priority,
			lc.dimension_key, lc.condition_operator, lc.expected_value
		FROM normative_limits nl
		LEFT JOIN limit_conditions lc ON nl.id = lc.limit_id
		WHERE nl.method_id = ?
		ORDER BY nl.priority DESC, nl.id
	`

	rows, err := r.db.QueryContext(ctx, query, methodID)
	if err != nil {
		return models.NormativeLimit{}, err
	}
	defer rows.Close()

	// Группируем лимиты и их условия в памяти
	limitsMap := make(map[string]*struct {
		Limit      models.NormativeLimit
		Conditions []models.LimitCondition
	})

	for rows.Next() {
		var (
			limitID, methodID, limitType string
			minValue, maxValue           sql.NullFloat64
			priority                     int
			dimKey, condOp, expectedVal  sql.NullString
		)

		err := rows.Scan(&limitID, &methodID, &limitType, &minValue, &maxValue, &priority,
			&dimKey, &condOp, &expectedVal)
		if err != nil {
			return models.NormativeLimit{}, err
		}

		// Инициализируем лимит, если видим впервые
		_, exists := limitsMap[limitID]
		if !exists {
			limit := models.NormativeLimit{
				ID:        limitID,
				MethodID:  methodID,
				LimitType: limitType,
				Priority:  priority,
			}
			if minValue.Valid {
				limit.MinValue = &minValue.Float64
			}
			if maxValue.Valid {
				limit.MaxValue = &maxValue.Float64
			}
			limitsMap[limitID] = &struct {
				Limit      models.NormativeLimit
				Conditions []models.LimitCondition
			}{Limit: limit, Conditions: []models.LimitCondition{}}
		}

		// Добавляем условие, если есть
		if dimKey.Valid {
			cond := models.LimitCondition{
				LimitID:           limitID,
				DimensionKey:      dimKey.String,
				ConditionOperator: "=",
				ExpectedValue:     expectedVal.String,
			}
			if condOp.Valid {
				cond.ConditionOperator = condOp.String
			}
			limitsMap[limitID].Conditions = append(limitsMap[limitID].Conditions, cond)
		}
	}

	if err := rows.Err(); err != nil {
		return models.NormativeLimit{}, err
	}

	// Преобразуем мапу в слайс и сортируем по приоритету (если нужно)
	// Но так как в запросе уже есть ORDER BY, просто ищем первое совпадение
	var defaultLimit *models.NormativeLimit

	for _, entry := range limitsMap {
		limit := entry.Limit
		conds := entry.Conditions

		// Лимит без условий — кандидат на дефолтный
		if len(conds) == 0 {
			if defaultLimit == nil {
				defaultLimit = &limit
			}
			continue
		}

		// Проверяем все условия
		match := true
		for _, cond := range conds {
			actualVal, exists := contextParams[cond.DimensionKey]
			if !exists {
				match = false
				break
			}
			switch cond.ConditionOperator {
			case "=":
				if actualVal != cond.ExpectedValue {
					match = false
				}
			case "!=":
				if actualVal == cond.ExpectedValue {
					match = false
				}
			case "IN":
				if !containsCSV(cond.ExpectedValue, actualVal) {
					match = false
				}
			}
			if !match {
				break
			}
		}

		if match {
			return limit, nil
		}
	}

	// Возвращаем дефолтный лимит (без условий), если специфичный не найден
	if defaultLimit != nil {
		return *defaultLimit, nil
	}

	return models.NormativeLimit{}, nil
}

// containsCSV проверяет наличие значения в строке "val1,val2,val3"
func containsCSV(csv, target string) bool {
	for _, v := range strings.Split(csv, ",") {
		if strings.TrimSpace(v) == target {
			return true
		}
	}
	return false
}

// GetTestMethod - без изменений, запрос простой и эффективный
func (r *standardRepo) GetTestMethod(ctx context.Context, methodID string) (models.TestMethod, error) {
	method := models.TestMethod{}
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, standard_id, code, name, formula_expr, unit, result_type, is_mandatory 
		 FROM test_methods WHERE id = ?`,
		methodID,
	).StructScan(&method)

	if err == sql.ErrNoRows {
		return models.TestMethod{}, nil
	}
	if err != nil {
		return models.TestMethod{}, err
	}

	return method, nil
}

// GetByMaterialID - без изменений
func (r *standardRepo) GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error) {
	var standards []models.Standard
	err := r.db.SelectContext(ctx, &standards,
		`SELECT id, material_id, name, description, valid_from, valid_to 
		 FROM standards WHERE material_id = ? ORDER BY name`,
		materialID,
	)
	if err != nil {
		return nil, err
	}
	// Устанавливаем MaterialID для каждого (если не загружается автоматически)
	for i := range standards {
		standards[i].MaterialID = materialID
	}
	return standards, nil
}

// GetMethodsByStandardID - используем sqlx.Select для чистоты кода
func (r *standardRepo) GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error) {
	var methods []models.TestMethod
	err := r.db.SelectContext(ctx, &methods,
		`SELECT id, standard_id, code, name, description, formula_expr, unit, result_type, is_mandatory 
		 FROM test_methods 
		 WHERE standard_id = ? 
		 ORDER BY name`,
		standardID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query methods: %w", err)
	}
	return methods, nil
}

// GetMethodInputs - используем sqlx.Select
func (r *standardRepo) GetMethodInputs(ctx context.Context, methodID string) ([]models.MethodInput, error) {
	var inputs []models.MethodInput
	err := r.db.SelectContext(ctx, &inputs,
		`SELECT id, method_id, param_key, label, unit, input_type, is_required 
		 FROM method_inputs 
		 WHERE method_id = ? 
		 ORDER BY param_key`,
		methodID,
	)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

type NormativeLimit struct {
	ID             string                 `db:"id" json:"id"`
	MethodID       string                 `db:"method_id" json:"method_id"`
	LimitType      string                 `db:"limit_type" json:"limit_type"`
	MinValue       *float64               `db:"min_value" json:"min_value,omitempty"`
	MaxValue       *float64               `db:"max_value" json:"max_value,omitempty"`
	DiscreteValues models.JSONStringSlice `db:"discrete_values" json:"discrete_values,omitempty"`
	Note           *string                `db:"note" json:"note,omitempty"`
	Priority       int                    `db:"priority" json:"priority"`
}

func (r *standardRepo) GetMethodLimits(ctx context.Context, methodID string) ([]models.NormativeLimit, error) {
	query := `SELECT id, method_id, limit_type, min_value, max_value, discrete_values, note, priority 
			  FROM normative_limits 
			  WHERE method_id = $1 
			  ORDER BY priority`

	var limits []NormativeLimit
	// Используем SelectContext для выполнения запроса
	err := r.db.SelectContext(ctx, &limits, query, methodID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch limits: %w", err)
	}

	// Возвращаем пустой слайс вместо nil, если ничего не найдено
	if limits == nil {
		return []models.NormativeLimit{}, nil
	}
	out := make([]models.NormativeLimit, len(limits))
	for i, limit := range limits {
		out[i] = limit.toModel()
	}

	return out, nil
}

func (nl NormativeLimit) toModel() models.NormativeLimit {
	return models.NormativeLimit{
		ID:             nl.ID,
		MethodID:       nl.MethodID,
		LimitType:      nl.LimitType,
		MinValue:       nl.MinValue,
		MaxValue:       nl.MaxValue,
		DiscreteValues: []string(nl.DiscreteValues), // Явное приведение типа
		Note:           nl.Note,
		Priority:       nl.Priority,
	}
}

func (r *standardRepo) GetLimitConditions(ctx context.Context, limitID string) ([]models.LimitCondition, error) {
	query := `SELECT id, limit_id, dimension_key, condition_operator, expected_value 
			  FROM limit_conditions WHERE limit_id = $1`

	var conditions []models.LimitCondition
	err := r.db.SelectContext(ctx, &conditions, query, limitID)
	if err != nil {
		return nil, err
	}

	if conditions == nil {
		conditions = []models.LimitCondition{}
	}

	return conditions, nil
}

// GetStandardDimensions - 🔥 ИСПРАВЛЕНО: context_dimensions вместо standard_dimensions
func (r *standardRepo) GetStandardDimensions(ctx context.Context, standardID string) ([]models.ContextDimension, error) {
	query := `SELECT id, standard_id, key_name, label, data_type, possible_values 
			  FROM context_dimensions 
			  WHERE standard_id = ? 
			  ORDER BY key_name`

	rows, err := r.db.QueryContext(ctx, query, standardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dimensions []models.ContextDimension

	for rows.Next() {
		var dim models.ContextDimension
		var possibleValuesRaw sql.NullString

		err := rows.Scan(
			&dim.ID, &dim.StandardID, &dim.KeyName, &dim.Label, &dim.DataType, &possibleValuesRaw,
		)
		if err != nil {
			return nil, err
		}

		// Парсим JSON только если поле не NULL и не пустое
		if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
			if err := json.Unmarshal([]byte(possibleValuesRaw.String), &dim.PossibleValues); err != nil {
				r.log.Warn("failed to parse possible_values",
					zap.String("dimension", dim.KeyName), zap.Error(err))
				// Не прерываем, возвращаем пустой слайс
				dim.PossibleValues = []string{}
			}
		} else {
			dim.PossibleValues = []string{}
		}

		dimensions = append(dimensions, dim)
	}

	return dimensions, rows.Err()
}

// GetMethodsFullByStandardID - ОПТИМИЗИРОВАННАЯ ВЕРСИЯ с sqlx
// Загружает методы, инпуты, лимиты и условия одним запросом
func (r *standardRepo) GetMethodsFullByStandardID(ctx context.Context, standardID string) (map[string]models.TestMethodFull, error) {
	query := `
		SELECT 
			tm.id, tm.code, tm.name, tm.description, tm.formula_expr, tm.unit, tm.result_type, tm.is_mandatory,
			mi.id as input_id, mi.param_key, mi.label as input_label, mi.unit as input_unit, mi.input_type, mi.is_required as input_is_req,
			nl.id as limit_id, nl.limit_type, nl.min_value, nl.max_value, nl.priority,
			lc.dimension_key, lc.condition_operator, lc.expected_value
		FROM test_methods tm
		LEFT JOIN method_inputs mi ON tm.id = mi.method_id
		LEFT JOIN normative_limits nl ON tm.id = nl.method_id
		LEFT JOIN limit_conditions lc ON nl.id = lc.limit_id
		WHERE tm.standard_id = ?
		ORDER BY tm.id, mi.param_key, nl.priority
	`

	rows, err := r.db.QueryxContext(ctx, query, standardID)
	if err != nil {
		return nil, fmt.Errorf("failed to query full methods: %w", err)
	}
	defer rows.Close()

	resultMap := make(map[string]models.TestMethodFull)

	for rows.Next() {
		var row struct {
			// Method
			MethodID    string         `db:"id"`
			Code        string         `db:"code"`
			Name        string         `db:"name"`
			Description sql.NullString `db:"description"`
			FormulaExpr sql.NullString `db:"formula_expr"`
			Unit        string         `db:"unit"`
			ResultType  string         `db:"result_type"`
			IsMandatory int            `db:"is_mandatory"`

			// Input (nullable)
			InputID    sql.NullString `db:"input_id"`
			ParamKey   sql.NullString `db:"param_key"`
			InputLabel sql.NullString `db:"input_label"`
			InputUnit  sql.NullString `db:"input_unit"`
			InputType  sql.NullString `db:"input_type"`
			InputIsReq sql.NullInt64  `db:"input_is_req"`

			// Limit (nullable)
			LimitID   sql.NullString  `db:"limit_id"`
			LimitType sql.NullString  `db:"limit_type"`
			MinValue  sql.NullFloat64 `db:"min_value"`
			MaxValue  sql.NullFloat64 `db:"max_value"`
			Priority  sql.NullInt64   `db:"priority"`

			// Condition (nullable)
			DimKey      sql.NullString `db:"dimension_key"`
			CondOp      sql.NullString `db:"condition_operator"`
			ExpectedVal sql.NullString `db:"expected_value"`
		}

		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Инициализация метода
		fullMethod, exists := resultMap[row.MethodID]
		if !exists {
			fullMethod = models.TestMethodFull{
				Method: models.TestMethod{
					ID:          row.MethodID,
					StandardID:  standardID,
					Code:        row.Code,
					Name:        row.Name,
					Unit:        row.Unit,
					ResultType:  row.ResultType,
					IsMandatory: row.IsMandatory == 1,
				},
				Inputs:          []models.MethodInput{},
				Limits:          []models.NormativeLimit{},
				LimitConditions: make(map[string][]models.LimitCondition),
			}
			if row.Description.Valid {
				fullMethod.Method.Description = &row.Description.String
			}
			if row.FormulaExpr.Valid {
				fullMethod.Method.FormulaExpr = &row.FormulaExpr.String
			}
		}

		// Добавляем Input
		if row.InputID.Valid {
			input := models.MethodInput{
				ID:        row.InputID.String,
				MethodID:  row.MethodID,
				ParamKey:  row.ParamKey.String,
				Label:     row.InputLabel.String,
				InputType: row.InputType.String,
			}
			if row.InputUnit.Valid {
				input.Unit = row.InputUnit.String
			}
			if row.InputIsReq.Valid {
				input.IsRequired = row.InputIsReq.Int64 == 1
			}
			fullMethod.Inputs = append(fullMethod.Inputs, input)
		}

		// Добавляем Limit
		if row.LimitID.Valid {
			// Проверяем, не добавлен ли уже этот лимит
			var currentLimit *models.NormativeLimit
			for i := range fullMethod.Limits {
				if fullMethod.Limits[i].ID == row.LimitID.String {
					currentLimit = &fullMethod.Limits[i]
					break
				}
			}

			if currentLimit == nil {
				newLimit := models.NormativeLimit{
					ID:       row.LimitID.String,
					MethodID: row.MethodID,
				}
				if row.LimitType.Valid {
					newLimit.LimitType = row.LimitType.String
				}
				if row.MinValue.Valid {
					newLimit.MinValue = &row.MinValue.Float64
				}
				if row.MaxValue.Valid {
					newLimit.MaxValue = &row.MaxValue.Float64
				}
				if row.Priority.Valid {
					newLimit.Priority = int(row.Priority.Int64)
				}
				fullMethod.Limits = append(fullMethod.Limits, newLimit)
				currentLimit = &fullMethod.Limits[len(fullMethod.Limits)-1]
				fullMethod.LimitConditions[currentLimit.ID] = []models.LimitCondition{}
			}

			// Добавляем Condition
			if row.DimKey.Valid {
				cond := models.LimitCondition{
					LimitID:       currentLimit.ID,
					DimensionKey:  row.DimKey.String,
					ExpectedValue: row.ExpectedVal.String,
				}
				if row.CondOp.Valid {
					cond.ConditionOperator = row.CondOp.String
				}
				fullMethod.LimitConditions[currentLimit.ID] = append(
					fullMethod.LimitConditions[currentLimit.ID], cond)
			}
		}

		resultMap[row.MethodID] = fullMethod
	}

	return resultMap, rows.Err()
}

// GetStandardFull - НОВЫЙ МЕТОД для эффективной загрузки всего стандарта для UI
// Возвращает стандарт с измерениями и методами (без глубокой вложенности лимитов для списка)
func (r *standardRepo) GetStandardFull(ctx context.Context, standardID string) (*models.StandardContext, error) {
	ctxData := &models.StandardContext{
		StandardID: standardID,
		Dimensions: []models.ContextDimension{},
		Methods:    make(map[string]models.TestMethodFull),
	}

	// 1. Загружаем измерения (параллельно можно, но для простоты последовательно)
	dims, err := r.GetStandardDimensions(ctx, standardID)
	if err != nil {
		return nil, err
	}
	ctxData.Dimensions = dims

	// 2. Загружаем методы с инпутами (лимиты не грузим для списка — только при валидации)
	// Отдельный запрос без условий лимитов для скорости
	query := `
		SELECT 
			tm.id, tm.code, tm.name, tm.description, tm.formula_expr, tm.unit, tm.result_type, tm.is_mandatory,
			mi.id as input_id, mi.param_key, mi.label as input_label, mi.unit as input_unit, mi.input_type, mi.is_required as input_is_req
		FROM test_methods tm
		LEFT JOIN method_inputs mi ON tm.id = mi.method_id
		WHERE tm.standard_id = ?
		ORDER BY tm.name, mi.param_key
	`

	rows, err := r.db.QueryxContext(ctx, query, standardID)
	if err != nil {
		return nil, fmt.Errorf("failed to query methods for standard: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row struct {
			// Method
			MethodID    string         `db:"id"`
			Code        string         `db:"code"`
			Name        string         `db:"name"`
			Description sql.NullString `db:"description"`
			FormulaExpr sql.NullString `db:"formula_expr"`
			Unit        string         `db:"unit"`
			ResultType  string         `db:"result_type"`
			IsMandatory int            `db:"is_mandatory"`
			// Input
			InputID    sql.NullString `db:"input_id"`
			ParamKey   sql.NullString `db:"param_key"`
			InputLabel sql.NullString `db:"input_label"`
			InputUnit  sql.NullString `db:"input_unit"`
			InputType  sql.NullString `db:"input_type"`
			InputIsReq sql.NullInt64  `db:"input_is_req"`
		}

		if err := rows.StructScan(&row); err != nil {
			return nil, err
		}

		fullMethod, exists := ctxData.Methods[row.MethodID]
		if !exists {
			fullMethod = models.TestMethodFull{
				Method: models.TestMethod{
					ID: row.MethodID, StandardID: standardID, Code: row.Code, Name: row.Name,
					Unit: row.Unit, ResultType: row.ResultType, IsMandatory: row.IsMandatory == 1,
				},
				Inputs: []models.MethodInput{},
			}
			if row.Description.Valid {
				fullMethod.Method.Description = &row.Description.String
			}
			if row.FormulaExpr.Valid {
				fullMethod.Method.FormulaExpr = &row.FormulaExpr.String
			}
		}

		if row.InputID.Valid {
			input := models.MethodInput{
				ID: row.InputID.String, MethodID: row.MethodID, ParamKey: row.ParamKey.String,
				Label: row.InputLabel.String, InputType: row.InputType.String,
			}
			if row.InputUnit.Valid {
				input.Unit = row.InputUnit.String
			}
			if row.InputIsReq.Valid {
				input.IsRequired = row.InputIsReq.Int64 == 1
			}
			fullMethod.Inputs = append(fullMethod.Inputs, input)
		}

		ctxData.Methods[row.MethodID] = fullMethod
	}

	return ctxData, rows.Err()
}
