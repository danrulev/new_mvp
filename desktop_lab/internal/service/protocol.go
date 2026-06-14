package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ProtocolService struct {
	protocolRepo ProtocolRepo
	sampleRepo   SampleRepo
	standardRepo StandardRepo
	groupRepo    ExperimentGroupRepo
	materialRepo MaterialRepo
	log          *zap.Logger
}

func NewProtocolService(
	pRepo ProtocolRepo,
	sRepo SampleRepo,
	stdRepo StandardRepo,
	gRepo ExperimentGroupRepo,
	mRepo MaterialRepo,
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

// CreateProtocolWithSample создает пробу, затем протокол с результатами
func (s *ProtocolService) CreateProtocolWithSample(ctx context.Context, req models.CreateProtocolRequest) (models.Protocol, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "CreateProtocolWithSample"),
		zap.String("group_id", req.GroupID),
		zap.String("material_id", req.Sample.MaterialID),
		zap.String("lab_name", req.LabName),
	)
	log.Info("starting protocol creation workflow")
	start := time.Now()
	defer func() {
		log.Debug("protocol creation workflow completed", zap.Duration("duration_ms", time.Since(start)))
	}()

	if req.Sample.CollectionDate == nil {
		now := time.Now()
		req.Sample.CollectionDate = &now
	}

	// 1. Создаем Пробу (Sample)
	sampleID := uuid.New().String()
	sample := models.Sample{
		ID:              sampleID,
		GroupID:         req.GroupID,
		MaterialID:      req.Sample.MaterialID,
		CollectionPlace: req.Sample.CollectionPlace,
		SampleNumber:    req.Sample.SampleNumber,
		CollectionDate:  *req.Sample.CollectionDate,
		ContextParams:   req.Sample.ContextParams,
		Note:            req.Sample.Note,
	}

	log.Debug("creating sample", zap.String("sample_id", sampleID))
	if err := s.sampleRepo.Create(ctx, sample); err != nil {
		log.Error("failed to create sample in repo", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка создания пробы: %w", err)
	}
	log.Debug("sample created successfully")

	// 2. Подготавливаем Протокол
	protocolID := uuid.New().String()
	now := time.Now()

	log.Debug("generating protocol number")
	protocolNumber, err := s.generateProtocolNumber(ctx, protocolID, req.Sample.MaterialID, req.GroupID, sampleID, now)
	if err != nil {
		log.Error("failed to generate protocol number", zap.Error(err))
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

	// Предзагрузка всех методов стандарта (оптимизация)
	if len(req.Results) > 0 {
		firstMethodID := req.Results[0].MethodID
		log.Debug("preloading standard methods cache", zap.String("first_method_id", firstMethodID))

		firstMethod, err := s.standardRepo.GetTestMethod(ctx, firstMethodID)
		if err != nil {
			log.Error("failed to determine standard from first method", zap.Error(err), zap.String("method_id", firstMethodID))
			return models.Protocol{}, fmt.Errorf("не удалось определить стандарт: %w", err)
		}
		standardID := firstMethod.StandardID

		methodsCache, err := s.standardRepo.GetMethodsFullByStandardID(ctx, standardID)
		if err != nil {
			log.Warn("failed to preload methods cache, falling back to individual queries",
				zap.Error(err), zap.String("standard_id", standardID))
			// Продолжаем работу в режиме legacy
		} else {
			log.Info("using optimized path with methods cache",
				zap.Int("methods_cached", len(methodsCache)))
			return s.createProtocolWithCache(ctx, protocol, sample, req.Results, methodsCache)
		}
	}

	log.Info("using legacy path with individual queries")
	return s.createProtocolLegacy(ctx, protocol, sample, req.Results)
}

// createProtocolWithCache - оптимизированная версия с кэшированием
func (s *ProtocolService) createProtocolWithCache(
	ctx context.Context,
	protocol models.Protocol,
	sample models.Sample,
	results []models.CreateResultDTO,
	methodsCache map[string]models.TestMethodFull,
) (models.Protocol, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "createProtocolWithCache"),
		zap.String("protocol_id", protocol.ID),
		zap.Int("results_count", len(results)),
	)
	log.Debug("processing protocol with cached methods")

	finalResults := make([]models.TestResult, 0, len(results))

	for i, inputRes := range results {
		methodLog := log.With(zap.Int("result_index", i), zap.String("method_id", inputRes.MethodID))

		fullMethod, exists := methodsCache[inputRes.MethodID]
		if !exists {
			methodLog.Error("method not found in cache", zap.String("method_id", inputRes.MethodID))
			return models.Protocol{}, fmt.Errorf("метод %s не найден в стандарте", inputRes.MethodID)
		}

		method := fullMethod.Method
		inputs := fullMethod.Inputs

		methodLog.Debug("processing method",
			zap.String("method_name", method.Name),
			zap.Bool("has_formula", method.FormulaExpr != ""),
			zap.Bool("is_mandatory", method.IsMandatory))

		var calculatedValue float64
		inputDataMap := make(map[string]interface{})

		if method.FormulaExpr != "" {
			params := make(map[string]interface{})
			for _, inp := range inputs {
				valStr, exists := inputRes.RawInputs[inp.ParamKey]

				if inp.IsRequired && (!exists || valStr == "") {
					methodLog.Error("missing required parameter",
						zap.String("param_key", inp.ParamKey),
						zap.String("param_label", inp.Label))
					return models.Protocol{}, fmt.Errorf("требуется параметр '%s' (%s) для метода '%s'", inp.Label, inp.ParamKey, method.Name)
				}
				if !exists || valStr == "" {
					methodLog.Debug("skipping optional empty parameter", zap.String("param_key", inp.ParamKey))
					continue
				}

				val, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					methodLog.Error("failed to parse parameter value",
						zap.Error(err),
						zap.String("param_key", inp.ParamKey),
						zap.String("raw_value", valStr))
					return models.Protocol{}, fmt.Errorf("некорректное число '%s' для параметра '%s': %w", valStr, inp.Label, err)
				}

				params[inp.ParamKey] = val
				inputDataMap[inp.ParamKey] = val
			}

			calcVal, err := s.calculateFormula(method.FormulaExpr, params)
			if err != nil {
				methodLog.Error("formula calculation failed", zap.Error(err), zap.String("formula", method.FormulaExpr))
				return models.Protocol{}, fmt.Errorf("ошибка расчета формулы '%s': %w", method.Name, err)
			}

			calculatedValue = math.Round(calcVal*100) / 100
			methodLog.Debug("formula calculated", zap.Float64("result", calculatedValue))

		} else {
			methodLog.Debug("using manual value (no formula)")
			calculatedValue = s.parseManualValue(inputRes.RawInputs, &inputDataMap)
			calculatedValue = math.Round(calculatedValue*100) / 100
		}

		// --- ВАЛИДАЦИЯ ---
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
			methodLog.Debug("limit applied",
				zap.String("limit_id", applicableLimit.ID),
				zap.String("limit_type", applicableLimit.LimitType),
				zap.Bool("is_compliant", isCompliant))
		} else {
			deviationMsg = "Норматив не применён (условия не найдены)"
			methodLog.Debug("no applicable limit found")
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

	log.Debug("all results processed, saving protocol")
	return s.saveProtocol(ctx, protocol, sample, finalResults)
}

