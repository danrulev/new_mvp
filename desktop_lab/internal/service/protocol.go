package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
	"fmt"

	"math"
	"strconv"
	"time"

	"github.com/Knetic/govaluate"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProtocolService struct {
	protocolRepo repository.ProtocolRepo
	sampleRepo   repository.SampleRepo
	standardRepo repository.StandardRepo // Нужен для загрузки методов и лимитов
	groupRepo    repository.ExperimentGroupRepo
	materialRepo repository.MaterialRepo
	log          *zap.Logger
}

func NewProtocolService(
	pRepo repository.ProtocolRepo,
	sRepo repository.SampleRepo,
	stdRepo repository.StandardRepo,
	gRepo repository.ExperimentGroupRepo,
	mRepo repository.MaterialRepo,
	log *zap.Logger,
) *ProtocolService {
	return &ProtocolService{
		protocolRepo: pRepo,
		sampleRepo:   sRepo,
		standardRepo: stdRepo,
		groupRepo:    gRepo,
		materialRepo: mRepo,
		log:          log,
	}
}

// CreateProtocolWithSample - аналог старого CreateProtocolWithSample
// Создает пробу, затем протокол с результатами, выполняя расчеты и валидацию
func (s *ProtocolService) CreateProtocolWithSample(ctx context.Context, req models.CreateProtocolRequest) (models.Protocol, error) {
	// 1. Создаем Пробу (Sample)
	sampleID := uuid.New().String()
	sample := models.Sample{
		ID:             sampleID,
		GroupID:        req.GroupID,
		MaterialID:     req.Sample.MaterialID,
		SampleNumber:   req.Sample.SampleNumber,
		CollectionDate: req.Sample.CollectionDate,
		ContextParams:  req.Sample.ContextParams, // Важно: контекст пробы!
		Note:           req.Sample.Note,
	}

	if err := s.sampleRepo.Create(ctx, sample); err != nil {
		s.log.Error("failed to create sample", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка создания пробы: %w", err)
	}

	// 2. Подготавливаем Протокол
	protocolID := uuid.New().String()
	now := time.Now()

	protocol := models.Protocol{
		ID:             protocolID,
		SampleID:       sampleID,
		ProtocolNumber: "", // Можно сгенерировать автоматически
		LabName:        req.LabName,
		OperatorName:   req.OperatorName,
		TestDate:       &now,
		Status:         "draft", // Или сразу "completed"
	}

	// 3. Обрабатываем результаты (Расчет + Валидация)
	finalResults := make([]models.TestResult, 0, len(req.Results))

	for _, inputRes := range req.Results {
		// Загружаем метод с формулой и входами
		method, err := s.standardRepo.GetMethodWithInputs(ctx, inputRes.MethodID)
		if err != nil {
			return models.Protocol{}, fmt.Errorf("метод %s не найден: %w", inputRes.MethodID, err)
		}

		var calculatedValue float64
		inputDataMap := make(map[string]interface{})

		// --- РАСЧЕТ ---
		if method.FormulaExpr != "" {
			// Парсим входные параметры
			params := make(map[string]interface{})

			for _, inp := range method.Inputs {
				valStr, exists := inputRes.RawInputs[inp.ParamKey]

				if inp.IsRequired && (!exists || valStr == "") {
					return models.Protocol{}, fmt.Errorf("требуется параметр '%s' для метода '%s'", inp.Label, method.Name)
				}

				if !exists || valStr == "" {
					continue // Опциональный параметр не введен
				}

				val, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					return models.Protocol{}, fmt.Errorf("некорректное число '%s' для параметра '%s'", valStr, inp.Label)
				}

				params[inp.ParamKey] = val
				inputDataMap[inp.ParamKey] = val
			}

			// Вычисляем формулу
			calculatedValue, err = s.calculateFormula(method.FormulaExpr, params)
			if err != nil {
				return models.Protocol{}, fmt.Errorf("ошибка расчета формулы для '%s': %w", method.Name, err)
			}
		} else {
			// Если формулы нет, ждем явное значение?
			// В старой модели было поле Value. В новой, если нет формулы, возможно, результат вводится вручную.
			// Для упрощения предположим, что если нет формулы, то первый ключ в RawInputs - это результат, или нужно доработать DTO.
			// Давайте предположим, что для ручного ввода мы передаем значение в ключе "value" или используем логику legacy.
			// Адаптация: если формулы нет, ищем ключ "result" или берем первое значение.
			// Но лучше изменить DTO для ручного ввода. Пока заглушка:
			if valStr, ok := inputRes.RawInputs["value"]; ok {
				v, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					return models.Protocol{}, fmt.Errorf("ошибка парсинга ручного значения: %w", err)
				}
				calculatedValue = v
				inputDataMap["value"] = v
			} else if len(inputRes.RawInputs) > 0 {
				// Берем первое попавшееся число как результат (fallback)
				for _, v := range inputRes.RawInputs {
					f, err := strconv.ParseFloat(v, 64)
					if err == nil {
						calculatedValue = f
						break
					}
				}
			}
		}

		// Округление (как в легаси)
		calculatedValue = math.Round(calculatedValue*100) / 100

		// --- ВАЛИДАЦИЯ (Новая логика) ---
		var isCompliant bool
		var deviationMsg string
		var appliedLimitID *string

		// Находим подходящий лимит на основе контекста пробы
		limit, err := s.standardRepo.GetApplicableLimit(ctx, method.ID, sample.ContextParams)
		if err != nil {
			s.log.Warn("error finding limit", zap.Error(err))
			// Не прерываем, просто не валидируем
		}

		switch limit.LimitType {
		case "min":
			if limit.MinValue != nil && calculatedValue < *limit.MinValue {
				isCompliant = false
				diff := *limit.MinValue - calculatedValue
				deviationMsg = fmt.Sprintf("Ниже нормы на %.2f %s (Мин: %.2f)", diff, method.Unit, *limit.MinValue)
			}
		case "max":
			if limit.MaxValue != nil && calculatedValue > *limit.MaxValue {
				isCompliant = false
				diff := calculatedValue - *limit.MaxValue
				deviationMsg = fmt.Sprintf("Выше нормы на %.2f %s (Макс: %.2f)", diff, method.Unit, *limit.MaxValue)
			}
		case "range":
			if limit.MinValue != nil && limit.MaxValue != nil {
				if calculatedValue < *limit.MinValue || calculatedValue > *limit.MaxValue {
					isCompliant = false
					if calculatedValue < *limit.MinValue {
						deviationMsg = fmt.Sprintf("Ниже диапазона [%.2f; %.2f]", *limit.MinValue, *limit.MaxValue)
					} else {
						deviationMsg = fmt.Sprintf("Выше диапазона [%.2f; %.2f]", *limit.MinValue, *limit.MaxValue)
					}
				}
			}
		default:
			isCompliant = true
			deviationMsg = "Норматив для данных условий не найден"
		}

		result := models.TestResult{
			MethodID:        method.ID,
			InputData:       inputDataMap,
			CalculatedValue: &calculatedValue,
			AppliedLimitID:  appliedLimitID,
			IsCompliant:     &isCompliant,
			DeviationMsg:    deviationMsg,
			Note:            inputRes.Note,
			// Поля для отображения заполним при чтении, либо можно заполнить тут именами
			MethodName: method.Name,
			MethodUnit: method.Unit,
		}

		result.MinNorm = limit.MinValue
		result.MaxNorm = limit.MaxValue

		finalResults = append(finalResults, result)
	}

	rawJSON, err := sample.ToJSON()
	if err != nil {
		return models.Protocol{}, fmt.Errorf("failed to marshal context: %w", err)
	}
	sample.RawContext = rawJSON // ✅ Заполняем сырой JSON

	// Аналогично для результатов:
	for i := range finalResults {
		rawInputs, err := finalResults[i].InputsToJSON()
		if err != nil {
			return models.Protocol{}, fmt.Errorf("failed to marshal inputs: %w", err)
		}
		finalResults[i].RawInputData = rawInputs
	}

	// 4. Сохраняем Протокол и Результаты одной транзакцией
	if err := s.protocolRepo.CreateFull(ctx, protocol, finalResults); err != nil {
		s.log.Error("failed to create protocol transaction", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка сохранения протокола: %w", err)
	}

	s.log.Info("protocol created with sample", zap.String("protocol_id", protocolID), zap.String("sample_id", sampleID))

	protocol.Sample = sample
	return protocol, nil
}

