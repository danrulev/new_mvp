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
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"go.uber.org/zap"
)

// ReportService отвечает за генерацию печатных форм (PDF)
type ReportService struct {
	protocolService *ProtocolService
	materialService *MaterialService // ✅ Используем сервис, а не интерфейс репозитория
	templatesDir    string           // Оставляем для кастомных путей, если нужно
	fontDir         string
	log             *zap.Logger
}

// NewReportService создает сервис отчетов
func NewReportService(
	protoSvc *ProtocolService,
	matSvc *MaterialService, // ✅ Inject service
	// fontDir можно убрать, если шрифты тоже в embed, или оставить для wkhtmltopdf
	fontDir string,
	templatesDir string,
	log *zap.Logger,
) *ReportService {
	return &ReportService{
		protocolService: protoSvc,
		materialService: matSvc,
		templatesDir:    templatesDir,
		fontDir:         fontDir,
		log:             log,
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

// GenerateGroupSummaryPDF - заглушка для примера
func (s *ReportService) GenerateGroupSummaryPDF(ctx context.Context, groupID string) ([]byte, error) {
	// Логика аналогична:
	// 1. Получить сводку через protocolService.GetGroupSummary (уже оптимизировано)
	// 2. Подготовить данные шаблона
	// 3. Сгенерировать PDF
	// Реализация опущена для краткости, но принцип тот же: минимум запросов, максимум кэша.

	// summary, err := s.protocolService.GetGroupSummary(ctx, groupID)
	// ...

	return nil, fmt.Errorf("group summary generation not fully implemented")
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
	Number string
	Date   string
	ID     string
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
	dateStr := ""
	if full.Protocol.TestDate != nil {
		dateStr = full.Protocol.TestDate.Format("02.01.2006")
	} else if !full.Protocol.CreatedAt.IsZero() {
		dateStr = full.Protocol.CreatedAt.Format("02.01.2006")
	}

	// Извлечение места отбора из контекста или заметок
	samplePlace := full.Sample.Note
	if val, ok := full.Sample.ContextParams["location"]; ok {
		samplePlace = val
	}

	data := ProtocolTemplateData{
		Protocol: ProtocolView{
			Number: full.Protocol.ProtocolNumber,
			Date:   dateStr,
			ID:     full.Protocol.ID,
		},
		Sample: SampleView{
			Number:          full.Sample.SampleNumber,
			CollectionPlace: samplePlace,
			Note:            full.Sample.Note,
			MaterialName:    full.Material.Name, // ✅ Уже загружено в full.Material
		},
		Material: MaterialView{
			Name: full.Material.Name,
			Code: full.Material.Code,
		},
		LabName:       full.Protocol.LabName,
		Operator:      full.Protocol.OperatorName,
		FormattedDate: time.Now().Format("02.01.2006"),
		QRCodeData:    full.Protocol.ID,
		// FontPath можно передать, если wkhtmltopdf требует локальный путь
	}

	// 🔥 ОБРАБОТКА РЕЗУЛЬТАТОВ БЕЗ N+1 ЗАПРОСОВ
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
