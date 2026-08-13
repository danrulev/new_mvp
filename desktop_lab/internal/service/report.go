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

// GenerateProtocolPDF генерирует PDF для конкретного протокола
func (s *ReportService) GenerateProtocolPDF(ctx context.Context, protocolID string) ([]byte, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GenerateProtocolPDF"),
		zap.String("protocol_id", protocolID),
	)
	log.Info("starting PDF generation for protocol")
	start := time.Now()
	defer func() {
		log.Info("PDF generation completed", zap.Duration("duration_ms", time.Since(start)))
	}()

	// 1. Получаем полные данные протокола
	log.Debug("fetching full protocol data")
	protocolFull, err := s.protocolService.GetProtocolFull(ctx, protocolID)
	if err != nil {
		log.Error("failed to fetch protocol data", zap.Error(err))
		return nil, fmt.Errorf("failed to get protocol data: %w", err)
	}
	if protocolFull.IsEmpty() {
		log.Warn("protocol not found", zap.String("searched_id", protocolID))
		return nil, fmt.Errorf("protocol %s not found", protocolID)
	}
	log.Debug("protocol data fetched successfully",
		zap.Int("results_count", len(protocolFull.Results)),
		zap.String("material_name", protocolFull.Material.Name))

	// 2. Предзагрузка методов для оптимизации
	standardID := ""
	if len(protocolFull.Results) > 0 {
		firstMethodID := protocolFull.Results[0].MethodID
		log.Debug("determining standard for methods cache", zap.String("first_method_id", firstMethodID))

		firstMethod, err := s.protocolService.standardRepo.GetTestMethod(ctx, firstMethodID)
		if err != nil {
			log.Warn("failed to determine standard, falling back to individual queries",
				zap.Error(err), zap.String("method_id", firstMethodID))
		} else {
			standardID = firstMethod.StandardID
		}
	}

	var methodsCache map[string]models.TestMethodFull
	if standardID != "" {
		log.Debug("preloading methods cache for standard", zap.String("standard_id", standardID))
		cacheStart := time.Now()
		methodsCache, err = s.protocolService.standardRepo.GetMethodsFullByStandardID(ctx, standardID)
		if err != nil {
			log.Warn("failed to preload methods cache", zap.Error(err), zap.Duration("cache_load_ms", time.Since(cacheStart)))
		} else {
			log.Info("methods cache loaded successfully",
				zap.Int("methods_count", len(methodsCache)),
				zap.Duration("cache_load_ms", time.Since(cacheStart)))
		}
	}

	// 3. Подготовка данных шаблона
	log.Debug("preparing template data")
	templateStart := time.Now()
	templateData, err := s.prepareProtocolTemplateData(ctx, protocolFull, methodsCache)
	if err != nil {
		log.Error("failed to prepare template data", zap.Error(err))
		return nil, fmt.Errorf("failed to prepare template data: %w", err)
	}
	log.Debug("template data prepared", zap.Duration("prep_duration_ms", time.Since(templateStart)))

	// 4. Рендеринг HTML
	log.Debug("rendering HTML template")
	htmlStart := time.Now()
	htmlContent, err := s.renderHTML("protocol_template.html", templateData)
	if err != nil {
		log.Error("failed to render HTML template", zap.Error(err))
		return nil, err
	}
	log.Debug("HTML rendered successfully",
		zap.Int("html_size_bytes", len(htmlContent)),
		zap.Duration("render_duration_ms", time.Since(htmlStart)))

	// 5. Генерация PDF
	log.Debug("generating PDF from HTML")
	pdfStart := time.Now()
	pdfBytes, err := s.generatePDFFromHTML(htmlContent)
	if err != nil {
		log.Error("failed to generate PDF", zap.Error(err))
		return nil, err
	}
	log.Info("PDF generated successfully",
		zap.Int("pdf_size_bytes", len(pdfBytes)),
		zap.Duration("pdf_generation_ms", time.Since(pdfStart)))

	return pdfBytes, nil
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