// calculateFormula использует govaluate (как в легаси)
func (s *ProtocolService) calculateFormula(exprStr string, params map[string]interface{}) (float64, error) {
	expr, err := govaluate.NewEvaluableExpression(exprStr)
	if err != nil {
		return 0, fmt.Errorf("синтаксическая ошибка формулы: %w", err)
	}

	// Добавляем математические функции
	funcs := map[string]interface{}{
		"abs":   math.Abs,
		"sqrt":  math.Sqrt,
		"log":   math.Log,
		"ln":    math.Log,
		"log10": math.Log10,
		"sin":   math.Sin,
		"cos":   math.Cos,
		"tan":   math.Tan,
		"pi":    math.Pi,
		"pow":   math.Pow,
		"min": func(a, b float64) float64 {
			if a < b {
				return a
			}
			return b
		},
		"max": func(a, b float64) float64 {
			if a > b {
				return a
			}
			return b
		},
	}

	// Мерджим параметры и функции
	allParams := make(map[string]interface{})
	for k, v := range params {
		allParams[k] = v
	}
	for k, v := range funcs {
		allParams[k] = v
	}

	result, err := expr.Evaluate(allParams)
	if err != nil {
		return 0, fmt.Errorf("ошибка вычисления: %w", err)
	}

	switch v := result.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("неподдерживаемый тип результата: %T", v)
	}
}

// GetProtocolByID загружает протокол с результатами и данными пробы

type GetProtocolByIDRequest struct {
	Protocol models.Protocol
	Results  []models.TestResult
}

