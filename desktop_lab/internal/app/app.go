package app

import (
	"context"
	"desktop_lab/data"
	"desktop_lab/internal/config"
	"desktop_lab/internal/db"
	"desktop_lab/internal/font"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
	"desktop_lab/internal/service"
	"desktop_lab/pkg/logger"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

// App - структура, экспортируемая в Wails
type App struct {
	services *service.Services
	ctx      context.Context
	log      *zap.Logger
}

// Init инициализирует все подсистемы и возвращает готовый App.
// fontFS передается из main.go (так как embed работает только там).
func Init(fontFS embed.FS, templateFS embed.FS) (*App, error) {
	// 1. Конфиг
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to init config: %w", err)
	}

	// 2. Логгер
	logInstance, err := logger.New(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger: %w", err)
	}

	logInstance.Info("Starting Lab Desktop Application")

	logInstance.Info("config", zap.Any("config", cfg))

	// 3. Шрифты
	fontDir, err := font.ExtractFonts(fontFS)
	if err != nil {
		logInstance.Fatal("Failed to extract fonts", zap.Error(err))
	}
	logInstance.Info("Fonts extracted", zap.String("path", fontDir))

	// 4. База данных
	dbConn, err := db.New(cfg.DB.Path, logInstance)
	if err != nil {
		logInstance.Fatal("DB connection failed", zap.Error(err))
	}
	logInstance.Info("Database connected")

	// 5. Миграции
	// Определяем путь к миграциям. В режиме dev это относительно корня, в prod - относительно exe.
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)

	migrationPaths := []string{
		filepath.Join("internal", "db", "migration"),
		filepath.Join(execDir, "internal", "db", "migration"),
		filepath.Join(execDir, "migration"),
	}

	migrationDir := ""
	for _, p := range migrationPaths {
		if _, err := os.Stat(p); err == nil {
			migrationDir = p
			break
		}
	}

	if migrationDir != "" {
		logInstance.Info("Running database migrations...", zap.String("path", migrationDir))
		if err := db.RunMigrations(dbConn, migrationDir, logInstance); err != nil {
			logInstance.Fatal("failed to run migrations", zap.Error(err))
		}
		logInstance.Info("Database migrations completed")
	} else {
		logInstance.Warn("Migration directory not found")
	}

	// 6. Репозитории (Используем НОВЫЕ репозитории)
	matRepo := repository.NewMaterialRepo(dbConn, logInstance)
	stdRepo := repository.NewStandardRepo(dbConn, logInstance)
	protRepo := repository.NewProtocolRepo(dbConn, logInstance)
	sampRepo := repository.NewSampleRepo(dbConn, logInstance)
	groupRepo := repository.NewExperimentGroupRepo(dbConn, logInstance)

	// 7. Сервисы (Используем НОВЫЕ сервисы)
	// Обратите внимание: NewServices теперь принимает другие аргументы
	services := service.NewServices(
		matRepo,
		stdRepo,
		protRepo,
		sampRepo,
		groupRepo,
		fontDir,
		"templates", // Путь к шаблонам на диске (рядом с exe)
		logInstance,
	)

	// 8. Сиды (Опционально, если нужно при каждом старте или по флагу)
	// Лучше вынести это в отдельный метод App или делать по кнопке в UI,
	// чтобы не нагружать старт. Но если нужно здесь:
	// data.SeedData(services, logInstance)

	if err := data.SeedData(services, logInstance); err != nil {
		logInstance.Error("Seed failed", zap.Error(err))
		os.Exit(1)
	}
	logInstance.Info("Data seed completed")

	return &App{
		services: services,
		ctx:      context.Background(),
		log:      logInstance,
	}, nil
}

// internal/app/app.go — ДОБАВИТЬ методы для Wails

// === МАТЕРИАЛЫ ===
func (a *App) GetMaterials() ([]models.Material, error) {
	return a.services.Materials.GetAll(a.ctx)
}

func (a *App) GetMaterialByID(id string) (models.Material, error) {
	return a.services.Materials.GetByID(a.ctx, id)
}

// === СТАНДАРТЫ ===
func (a *App) GetStandardsByMaterialID(materialID string) ([]models.Standard, error) {
	a.log.Debug("Getting standards for material", zap.String("material_id", materialID))

	stds, err := a.services.Standards.GetByMaterialID(a.ctx, materialID)
	if err != nil {
		a.log.Error("Failed to get standards", zap.Error(err))
		return nil, err
	}

	a.log.Debug("Found standards", zap.Int("count", len(stds)))
	return stds, nil
}

func (a *App) GetMethodDetails(methodID string) (models.TestMethod, error) {
	return a.services.Standards.GetMethodDetails(a.ctx, methodID)
}