// GenerateGroupSummaryPDF генерирует сводный PDF-отчёт по группе
func (s *ReportService) GenerateGroupSummaryPDF(ctx context.Context, groupID string) ([]byte, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GenerateGroupSummaryPDF"),
		zap.String("group_id", groupID),
	)
	log.Info("starting group summary PDF generation")
	start := time.Now()
	defer func() {
		log.Info("group summary PDF generation completed", zap.Duration("total_duration_ms", time.Since(start)))
	}()

	// 1. Получаем информацию о группе
	log.Debug("fetching group data")
	group, err := s.protocolService.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		log.Error("failed to fetch group", zap.Error(err))
		return nil, fmt.Errorf("failed to get group: %w", err)
	}
	if group.ID == "" {
		log.Warn("group not found", zap.String("searched_id", groupID))
		return nil, fmt.Errorf("group %s not found", groupID)
	}
	log.Debug("group data fetched", zap.String("group_name", group.Name))

	// 2. Получаем материал
	log.Debug("fetching material data")
	material, err := s.materialService.GetByID(ctx, group.MaterialID)
	if err != nil {
		log.Warn("failed to load material, using fallback", zap.Error(err), zap.String("material_id", group.MaterialID))
		material = models.Material{ID: group.MaterialID, Name: "Неизвестный материал"}
	}

	// 3. Получаем сводную статистику
	log.Debug("fetching group summary statistics")
	summary, err := s.protocolService.GetGroupSummary(ctx, groupID)
	if err != nil {
		log.Error("failed to fetch group summary", zap.Error(err))
		return nil, fmt.Errorf("failed to get group summary: %w", err)
	}
	log.Debug("summary fetched",
		zap.Int("methods_count", len(summary.Results)),
		zap.Float64("compliant_rate", summary.CompliantRate))

	// 4. Кэширование методов
	standardID := ""
	if len(summary.Results) > 0 {
		firstMethod, err := s.protocolService.standardRepo.GetTestMethod(ctx, summary.Results[0].MethodID)
		if err == nil && firstMethod.StandardID != "" {
			standardID = firstMethod.StandardID
		}
	}

	methodsCache := make(map[string]models.TestMethodFull)
	if standardID != "" {
		log.Debug("preloading methods cache for group report", zap.String("standard_id", standardID))
		cache, err := s.protocolService.standardRepo.GetMethodsFullByStandardID(ctx, standardID)
		if err != nil {
			log.Warn("failed to preload methods cache", zap.Error(err))
		} else {
			methodsCache = cache
			log.Debug("methods cache loaded", zap.Int("cached_methods", len(cache)))
		}
	}

	// 5. Получаем протоколы группы
	log.Debug("fetching protocols for group")
	protocols, err := s.protocolService.protocolRepo.GetByGroupID(ctx, groupID)
	if err != nil {
		log.Warn("failed to load protocols, continuing with empty list", zap.Error(err))
		protocols = []models.Protocol{}
	} else {
		log.Debug("protocols loaded", zap.Int("count", len(protocols)))
	}

	// 6. Подготовка данных шаблона
	log.Debug("preparing group summary template data")
	templateData, err := s.prepareGroupSummaryTemplateData(ctx, group, material, summary, methodsCache, protocols)
	if err != nil {
		log.Error("failed to prepare group template data", zap.Error(err))
		return nil, fmt.Errorf("failed to prepare group template data: %w", err)
	}

	// 7. Рендеринг HTML
	log.Debug("rendering group summary HTML")
	htmlContent, err := s.renderHTML("group_summary_template.html", templateData)
	if err != nil {
		log.Error("failed to render group summary HTML", zap.Error(err))
		return nil, err
	}

	// 8. Генерация PDF
	log.Debug("generating PDF from HTML")
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
	createdAt := group.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	dateStr := createdAt.Format("02.01.2006")
	formattedFullDate := createdAt.Format("02.01.2006 15:04")

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
			TotalProtocols: len(completedProtocolsMap),
		},
		FormattedDate: formattedFullDate,
		QRCodeData:    group.ID,
		GeneratedAt:   time.Now().Format("02.01.2006 15:04"),
	}

	tempMethodResults := make([]MethodSummaryView, 0)
	totalTests := 0
	compliantTests := 0

	for _, res := range summary.Results {
		methodFull, exists := methodsCache[res.MethodID]
		if !exists {
			methodBasic, err := s.protocolService.standardRepo.GetTestMethod(ctx, res.MethodID)
			if err != nil {
				s.log.Warn("failed to load method for group report",
					zap.String("method_id", res.MethodID),
					zap.Error(err))
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
			limit := methodFull.Limits[0]
			switch limit.LimitType {
			case "range":
				if limit.MinValue != nil && limit.MaxValue != nil {
					normStr = fmt.Sprintf("%.2f – %.2f", *limit.MinValue, *limit.MaxValue)
				}
			case "min":
				if limit.MinValue != nil {
					normStr = fmt.Sprintf("≥ %.2f", *limit.MinValue)
				}
			case "max":
				if limit.MaxValue != nil {
					normStr = fmt.Sprintf("≤ %.2f", *limit.MaxValue)
				}
			}
			if len(limit.DiscreteValues) > 0 && normStr == "—" {
				normStr = strings.Join(limit.DiscreteValues, ", ")
			}
		}

		protocolTrials := make([]ProtocolTrialView, 0)
		values := make([]float64, 0)

		for _, trial := range res.Trials {
			proto, exists := completedProtocolsMap[trial.ProtocolID]
			if !exists {
				continue
			}

			if !math.IsNaN(trial.Value) && !math.IsInf(trial.Value, 0) {
				values = append(values, trial.Value)
			}

			testDate := proto.TestDate
			if testDate.IsZero() {
				testDate = proto.CreatedAt
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

		if len(protocolTrials) == 0 {
			continue
		}

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
			IsCompliant:  res.IsCompliant,
			TrialsCount:  len(protocolTrials),
			Protocols:    protocolTrials,
		}
		tempMethodResults = append(tempMethodResults, methodView)
	}

	data.MethodResults = tempMethodResults
	data.Statistics = StatisticsView{
		CompliantRate:    summary.CompliantRate,
		CompliantPercent: fmt.Sprintf("%.1f%%", summary.CompliantRate),
		TotalSamples:     len(completedProtocolsMap),
		TotalTests:       totalTests,
		CompliantTests:   compliantTests,
	}

	return data, nil
}

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

	// Расширенные поля образца
	PhotoURL     string
	LengthMM     float64
	WidthMM      float64
	HeightMM     float64
	Shape        string
	WeightGrams  float64
	Color        string
	BatchNumber  string
	Manufacturer string
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

// prepareProtocolTemplateData - оптимизированная версия с кэшированием
func (s *ReportService) prepareProtocolTemplateData(
	ctx context.Context,
	full models.ProtocolFull,
	methodsCache map[string]models.TestMethodFull,
) (ProtocolTemplateData, error) {
	var reportTime time.Time
	if !full.Protocol.TestDate.IsZero() {
		reportTime = full.Protocol.TestDate.Local()
	} else if !full.Protocol.CreatedAt.IsZero() {
		reportTime = full.Protocol.CreatedAt.Local()
	} else {
		reportTime = time.Now()
	}

	dateStr := reportTime.Format("02.01.2006")
	formattedFullDate := reportTime.Format("02.01.2006 15:04")

	var lengthMM, widthMM, heightMM, weightGrams float64
	if full.Sample.LengthMM != nil {
		lengthMM = *full.Sample.LengthMM
	}
	if full.Sample.WidthMM != nil {
		widthMM = *full.Sample.WidthMM
	}
	if full.Sample.HeightMM != nil {
		heightMM = *full.Sample.HeightMM
	}
	if full.Sample.WeightGrams != nil {
		weightGrams = *full.Sample.WeightGrams
	}

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
			MaterialName:    full.Material.Name,
			// Расширенные поля образца
			PhotoURL:     full.Sample.PhotoURL,
			LengthMM:     lengthMM,
			WidthMM:      widthMM,
			HeightMM:     heightMM,
			Shape:        full.Sample.Shape,
			WeightGrams:  weightGrams,
			Color:        full.Sample.Color,
			BatchNumber:  full.Sample.BatchNumber,
			Manufacturer: full.Sample.Manufacturer,
		},
		Material: MaterialView{
			Name: full.Material.Name,
			Code: full.Material.Code,
		},
		LabName:       full.Protocol.LabName,
		Operator:      full.Protocol.OperatorName,
		FormattedDate: formattedFullDate,
		QRCodeData:    full.Protocol.ID,
	}

	for i, res := range full.Results {
		var method models.TestMethod
		var applicableLimit models.NormativeLimit
		cacheHit := false

		if methodsCache != nil {
			if fullMethod, ok := methodsCache[res.MethodID]; ok {
				method = fullMethod.Method
				applicableLimit = s.findMatchingLimitInMemory(fullMethod.Limits, fullMethod.LimitConditions, full.Sample.ContextParams)
				cacheHit = true
			}
		}

		if method.ID == "" {
			s.log.Debug("method cache miss, fetching from DB",
				zap.String("method_id", res.MethodID),
				zap.Int("result_index", i))

			var err error
			method, err = s.protocolService.standardRepo.GetTestMethod(ctx, res.MethodID)
			if err != nil {
				s.log.Warn("failed to load method for report",
					zap.String("method_id", res.MethodID),
					zap.Error(err))
				continue
			}
			applicableLimit, _ = s.protocolService.standardRepo.GetApplicableLimit(ctx, res.MethodID, full.Sample.ContextParams)
		} else if cacheHit {
			s.log.Debug("method cache hit",
				zap.String("method_id", res.MethodID),
				zap.Int("result_index", i))
		}

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

		complianceStr := "Соответствует"
		if res.IsCompliant != nil && !*res.IsCompliant {
			complianceStr = "Не соответствует"
		}

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
			Deviation:  res.DeviationMsg,
			RawInputs:  res.InputData,
		}
		data.Results = append(data.Results, row)
	}

	s.log.Debug("template data prepared",
		zap.Int("results_processed", len(data.Results)),
		zap.String("protocol_number", data.Protocol.Number))

	return data, nil
}

