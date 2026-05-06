package service

import (
	"bytes"
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"go.uber.org/zap"
)

// ReportService отвечает за генерацию печатных форм (PDF)
type ReportService struct {
	protocolService    *ProtocolService
	materialService    *MaterialService // ✅ Используем сервис, а не интерфейс репозитория
	templatesDir       string           // Оставляем для кастомных путей, если нужно
	fontDir            string
	wkhtmltopdfWindows []byte
	log                *zap.Logger
}

// NewReportService создает сервис отчетов
func NewReportService(
	protoSvc *ProtocolService,
	matSvc *MaterialService, // ✅ Inject service
	// fontDir можно убрать, если шрифты тоже в embed, или оставить для wkhtmltopdf
	fontDir string,
	templatesDir string,
	wkhtmltopdfWindows []byte,
	log *zap.Logger,
) *ReportService {
	return &ReportService{
		protocolService:    protoSvc,
		materialService:    matSvc,
		templatesDir:       templatesDir,
		fontDir:            fontDir,
		wkhtmltopdfWindows: wkhtmltopdfWindows,
		log:                log,
	}
}

// GenerateProtocolPDF генерирует PDF для конкретного протокола (ОПТИМИЗИРОВАНО)
func (s *ReportService) GenerateProtocolPDF(ctx context.Context, protocolID string) ([]byte, error) {
	s.log.Info("Generating PDF for protocol", zap.String("id", protocolID))

	// 🔥 1. ОДИН ЗАПРОС вместо четырёх
	// Получаем протокол, пробу, материал и результаты сразу
	// Это возможно благодаря методу GetProtocolFull, который мы реализовали ранее
	protocolFull, err := s.protocolService.GetProtocolFull(ctx, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol data: %w", err)
	}
	if protocolFull.IsEmpty() {
		return nil, fmt.Errorf("protocol %s not found", protocolID)
	}

	// 🔥 2. Предзагрузка методов для результатов (Batch Load)
	// Чтобы избежать N+1 запросов в цикле prepareProtocolTemplateData,
	// загружаем все методы, которые встречаются в результатах, одним махом.
	// Если StandardID неизвестен, можно загрузить по списку ID методов.

	// Собираем уникальные MethodID из результатов
	methodIDs := make(map[string]bool)
	for _, res := range protocolFull.Results {
		methodIDs[res.MethodID] = true
	}

	// Если у нас есть доступ к стандарту (он есть в протоколе через пробу),
	// можно загрузить методы стандарта целиком (как мы делали в ProtocolService).
	// Но для универсальности загрузим только нужные методы.
	// Примечание: В идеале нужен метод standardRepo.GetMethodsByIDs(ctx, []string)
	// Пока реализуем кэширование внутри prepare... или загрузим стандарт полностью.

	// Для простоты и максимальной скорости: если все результаты одного стандарта,
	// загружаем контекст стандарта целиком.
	standardID := ""
	if len(protocolFull.Results) > 0 {
		// Быстрый хак: получаем StandardID по первому методу (один легкий запрос)
		// В продакшене лучше передавать StandardID в ответе GetProtocolFull
		firstMethod, _ := s.protocolService.standardRepo.GetTestMethod(ctx, protocolFull.Results[0].MethodID)
		standardID = firstMethod.StandardID
	}

	var methodsCache map[string]models.TestMethodFull
	if standardID != "" {
		methodsCache, _ = s.protocolService.standardRepo.GetMethodsFullByStandardID(ctx, standardID)
		// Если ошибка - просто проигнорируем и будем грузить по одному (fallback)
	}

	// 3. Преобразуем данные в формат для шаблона
	// Передаем кэш методов, чтобы функция не делала лишние запросы
	templateData, err := s.prepareProtocolTemplateData(ctx, protocolFull, methodsCache)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare template data: %w", err)
	}

	// 4. Рендерим HTML (используем embed FS)
	htmlContent, err := s.renderHTML("protocol_template.html", templateData)
	if err != nil {
		return nil, err
	}

	// 5. Генерируем PDF
	return s.generatePDFFromHTML(htmlContent)
}

type GroupSummaryTemplateData struct {
	Group         GroupView
	Statistics    StatisticsView
	MethodResults []MethodSummaryView
	FormattedDate string
	QRCodeData    string
	LabName       string // Опционально: можно передать из контекста
	GeneratedAt   string
}

type GroupView struct {
	ID             string
	Name           string
	MaterialName   string
	MaterialCode   string
	Project        string
	Location       string
	CreatedAt      string
	TotalProtocols int
}