// createProtocolLegacy - фоллбэк с индивидуальными запросами
func (s *ProtocolService) createProtocolLegacy(
	ctx context.Context,
	protocol models.Protocol,
	sample models.Sample,
	results []models.CreateResultDTO,
) (models.Protocol, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "createProtocolLegacy"),
		zap.String("protocol_id", protocol.ID),
	)
	log.Warn("using legacy path - individual DB queries per method")

	finalResults := make([]models.TestResult, 0, len(results))

	for i, inputRes := range results {
		methodLog := log.With(zap.Int("result_index", i), zap.String("method_id", inputRes.MethodID))

		method, err := s.standardRepo.GetTestMethod(ctx, inputRes.MethodID)
		if err != nil {
			methodLog.Error("failed to fetch method", zap.Error(err))
			return models.Protocol{}, fmt.Errorf("метод %s не найден: %w", inputRes.MethodID, err)
		}

		inputs, err := s.standardRepo.GetMethodInputs(ctx, inputRes.MethodID)
		if err != nil {
			methodLog.Error("failed to fetch method inputs", zap.Error(err))
			return models.Protocol{}, fmt.Errorf("ошибка загрузки инпутов для %s: %w", inputRes.MethodID, err)
		}

		var calculatedValue float64
		inputDataMap := make(map[string]interface{})

		if method.FormulaExpr != "" {
			params := make(map[string]interface{})
			for _, inp := range inputs {
				valStr, exists := inputRes.RawInputs[inp.ParamKey]
				if inp.IsRequired && (!exists || valStr == "") {
					methodLog.Error("missing required parameter", zap.String("param_key", inp.ParamKey))
					return models.Protocol{}, fmt.Errorf("требуется параметр '%s' для метода '%s'", inp.Label, method.Name)
				}
				if !exists || valStr == "" {
					continue
				}
				val, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					methodLog.Error("failed to parse parameter", zap.Error(err), zap.String("value", valStr))
					return models.Protocol{}, fmt.Errorf("некорректное число '%s': %w", valStr, err)
				}
				params[inp.ParamKey] = val
				inputDataMap[inp.ParamKey] = val
			}
			calculatedValue, err = s.calculateFormula(method.FormulaExpr, params)
			if err != nil {
				methodLog.Error("formula calculation failed", zap.Error(err))
				return models.Protocol{}, fmt.Errorf("ошибка расчета формулы '%s': %w", method.Name, err)
			}
			calculatedValue = math.Round(calculatedValue*100) / 100
		} else {
			calculatedValue = s.parseManualValue(inputRes.RawInputs, &inputDataMap)
			calculatedValue = math.Round(calculatedValue*100) / 100
		}

		// Валидация через БД
		limit, err := s.standardRepo.GetApplicableLimit(ctx, method.ID, sample.ContextParams)
		if err != nil {
			methodLog.Warn("error finding limit", zap.Error(err))
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

// findMatchingLimit - поиск лимита в памяти (без логирования, чистая функция)
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

// Вспомогательные функции (без логирования - чистые утилиты)
func containsValue(csv, target string) bool {
	for _, v := range splitCSV(csv) {
		if v == target {
			return true
		}
	}
	return false
}

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

func (s *ProtocolService) parseManualValue(rawInputs map[string]string, outMap *map[string]interface{}) float64 {
	if valStr, ok := rawInputs["value"]; ok {
		if v, err := strconv.ParseFloat(valStr, 64); err == nil {
			(*outMap)["value"] = v
			return v
		}
	}
	for k, v := range rawInputs {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			(*outMap)[k] = f
			return f
		}
	}
	return 0
}