func (s *ProtocolService) GetProtocolByID(ctx context.Context, id string) (GetProtocolByIDRequest, error) {
	p, err := s.protocolRepo.GetByID(ctx, id)
	if err != nil {
		return GetProtocolByIDRequest{}, err
	}
	if p.ID == "" {
		return GetProtocolByIDRequest{}, fmt.Errorf("protocol not found")
	}

	results, err := s.protocolRepo.GetResultsByProtocolID(ctx, id)
	if err != nil {
		return GetProtocolByIDRequest{}, err
	}

	for i := range results {
		if results[i].MethodName == "" {
			m, err := s.standardRepo.GetMethodWithInputs(ctx, results[i].MethodID)
			if err == nil {
				results[i].MethodName = m.Name
				results[i].MethodUnit = m.Unit
			}
		}
	}

	out := GetProtocolByIDRequest{Protocol: p, Results: results}

	return out, nil
}

// GetProtocolsByGroupID - аналог получения списка для сводки
func (s *ProtocolService) GetProtocolsByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
	return s.protocolRepo.GetByGroupID(ctx, groupID)
}

func (s *ProtocolService) GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error) {
	return s.protocolRepo.GetList(ctx, limit, offset)
}

// GetGroupSummary нужно реализовать аналогично старому коду, но с новой схемой
// internal/service/protocol.go

func (s *ProtocolService) GetGroupSummary(ctx context.Context, groupID string) (*models.GroupSummary, error) {
	// 1. Получаем группу
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if group.ID == "" {
		return nil, fmt.Errorf("group not found")
	}

	// 2. Получаем все протоколы группы
	// Примечание: нужен метод в репозитории, возвращающий только ID или краткие данные
	protocols, err := s.protocolRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	if len(protocols) == 0 {
		return &models.GroupSummary{
			GroupID:       group.ID,
			GroupName:     group.Name,
			MaterialID:    group.MaterialID,
			TotalSamples:  0,
			CompliantRate: 100.0,
			Results:       []models.MethodResultSummary{},
		}, nil
	}

	// 3. Агрегация результатов
	// Map: MethodID -> Summary
	methodMap := make(map[string]*models.MethodResultSummary)
	totalTests := 0
	compliantTests := 0

	for _, proto := range protocols {
		// Загружаем результаты конкретного протокола
		results, err := s.protocolRepo.GetResultsByProtocolID(ctx, proto.ID) // Нужен такой метод в репо
		if err != nil {
			s.log.Warn("failed to load results", zap.String("protocol", proto.ID), zap.Error(err))
			continue
		}

		for _, res := range results {
			// Загружаем детали метода (имя, единицы)
			method, err := s.standardRepo.GetMethodWithInputs(ctx, res.MethodID)
			if err != nil {
				continue
			}

			if _, exists := methodMap[method.ID]; !exists {
				// Инициализируем сводку по методу
				// Нормы берем из результата (если мы их сохранили денормализованно) или ищем лимит снова
				methodMap[method.ID] = &models.MethodResultSummary{
					MethodID:    method.ID,
					MethodName:  method.Name,
					Unit:        method.Unit,
					MinValue:    res.MinNorm, // Предполагаем, что в TestResult есть поля MinNorm/MaxNorm
					MaxValue:    res.MaxNorm,
					IsCompliant: true,
					Trials:      []models.MethodTrial{},
				}
			}

			entry := methodMap[method.ID]

			isComp := false
			if res.IsCompliant != nil {
				isComp = *res.IsCompliant
			}

			samples, err := s.sampleRepo.GetByGroupID(ctx, groupID)
			if err != nil {
				s.log.Warn("failed to load samples for summary", zap.Error(err))
			}
			sampleMap := make(map[string]string) // ID -> SampleNumber
			for _, samp := range samples {
				sampleMap[samp.ID] = samp.SampleNumber
			}

			trial := models.MethodTrial{
				ProtocolID:   proto.ID,
				SampleNumber: "", // Нужно подгрузить номер пробы из Proto.Sample
				Value:        0.0,
				IsCompliant:  isComp,
			}

			if res.CalculatedValue != nil {
				trial.Value = *res.CalculatedValue
			}
			if res.DeviationMsg != "" {
				trial.Deviation = &res.DeviationMsg
			}

			// TODO: Подгрузить SampleNumber из протокола или кэша, чтобы не делать лишний запрос в цикле

			entry.Trials = append(entry.Trials, trial)

			totalTests++
			if isComp {
				compliantTests++
			} else {
				entry.IsCompliant = false
			}
		}
	}

	// Преобразуем мапу в слайс
	var summaries []models.MethodResultSummary
	for _, v := range methodMap {
		summaries = append(summaries, *v)
	}

	rate := 0.0
	if totalTests > 0 {
		rate = float64(compliantTests) / float64(totalTests) * 100.0
	}

	return &models.GroupSummary{
		GroupID:       group.ID,
		GroupName:     group.Name,
		MaterialID:    group.MaterialID,
		TotalSamples:  len(protocols),
		CompliantRate: rate,
		Results:       summaries,
	}, nil
}