func (a *App) GetMethodsByStandardID(standardID string) ([]models.TestMethod, error) {
	a.log.Debug("Getting methods for standard", zap.String("standard_id", standardID))

	methods, err := a.services.Standards.GetMethodsByStandardID(a.ctx, standardID)
	if err != nil {
		a.log.Error("Failed to get methods", zap.Error(err))
		return nil, err
	}

	a.log.Debug("Found methods", zap.Int("count", len(methods)))
	return methods, nil
}

// === ГРУППЫ ===
func (a *App) CreateGroup(name, projectName, location, materialID string) (models.ExperimentGroup, error) {
	return a.services.Groups.Create(a.ctx, name, projectName, location, materialID)
}

func (a *App) GetGroups(limit, offset int64) (models.GroupListResponse, error) {
	items, total, err := a.services.Groups.GetList(a.ctx, limit, offset)
	if err != nil {
		return models.GroupListResponse{}, err
	}
	meta := models.PaginatedMetadata{
		Total:      total,
		Page:       offset/limit + 1,
		PageSize:   limit,
		TotalPages: (total + limit - 1) / limit,
	}
	return models.GroupListResponse{Items: items, Meta: meta}, nil
}

func (a *App) GetGroupByID(id string) (models.ExperimentGroup, error) {
	return a.services.Groups.GetByID(a.ctx, id)
}

// === ПРОТОКОЛЫ ===
func (a *App) CreateProtocolWithSample(req models.CreateProtocolRequest) (models.Protocol, error) {
	return a.services.Protocols.CreateProtocolWithSample(a.ctx, req)
}

func (a *App) GetProtocolByID(id string) (service.GetProtocolByIDRequest, error) {
	return a.services.Protocols.GetProtocolByID(a.ctx, id)
}

func (a *App) GetProtocols(limit, offset int64) (models.ProtocolListResponse, error) {
	items, total, err := a.services.Protocols.GetList(a.ctx, limit, offset)
	if err != nil {
		return models.ProtocolListResponse{}, err
	}

	enhanced := make([]models.ProtocolListItem, 0, len(items))
	for _, p := range items {
		mat, _ := a.services.Materials.GetByID(a.ctx, p.Sample.MaterialID)
		enhanced = append(enhanced, models.ProtocolListItem{
			Protocol:     p,
			MaterialName: mat.Name,
		})
	}

	meta := models.PaginatedMetadata{
		Total:      total,
		Page:       offset/limit + 1,
		PageSize:   limit,
		TotalPages: (total + limit - 1) / limit,
	}
	return models.ProtocolListResponse{Items: enhanced, Meta: meta}, nil
}

func (a *App) GetGroupSummary(groupID string) (*models.GroupSummary, error) {
	return a.services.Protocols.GetGroupSummary(a.ctx, groupID)
}

// === ОТЧЁТЫ ===
func (a *App) GenerateProtocolPDF(protocolID string) (string, error) {

	_, err := a.services.Reports.GenerateProtocolPDF(a.ctx, protocolID)
	if err != nil {
		return "", err
	}
	// Возвращаем сообщение (в продакшене — путь к файлу)
	return "PDF сгенерирован (реализуйте сохранение через диалог)", nil
}

func (a *App) GenerateGroupSummaryPDF(groupID string) (string, error) {
	_, err := a.services.Reports.GenerateGroupSummaryPDF(a.ctx, groupID)
	if err != nil {
		return "", err
	}
	return "Сводный PDF сгенерирован", nil
}

func (a *App) SaveProtocolPDFWithDialog(protocolID string) (string, error) {
	// 1. Генерируем PDF
	pdfBytes, err := a.services.Reports.GenerateProtocolPDF(a.ctx, protocolID)
	if err != nil {
		return "", err
	}

	// 2. Открываем диалог сохранения
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Сохранить протокол",
		Filters: []runtime.FileFilter{
			{Pattern: "*.pdf", DisplayName: "PDF Files"},
		},
		DefaultFilename: "protocol.pdf",
	})
	if err != nil || filePath == "" {
		return "", nil // Отменено пользователем
	}

	// 3. Сохраняем файл
	if err := os.WriteFile(filePath, pdfBytes, 0644); err != nil {
		return "", fmt.Errorf("ошибка записи файла: %w", err)
	}

	return filePath, nil
}

func (a *App) SaveGroupPDFWithDialog(groupID string) (string, error) {
	pdfBytes, err := a.services.Reports.GenerateGroupSummaryPDF(a.ctx, groupID)
	if err != nil {
		return "", err
	}

	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "Сохранить сводку",
		Filters: []runtime.FileFilter{
			{Pattern: "*.pdf", DisplayName: "PDF Files"},
		},
		DefaultFilename: "group_summary.pdf",
	})
	if err != nil || filePath == "" {
		return "", nil
	}

	if err := os.WriteFile(filePath, pdfBytes, 0644); err != nil {
		return "", fmt.Errorf("ошибка записи файла: %w", err)
	}

	return filePath, nil
}