type StatisticsView struct {
	CompliantRate    float64 // 0.0–1.0
	CompliantPercent string  // "85.3%"
	TotalSamples     int
	TotalTests       int
	CompliantTests   int
}

type MethodSummaryView struct {
	MethodID     string
	MethodName   string
	Unit         string
	AverageValue float64
	MinValue     float64
	MaxValue     float64
	NormDisplay  string
	IsCompliant  bool
	TrialsCount  int
	Protocols    []ProtocolTrialView
}

type ProtocolTrialView struct {
	ProtocolNumber string
	SampleNumber   string
	TestDate       string
	Value          float64
	IsCompliant    bool
	Deviation      string
	LabName        string
}

// GenerateGroupSummaryPDF генерирует сводный PDF-отчёт по группе испытаний
func (s *ReportService) GenerateGroupSummaryPDF(ctx context.Context, groupID string) ([]byte, error) {
	s.log.Info("Generating group summary PDF", zap.String("group_id", groupID))

	// 1. Получаем базовую информацию о группе
	group, err := s.protocolService.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group.ID == "" {
		return nil, fmt.Errorf("group %s not found", groupID)
	}

	// 2. Получаем материал для отображения
	material, err := s.materialService.GetByID(ctx, group.MaterialID)
	if err != nil {
		s.log.Warn("failed to load material for group report", zap.Error(err))
		material = models.Material{ID: group.MaterialID, Name: "Неизвестный материал"}
	}

	// 3. Получаем сводную статистику
	summary, err := s.protocolService.GetGroupSummary(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group summary: %w", err)
	}

	// 4. 🔥 КЭШИРОВАНИЕ МЕТОДОВ через GetMethodsFullByStandardID
	methodsCache := make(map[string]models.TestMethodFull)

	// Получаем StandardID по первому результату (если есть)
	standardID := ""
	if len(summary.Results) > 0 {
		firstMethod, err := s.protocolService.standardRepo.GetTestMethod(ctx, summary.Results[0].MethodID)
		if err == nil && firstMethod.StandardID != "" {
			standardID = firstMethod.StandardID
		}
	}

	// Загружаем ВСЕ методы стандарта одним запросом (оптимизация)
	if standardID != "" {
		cache, err := s.protocolService.standardRepo.GetMethodsFullByStandardID(ctx, standardID)
		if err != nil {
			s.log.Warn("failed to preload methods for group report", zap.Error(err))
		} else {
			methodsCache = cache
		}
	}

	// 5. Получаем протоколы группы для детализации испытаний
	protocols, err := s.protocolService.protocolRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		s.log.Warn("failed to load protocols for group report", zap.Error(err))
		protocols = []models.Protocol{}
	}

	// 6. Подготавливаем данные для шаблона
	templateData, err := s.prepareGroupSummaryTemplateData(ctx, group, material, summary, methodsCache, protocols)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare group template  %w", err)
	}

	// 7. Рендерим HTML
	htmlContent, err := s.renderHTML("group_summary_template.html", templateData)
	if err != nil {
		return nil, err
	}

	// 8. Генерируем PDF
	return s.generatePDFFromHTML(htmlContent)
}

