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
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/jmoiron/sqlx"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.uber.org/zap"
)

// App - главная структура приложения, экспортируемая в Wails.
// Все публичные методы с указателем на App автоматически становятся доступными в JavaScript.
type App struct {
	ctx      context.Context
	log      *zap.Logger
	services *service.Services

	// Wails lifecycle
	wailsCtx context.Context
	appReady bool
	mu       sync.RWMutex // для потокобезопасности при доступе из JS
}

// ============================================================================
// ИНИЦИАЛИЗАЦИЯ
// ============================================================================

// New создает новый экземпляр приложения.
// Вызывается автоматически при старте Wails.
func New() *App {
	return &App{
		ctx: context.Background(),
	}
}

// Init инициализирует все подсистемы приложения.
// Вызывается из main.go после создания экземпляра App.
// fontFS и templateFS передаются через embed из main (единственное место, где это возможно).
func (a *App) Init(fontFS embed.FS, templateFS embed.FS) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// 1. Конфигурация
	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}

	// 2. Логгер
	logInstance, err := logger.New(cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	a.log = logInstance

	a.log.Info("Starting Lab Desktop Application")

	// 3. Шрифты для PDF
	fontDir, err := font.ExtractFonts(fontFS)
	if err != nil {
		a.log.Fatal("Failed to extract fonts", zap.Error(err))
	}
	a.log.Info("Fonts extracted", zap.String("path", fontDir))

	// 4. База данных
	dbConn, err := db.New(cfg.DB.Path, a.log)
	if err != nil {
		a.log.Fatal("DB connection failed", zap.Error(err))
	}
	a.log.Info("🗄️ Database connected", zap.String("path", cfg.DB.Path))

	// 5. Миграции
	if err := a.runMigrations(dbConn); err != nil {
		a.log.Fatal("Migration failed", zap.Error(err))
	}

	// 6. Репозитории
	matRepo := repository.NewMaterialRepo(dbConn, a.log)
	stdRepo := repository.NewStandardRepo(dbConn, a.log)
	protRepo := repository.NewProtocolRepo(dbConn, a.log)
	sampRepo := repository.NewSampleRepo(dbConn, a.log)
	groupRepo := repository.NewExperimentGroupRepo(dbConn, a.log)

	// 7. Сервисы
	a.services = service.NewServices(
		matRepo, stdRepo, protRepo, sampRepo, groupRepo,
		fontDir, "templates", a.log,
	)

	if err := data.SeedData(a.services, a.log); err != nil {
		a.log.Error("Seed failed", zap.Error(err))
		// Не прерываем запуск, но логируем ошибку
	} else {
		a.log.Info("Data seed completed")
	}

	a.appReady = true
	a.log.Info("Application initialized successfully")
	return nil
}

// runMigrations выполняет миграции БД с поиском пути в разных локациях
func (a *App) runMigrations(dbConn *sqlx.DB) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)

	// Порядок поиска: dev → prod рядом с exe → prod в подпапке
	paths := []string{
		filepath.Join("internal", "db", "migration"),          // go run / dev
		filepath.Join(execDir, "internal", "db", "migration"), // prod: рядом с exe
		filepath.Join(execDir, "migration"),                   // prod: отдельная папка
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			a.log.Info("🔄 Running migrations", zap.String("path", p))
			if err := db.RunMigrations(dbConn, p, a.log); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
			a.log.Info("Migrations completed")
			return nil
		}
	}

	a.log.Warn("Migration directory not found - skipping")
	return nil
}

// ============================================================================
// WAILS LIFECYCLE METHODS
// ============================================================================

// Startup вызывается после загрузки DOM, но до показа окна.
// Идеально для финальных проверок перед стартом UI.
func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.wailsCtx = ctx
	a.mu.Unlock()

	a.log.Info("Wails startup: context attached")

	// Можно отправить событие в фронтенд о готовности
	// runtime.EventsEmit(ctx, "app:ready", map[string]interface{}{"status": "ok"})
}

// DomReady вызывается когда DOM полностью загружен.
// Можно инициализировать слушатели событий или отправить данные во фронтенд.
func (a *App) DomReady(ctx context.Context) {
	a.log.Info("DOM ready")

	// Пример: отправить конфигурацию во фронтенд
	// runtime.EventsEmit(ctx, "app:config", map[string]interface{}{
	// 	"version": config.Version,
	// 	"env":     config.Environment,
	// })
}

