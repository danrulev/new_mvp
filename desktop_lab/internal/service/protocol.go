package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProtocolService struct {
	protocolRepo repository.ProtocolRepo
	sampleRepo   repository.SampleRepo
	standardRepo repository.StandardRepo
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

// CreateProtocolWithSample создает пробу, затем протокол с результатами,
// выполняя расчеты и валидацию с оптимизированной загрузкой методов
func (s *ProtocolService) CreateProtocolWithSample(ctx context.Context, req models.CreateProtocolRequest) (models.Protocol, error) {
	// 1. Создаем Пробу (Sample)
	sampleID := uuid.New().String()
	sample := models.Sample{
		ID:              sampleID,
		GroupID:         req.GroupID,
		MaterialID:      req.Sample.MaterialID,
		CollectionPlace: req.Sample.CollectionPlace,
		SampleNumber:    req.Sample.SampleNumber,
		CollectionDate:  req.Sample.CollectionDate,
		ContextParams:   req.Sample.ContextParams,
		Note:            req.Sample.Note,
	}

	if err := s.sampleRepo.Create(ctx, sample); err != nil {
		s.log.Error("failed to create sample", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка создания пробы: %w", err)
	}

	// 2. Подготавливаем Протокол
	protocolID := uuid.New().String()
	now := time.Now()
	protocolNumber, err := s.generateProtocolNumber(ctx, protocolID, req.Sample.MaterialID, req.GroupID, sampleID, now)
	if err != nil {
		s.log.Error("failed to generate protocol number", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка генерации номера протокола: %w", err)
	}

	protocol := models.Protocol{
		ID:             protocolID,
		SampleID:       sampleID,
		ProtocolNumber: protocolNumber,
		LabName:        req.LabName,
		OperatorName:   req.OperatorName,
		TestDate:       now,
		Status:         "draft",
		Note:           req.Note,
	}

	// Предзагрузка всех методов стандарта
	// Определяем StandardID по первому методу (или можно передавать в запросе)
	standardID := ""
	if len(req.Results) > 0 {
		firstMethod, err := s.standardRepo.GetTestMethod(ctx, req.Results[0].MethodID)
		if err != nil {
			return models.Protocol{}, fmt.Errorf("не удалось определить стандарт: %w", err)
		}
		standardID = firstMethod.StandardID

		// Загружаем ВСЕ методы, инпуты и лимиты стандарта ОДИН запросом
		methodsCache, err := s.standardRepo.GetMethodsFullByStandardID(ctx, standardID)
		if err != nil {
			s.log.Warn("failed to preload methods, falling back to individual queries",
				zap.Error(err), zap.String("standard_id", standardID))
			// Продолжаем работу, в цикле ниже будут индивидуальные запросы
		} else {
			// Используем кэш в цикле обработки результатов
			return s.createProtocolWithCache(ctx, protocol, sample, req.Results, methodsCache)
		}
	}

	// Фоллбэк: старая логика с индивидуальными запросами (если кэш не сработал)
	return s.createProtocolLegacy(ctx, protocol, sample, req.Results)
}

// createProtocolWithCache - оптимизированная версия с использованием предзагруженных данных
func (s *ProtocolService) createProtocolWithCache(
	ctx context.Context,
	protocol models.Protocol,
	sample models.Sample,
	results []models.CreateResultDTO,
	methodsCache map[string]models.TestMethodFull,
) (models.Protocol, error) {
	finalResults := make([]models.TestResult, 0, len(results))

	for _, inputRes := range results {
		// Быстрый доступ из кэша
		fullMethod, exists := methodsCache[inputRes.MethodID]
		if !exists {
			return models.Protocol{}, fmt.Errorf("метод %s не найден в стандарте", inputRes.MethodID)
		}

		method := fullMethod.Method
		inputs := fullMethod.Inputs

		s.log.Debug("Processing method",
			zap.String("method_id", method.ID),
			zap.String("formula", method.FormulaExpr),
			zap.Any("raw_inputs", inputRes.RawInputs),
			zap.Bool("is_mandatory", method.IsMandatory))

		var calculatedValue float64
		inputDataMap := make(map[string]interface{})

		// Проверка: есть ли формула?
		hasFormula := method.FormulaExpr != ""

		if hasFormula {
			params := make(map[string]interface{})

			for _, inp := range inputs {
				valStr, exists := inputRes.RawInputs[inp.ParamKey]

				s.log.Debug("Checking input param",
					zap.String("key", inp.ParamKey),
					zap.String("found_value", valStr),
					zap.Bool("exists", exists))

				if inp.IsRequired && (!exists || valStr == "") {
					return models.Protocol{}, fmt.Errorf("требуется параметр '%s' (%s) для метода '%s'", inp.Label, inp.ParamKey, method.Name)
				}
				if !exists || valStr == "" {
					continue
				}

				val, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					return models.Protocol{}, fmt.Errorf("некорректное число '%s' для параметра '%s': %w", valStr, inp.Label, err)
				}

				params[inp.ParamKey] = val
				inputDataMap[inp.ParamKey] = val
			}

			s.log.Debug("Calling calculateFormula",
				zap.String("expr", method.FormulaExpr),
				zap.Any("params", params))

			calcVal, err := s.calculateFormula(method.FormulaExpr, params)
			if err != nil {
				s.log.Error("Formula calculation failed", zap.Error(err), zap.String("method", method.Name))
				return models.Protocol{}, fmt.Errorf("ошибка расчета формулы '%s': %w", method.Name, err)
			}

			calculatedValue = math.Round(calcVal*100) / 100
			s.log.Debug("Calculation result", zap.Float64("value", calculatedValue))

		} else {
			// Ветка ручного ввода
			s.log.Debug("No formula found, using manual value")
			calculatedValue = s.parseManualValue(inputRes.RawInputs, &inputDataMap)
			calculatedValue = math.Round(calculatedValue*100) / 100
		}

		// --- ВАЛИДАЦИЯ (с использованием предзагруженных лимитов) ---
		applicableLimit := s.findMatchingLimit(fullMethod.Limits, fullMethod.LimitConditions, sample.ContextParams)

		isCompliant := true
		var deviationMsg string
		var appliedLimitID *string

		if applicableLimit.ID != "" {
			appliedLimitID = &applicableLimit.ID

			switch applicableLimit.LimitType {
			case "min":
				if applicableLimit.MinValue != nil && calculatedValue < *applicableLimit.MinValue {
					isCompliant = false
					deviationMsg = fmt.Sprintf("Ниже нормы на %.2f %s", *applicableLimit.MinValue-calculatedValue, method.Unit)
				}
			case "max":
				if applicableLimit.MaxValue != nil && calculatedValue > *applicableLimit.MaxValue {
					isCompliant = false
					deviationMsg = fmt.Sprintf("Выше нормы на %.2f %s", calculatedValue-*applicableLimit.MaxValue, method.Unit)
				}
			case "range":
				if applicableLimit.MinValue != nil && applicableLimit.MaxValue != nil {
					if calculatedValue < *applicableLimit.MinValue || calculatedValue > *applicableLimit.MaxValue {
						isCompliant = false
						if calculatedValue < *applicableLimit.MinValue {
							deviationMsg = fmt.Sprintf("Ниже диапазона [%.2f; %.2f]", *applicableLimit.MinValue, *applicableLimit.MaxValue)
						} else {
							deviationMsg = fmt.Sprintf("Выше диапазона [%.2f; %.2f]", *applicableLimit.MinValue, *applicableLimit.MaxValue)
						}
					}
				}
			}
		} else {
			deviationMsg = "Норматив не применён (условия не найдены)"
		}

		result := models.TestResult{
			MethodID:        method.ID,
			InputData:       inputDataMap,
			CalculatedValue: &calculatedValue,
			AppliedLimitID:  appliedLimitID,
			IsCompliant:     &isCompliant,
			DeviationMsg:    deviationMsg,
			Note:            inputRes.Note,
		}
		finalResults = append(finalResults, result)
	}

	return s.saveProtocol(ctx, protocol, sample, finalResults)
}

// createProtocolLegacy - фоллбэк-логика с индивидуальными запросами к БД
func (s *ProtocolService) createProtocolLegacy(
	ctx context.Context,
	protocol models.Protocol,
	sample models.Sample,
	results []models.CreateResultDTO,
) (models.Protocol, error) {
	finalResults := make([]models.TestResult, 0, len(results))

	for _, inputRes := range results {
		method, err := s.standardRepo.GetTestMethod(ctx, inputRes.MethodID)
		if err != nil {
			return models.Protocol{}, fmt.Errorf("метод %s не найден: %w", inputRes.MethodID, err)
		}

		inputs, err := s.standardRepo.GetMethodInputs(ctx, inputRes.MethodID)
		if err != nil {
			return models.Protocol{}, fmt.Errorf("ошибка загрузки инпутов для %s: %w", inputRes.MethodID, err)
		}

		var calculatedValue float64
		inputDataMap := make(map[string]interface{})

		if method.FormulaExpr != "" {
			params := make(map[string]interface{})
			for _, inp := range inputs {
				valStr, exists := inputRes.RawInputs[inp.ParamKey]
				if inp.IsRequired && (!exists || valStr == "") {
					return models.Protocol{}, fmt.Errorf("требуется параметр '%s' для метода '%s'", inp.Label, method.Name)
				}
				if !exists || valStr == "" {
					continue
				}
				val, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					return models.Protocol{}, fmt.Errorf("некорректное число '%s': %w", valStr, err)
				}
				params[inp.ParamKey] = val
				inputDataMap[inp.ParamKey] = val
			}
			calculatedValue, err = s.calculateFormula(method.FormulaExpr, params)
			if err != nil {
				return models.Protocol{}, fmt.Errorf("ошибка расчета формулы '%s': %w", method.Name, err)
			}
			calculatedValue = math.Round(calculatedValue*100) / 100
		} else {
			calculatedValue = s.parseManualValue(inputRes.RawInputs, &inputDataMap)
			calculatedValue = math.Round(calculatedValue*100) / 100
		}

		// Валидация через запрос к БД
		limit, err := s.standardRepo.GetApplicableLimit(ctx, method.ID, sample.ContextParams)
		if err != nil {
			s.log.Warn("error finding limit", zap.Error(err))
		}

		isCompliant := true
		var deviationMsg string
		var appliedLimitID *string

		if limit.ID != "" {
			appliedLimitID = &limit.ID
			switch limit.LimitType {
			case "min":
				if limit.MinValue != nil && calculatedValue < *limit.MinValue {
					isCompliant = false
					deviationMsg = fmt.Sprintf("Ниже нормы на %.2f %s", *limit.MinValue-calculatedValue, method.Unit)
				}
			case "max":
				if limit.MaxValue != nil && calculatedValue > *limit.MaxValue {
					isCompliant = false
					deviationMsg = fmt.Sprintf("Выше нормы на %.2f %s", calculatedValue-*limit.MaxValue, method.Unit)
				}
			case "range":
				if limit.MinValue != nil && limit.MaxValue != nil {
					if calculatedValue < *limit.MinValue || calculatedValue > *limit.MaxValue {
						isCompliant = false
						deviationMsg = fmt.Sprintf("Вне диапазона [%.2f; %.2f]", *limit.MinValue, *limit.MaxValue)
					}
				}
			}
		} else {
			deviationMsg = "Норматив не применён"
		}

		result := models.TestResult{
			MethodID:        method.ID,
			InputData:       inputDataMap,
			CalculatedValue: &calculatedValue,
			AppliedLimitID:  appliedLimitID,
			IsCompliant:     &isCompliant,
			DeviationMsg:    deviationMsg,
			Note:            inputRes.Note,
		}
		finalResults = append(finalResults, result)
	}

	return s.saveProtocol(ctx, protocol, sample, finalResults)
}

// findMatchingLimit ищет подходящий лимит в предзагруженных данных (работает в памяти)
func (s *ProtocolService) findMatchingLimit(
	limits []models.NormativeLimit,
	conditionsMap map[string][]models.LimitCondition,
	contextParams map[string]string,
) models.NormativeLimit {
	var defaultLimit *models.NormativeLimit

	for i := range limits {
		limit := limits[i]
		conds := conditionsMap[limit.ID]

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
				// Простая реализация: ожидаемое значение - список через запятую
				// Можно улучшить парсингом JSON-массива
				if !containsValue(cond.ExpectedValue, actualVal) {
					match = false
				}
			}
			if !match {
				break
			}
		}

		if match {
			return limit
		}
	}

	if defaultLimit != nil {
		return *defaultLimit
	}

	return models.NormativeLimit{}
}