// prepareGroupSummaryTemplateData подготавливает данные для шаблона сводки группы
func (s *ReportService) prepareGroupSummaryTemplateData(
	ctx context.Context,
	group models.ExperimentGroup,
	material models.Material,
	summary models.GroupSummary,
	methodsCache map[string]models.TestMethodFull,
	protocols []models.Protocol,
) (GroupSummaryTemplateData, error) {
	// Форматирование даты
	createdAt := group.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	dateStr := createdAt.Format("02.01.2006")
	formattedFullDate := createdAt.Format("02.01.2006 15:04")

	// 🔹 Создаем карту только завершенных протоколов
	completedProtocolsMap := make(map[string]models.Protocol)
	for _, p := range protocols {
		if p.Status == "completed" {
			completedProtocolsMap[p.ID] = p
		}
	}

	data := GroupSummaryTemplateData{
		Group: GroupView{
			ID:             group.ID,
			Name:           group.Name,
			MaterialName:   material.Name,
			MaterialCode:   material.Code,
			Project:        group.ProjectName,
			Location:       group.Location,
			CreatedAt:      dateStr,
			TotalProtocols: len(completedProtocolsMap), // Показываем только завершенные
		},
		FormattedDate: formattedFullDate,
		QRCodeData:    group.ID,
		GeneratedAt:   time.Now().Format("02.01.2006 15:04"),
	}

	// Временный срез для накопления методов
	tempMethodResults := make([]MethodSummaryView, 0)

	// Переменные для общей статистики (только по методам, которые войдут в отчет)
	totalTests := 0
	compliantTests := 0

	// Обработка результатов по методам
	for _, res := range summary.Results {
		methodFull, exists := methodsCache[res.MethodID]
		if !exists {
			methodBasic, err := s.protocolService.standardRepo.GetTestMethod(ctx, res.MethodID)
			if err != nil {
				s.log.Warn("failed to load method for group report", zap.String("method_id", res.MethodID), zap.Error(err))
				continue
			}
			methodFull = models.TestMethodFull{
				Method: methodBasic,
				Inputs: []models.MethodInput{},
				Limits: []models.NormativeLimit{},
			}
		}
		method := methodFull.Method
		var normStr string = "—"

		if len(methodFull.Limits) > 0 {
			// Простая логика: берем первый лимит, у которого нет условий, или просто первый, если условий нет ни у кого
			// В идеале здесь нужна та же логика findMatchingLimit, но с пустым или усредненным контекстом.
			// Для простоты возьмем первый лимит, считая его основным для метода.
			limit := methodFull.Limits[0]

			if limit.LimitType == "range" && limit.MinValue != nil && limit.MaxValue != nil {
				normStr = fmt.Sprintf("%.2f – %.2f", *limit.MinValue, *limit.MaxValue)
			} else if limit.LimitType == "min" && limit.MinValue != nil {
				normStr = fmt.Sprintf("≥ %.2f", *limit.MinValue)
			} else if limit.LimitType == "max" && limit.MaxValue != nil {
				normStr = fmt.Sprintf("≤ %.2f", *limit.MaxValue)
			} else if len(limit.DiscreteValues) > 0 {
				normStr = strings.Join(limit.DiscreteValues, ", ")
			}
		}

		// Собираем детализацию ТОЛЬКО по завершенным протоколам
		protocolTrials := make([]ProtocolTrialView, 0)
		values := make([]float64, 0)

		for _, trial := range res.Trials {
			// 🔹 ГЛАВНЫЙ ФИЛЬТР: Пропускаем черновики
			proto, exists := completedProtocolsMap[trial.ProtocolID]
			if !exists {
				continue
			}

			// Добавляем значение для статистики метода
			if !math.IsNaN(trial.Value) && !math.IsInf(trial.Value, 0) {
				values = append(values, trial.Value)
			}

			// Формируем строку для таблицы
			testDate := proto.TestDate
			if testDate == nil || testDate.IsZero() {
				testDate = &proto.CreatedAt
			}

			deviationStr := ""
			if trial.Deviation != nil {
				deviationStr = *trial.Deviation
			}

			protocolTrials = append(protocolTrials, ProtocolTrialView{
				ProtocolNumber: proto.ProtocolNumber,
				SampleNumber:   trial.SampleNumber,
				TestDate:       testDate.Format("02.01.2006"),
				Value:          math.Round(trial.Value*100) / 100,
				IsCompliant:    trial.IsCompliant,
				Deviation:      deviationStr,
				LabName:        proto.LabName,
			})
		}

		// 🔹 ИСКЛЮЧЕНИЕ МЕТОДА: Если нет завершенных протоколов, пропускаем этот метод полностью
		if len(protocolTrials) == 0 {
			continue
		}

		// Расчет статистики для этого метода
		var avg, min, max float64

		if len(values) > 0 {
			sum := 0.0
			min = values[0]
			max = values[0]
			for _, v := range values {
				sum += v
				if v < min {
					min = v
				}
				if v > max {
					max = v
				}
			}
			avg = sum / float64(len(values))
		}

		// Обновляем общую статистику (теперь мы уверены, что метод имеет данные)
		for _, trial := range protocolTrials {
			totalTests++
			if trial.IsCompliant {
				compliantTests++
			}
		}

		methodView := MethodSummaryView{
			MethodID:     res.MethodID,
			MethodName:   method.Name,
			Unit:         method.Unit,
			AverageValue: math.Round(avg*100) / 100,
			MinValue:     math.Round(min*100) / 100,
			MaxValue:     math.Round(max*100) / 100,
			NormDisplay:  normStr,
			IsCompliant:  res.IsCompliant, // Статус соответствия метода (из сервиса)
			TrialsCount:  len(protocolTrials),
			Protocols:    protocolTrials,
		}
		tempMethodResults = append(tempMethodResults, methodView)
	}

	// Записываем отфильтрованные результаты
	data.MethodResults = tempMethodResults

	// Записываем пересчитанную статистику
	data.Statistics = StatisticsView{
		CompliantRate:    summary.CompliantRate, // Берем из сервиса (там логика только по completed)
		CompliantPercent: fmt.Sprintf("%.1f%%", summary.CompliantRate*100),
		TotalSamples:     len(completedProtocolsMap),
		TotalTests:       totalTests,
		CompliantTests:   compliantTests,
	}

	return data, nil
}