// findMatchingLimitInMemory - чистая функция, без логирования
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

// renderHTML - рендеринг HTML-шаблона
func (s *ReportService) renderHTML(templateName string, data interface{}) (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	tmplPath := filepath.Join(execDir, "templates", "protocols", templateName)

	if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
		tmplPath = filepath.Join("templates", "protocols", templateName)
		if _, err := os.Stat(tmplPath); os.IsNotExist(err) {
			return "", fmt.Errorf("template file not found at %s or %s",
				filepath.Join(execDir, "templates", "protocols", templateName),
				filepath.Join("templates", "protocols", templateName))
		}
	}

	s.log.Debug("parsing HTML template", zap.String("template_path", tmplPath))
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		s.log.Error("failed to parse template", zap.Error(err), zap.String("template", templateName))
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		s.log.Error("failed to execute template", zap.Error(err))
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	s.log.Debug("HTML template rendered successfully", zap.Int("output_size_bytes", buf.Len()))
	return buf.String(), nil
}

// generatePDFFromHTML - генерация PDF через wkhtmltopdf
func (s *ReportService) generatePDFFromHTML(htmlContent string) ([]byte, error) {
	s.log.Debug("initializing wkhtmltopdf")
	wkPath, err := GetWkhtmltopdfPath(s.wkhtmltopdfWindows)
	if err != nil {
		s.log.Error("failed to prepare wkhtmltopdf binary", zap.Error(err))
		return nil, fmt.Errorf("failed to prepare wkhtmltopdf: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "protocol_*.html")
	if err != nil {
		s.log.Error("failed to create temp HTML file", zap.Error(err))
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tmpFile.Name()
	defer func() {
		os.Remove(tempPath)
		s.log.Debug("cleaned up temp HTML file", zap.String("path", tempPath))
	}()

	if _, err := tmpFile.Write([]byte(htmlContent)); err != nil {
		tmpFile.Close()
		s.log.Error("failed to write HTML to temp file", zap.Error(err))
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	wkhtmltopdf.SetPath(wkPath)
	s.log.Debug("creating PDF generator instance")

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		s.log.Error("failed to initialize PDF generator", zap.Error(err))
		return nil, fmt.Errorf("failed to init wkhtmltopdf: %w", err)
	}

	page := wkhtmltopdf.NewPage(tempPath)
	page.EnableLocalFileAccess.Set(true)
	page.Encoding.Set("UTF-8")
	pdfg.AddPage(page)

	// Настройки PDF
	pdfg.Dpi.Set(300)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.MarginTop.Set(10)
	pdfg.MarginBottom.Set(10)
	pdfg.MarginLeft.Set(10)
	pdfg.MarginRight.Set(10)

	s.log.Debug("generating PDF content")
	if err := pdfg.Create(); err != nil {
		s.log.Error("failed to create PDF content", zap.Error(err))
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	pdfBytes := pdfg.Bytes()
	s.log.Debug("PDF content generated", zap.Int("size_bytes", len(pdfBytes)))
	return pdfBytes, nil
}

// GetWkhtmltopdfPath - утилита для извлечения бинарника
func GetWkhtmltopdfPath(wkhtmltopdfWindows []byte) (string, error) {
	var filename string
	filename = "wkhtmltopdf.exe"

	tempDir := os.TempDir()
	binaryPath := filepath.Join(tempDir, "desktop_lab_wkhtmltopdf", filename)

	if _, err := os.Stat(binaryPath); err == nil {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return "", fmt.Errorf("failed to chmod binary: %w", err)
		}
		return binaryPath, nil
	}

	dir := filepath.Dir(binaryPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	if err := os.WriteFile(binaryPath, wkhtmltopdfWindows, 0755); err != nil {
		return "", fmt.Errorf("failed to write wkhtmltopdf binary: %w", err)
	}

	return binaryPath, nil
}
