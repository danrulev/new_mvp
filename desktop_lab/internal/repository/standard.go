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

type StandardRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewStandardRepo(db *sqlx.DB, log *zap.Logger) *StandardRepo {
	return &StandardRepo{db: db, log: log}
}

// CreateFull - ОБНОВЛЕННАЯ ВЕРСИЯ для новой схемы БД
func (r *StandardRepo) CreateFull(ctx context.Context, req models.CreateStandardRequest) (string, error) {
	log := logQuery(ctx, r.log, "INSERT (TX)", "standards + test_methods + method_inputs + normative_limits + limit_conditions",
		zap.String("material_id", req.MaterialID),
		zap.String("standard_name", req.Name),
		zap.Int("dimensions_count", len(req.Dimensions)),
		zap.Int("methods_count", len(req.Methods)),
	)
	log.Info("starting standard creation transaction")

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("transaction rollback failed", zap.Error(rbErr))
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	stdID := uuid.New().String()

	_, err = tx.ExecContext(ctx,
		`INSERT INTO standards (id, material_id, name, description) VALUES (?, ?, ?, ?)`,
		stdID, req.MaterialID, req.Name, req.Description,
	)
	if err != nil {
		log.Error("failed to insert standard", zap.Error(err))
		return "", fmt.Errorf("failed to insert standard: %w", err)
	}

	if len(req.Dimensions) > 0 {
		stmtLink, err := tx.PreparexContext(ctx,
			`INSERT INTO standard_context_dims (id, standard_id, dimension_id) 
			 VALUES (?, ?, ?)`,
		)
		if err != nil {
			log.Error("failed to prepare dimension link statement", zap.Error(err))
			return "", fmt.Errorf("failed to prepare dimension link statement: %w", err)
		}

		for i, dimDTO := range req.Dimensions {
			dimLog := log.With(zap.Int("dimension_index", i), zap.String("dimension_key", dimDTO.KeyName))

			var dimID string
			err := tx.GetContext(ctx, &dimID, `SELECT id FROM context_dimensions WHERE key_name = ?`, dimDTO.KeyName)

			if err == sql.ErrNoRows {
				dimLog.Warn("dimension not found in global registry", zap.String("key_name", dimDTO.KeyName))
				return "", fmt.Errorf("dimension with key_name '%s' not found in global registry", dimDTO.KeyName)
			} else if err != nil {
				dimLog.Error("failed to find dimension", zap.Error(err))
				_ = stmtLink.Close()
				return "", fmt.Errorf("failed to find dimension %s: %w", dimDTO.KeyName, err)
			}

			linkID := uuid.New().String()
			_, err = stmtLink.ExecContext(ctx, linkID, stdID, dimID)
			if err != nil {
				_ = stmtLink.Close()
				return "", fmt.Errorf("failed to link dimension %s: %w", dimDTO.KeyName, err)
			}
		}
		_ = stmtLink.Close()
	}

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
		isMandatory := 0
		if mReq.IsMandatory {
			isMandatory = 1
		}

		_, err = stmtMethod.ExecContext(ctx,
			methodID, stdID, mReq.Code, mReq.Name, mReq.FormulaExpr, mReq.Unit, "scalar", isMandatory,
		)
		if err != nil {
			return "", fmt.Errorf("failed to insert method %s: %w", mReq.Name, err)
		}

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
		log.Error("transaction commit failed", zap.Error(err))
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("Standard created successfully", zap.String("id", stdID), zap.Int("methods_inserted", len(req.Methods)))
	return stdID, nil
}