// saveProtocol - сохранение протокола с транзакцией
func (s *ProtocolService) saveProtocol(
	ctx context.Context,
	protocol models.Protocol,
	sample models.Sample,
	results []models.TestResult,
) (models.Protocol, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "saveProtocol"),
		zap.String("protocol_id", protocol.ID),
		zap.String("sample_id", sample.ID),
		zap.Int("results_count", len(results)),
	)
	log.Debug("serializing and saving protocol")

	rawJSON, err := sample.ToJSON()
	if err != nil {
		log.Error("failed to marshal sample context", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("failed to marshal context: %w", err)
	}
	sample.RawContext = rawJSON

	for i := range results {
		rawInputs, err := results[i].InputsToJSON()
		if err != nil {
			log.Error("failed to marshal result inputs", zap.Error(err), zap.Int("result_index", i))
			return models.Protocol{}, fmt.Errorf("failed to marshal inputs: %w", err)
		}
		results[i].RawInputData = rawInputs
	}

	if err := s.protocolRepo.CreateFull(ctx, protocol, results); err != nil {
		log.Error("failed to create protocol in transaction", zap.Error(err))
		return models.Protocol{}, fmt.Errorf("ошибка сохранения протокола: %w", err)
	}

	log.Info("protocol and results saved successfully",
		zap.String("protocol_number", protocol.ProtocolNumber),
		zap.String("status", protocol.Status))

	return protocol, nil
}