// containsValue проверяет наличие значения в строке "val1,val2,val3"
func containsValue(csv, target string) bool {
	for _, v := range splitCSV(csv) {
		if v == target {
			return true
		}
	}
	return false
}

// splitCSV простая реализация разделения строки по запятым
func splitCSV(s string) []string {
	var result []string
	var current string
	for _, r := range s {
		if r == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// parseManualValue парсит ручное значение из RawInputs
func (s *ProtocolService) parseManualValue(rawInputs map[string]string, outMap *map[string]interface{}) float64 {
	if valStr, ok := rawInputs["value"]; ok {
		if v, err := strconv.ParseFloat(valStr, 64); err == nil {
			(*outMap)["value"] = v
			return v
		}
	}
	// Fallback: берем первое валидное число
	for k, v := range rawInputs {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			(*outMap)[k] = f
			return f
		}
	}
	return 0
}

// saveProtocol сериализует и сохраняет протокол с результатами
func (s *ProtocolService) saveProtocol(
	ctx context.Context,
	protocol models.Protocol,
	sample models.Sample,
	results []models.TestResult,
) (models.Protocol, error) {
	// Сериализация контекста пробы
	rawJSON, err := sample.ToJSON()
	if err != nil {
		return models.Protocol{}, fmt.Errorf("failed to marshal context: %w", err)
	}
	sample.RawContext = rawJSON

	// Сериализация входных данных результатов
	for i := range results {
		rawInputs, err := results[i].InputsToJSON()
		if err != nil {
			return models.Protocol{}, fmt.Errorf("failed to marshal inputs: %w", err)
		}
		results[i].RawInputData = rawInputs
	}

	// Сохранение в транзакции
	if err := s.protocolRepo.CreateFull(ctx, protocol, results); err != nil {
		s.log.Error("failed to create protocol transaction", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка сохранения протокола: %w", err)
	}

	s.log.Info("protocol created successfully",
		zap.String("protocol_id", protocol.ID),
		zap.String("sample_id", sample.ID))

	return protocol, nil
}

// GetProtocolFull загружает полный протокол с пробой, материалом и результатами (ОПТИМИЗИРОВАНО)
func (s *ProtocolService) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
	full, err := s.protocolRepo.GetProtocolFull(ctx, id)
	if err != nil {
		return models.ProtocolFull{}, err
	}
	if full.IsEmpty() {
		return models.ProtocolFull{}, fmt.Errorf("protocol not found")
	}
	return full, nil
}

// GetProtocolsByGroupID возвращает список протоколов группы
func (s *ProtocolService) GetProtocolsByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
	return s.protocolRepo.GetByGroupID(ctx, groupID)
}