func (r *StandardRepo) GetApplicableLimit(ctx context.Context, methodID string, contextParams map[string]string) (models.NormativeLimit, error) {
	log := logQuery(ctx, r.log, "SELECT (JOIN)", "normative_limits + limit_conditions",
		zap.String("method_id", methodID),
		zap.Int("context_params_count", len(contextParams)),
	)
	log.Debug("fetching applicable limit for method")

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
		log.Error("rows iteration error", zap.Error(err))
		return models.NormativeLimit{}, err
	}

	var defaultLimit *models.NormativeLimit

	for _, entry := range limitsMap {
		limit := entry.Limit
		conds := entry.Conditions

		if len(conds) == 0 {
			if defaultLimit == nil {
				defaultLimit = &limit
			}
			continue
		}

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
			log.Debug("matching limit found",
				zap.String("limit_id", limit.ID),
				zap.String("limit_type", limit.LimitType))
			return limit, nil
		}
	}

	if defaultLimit != nil {
		log.Debug("using default limit (no conditions)",
			zap.String("limit_id", defaultLimit.ID))
		return *defaultLimit, nil
	}

	log.Debug("no applicable limit found")
	return models.NormativeLimit{}, nil
}

func containsCSV(csv, target string) bool {
	for _, v := range strings.Split(csv, ",") {
		if strings.TrimSpace(v) == target {
			return true
		}
	}
	return false
}

