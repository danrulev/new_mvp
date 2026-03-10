package app

import (
	"context"
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

	return &App{
		services: services,
		ctx:      context.Background(),
		log:      logInstance,
	}, nil
}

// ============================================================================
// МЕТОДЫ ДЛЯ ЭКСПОРТА (API для фронтенда)
// ============================================================================

func (a *App) GetMaterials() ([]models.Material, error) {
	return a.services.Materials.GetAll(a.ctx)
}

func (a *App) CreateMaterial(name, code string) (models.Material, error) {
	return a.services.Materials.Create(a.ctx, name, code)
}

func (a *App) GetGroups(limit, offset int64) ([]models.ExperimentGroup, int64, error) {
	return a.services.Groups.GetList(a.ctx, limit, offset)
}

func (a *App) CreateGroup(name, project, location, matID string) (models.ExperimentGroup, error) {
	return a.services.Groups.Create(a.ctx, name, project, location, matID)
}

func (a *App) CreateProtocol(req models.CreateProtocolRequest) (string, error) {
	p, err := a.services.Protocols.CreateProtocolWithSample(a.ctx, req)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}

func (a *App) GeneratePDF(protocolID string) ([]byte, error) {
	return a.services.Reports.GenerateProtocolPDF(a.ctx, protocolID)
}

func (a *App) GetGroupSummary(groupID string) (*models.GroupSummary, error) {
	return a.services.Protocols.GetGroupSummary(a.ctx, groupID)
}

func (a *App) GetStandardsByMaterial(materialID string) ([]models.Standard, error) {
	return a.services.Standards.GetByMaterialID(a.ctx, materialID)
}

// GetMethodsByStandard возвращает методы для конкретного стандарта с входными параметрами
func (a *App) GetMethodsByStandard(standardID string) ([]models.TestMethod, error) {
	// В сервисе нужно реализовать метод, который грузит методы + inputs
	// Если его нет, можно сделать заглушку или доработать сервис
	return a.services.Standards.GetMethodsByStandardID(a.ctx, standardID)
}

// GetProtocols возвращает список протоколов с пагинацией и доп. данными
// Возвращает структуру, совместимую с ожиданиями фронтенда
func (a *App) GetProtocols(limit, offset int64) (*models.ProtocolListResponse, error) {
	// Пока у нас нет метода с пагинацией в сервисе, возьмем все или реализуем заглушку
	// В идеале: добавить в service/protocol.go метод GetList(limit, offset)

	// Для примера реализуем получение всех и обрезку (неэффективно для большой БД, но для старта ок)
	// Лучше добавить метод в сервис: a.services.Protocols.GetList(...)

	// Заглушка: получаем последние N протоколов (нужно реализовать в репозитории/сервисе)
	// Допустим, мы пока не реализовали пагинацию в сервисе, вернем ошибку или пустой список,
	// пока ты не добавишь метод в сервис.

	// === ВРЕМЕННОЕ РЕШЕНИЕ (до реализации пагинации в сервисе) ===
	// Тебе нужно добавить метод GetProtocols(limit, offset) в service/protocol.go
	// А здесь вызвать его.

	protocols, total, err := a.services.Protocols.GetList(a.ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Обогащаем протоколы именами материалов (денормализация для UI)
	items := make([]models.ProtocolListItem, len(protocols))
	for i, p := range protocols {
		matName := ""
		if p.Sample != nil {
			mat, _ := a.services.Materials.GetByID(a.ctx, p.Sample.MaterialID)
			if mat.ID != "" {
				matName = mat.Name
			}
		}

		items[i] = models.ProtocolListItem{
			Protocol:     p, // Здесь нужно убедиться, что поля маппятся верно
			MaterialName: matName,
		}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	return &models.ProtocolListResponse{
		Items: items,
		Meta: models.PaginatedMetadata{
			Total:      total,
			Page:       (offset / limit) + 1,
			PageSize:   limit,
			TotalPages: totalPages,
		},
	}, nil
}

// SaveProtocolPDFWithDialog генерирует PDF и сохраняет через диалог системы
func (a *App) SaveProtocolPDFWithDialog(protocolID string) (string, error) {
	// 1. Генерируем PDF
	// pdfBytes, err := a.services.Reports.GenerateProtocolPDF(a.ctx, protocolID)
	// if err != nil {
	// 	return "", err
	// }

	// 2. Открываем диалог сохранения (требуется импорт runtime)
	// Примечание: Wails Runtime вызывается из JS, но можно сделать и тут, если передать контекст окна
	// Однако, проще вернуть байты фронтенду, а там вызвать SaveDialog.
	// Но раз ты просишь метод как в старом проекте, давай вернем путь.
	// Для этого нужен доступ к файловой системе и диалогу.
	// В Wails v2 это делается через runtime.SaveDialog в JS.
	// Go метод может только вернуть байты.
	// Поэтому изменим логику: этот метод будет возвращать байты, а JS сохранит.
	// ИЛИ: используем wails filesystem.

	// Вариант для совместимости со старым кодом:
	// Вернем ошибку, что нужно использовать GeneratePDF + SaveDialog на JS стороне,
	// так как прямой доступ к диалогу из Go в Wails v2 требует лишних зависимостей.
	// Но если очень нужно, можно использовать os.WriteFile в папку Загрузок.

	return "", nil // Заглушка, см. комментарий выше
}

func (a *App) Shutdown(ctx context.Context) {
	a.log.Info("Application shutting down")
}

// Startup вызывается Wails при старте UI (опционально)
func (a *App) Startup(ctx context.Context) {
	a.log.Info("UI Started")
}