// ============================================================================
// СТРУКТУРЫ ДАННЫХ ДЛЯ ШАБЛОНА (VIEW MODELS)
// ============================================================================

type ProtocolTemplateData struct {
	Protocol      ProtocolView
	Sample        SampleView
	Material      MaterialView
	Results       []ResultRowView
	FormattedDate string
	QRCodeData    string
	FontPath      string // Можно убрать, если шрифты в base64 или embed
	LabName       string
	Operator      string
	Project       string
}

type ProtocolView struct {
	Number   string
	Date     string
	ID       string
	LabName  string
	Operator string
	Project  string
}

type SampleView struct {
	Number          string
	CollectionPlace string
	Note            string
	MaterialName    string
}

type MaterialView struct {
	Name string
	Code string
}

type ResultRowView struct {
	MethodName string
	Value      float64
	Unit       string
	Norm       string // "≥ 10.5" или "—"
	Compliance string // "Соответствует" / "Не соответствует"
	Deviation  string
	RawInputs  map[string]interface{}
}

// ============================================================================
// ВНУТРЕННЯЯ ЛОГИКА
// ============================================================================

// prepareProtocolTemplateData - ОПТИМИЗИРОВАННАЯ ВЕРСИЯ
// Принимает ProtocolFull и кэш методов, чтобы избежать запросов в цикле
func (s *ReportService) prepareProtocolTemplateData(
	ctx context.Context,
	full models.ProtocolFull,
	methodsCache map[string]models.TestMethodFull, // 🔥 Кэш для ускорения
) (ProtocolTemplateData, error) {
	// Форматирование даты
	var reportTime time.Time

	if full.Protocol.TestDate != nil && !full.Protocol.TestDate.IsZero() {
		reportTime = full.Protocol.TestDate.Local()
	} else if !full.Protocol.CreatedAt.IsZero() {
		reportTime = full.Protocol.CreatedAt.Local()
	} else {
		reportTime = time.Now()
	}

	dateStr := reportTime.Format("02.01.2006")
	formattedFullDate := reportTime.Format("02.01.2006 15:04")

	data := ProtocolTemplateData{
		Protocol: ProtocolView{
			Number:   full.Protocol.ProtocolNumber,
			Date:     dateStr,
			ID:       full.Protocol.ID,
			LabName:  full.Protocol.LabName,
			Operator: full.Protocol.OperatorName,
			Project:  full.Sample.GroupID,
		},
		Sample: SampleView{
			Number:          full.Sample.SampleNumber,
			CollectionPlace: full.Sample.CollectionPlace,
			Note:            full.Sample.Note,
			MaterialName:    full.Material.Name, // ✅ Уже загружено в full.Material
		},
		Material: MaterialView{
			Name: full.Material.Name,
			Code: full.Material.Code,
		},
		LabName:       full.Protocol.LabName,
		Operator:      full.Protocol.OperatorName,
		FormattedDate: formattedFullDate,
		QRCodeData:    full.Protocol.ID,
		// FontPath можно передать, если wkhtmltopdf требует локальный путь
	}

	// ОБРАБОТКА РЕЗУЛЬТАТОВ
	for _, res := range full.Results {
		var method models.TestMethod
		var applicableLimit models.NormativeLimit

		// Пытаемся взять метод из предзагруженного кэша
		if methodsCache != nil {
			if fullMethod, ok := methodsCache[res.MethodID]; ok {
				method = fullMethod.Method
				// Ищем применимый лимит в памяти (быстро)
				// Используем ту же логику, что и в ProtocolService.findMatchingLimit
				applicableLimit = s.findMatchingLimitInMemory(fullMethod.Limits, fullMethod.LimitConditions, full.Sample.ContextParams)
			}
		}

		// Fallback: если кэш не сработал (или пуст), грузим из БД (медленно, но надежно)
		if method.ID == "" {
			var err error
			method, err = s.protocolService.standardRepo.GetTestMethod(ctx, res.MethodID)
			if err != nil {
				s.log.Warn("failed to load method for report", zap.String("method_id", res.MethodID), zap.Error(err))
				continue // Пропускаем этот результат, чтобы не ломать весь отчет
			}
			// Для лимитов в фоллбэке тоже делаем запрос
			applicableLimit, _ = s.protocolService.standardRepo.GetApplicableLimit(ctx, res.MethodID, full.Sample.ContextParams)
		}

		// Формирование строки нормы
		normStr := "—"
		if applicableLimit.ID != "" {
			if applicableLimit.MinValue != nil && applicableLimit.MaxValue != nil {
				normStr = fmt.Sprintf("%.2f – %.2f", *applicableLimit.MinValue, *applicableLimit.MaxValue)
			} else if applicableLimit.MinValue != nil {
				normStr = fmt.Sprintf("≥ %.2f", *applicableLimit.MinValue)
			} else if applicableLimit.MaxValue != nil {
				normStr = fmt.Sprintf("≤ %.2f", *applicableLimit.MaxValue)
			}
		}

		// Статус соответствия
		// ВАЖНО: Используем статус, который УЖЕ рассчитан и сохранен в БД (res.IsCompliant)
		// Не нужно пересчитывать его заново!
		complianceStr := "Соответствует"
		if res.IsCompliant != nil && !*res.IsCompliant {
			complianceStr = "Не соответствует"
		}

		// Значение
		val := 0.0
		if res.CalculatedValue != nil {
			val = *res.CalculatedValue
		}

		row := ResultRowView{
			MethodName: method.Name,
			Value:      math.Round(val*100) / 100,
			Unit:       method.Unit,
			Norm:       normStr,
			Compliance: complianceStr,
			Deviation:  res.DeviationMsg, // Берем сохраненное сообщение об ошибке
			RawInputs:  res.InputData,
		}
		data.Results = append(data.Results, row)
	}

	return data, nil
}