// GetList возвращает список протоколов с пагинацией
func (s *ProtocolService) GetList(ctx context.Context, limit, offset int64) (models.ProtocolListResponse, error) {
	protocols, total, err := s.protocolRepo.GetList(ctx, limit, offset)
	if err != nil {
		return models.ProtocolListResponse{}, err
	}

	return models.ProtocolListResponse{
		Items: protocols,
		Meta:  models.MakePaginatedMetadata(limit, offset, total),
	}, nil
}

// GetGroupSummary формирует сводный отчет по группе испытаний (ОПТИМИЗИРОВАНО)
func (s *ProtocolService) GetGroupSummary(ctx context.Context, groupID string) (models.GroupSummary, error) {
	// 1. Получаем группу
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return models.GroupSummary{}, fmt.Errorf("failed to get group: %w", err)
	}
	if group.ID == "" {
		return models.GroupSummary{}, fmt.Errorf("group not found")
	}

	// 2. Получаем протоколы группы (оптимизированный запрос с JOIN)
	protocols, err := s.protocolRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		return models.GroupSummary{}, fmt.Errorf("failed to get protocols: %w", err)
	}

	if len(protocols) == 0 {
		return models.GroupSummary{
			GroupID:       group.ID,
			GroupName:     group.Name,
			MaterialID:    group.MaterialID,
			TotalSamples:  0,
			CompliantRate: 100.0,
			Results:       []models.MethodResultSummary{},
		}, nil
	}

	// 3. 🔥 ПРЕДЗАГРУЗКА: Все пробы группы для быстрого маппинга
	samples, err := s.sampleRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		s.log.Warn("failed to load samples for summary", zap.Error(err))
	}
	sampleMap := make(map[string]string, len(samples))
	for _, samp := range samples {
		sampleMap[samp.ID] = samp.SampleNumber
	}

	// 4. Агрегация результатов
	methodMap := make(map[string]*models.MethodResultSummary) // pointer для мутаций
	totalTests := 0
	compliantTests := 0

	// Кэш методов, чтобы не грузить одно и то же много раз
	methodCache := make(map[string]models.TestMethod)

	for _, proto := range protocols {
		results, err := s.protocolRepo.GetResultsByProtocolID(ctx, proto.ID)
		if err != nil {
			s.log.Warn("failed to load results", zap.String("protocol_id", proto.ID), zap.Error(err))
			continue
		}

		for _, res := range results {
			// Получаем метод из кэша или БД
			method, exists := methodCache[res.MethodID]
			if !exists {
				method, err = s.standardRepo.GetTestMethod(ctx, res.MethodID)
				if err != nil {
					s.log.Warn("failed to load method", zap.String("method_id", res.MethodID), zap.Error(err))
					continue
				}
				methodCache[res.MethodID] = method
			}

			// Инициализируем сводку по методу при первом появлении
			summary, exists := methodMap[method.ID]
			if !exists {
				summary = &models.MethodResultSummary{
					MethodID:    method.ID,
					MethodName:  method.Name,
					Unit:        method.Unit,
					IsCompliant: true,
					Trials:      make([]models.MethodTrial, 0),
				}
				methodMap[method.ID] = summary
			}

			// Формируем запись испытания
			isComp := false
			if res.IsCompliant != nil {
				isComp = *res.IsCompliant
			}

			trial := models.MethodTrial{
				ProtocolID:   proto.ID,
				SampleNumber: sampleMap[proto.SampleID], // ✅ Быстрый доступ из предзагруженной мапы
				IsCompliant:  isComp,
			}

			if res.CalculatedValue != nil {
				trial.Value = *res.CalculatedValue
			}
			if res.DeviationMsg != "" {
				trial.Deviation = &res.DeviationMsg
			}

			summary.Trials = append(summary.Trials, trial)

			totalTests++
			if isComp {
				compliantTests++
			} else {
				summary.IsCompliant = false
			}
		}
	}

	// Преобразуем мапу в слайс для ответа
	summaries := make([]models.MethodResultSummary, 0, len(methodMap))
	for _, v := range methodMap {
		summaries = append(summaries, *v)
	}

	// Расчет процента соответствия
	rate := 100.0
	if totalTests > 0 {
		rate = float64(compliantTests) / float64(totalTests) * 100.0
	}

	return models.GroupSummary{
		GroupID:       group.ID,
		GroupName:     group.Name,
		MaterialID:    group.MaterialID,
		TotalSamples:  len(protocols),
		CompliantRate: rate,
		Results:       summaries,
	}, nil
}

func (s *ProtocolService) generateProtocolNumber(ctx context.Context, protocolID, materialID, groupID, sampleID string, createdAt time.Time) (string, error) {
	mat, err := s.materialRepo.GetByID(ctx, materialID)
	if err != nil {
		return "", err
	}
	isGroup := ""
	if groupID != "" {
		isGroup = "G"
	}
	return fmt.Sprintf("%s%s-%s-%s-%s", isGroup, mat.Code[:8], sampleID[:8], protocolID[:8], createdAt.Format("20060102")), nil
}

func (s *ProtocolService) UpdateProtocol(ctx context.Context, id string, req models.UpdateProtocolRequest) error {
	prot, err := s.protocolRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if prot.Status != "draft" {
		return fmt.Errorf("cannot update protocol with status %s", prot.Status)
	}

	return s.protocolRepo.UpdateProtocol(ctx, id, req)
}

func (s *ProtocolService) UpdateProtocolStatus(ctx context.Context, id string, status string) error {
	return s.protocolRepo.UpdateStatus(ctx, id, status)
}

func (s *ProtocolService) DeleteProtocol(ctx context.Context, id string) error {
	return s.protocolRepo.DeleteProtocol(ctx, id)
}