// BeforeClose вызывается перед закрытием приложения.
// Можно показать диалог подтверждения или сохранить состояние.
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.log.Info("App closing...")

	// Пример: предотвратить закрытие если есть несохранённые данные
	// if hasUnsavedChanges() {
	//     dialog, _ := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
	//         Type:    runtime.QuestionDialog,
	//         Title:   "Подтверждение",
	//         Message: "Есть несохранённые изменения. Закрыть?",
	//         Buttons: []string{"Да", "Нет"},
	//     })
	//     return dialog == "Нет"
	// }

	return false // разрешить закрытие
}

// Shutdown вызывается при завершении работы приложения.
// Здесь нужно освободить ресурсы: закрыть БД, остановить горутин и т.д.
func (a *App) Shutdown(ctx context.Context) {
	a.log.Info("Application shutdown")

	// Закрываем соединение с БД (если репозиторий поддерживает)
	// db.GetDB().Close()

	a.log.Info("Shutdown complete")
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ
// ============================================================================

// isReady проверяет, инициализировано ли приложение
// Потокобезопасная проверка для методов, вызываемых из JS
func (a *App) isReady() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.appReady {
		return fmt.Errorf("application not initialized")
	}
	return nil
}

// wrapError оборачивает ошибку в формат, понятный JavaScript
// Логирует ошибку на сервере и возвращает чистое сообщение для фронтенда
func (a *App) wrapError(op string, err error) error {
	if err == nil {
		return nil
	}

	a.log.Error(op, zap.Error(err))

	// Для продакшена: не возвращать детали внутренних ошибок
	// return fmt.Errorf("operation failed: %s", op)

	// Для разработки: возвращать полную ошибку для отладки
	return err
}

// ============================================================================
// === МАТЕРИАЛЫ ===
// ============================================================================

// GetMaterials возвращает список всех материалов
// @returns {Promise<Material[]>}
func (a *App) GetMaterials() ([]models.Material, error) {
	if err := a.isReady(); err != nil {
		return nil, a.wrapError("GetMaterials", err)
	}

	a.log.Debug("GetMaterials: fetching all materials")

	mats, err := a.services.Materials.GetAll(a.wailsCtx)
	if err != nil {
		return nil, a.wrapError("GetMaterials", err)
	}

	a.log.Debug("GetMaterials: success", zap.Int("count", len(mats)))
	return mats, nil
}

// GetMaterialByID возвращает материал по ID
// @param {string} id - UUID материала
// @returns {Promise<Material>}
func (a *App) GetMaterialByID(id string) (models.Material, error) {
	if err := a.isReady(); err != nil {
		return models.Material{}, a.wrapError("GetMaterialByID", err)
	}
	if id == "" {
		return models.Material{}, fmt.Errorf("material ID is required")
	}

	mat, err := a.services.Materials.GetByID(a.wailsCtx, id)
	if err != nil {
		return models.Material{}, a.wrapError("GetMaterialByID", err)
	}
	if mat.ID == "" {
		return models.Material{}, fmt.Errorf("material not found: %s", id)
	}

	return mat, nil
}

// CreateMaterial создаёт новый материал
// @param {string} name - Название материала (обязательно)
// @param {string} code - Код материала (опционально)
// @returns {Promise<Material>}
func (a *App) CreateMaterial(name, code string) (models.Material, error) {
	if err := a.isReady(); err != nil {
		return models.Material{}, a.wrapError("CreateMaterial", err)
	}
	if name == "" {
		return models.Material{}, fmt.Errorf("material name is required")
	}

	mat, err := a.services.Materials.Create(a.wailsCtx, name, code)
	if err != nil {
		return models.Material{}, a.wrapError("CreateMaterial", err)
	}

	a.log.Info("Material created", zap.String("id", mat.ID), zap.String("name", mat.Name))
	return mat, nil
}

// ============================================================================
// === СТАНДАРТЫ (ГОСТы) ===
// ============================================================================

// GetStandardsByMaterialID возвращает стандарты для материала
// @param {string} materialID - UUID материала
// @returns {Promise<Standard[]>}
func (a *App) GetStandardsByMaterialID(materialID string) ([]models.Standard, error) {
	if err := a.isReady(); err != nil {
		return nil, a.wrapError("GetStandardsByMaterialID", err)
	}
	if materialID == "" {
		return nil, fmt.Errorf("material ID is required")
	}

	a.log.Debug("GetStandardsByMaterialID", zap.String("material_id", materialID))

	stds, err := a.services.Standards.GetByMaterialID(a.wailsCtx, materialID)
	if err != nil {
		return nil, a.wrapError("GetStandardsByMaterialID", err)
	}

	a.log.Debug("GetStandardsByMaterialID: success", zap.Int("count", len(stds)))
	return stds, nil
}

