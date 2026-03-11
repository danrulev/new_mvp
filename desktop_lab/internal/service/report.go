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
	materialRepo    interface {
		GetByID(ctx context.Context, id string) (models.Material, error)
	}
	fontDir      string
	templatesDir string
	log          *zap.Logger
}

// NewReportService создает сервис отчетов
func NewReportService(
	protoSvc *ProtocolService,
	matRepo interface {
		GetByID(context.Context, string) (models.Material, error)
	},
	fontDir string,
	templatesDir string,
	log *zap.Logger,
) *ReportService {
	return &ReportService{
		protocolService: protoSvc,
		materialRepo:    matRepo,
		fontDir:         fontDir,
		templatesDir:    templatesDir,
		log:             log,
	}
}

// GenerateProtocolPDF генерирует PDF для конкретного протокола
func (s *ReportService) GenerateProtocolPDF(ctx context.Context, protocolID string) ([]byte, error) {
	s.log.Info("Generating PDF for protocol", zap.String("id", protocolID))

	// 1. Получаем полные данные протокола через сервис
	protocol, err := s.protocolService.GetProtocolByID(ctx, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol data: %w", err)
	}

	// 2. Получаем материал для названия

	sample, err := s.protocolService.sampleRepo.GetByID(ctx, protocol.Protocol.SampleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load sample: %w", err)
	}
	protocol.Protocol.Sample = sample
	// ✅ Теперь ищем материал по правильному ID
	material, err := s.materialRepo.GetByID(ctx, protocol.Protocol.Sample.MaterialID)

	results, err := s.protocolService.protocolRepo.GetResultsByProtocolID(ctx, protocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to load protocol results: %w", err)
	}

	// 3. Преобразуем данные в формат для шаблона (адаптация новых моделей под старый шаблон)
	templateData, err := s.prepareProtocolTemplateData(protocol.Protocol, results, material)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare template data: %w", err)
	}

	// 4. Рендерим HTML
	htmlContent, err := s.renderHTML("protocol_template.html", templateData)
	if err != nil {
		return nil, err
	}

	// 5. Генерируем PDF из HTML
	return s.generatePDFFromHTML(htmlContent)
}

// GenerateGroupSummaryPDF генерирует сводный отчет по группе (если нужен)
func (s *ReportService) GenerateGroupSummaryPDF(ctx context.Context, groupID string) ([]byte, error) {
	// Логика аналогична, но требует агрегации данных по группе.
	// Для краткости опущу детальную реализацию агрегации, сосредоточимся на основном протоколе.
	// Можно реализовать по аналогии с PrepareProtocolTemplateData, но собирая данные из всех протоколов группы.
	return nil, fmt.Errorf("group summary generation not fully implemented in this snippet")
}

// --- Внутренние методы ---

// TemplateData структура, соответствующая полям в твоем старом шаблоне protocol_template.html
// Я добавил поля, которые могут понадобиться, исходя из новой модели
type ProtocolTemplateData struct {
	Protocol      ProtocolView
	Sample        SampleView
	Material      MaterialView
	Results       []ResultRowView
	FormattedDate string
	QRCodeData    string // Опционально, если шаблон использует QR
	FontPath      string // Путь к шрифтам для CSS
	LabName       string
	Operator      string
	Project       string
}

type ProtocolView struct {
	Number string
	Date   string
	ID     string // Для внутренних нужд или QR
}

type SampleView struct {
	Number          string
	CollectionPlace string // В новой модели это может быть Note или часть Context
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
	Norm       string                 // Форматированная строка нормы (например "≥ 10.5")
	Compliance string                 // "Соответствует" / "Не соответствует"
	Deviation  string                 // Сообщение об отклонении
	RawInputs  map[string]interface{} // Если шаблон выводит входные данные
}