// GetProtocolFull - загрузка полного протокола
func (s *ProtocolService) GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetProtocolFull"),
		zap.String("protocol_id", id),
	)
	log.Debug("fetching full protocol")

	full, err := s.protocolRepo.GetProtocolFull(ctx, id)
	if err != nil {
		log.Error("failed to fetch full protocol from repo", zap.Error(err))
		return models.ProtocolFull{}, err
	}
	if full.IsEmpty() {
		log.Warn("protocol not found", zap.String("searched_id", id))
		return models.ProtocolFull{}, fmt.Errorf("protocol not found")
	}

	log.Debug("full protocol retrieved successfully")
	return full, nil
}

// GetProtocolsByGroupID - список протоколов группы
func (s *ProtocolService) GetProtocolsByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetProtocolsByGroupID"),
		zap.String("group_id", groupID),
	)
	log.Debug("fetching protocols by group")

	protocols, err := s.protocolRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		log.Error("failed to fetch protocols by group", zap.Error(err))
		return nil, err
	}

	log.Debug("protocols retrieved", zap.Int("count", len(protocols)))
	return protocols, nil
}

// GetList - пагинированный список протоколов
func (s *ProtocolService) GetList(ctx context.Context, limit, offset int64) (models.ProtocolListResponse, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetList"),
		zap.Int64("limit", limit),
		zap.Int64("offset", offset),
	)
	log.Debug("fetching paginated protocols list")

	protocols, total, err := s.protocolRepo.GetList(ctx, limit, offset)
	if err != nil {
		log.Error("failed to fetch protocols list", zap.Error(err))
		return models.ProtocolListResponse{}, err
	}

	log.Debug("protocols list retrieved",
		zap.Int("returned", len(protocols)),
		zap.Int64("total", total))

	return models.ProtocolListResponse{
		Items: protocols,
		Meta:  models.MakePaginatedMetadata(limit, offset, total),
	}, nil
}