func (r *StandardRepo) GetTestMethod(ctx context.Context, methodID string) (models.TestMethod, error) {
	log := logQuery(ctx, r.log, "SELECT", "test_methods",
		zap.String("method_id", methodID))
	log.Debug("fetching test method by ID")
	method := models.TestMethod{}
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, standard_id, code, name, formula_expr, unit, result_type, is_mandatory 
		 FROM test_methods WHERE id = ?`,
		methodID,
	).StructScan(&method)

	if err == sql.ErrNoRows {
		log.Debug("method not found")
		return models.TestMethod{}, nil
	}
	if err != nil {
		return models.TestMethod{}, err
	}
	log.Debug("method retrieved successfully",
		zap.String("method_name", method.Name))
	return method, nil
}

func (r *StandardRepo) GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error) {
	log := logQuery(ctx, r.log, "SELECT", "standards",
		zap.String("material_id", materialID))
	log.Debug("fetching standards for material")

	var standards []models.Standard
	err := r.db.SelectContext(ctx, &standards,
		`SELECT id, material_id, name, description, valid_from, valid_to 
		 FROM standards WHERE material_id = ? ORDER BY name`,
		materialID,
	)
	if err != nil {
		return nil, err
	}
	for i := range standards {
		standards[i].MaterialID = materialID
	}

	log.Debug("standards retrieved successfully",
		zap.Int("count", len(standards)))
	return standards, nil
}

func (r *StandardRepo) GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error) {
	log := logQuery(ctx, r.log, "SELECT", "test_methods",
		zap.String("standard_id", standardID))
	log.Debug("fetching methods for standard")

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
	log.Debug("methods retrieved successfully",
		zap.Int("count", len(methods)))
	return methods, nil
}

func (r *StandardRepo) GetMethodInputs(ctx context.Context, methodID string) ([]models.MethodInput, error) {
	log := logQuery(ctx, r.log, "SELECT", "method_inputs",
		zap.String("method_id", methodID))
	log.Debug("fetching inputs for method")

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
	log.Debug("inputs retrieved successfully",
		zap.Int("count", len(inputs)))
	return inputs, nil
}

type NormativeLimitDB struct {
	ID             string                 `db:"id"`
	MethodID       string                 `db:"method_id"`
	LimitType      string                 `db:"limit_type"`
	MinValue       *float64               `db:"min_value"`
	MaxValue       *float64               `db:"max_value"`
	DiscreteValues models.JSONStringSlice `db:"discrete_values"`
	Note           *string                `db:"note"`
	Priority       int                    `db:"priority"`
}

func (r *StandardRepo) GetMethodLimits(ctx context.Context, methodID string) ([]models.NormativeLimit, error) {
	log := logQuery(ctx, r.log, "SELECT", "normative_limits",
		zap.String("method_id", methodID))
	log.Debug("fetching limits for method")

	query := `SELECT id, method_id, limit_type, min_value, max_value, discrete_values, note, priority 
			  FROM normative_limits 
			  WHERE method_id = ? 
			  ORDER BY priority`

	var limits []NormativeLimitDB
	err := r.db.SelectContext(ctx, &limits, query, methodID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch limits: %w", err)
	}

	if limits == nil {
		log.Debug("no limits found for method")
		return []models.NormativeLimit{}, nil
	}

	out := make([]models.NormativeLimit, len(limits))
	for i, limit := range limits {
		out[i] = limit.toModel()
	}

	log.Debug("limits retrieved successfully",
		zap.Int("count", len(out)))
	return out, nil
}

func (nl NormativeLimitDB) toModel() models.NormativeLimit {
	return models.NormativeLimit{
		ID:             nl.ID,
		MethodID:       nl.MethodID,
		LimitType:      nl.LimitType,
		MinValue:       nl.MinValue,
		MaxValue:       nl.MaxValue,
		DiscreteValues: []string(nl.DiscreteValues),
		Note:           nl.Note,
		Priority:       nl.Priority,
	}
}

func (r *StandardRepo) GetLimitConditions(ctx context.Context, limitID string) ([]models.LimitCondition, error) {
	log := logQuery(ctx, r.log, "SELECT", "limit_conditions",
		zap.String("limit_id", limitID))
	log.Debug("fetching conditions for limit")
	query := `SELECT id, limit_id, dimension_key, condition_operator, expected_value 
			  FROM limit_conditions WHERE limit_id = ?`

	var conditions []models.LimitCondition
	err := r.db.SelectContext(ctx, &conditions, query, limitID)
	if err != nil {
		return nil, err
	}

	if conditions == nil {
		conditions = []models.LimitCondition{}
	}
	log.Debug("conditions retrieved successfully",
		zap.Int("count", len(conditions)))
	return conditions, nil
}

func (r *StandardRepo) GetStandardDimensions(ctx context.Context, standardID string) ([]models.ContextDimension, error) {
	log := logQuery(ctx, r.log, "SELECT (JOIN)", "context_dimensions + standard_context_dims",
		zap.String("standard_id", standardID))
	log.Debug("fetching dimensions for standard")

	query := `
		SELECT cd.id, cd.key_name, cd.label, cd.data_type, cd.possible_values, cd.description
		FROM context_dimensions cd
		JOIN standard_context_dims scd ON cd.id = scd.dimension_id
		WHERE scd.standard_id = ? 
		ORDER BY cd.label
	`

	rows, err := r.db.QueryContext(ctx, query, standardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dimensions []models.ContextDimension

	for rows.Next() {
		var dim models.ContextDimension
		var possibleValuesRaw, description sql.NullString

		err := rows.Scan(
			&dim.ID, &dim.KeyName, &dim.Label, &dim.DataType, &possibleValuesRaw, &description,
		)
		if err != nil {
			return nil, err
		}

		if possibleValuesRaw.Valid && possibleValuesRaw.String != "" && possibleValuesRaw.String != "null" {
			if err := json.Unmarshal([]byte(possibleValuesRaw.String), &dim.PossibleValues); err != nil {
				r.log.Warn("failed to parse possible_values",
					zap.String("dimension", dim.KeyName), zap.Error(err))
				dim.PossibleValues = []string{}
			}
		} else {
			dim.PossibleValues = []string{}
		}

		dimensions = append(dimensions, dim)
	}

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error", zap.Error(err))
		return nil, err
	}

	log.Debug("dimensions retrieved successfully",
		zap.Int("count", len(dimensions)))
	return dimensions, rows.Err()
}

func (r *StandardRepo) GetMethodsFullByStandardID(ctx context.Context, standardID string) (map[string]models.TestMethodFull, error) {
	log := logQuery(ctx, r.log, "SELECT (FULL JOIN)", "test_methods + method_inputs + normative_limits + limit_conditions",
		zap.String("standard_id", standardID))
	log.Debug("fetching full methods with cache-friendly query")
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
			MethodID    string         `db:"id"`
			Code        string         `db:"code"`
			Name        string         `db:"name"`
			Description sql.NullString `db:"description"`
			FormulaExpr sql.NullString `db:"formula_expr"`
			Unit        string         `db:"unit"`
			ResultType  string         `db:"result_type"`
			IsMandatory int            `db:"is_mandatory"`

			InputID    sql.NullString `db:"input_id"`
			ParamKey   sql.NullString `db:"param_key"`
			InputLabel sql.NullString `db:"input_label"`
			InputUnit  sql.NullString `db:"input_unit"`
			InputType  sql.NullString `db:"input_type"`
			InputIsReq sql.NullInt64  `db:"input_is_req"`

			LimitID   sql.NullString  `db:"limit_id"`
			LimitType sql.NullString  `db:"limit_type"`
			MinValue  sql.NullFloat64 `db:"min_value"`
			MaxValue  sql.NullFloat64 `db:"max_value"`
			Priority  sql.NullInt64   `db:"priority"`

			DimKey      sql.NullString `db:"dimension_key"`
			CondOp      sql.NullString `db:"condition_operator"`
			ExpectedVal sql.NullString `db:"expected_value"`
		}

		if err := rows.StructScan(&row); err != nil {
			log.Error("failed to scan row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

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
				fullMethod.Method.FormulaExpr = row.FormulaExpr.String
			}
		}

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

		if row.LimitID.Valid {
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

	log.Info("full methods retrieved successfully",
		zap.Int("methods_count", len(resultMap)))
	return resultMap, rows.Err()
}

func (r *StandardRepo) GetStandardFull(ctx context.Context, standardID string) (models.StandardContext, error) {
	log := logQuery(ctx, r.log, "SELECT (MULTI)", "standard_context_dims + test_methods + method_inputs",
		zap.String("standard_id", standardID))
	log.Debug("fetching full standard context")
	ctxData := models.StandardContext{
		StandardID: standardID,
		Dimensions: []models.ContextDimension{},
		Methods:    make(map[string]models.TestMethodFull),
	}

	dims, err := r.GetStandardDimensions(ctx, standardID)
	if err != nil {
		return models.StandardContext{}, err
	}
	ctxData.Dimensions = dims

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
		return models.StandardContext{}, fmt.Errorf("failed to query methods for standard: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row struct {
			MethodID    string         `db:"id"`
			Code        string         `db:"code"`
			Name        string         `db:"name"`
			Description sql.NullString `db:"description"`
			FormulaExpr sql.NullString `db:"formula_expr"`
			Unit        string         `db:"unit"`
			ResultType  string         `db:"result_type"`
			IsMandatory int            `db:"is_mandatory"`
			InputID     sql.NullString `db:"input_id"`
			ParamKey    sql.NullString `db:"param_key"`
			InputLabel  sql.NullString `db:"input_label"`
			InputUnit   sql.NullString `db:"input_unit"`
			InputType   sql.NullString `db:"input_type"`
			InputIsReq  sql.NullInt64  `db:"input_is_req"`
		}

		if err := rows.StructScan(&row); err != nil {
			return models.StandardContext{}, err
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
				fullMethod.Method.FormulaExpr = row.FormulaExpr.String
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

	if err := rows.Err(); err != nil {
		log.Error("rows iteration error for methods", zap.Error(err))
		return models.StandardContext{}, err
	}
	log.Info("full standard context loaded successfully",
		zap.Int("dimensions_count", len(dims)))

	return ctxData, nil
}

// LinkDimensionToStandard связывает измерение со стандартом
func (r *StandardRepo) LinkDimensionToStandard(ctx context.Context, standardID, dimensionID string) error {
	log := logQuery(ctx, r.log, "INSERT", "standard_context_dims",
		zap.String("standard_id", standardID),
		zap.String("dimension_id", dimensionID))
	log.Debug("linking dimension to standard")

	id := uuid.New().String()

	query := `INSERT OR IGNORE INTO standard_context_dims (id, standard_id, dimension_id) VALUES (?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, id, standardID, dimensionID)

	log.Info("dimension linked to standard successfully",
		zap.String("link_id", id))
	return err
}
