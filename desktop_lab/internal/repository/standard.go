package repository

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"encoding/json"
	"fmt"

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

// CreateFull реализует атомарное создание стандарта со всей иерархией
func (r *standardRepo) CreateFull(ctx context.Context, req models.CreateStandardRequest) (string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Откат при любой ошибке
	defer func() {
		if err != nil {
			tx.Rollback()
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

	// 2. Создаем Dimensions (если есть)
	for _, dim := range req.Dimensions {
		dimID := uuid.New().String()
		valuesJSON := "null"
		if len(dim.PossibleValues) > 0 {
			b, _ := json.Marshal(dim.PossibleValues)
			valuesJSON = string(b)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO context_dimensions (id, standard_id, key_name, label, data_type, possible_values) 
			 VALUES (?, ?, ?, ?, ?, ?)`,
			dimID, stdID, dim.KeyName, dim.Label, dim.DataType, valuesJSON,
		)
		if err != nil {
			return "", fmt.Errorf("failed to insert dimension %s: %w", dim.KeyName, err)
		}
	}

	// 3. Создаем Методы и их вложенные сущности
	for _, mReq := range req.Methods {
		methodID := uuid.New().String()
		isMandatory := 1 // По умолчанию обязательный

		_, err = tx.ExecContext(ctx,
			`INSERT INTO test_methods (id, standard_id, code, name, formula_expr, unit, result_type, is_mandatory)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			methodID, stdID, mReq.Code, mReq.Name, mReq.FormulaExpr, mReq.Unit, "scalar", isMandatory,
		)
		if err != nil {
			return "", fmt.Errorf("failed to insert method %s: %w", mReq.Name, err)
		}

		// 3.1 Inputs
		for _, inp := range mReq.Inputs {
			inpID := uuid.New().String()
			isReq := 0
			if inp.IsRequired {
				isReq = 1
			}
			_, err = tx.ExecContext(ctx,
				`INSERT INTO method_inputs (id, method_id, param_key, label, unit, input_type, is_required)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				inpID, methodID, inp.ParamKey, inp.Label, inp.Unit, inp.InputType, isReq,
			)
			if err != nil {
				return "", fmt.Errorf("failed to insert input %s: %w", inp.ParamKey, err)
			}
		}

		// 3.2 Limits и Conditions
		for _, lim := range mReq.Limits {
			limitID := uuid.New().String()
			discreteJSON := "null"

			_, err = tx.ExecContext(ctx,
				`INSERT INTO normative_limits (id, method_id, limit_type, min_value, max_value, discrete_values, priority)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				limitID, methodID, lim.LimitType, lim.MinValue, lim.MaxValue, discreteJSON, 0,
			)
			if err != nil {
				return "", fmt.Errorf("failed to insert limit: %w", err)
			}

			// 3.3 Conditions для лимита
			for _, cond := range lim.Conditions {
				condID := uuid.New().String()
				_, err = tx.ExecContext(ctx,
					`INSERT INTO limit_conditions (id, limit_id, dimension_key, condition_operator, expected_value)
					 VALUES (?, ?, ?, ?, ?)`,
					condID, limitID, cond.DimensionKey, cond.Operator, cond.ExpectedValue,
				)
				if err != nil {
					return "", fmt.Errorf("failed to insert condition: %w", err)
				}
			}
		}
	}

	// Коммит транзакции
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.log.Info("Standard created successfully", zap.String("id", stdID))
	return stdID, nil
}

// GetApplicableLimit находит лимит, условия которого совпадают с контекстом пробы
func (r *standardRepo) GetApplicableLimit(ctx context.Context, methodID string, contextParams map[string]string) (models.NormativeLimit, error) {
	// Получаем все лимиты для метода
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, method_id, limit_type, min_value, max_value, priority 
		  FROM normative_limits 
		  WHERE method_id = ? 
		  ORDER BY priority DESC`, // Приоритетные сначала
		methodID,
	)
	if err != nil {
		return models.NormativeLimit{}, err
	}
	defer rows.Close()

	var limits []models.NormativeLimit
	for rows.Next() {
		var l models.NormativeLimit
		err := rows.Scan(&l.ID, &l.MethodID, &l.LimitType, &l.MinValue, &l.MaxValue, &l.Priority)
		if err != nil {
			return models.NormativeLimit{}, err
		}
		limits = append(limits, l)
	}

	if len(limits) == 0 {
		return models.NormativeLimit{}, nil // Нет лимитов для этого метода
	}

	// Для каждого лимита проверяем условия
	for i := range limits {
		limit := limits[i]

		// Загружаем условия для этого лимита
		condRows, err := r.db.QueryContext(ctx,
			`SELECT dimension_key, condition_operator, expected_value 
			  FROM limit_conditions 
			  WHERE limit_id = ?`,
			limit.ID,
		)
		if err != nil {
			return models.NormativeLimit{}, err
		}

		match := true
		for condRows.Next() {
			var key, op, expected string
			if err := condRows.Scan(&key, &op, &expected); err != nil {
				condRows.Close()
				return models.NormativeLimit{}, err
			}

			// Проверяем, есть ли такой ключ в контексте пробы
			actualVal, exists := contextParams[key]
			if !exists {
				match = false
				break
			}

			// Простейшая проверка равенства (можно расширить для IN, !=)
			if op == "=" && actualVal != expected {
				match = false
				break
			}
			if op == "!=" && actualVal == expected {
				match = false
				break
			}
		}
		condRows.Close()

		if match {
			// Нашли подходящий лимит! Загружаем для него полные данные (условия текстом для отладки)
			// В реальном коде можно сразу вернуть limit, но давайте заполним Conditions для полноты
			return limit, nil
		}
	}

	// Если ни один лимит не подошел по условиям, возвращаем первый (дефолтный) или nil?
	// Логика лаборатории: если нет специфического лимита, возможно, норма не применима.
	// Но часто бывает дефолтный лимит без условий. Давайте вернем первый, у которого нет условий.
	for i := range limits {
		// Проверим, есть ли у него условия вообще
		count := 0
		err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM limit_conditions WHERE limit_id = ?`, limits[i].ID).Scan(&count)
		if err != nil {
			continue
		}
		if count == 0 {
			return limits[i], nil
		}
	}

	return models.NormativeLimit{}, nil // Подходящего лимита не найдено
}