func (s *ReportService) prepareProtocolTemplateData(proto models.Protocol, results []models.TestResult, mat models.Material) (ProtocolTemplateData, error) {

	// Форматируем дату
	dateStr := ""
	if proto.TestDate != nil {
		dateStr = proto.TestDate.Format("02.01.2006")
	} else if !proto.CreatedAt.IsZero() {
		dateStr = proto.CreatedAt.Format("02.01.2006")
	}

	// Подготовка пробы
	samplePlace := proto.Sample.Note // Или можно взять из контекста, если там есть место отбора
	if val, ok := proto.Sample.ContextParams["location"]; ok {
		samplePlace = val
	}

	data := ProtocolTemplateData{
		Protocol: ProtocolView{
			Number: proto.ProtocolNumber,
			Date:   dateStr,
			ID:     proto.ID,
		},
		Sample: SampleView{
			Number:          proto.Sample.SampleNumber,
			CollectionPlace: samplePlace,
			Note:            proto.Sample.Note,
			MaterialName:    mat.Name,
		},
		Material: MaterialView{
			Name: mat.Name,
			Code: mat.Code,
		},
		LabName:       proto.LabName,
		Operator:      proto.OperatorName,
		Project:       "", // Можно добавить поле Project в Protocol, если нужно
		FormattedDate: time.Now().Format("02.01.2006"),
		FontPath:      s.fontDir,
		QRCodeData:    proto.ID, // Пример данных для QR
	}

	// Преобразование результатов
	for _, res := range results {
		// Формируем строку нормы
		normStr := "—"
		if res.MinNorm != nil && res.MaxNorm != nil {
			normStr = fmt.Sprintf("%.2f – %.2f", *res.MinNorm, *res.MaxNorm)
		} else if res.MinNorm != nil {
			normStr = fmt.Sprintf("≥ %.2f", *res.MinNorm)
		} else if res.MaxNorm != nil {
			normStr = fmt.Sprintf("≤ %.2f", *res.MaxNorm)
		}

		// Статус соответствия
		complianceStr := "Соответствует"
		if res.IsCompliant != nil && !*res.IsCompliant {
			complianceStr = "Не соответствует"
		}

		// Значение (защита от nil)
		val := 0.0
		if res.CalculatedValue != nil {
			val = *res.CalculatedValue
		}

		row := ResultRowView{
			MethodName: res.MethodName,
			Value:      math.Round(val*100) / 100,
			Unit:       res.MethodUnit,
			Norm:       normStr,
			Compliance: complianceStr,
			Deviation:  res.DeviationMsg,
			RawInputs:  res.InputData,
		}
		data.Results = append(data.Results, row)
	}

	return data, nil
}

func (s *ReportService) renderHTML(templateName string, data interface{}) (string, error) {
	// Путь к шаблону: корень проекта / templates / protocols / templateName
	// Так как мы запускаемся из exe, путь может отличаться.
	// Лучше использовать абсолютный путь относительно директории запуска или жестко заданный.
	// Предположим, что шаблоны лежат рядом с exe в папке templates/protocols

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

func (s *ReportService) generatePDFFromHTML(htmlContent string) ([]byte, error) {
	// Создаем временный файл для HTML (решение проблемы с локальными ресурсами wkhtmltopdf)
	tmpFile, err := os.CreateTemp("", "protocol_*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempPath := tmpFile.Name()
	defer os.Remove(tempPath) // Удаляем после генерации

	if _, err := tmpFile.Write([]byte(htmlContent)); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	// Инициализация генератора PDF
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to init wkhtmltopdf: %w", err)
	}

	// Создание страницы из файла
	page := wkhtmltopdf.NewPage(tempPath)

	// Критические настройки
	page.EnableLocalFileAccess.Set(true) // Разрешить доступ к локальным файлам (шрифты, картинки)
	page.Encoding.Set("UTF-8")

	// Настройки страницы
	pdfg.AddPage(page)
	pdfg.Dpi.Set(300)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)
	pdfg.MarginTop.Set(10)
	pdfg.MarginBottom.Set(10)
	pdfg.MarginLeft.Set(10)
	pdfg.MarginRight.Set(10)

	// Генерация
	if err := pdfg.Create(); err != nil {
		return nil, fmt.Errorf("failed to create PDF: %w", err)
	}

	return pdfg.Bytes(), nil
}