// GetMethodsByStandardID возвращает методы стандарта
// @param {string} standardID - UUID стандарта
// @returns {Promise<TestMethod[]>}
func (a *App) GetMethodsByStandardID(standardID string) ([]models.TestMethod, error) {
	if err := a.isReady(); err != nil {
		return nil, a.wrapError("GetMethodsByStandardID", err)
	}
	if standardID == "" {
		return nil, fmt.Errorf("standard ID is required")
	}

	a.log.Debug("GetMethodsByStandardID", zap.String("standard_id", standardID))

	methods, err := a.services.Standards.GetMethodsByStandardID(a.wailsCtx, standardID)
	if err != nil {
		return nil, a.wrapError("GetMethodsByStandardID", err)
	}

	a.log.Debug("GetMethodsByStandardID: success", zap.Int("count", len(methods)))
	return methods, nil
}

// GetMethodDetails возвращает детали метода с входными параметрами
// @param {string} methodID - UUID метода
// @returns {Promise<TestMethodFull>}  ✅ ИСПРАВЛЕНО: был TestMethod
func (a *App) GetMethodDetails(methodID string) (models.TestMethodFull, error) {
	if err := a.isReady(); err != nil {
		return models.TestMethodFull{}, a.wrapError("GetMethodDetails", err)
	}
	if methodID == "" {
		return models.TestMethodFull{}, fmt.Errorf("method ID is required")
	}

	// ✅ Вызываем сервис который возвращает TestMethodFull
	methodFull, err := a.services.Standards.GetMethodDetails(a.wailsCtx, methodID)
	if err != nil {
		return models.TestMethodFull{}, a.wrapError("GetMethodDetails", err)
	}
	if methodFull.Method.ID == "" {
		return models.TestMethodFull{}, fmt.Errorf("method not found: %s", methodID)
	}

	return methodFull, nil
}

// GetStandardDimensions возвращает измерения контекста для стандарта
// @param {string} standardID - UUID стандарта
// @returns {Promise<ContextDimension[]>}
func (a *App) GetStandardDimensions(standardID string) ([]models.ContextDimension, error) {
	if err := a.isReady(); err != nil {
		return nil, a.wrapError("GetStandardDimensions", err)
	}
	if standardID == "" {
		return nil, fmt.Errorf("standard ID is required")
	}

	dims, err := a.services.Standards.GetStandardDimensions(a.wailsCtx, standardID)
	if err != nil {
		return nil, a.wrapError("GetStandardDimensions", err)
	}

	return dims, nil
}

// CreateStandard создаёт новый стандарт с методами и лимитами
// @param {CreateStandardRequest} req - Данные стандарта
// @returns {Promise<string>} - ID созданного стандарта
func (a *App) CreateStandard(req models.CreateStandardRequest) (string, error) {
	if err := a.isReady(); err != nil {
		return "", a.wrapError("CreateStandard", err)
	}

	id, err := a.services.Standards.CreateStandard(a.wailsCtx, req)
	if err != nil {
		return "", a.wrapError("CreateStandard", err)
	}

	a.log.Info("Standard created", zap.String("id", id), zap.String("name", req.Name))
	return id, nil
}

// InvalidateStandardCache очищает кэш стандарта (для админ-панели)
// @param {string} standardID - UUID стандарта
// @returns {Promise<void>}
func (a *App) InvalidateStandardCache(standardID string) error {
	if err := a.isReady(); err != nil {
		return a.wrapError("InvalidateStandardCache", err)
	}

	a.services.Standards.InvalidateStandardCache(standardID)
	return nil
}

// ============================================================================
// === ГРУППЫ ЭКСПЕРИМЕНТОВ ===
// ============================================================================