// GetGroupSummary - сводный отчёт по группе (с кэшированием)
func (s *ProtocolService) GetGroupSummary(ctx context.Context, groupID string) (models.GroupSummary, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetGroupSummary"),
		zap.String("group_id", groupID),
	)
	log.Info("generating group summary report")
	start := time.Now()
	defer func() {
		log.Debug("summary generation completed", zap.Duration("duration_ms", time.Since(start)))
	}()

	// 1. Получаем группу
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		log.Error("failed to fetch group", zap.Error(err))
		return models.GroupSummary{}, fmt.Errorf("failed to get group: %w", err)
	}
	if group.ID == "" {
		log.Warn("group not found", zap.String("searched_id", groupID))
		return models.GroupSummary{}, fmt.Errorf("group not found")
	}

	// 2. Получаем протоколы
	protocols, err := s.protocolRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		log.Error("failed to fetch protocols for group", zap.Error(err))
		return models.GroupSummary{}, fmt.Errorf("failed to get protocols: %w", err)
	}

	if len(protocols) == 0 {
		log.Info("no protocols found for group, returning empty summary")
		return models.GroupSummary{
			GroupID:       group.ID,
			GroupName:     group.Name,
			MaterialID:    group.MaterialID,
			TotalSamples:  0,
			CompliantRate: 100.0,
			Results:       []models.MethodResultSummary{},
		}, nil
	}

	// 3. ПРЕДЗАГРУЗКА проб для быстрого маппинга
	samples, err := s.sampleRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		log.Warn("failed to load samples for summary, continuing without sample numbers", zap.Error(err))
	}
	sampleMap := make(map[string]string, len(samples))
	for _, samp := range samples {
		sampleMap[samp.ID] = samp.SampleNumber
	}
	log.Debug("samples preloaded for mapping", zap.Int("samples_count", len(sampleMap)))

	// 4. Агрегация результатов
	methodMap := make(map[string]*models.MethodResultSummary)
	totalTests := 0
	compliantTests := 0
	methodCache := make(map[string]models.TestMethod)

	for _, proto := range protocols {
		results, err := s.protocolRepo.GetResultsByProtocolID(ctx, proto.ID)
		if err != nil {
			log.Warn("failed to load results for protocol",
				zap.String("protocol_id", proto.ID),
				zap.Error(err))
			continue
		}

		for _, res := range results {
			// Кэш методов
			method, exists := methodCache[res.MethodID]
			if !exists {
				method, err = s.standardRepo.GetTestMethod(ctx, res.MethodID)
				if err != nil {
					log.Warn("failed to load method definition",
						zap.String("method_id", res.MethodID),
						zap.Error(err))
					continue
				}
				methodCache[res.MethodID] = method
			}

			// Инициализация сводки
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

			// Формирование испытания
			isComp := false
			if res.IsCompliant != nil {
				isComp = *res.IsCompliant
			}

			trial := models.MethodTrial{
				ProtocolID:   proto.ID,
				SampleNumber: sampleMap[proto.SampleID],
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

	// Преобразование в слайс
	summaries := make([]models.MethodResultSummary, 0, len(methodMap))
	for _, v := range methodMap {
		summaries = append(summaries, *v)
	}

	// Расчёт процента
	rate := 100.0
	if totalTests > 0 {
		rate = float64(compliantTests) / float64(totalTests) * 100.0
	}

	log.Info("group summary generated successfully",
		zap.Int("protocols_processed", len(protocols)),
		zap.Int("methods_aggregated", len(summaries)),
		zap.Int("total_tests", totalTests),
		zap.Float64("compliant_rate", rate))

	return models.GroupSummary{
		GroupID:       group.ID,
		GroupName:     group.Name,
		MaterialID:    group.MaterialID,
		TotalSamples:  len(protocols),
		CompliantRate: rate,
		Results:       summaries,
	}, nil
}

// generateProtocolNumber - генерация номера (вспомогательная, без логирования)
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

// UpdateProtocol - обновление черновика
func (s *ProtocolService) UpdateProtocol(ctx context.Context, id string, req models.UpdateProtocolRequest) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "UpdateProtocol"),
		zap.String("protocol_id", id),
	)
	log.Debug("updating protocol")

	prot, err := s.protocolRepo.GetByID(ctx, id)
	if err != nil {
		log.Error("failed to fetch protocol for update", zap.Error(err))
		return err
	}

	if prot.Status != "draft" {
		log.Warn("cannot update protocol - not in draft status", zap.String("current_status", prot.Status))
		return fmt.Errorf("cannot update protocol with status %s", prot.Status)
	}

	if err := s.protocolRepo.UpdateProtocol(ctx, id, req); err != nil {
		log.Error("failed to update protocol in repo", zap.Error(err))
		return err
	}

	log.Info("protocol updated successfully")
	return nil
}

// UpdateProtocolStatus - смена статуса
func (s *ProtocolService) UpdateProtocolStatus(ctx context.Context, id string, status string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "UpdateProtocolStatus"),
		zap.String("protocol_id", id),
		zap.String("new_status", status),
	)
	log.Info("updating protocol status")

	if err := s.protocolRepo.UpdateStatus(ctx, id, status); err != nil {
		log.Error("failed to update protocol status in repo", zap.Error(err))
		return err
	}

	log.Info("protocol status updated successfully")
	return nil
}

// DeleteProtocol - удаление
func (s *ProtocolService) DeleteProtocol(ctx context.Context, id string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "DeleteProtocol"),
		zap.String("protocol_id", id),
	)
	log.Info("deleting protocol")

	if err := s.protocolRepo.DeleteProtocol(ctx, id); err != nil {
		log.Error("failed to delete protocol from repo", zap.Error(err))
		return err
	}

	log.Info("protocol deleted successfully")
	return nil
}