// findMatchingLimitInMemory - вспомогательная функция для поиска лимита в кэше
// (Дублирует логику из ProtocolService, можно вынести в утилиты)
func (s *ReportService) findMatchingLimitInMemory(
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
			if cond.ConditionOperator == "=" && actualVal != cond.ExpectedValue {
				match = false
				break
			}
			if cond.ConditionOperator == "!=" && actualVal == cond.ExpectedValue {
				match = false
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

// renderHTML - ОПТИМИЗИРОВАННАЯ ВЕРСИЯ С EMBED
func (s *ReportService) renderHTML(templateName string, data interface{}) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Конструируем полный путь
	tmplPath := filepath.Join(execDir, "templates", "protocols", templateName)

	// Проверка существования файла
	if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
		// Попытка найти в текущей рабочей директории (для режима разработки go run)
		tmplPath = filepath.Join("templates", "protocols", templateName)
		if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
			return "", fmt.Errorf("template file not found at %s or %s",
				filepath.Join(execDir, "templates", "protocols", templateName),
				filepath.Join("templates", "protocols", templateName))
		}
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generatePDFFromHTML (без изменений, логика wkhtmltopdf)
func (s *ReportService) generatePDFFromHTML(htmlContent string) ([]byte, error) {
	wkPath, err := GetWkhtmltopdfPath(s.wkhtmltopdfWindows)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare wkhtmltopdf: %w", err)
	}
	tmpFile, err := os.CreateTemp("", "protocol_*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tmpFile.Name()
	defer os.Remove(tempPath)

	if _, err := tmpFile.Write([]byte(htmlContent)); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	wkhtmltopdf.SetPath(wkPath)

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to init wkhtmltopdf: %w", err)
	}

	page := wkhtmltopdf.NewPage(tempPath)
	page.EnableLocalFileAccess.Set(true)
	page.Encoding.Set("UTF-8")

	pdfg.AddPage(page)
	pdfg.Dpi.Set(300)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.MarginTop.Set(10)
	pdfg.MarginBottom.Set(10)
	pdfg.MarginLeft.Set(10)
	pdfg.MarginRight.Set(10)

	if err := pdfg.Create(); err != nil {
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	return pdfg.Bytes(), nil
}

func GetWkhtmltopdfPath(wkhtmltopdfWindows []byte) (string, error) {
	var binary []byte
	var filename string

	binary = wkhtmltopdfWindows
	filename = "wkhtmltopdf.exe"

	// Проверяем, уже ли извлечён файл
	tempDir := os.TempDir()
	binaryPath := filepath.Join(tempDir, "desktop_lab_wkhtmltopdf", filename)

	if _, err := os.Stat(binaryPath); err == nil {
		// Файл существует — проверяем, что он исполняемый
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return "", fmt.Errorf("failed to chmod binary: %w", err)
		}
		return binaryPath, nil
	}

	// Создаём директорию
	dir := filepath.Dir(binaryPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Записываем бинарник
	if err := os.WriteFile(binaryPath, binary, 0755); err != nil {
		return "", fmt.Errorf("failed to write wkhtmltopdf binary: %w", err)
	}

	return binaryPath, nil
}