// CreateGroup создаёт новую группу экспериментов
// @param {string} name - Название группы
// @param {string} projectName - Название проекта
// @param {string} location - Место проведения
// @param {string} materialID - UUID материала
// @returns {Promise<ExperimentGroup>}
func (a *App) CreateGroup(name, projectName, location, materialID string) (models.ExperimentGroup, error) {
	if err := a.isReady(); err != nil {
		return models.ExperimentGroup{}, a.wrapError("CreateGroup", err)
	}
	if name == "" || materialID == "" {
		return models.ExperimentGroup{}, fmt.Errorf("name and materialID are required")
	}

	group, err := a.services.Groups.Create(a.wailsCtx, name, projectName, location, materialID)
	if err != nil {
		return models.ExperimentGroup{}, a.wrapError("CreateGroup", err)
	}

	a.log.Info("Group created", zap.String("id", group.ID), zap.String("name", group.Name))
	return group, nil
}

// GetGroups возвращает список групп с пагинацией
// @param {number} limit - Количество записей на странице
// @param {number} offset - Смещение
// @returns {Promise<GroupListResponse>}
func (a *App) GetGroups(limit, offset int64) (models.GroupListResponse, error) {
	if err := a.isReady(); err != nil {
		return models.GroupListResponse{}, a.wrapError("GetGroups", err)
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := a.services.Groups.GetList(a.wailsCtx, limit, offset)
	if err != nil {
		return models.GroupListResponse{}, a.wrapError("GetGroups", err)
	}

	meta := models.PaginatedMetadata{
		Total:      total,
		Page:       offset/limit + 1,
		PageSize:   limit,
		TotalPages: (total + limit - 1) / limit,
	}

	return models.GroupListResponse{Items: items, Meta: meta}, nil
}

// GetGroupByID возвращает группу по ID
// @param {string} id - UUID группы
// @returns {Promise<ExperimentGroup>}
func (a *App) GetGroupByID(id string) (models.ExperimentGroup, error) {
	if err := a.isReady(); err != nil {
		return models.ExperimentGroup{}, a.wrapError("GetGroupByID", err)
	}
	if id == "" {
		return models.ExperimentGroup{}, fmt.Errorf("group ID is required")
	}

	group, err := a.services.Groups.GetByID(a.wailsCtx, id)
	if err != nil {
		return models.ExperimentGroup{}, a.wrapError("GetGroupByID", err)
	}
	if group.ID == "" {
		return models.ExperimentGroup{}, fmt.Errorf("group not found: %s", id)
	}

	return group, nil
}

// ============================================================================
// === ПРОТОКОЛЫ И ПРОБЫ ===
// ============================================================================

// CreateProtocolWithSample создаёт пробу и протокол с результатами
// @param {CreateProtocolRequest} req - Данные протокола
// @returns {Promise<Protocol>}
func (a *App) CreateProtocolWithSample(req models.CreateProtocolRequest) (models.Protocol, error) {
	if err := a.isReady(); err != nil {
		return models.Protocol{}, a.wrapError("CreateProtocolWithSample", err)
	}

	protocol, err := a.services.Protocols.CreateProtocolWithSample(a.wailsCtx, req)
	if err != nil {
		return models.Protocol{}, a.wrapError("CreateProtocolWithSample", err)
	}

	a.log.Info("Protocol created",
		zap.String("protocol_id", protocol.ID),
		zap.String("sample_id", protocol.SampleID))

	return protocol, nil
}

// GetProtocolByID возвращает протокол с результатами
// @param {string} id - UUID протокола
// @returns {Promise<GetProtocolByIDRequest>}
func (a *App) GetProtocolByID(id string) (models.GetProtocolByIDRequest, error) {
	if err := a.isReady(); err != nil {
		return models.GetProtocolByIDRequest{}, a.wrapError("GetProtocolByID", err)
	}
	if id == "" {
		return models.GetProtocolByIDRequest{}, fmt.Errorf("protocol ID is required")
	}

	result, err := a.services.Protocols.GetProtocolByID(a.wailsCtx, id)
	if err != nil {
		return models.GetProtocolByIDRequest{}, a.wrapError("GetProtocolByID", err)
	}
	if result.Protocol.ID == "" {
		return models.GetProtocolByIDRequest{}, fmt.Errorf("protocol not found: %s", id)
	}

	return result, nil
}

// GetProtocolFull возвращает полный протокол с пробой и материалом (для отчетов)
// @param {string} id - UUID протокола
// @returns {Promise<ProtocolFull>}
func (a *App) GetProtocolFull(id string) (models.ProtocolFull, error) {
	if err := a.isReady(); err != nil {
		return models.ProtocolFull{}, a.wrapError("GetProtocolFull", err)
	}
	if id == "" {
		return models.ProtocolFull{}, fmt.Errorf("protocol ID is required")
	}

	full, err := a.services.Protocols.GetProtocolFull(a.wailsCtx, id)
	if err != nil {
		return models.ProtocolFull{}, a.wrapError("GetProtocolFull", err)
	}
	if full.IsEmpty() {
		return models.ProtocolFull{}, fmt.Errorf("protocol not found: %s", id)
	}

	return full, nil
}

// GetProtocols возвращает список протоколов с пагинацией
// @param {number} limit - Количество записей
// @param {number} offset - Смещение
// @returns {Promise<ProtocolListResponse>}
func (a *App) GetProtocols(limit, offset int64) (models.ProtocolListResponse, error) {
	if err := a.isReady(); err != nil {
		return models.ProtocolListResponse{}, a.wrapError("GetProtocols", err)
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := a.services.Protocols.GetList(a.wailsCtx, limit, offset)
	if err != nil {
		return models.ProtocolListResponse{}, a.wrapError("GetProtocols", err)
	}

	meta := models.PaginatedMetadata{
		Total:      total,
		Page:       offset/limit + 1,
		PageSize:   limit,
		TotalPages: (total + limit - 1) / limit,
	}

	return models.ProtocolListResponse{Items: items, Meta: meta}, nil
}

// GetProtocolsByGroupID возвращает протоколы конкретной группы
// @param {string} groupID - UUID группы
// @returns {Promise<Protocol[]>}
func (a *App) GetProtocolsByGroupID(groupID string) ([]models.Protocol, error) {
	if err := a.isReady(); err != nil {
		return nil, a.wrapError("GetProtocolsByGroupID", err)
	}
	if groupID == "" {
		return nil, fmt.Errorf("group ID is required")
	}

	protocols, err := a.services.Protocols.GetProtocolsByGroupID(a.wailsCtx, groupID)
	if err != nil {
		return nil, a.wrapError("GetProtocolsByGroupID", err)
	}

	return protocols, nil
}

// GetGroupSummary возвращает сводную статистику по группе
// @param {string} groupID - UUID группы
// @returns {Promise<GroupSummary>}
func (a *App) GetGroupSummary(groupID string) (models.GroupSummary, error) {
	if err := a.isReady(); err != nil {
		return models.GroupSummary{}, a.wrapError("GetGroupSummary", err)
	}
	if groupID == "" {
		return models.GroupSummary{}, fmt.Errorf("group ID is required")
	}

	summary, err := a.services.Protocols.GetGroupSummary(a.wailsCtx, groupID)
	if err != nil {
		return models.GroupSummary{}, a.wrapError("GetGroupSummary", err)
	}

	return summary, nil
}

// ============================================================================
// === ОТЧЁТЫ (PDF) ===
// ============================================================================

// GenerateProtocolPDF генерирует PDF протокола и возвращает байты (base64)
// @param {string} protocolID - UUID протокола
// @returns {Promise<string>} - Base64-строка с PDF
func (a *App) GenerateProtocolPDF(protocolID string) (string, error) {
	if err := a.isReady(); err != nil {
		return "", a.wrapError("GenerateProtocolPDF", err)
	}
	if protocolID == "" {
		return "", fmt.Errorf("protocol ID is required")
	}

	pdfBytes, err := a.services.Reports.GenerateProtocolPDF(a.wailsCtx, protocolID)
	if err != nil {
		return "", a.wrapError("GenerateProtocolPDF", err)
	}

	// Конвертируем в base64 для передачи в JavaScript
	return base64.StdEncoding.EncodeToString(pdfBytes), nil
}

// GenerateGroupSummaryPDF генерирует сводный PDF по группе
// @param {string} groupID - UUID группы
// @returns {Promise<string>} - Base64-строка с PDF
func (a *App) GenerateGroupSummaryPDF(groupID string) (string, error) {
	if err := a.isReady(); err != nil {
		return "", a.wrapError("GenerateGroupSummaryPDF", err)
	}
	if groupID == "" {
		return "", fmt.Errorf("group ID is required")
	}

	pdfBytes, err := a.services.Reports.GenerateGroupSummaryPDF(a.wailsCtx, groupID)
	if err != nil {
		return "", a.wrapError("GenerateGroupSummaryPDF", err)
	}

	return base64.StdEncoding.EncodeToString(pdfBytes), nil
}

// SaveProtocolPDFWithDialog открывает диалог сохранения и записывает PDF на диск
// @param {string} protocolID - UUID протокола
// @returns {Promise<string>} - Путь к сохранённому файлу или пустая строка если отменено
func (a *App) SaveProtocolPDFWithDialog(protocolID string) (string, error) {
	if err := a.isReady(); err != nil {
		return "", a.wrapError("SaveProtocolPDFWithDialog", err)
	}
	if protocolID == "" {
		return "", fmt.Errorf("protocol ID is required")
	}

	// Генерируем PDF
	pdfBytes, err := a.services.Reports.GenerateProtocolPDF(a.wailsCtx, protocolID)
	if err != nil {
		return "", a.wrapError("SaveProtocolPDFWithDialog", err)
	}

	// Открываем диалог сохранения
	filePath, err := runtime.SaveFileDialog(a.wailsCtx, runtime.SaveDialogOptions{
		Title: "Сохранить протокол",
		Filters: []runtime.FileFilter{
			{Pattern: "*.pdf", DisplayName: "PDF Files"},
		},
		DefaultFilename: "protocol.pdf",
	})
	if err != nil {
		return "", a.wrapError("SaveProtocolPDFWithDialog: dialog", err)
	}
	if filePath == "" {
		// Пользователь отменил сохранение
		return "", nil
	}

	// Записываем файл
	if err := os.WriteFile(filePath, pdfBytes, 0644); err != nil {
		return "", a.wrapError("SaveProtocolPDFWithDialog: write",
			fmt.Errorf("failed to write file: %w", err))
	}

	a.log.Info("PDF saved", zap.String("path", filePath))
	return filePath, nil
}

// SaveGroupPDFWithDialog открывает диалог сохранения для сводного отчёта
// @param {string} groupID - UUID группы
// @returns {Promise<string>} - Путь к файлу или пустая строка
func (a *App) SaveGroupPDFWithDialog(groupID string) (string, error) {
	if err := a.isReady(); err != nil {
		return "", a.wrapError("SaveGroupPDFWithDialog", err)
	}
	if groupID == "" {
		return "", fmt.Errorf("group ID is required")
	}

	pdfBytes, err := a.services.Reports.GenerateGroupSummaryPDF(a.wailsCtx, groupID)
	if err != nil {
		return "", a.wrapError("SaveGroupPDFWithDialog", err)
	}

	filePath, err := runtime.SaveFileDialog(a.wailsCtx, runtime.SaveDialogOptions{
		Title: "Сохранить сводный отчёт",
		Filters: []runtime.FileFilter{
			{Pattern: "*.pdf", DisplayName: "PDF Files"},
		},
		DefaultFilename: "group_summary.pdf",
	})
	if err != nil || filePath == "" {
		return "", err
	}

	if err := os.WriteFile(filePath, pdfBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	a.log.Info("Group PDF saved", zap.String("path", filePath))
	return filePath, nil
}

// ============================================================================
// === СИСТЕМНЫЕ МЕТОДЫ ===
// ============================================================================

// LogFrontendEvent принимает лог-событие от фронтенда для отладки
// @param {string} level - Уровень лога: 'debug', 'info', 'warn', 'error'
// @param {string} message - Сообщение
// @param {object} data - Дополнительные данные
// @returns {Promise<void>}
func (a *App) LogFrontendEvent(level, message string, data map[string]interface{}) {
	switch level {
	case "error":
		a.log.Error("[JS] "+message, zap.Any("data", data))
	case "warn":
		a.log.Warn("[JS] "+message, zap.Any("data", data))
	case "debug":
		a.log.Debug("[JS] "+message, zap.Any("data", data))
	default:
		a.log.Info("[JS] "+message, zap.Any("data", data))
	}
}

// ============================================================================
// === СОБЫТИЯ (Events) ===
// ============================================================================

// EmitEvent отправляет событие во фронтенд через Wails Events
// Используется для push-уведомлений о завершении долгих операций
// @param {string} eventName - Имя события
// @param {any} payload - Данные события (должны быть JSON-сериализуемы)
func (a *App) EmitEvent(eventName string, payload interface{}) {
	if a.wailsCtx == nil {
		a.log.Warn("Cannot emit event: wails context not set")
		return
	}
	runtime.EventsEmit(a.wailsCtx, eventName, payload)
	a.log.Debug("Event emitted", zap.String("event", eventName))
}

// SubscribeToEvents (опционально) - можно реализовать механизм подписки фронтенда
// на определённые события, если нужна сложная логика подписок.