// GetMethodWithInputs загружает метод и его параметры ввода
func (r *standardRepo) GetMethodWithInputs(ctx context.Context, methodID string) (models.TestMethod, error) {
	method := models.TestMethod{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, standard_id, code, name, formula_expr, unit, result_type, is_mandatory 
		 FROM test_methods WHERE id = ?`,
		methodID,
	).Scan(&method.ID, &method.StandardID, &method.Code, &method.Name, &method.FormulaExpr, &method.Unit, &method.ResultType, &method.IsMandatory)

	if err == sql.ErrNoRows {
		return models.TestMethod{}, nil
	}
	if err != nil {
		return models.TestMethod{}, err
	}

	// Загружаем инпуты
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, param_key, label, unit, input_type, is_required 
		 FROM method_inputs WHERE method_id = ?`,
		methodID,
	)
	if err != nil {
		return models.TestMethod{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var inp models.MethodInput
		var isReq int
		err := rows.Scan(&inp.ID, &inp.ParamKey, &inp.Label, &inp.Unit, &inp.InputType, &isReq)
		if err != nil {
			return models.TestMethod{}, err
		}
		inp.IsRequired = isReq == 1
		inp.MethodID = methodID
		method.Inputs = append(method.Inputs, inp)
	}

	return method, nil
}

// GetByMaterialID загружает стандарты с методами (упрощенно, без глубокой вложенности лимитов для списка)
func (r *standardRepo) GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, description FROM standards WHERE material_id = ?`,
		materialID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var standards []models.Standard
	for rows.Next() {
		var s models.Standard
		s.MaterialID = materialID
		if err := rows.Scan(&s.ID, &s.Name, &s.Description); err != nil {
			return nil, err
		}

		// Загружаем методы для этого стандарта
		mRows, err := r.db.QueryContext(ctx,
			`SELECT id, code, name, unit, formula_expr FROM test_methods WHERE standard_id = ?`,
			s.ID,
		)
		if err != nil {
			return nil, err
		}

		for mRows.Next() {
			var m models.TestMethod
			if err := mRows.Scan(&m.ID, &m.Code, &m.Name, &m.Unit, &m.FormulaExpr); err != nil {
				mRows.Close()
				return nil, err
			}
			m.StandardID = s.ID
			s.Methods = append(s.Methods, m)
		}
		mRows.Close()

		standards = append(standards, s)
	}

	return standards, nil
}
func (r *standardRepo) GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error) {
	// 1. Получаем список методов
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, standard_id, code, name, description, formula_expr, unit, result_type, is_mandatory 
		 FROM test_methods 
		 WHERE standard_id = ? 
		 ORDER BY name`,
		standardID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query methods: %w", err)
	}
	defer rows.Close()

	var methods []models.TestMethod

	for rows.Next() {
		var m models.TestMethod
		var isMandatoryInt int

		err := rows.Scan(
			&m.ID, &m.StandardID, &m.Code, &m.Name, &m.Description,
			&m.FormulaExpr, &m.Unit, &m.ResultType, &isMandatoryInt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan method: %w", err)
		}

		m.IsMandatory = isMandatoryInt == 1

		// 2. Для каждого метода загружаем Inputs
		inputs, err := r.getMethodInputs(ctx, m.ID)
		if err != nil {
			r.log.Warn("failed to load inputs for method", zap.String("method_id", m.ID), zap.Error(err))
			// Не прерываем работу, просто оставляем пустой список инпутов
			inputs = []models.MethodInput{}
		}
		m.Inputs = inputs

		methods = append(methods, m)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return methods, nil
}

// getMethodInputs - вспомогательный приватный метод для загрузки параметров
func (r *standardRepo) getMethodInputs(ctx context.Context, methodID string) ([]models.MethodInput, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, method_id, param_key, label, unit, input_type, is_required 
		 FROM method_inputs 
		 WHERE method_id = ? 
		 ORDER BY param_key`, // Или по порядку ввода, если есть поле order
		methodID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var inputs []models.MethodInput
	for rows.Next() {
		var inp models.MethodInput
		var isRequiredInt int

		err := rows.Scan(
			&inp.ID, &inp.MethodID, &inp.ParamKey, &inp.Label,
			&inp.Unit, &inp.InputType, &isRequiredInt,
		)
		if err != nil {
			return nil, err
		}

		inp.IsRequired = isRequiredInt == 1
		inputs = append(inputs, inp)
	}

	return inputs, nil
}
