package app

import (
	"context"
	"desktop_lab/data"
	"desktop_lab/internal/config"
	"desktop_lab/internal/db"
	"desktop_lab/internal/font"
	"desktop_lab/internal/handler"
	"desktop_lab/internal/repository"
	"desktop_lab/internal/server"
	"desktop_lab/internal/service"
	"desktop_lab/pkg/logger"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/gen2brain/dlgs"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// App представляет основное приложение с управлением состоянием.
type App struct {
	dbConn  *sqlx.DB
	cfgMu   sync.RWMutex
	cfg     *config.Config
	log     *zap.Logger
	fontDir string
	mu      sync.RWMutex
	ready   bool
}

// NewApp инициализирует и запускает приложение.
func NewApp(wkhtmltopdfWindows []byte, fontFS, frontendFS embed.FS) error {
	a := &App{}

	// 1. Конфигурация
	cfg, err := config.NewConfig()
	if err != nil {
		return fmt.Errorf("failed to init config: %w", err)
	}
	a.cfg = cfg

	// 2. Логгер
	logInstance, err := logger.New(cfg.Logger)
	if err != nil {
		return fmt.Errorf("failed to init logger: %w", err)
	}
	a.log = logInstance
	a.log.Info("Starting Lab Desktop Application (HTTP Mode)")

	// 3. Выбор БД (если разрешено)
	if a.cfg.DB.AllowSelection {
		if err := a.selectDatabase(); err != nil {
			return err
		}
	}

	// 4. Шрифты
	fontDir, err := font.ExtractFonts(fontFS)
	if err != nil {
		return fmt.Errorf("failed to extract fonts: %w", err)
	}
	a.fontDir = fontDir

	// 5. Подключение к БД
	dbConn, err := db.New(a.cfg.DB.Path, a.log)
	if err != nil {
		return fmt.Errorf("db connection failed: %w", err)
	}
	a.dbConn = dbConn

	// 6. Миграции
	if err := a.runMigrations(dbConn); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// 7. Инициализация слоев приложения
	repos := repository.NewRepository(dbConn, a.log)
	svc := a.initServices(repos, wkhtmltopdfWindows)
	handl := a.initHandlers(svc, frontendFS)

	a.ready = true
	a.log.Info("Application initialized successfully")

	// 8. Запуск сервера
	return a.runServer(handl)
}

// selectDatabase открывает диалог выбора файла БД.
func (a *App) selectDatabase() error {
	selectedPath, ok, err := dlgs.File(
		"Выберите файл базы данных",
		"*.db *.sqlite",
		false,
	)
	if err != nil {
		return fmt.Errorf("file dialog error: %w", err)
	}
	if !ok {
		a.log.Info("Database selection cancelled by user")
		return nil
	}

	if _, statErr := os.Stat(selectedPath); os.IsNotExist(statErr) {
		a.log.Warn("Selected database file does not exist, will create new",
			zap.String("path", selectedPath))
	}

	a.cfg.DB.Path = selectedPath
	a.log.Info("Database selected", zap.String("path", selectedPath))
	return nil
}

// initServices инициализирует сервисы.
func (a *App) initServices(repos *repository.Repository, wkhtmltopdfWindows []byte) *service.Services {
	svc := service.NewServices(
		repos.Material, repos.Standard, repos.Protocol, repos.Sample, repos.Group, repos.Token, repos.User, repos.Dimension, repos.Organization, repos.OrganizationTests, repos.OrganizationUser, repos.Order, repos.Invitation,
		a.fontDir, "templates", wkhtmltopdfWindows, *a.cfg, a.log,
	)

	if err := data.SeedData(svc, a.log); err != nil {
		a.log.Error("Seed failed", zap.Error(err))
	} else {
		a.log.Info("Data seed completed")
	}

	return svc
}

// initHandlers инициализирует обработчики.
func (a *App) initHandlers(svc *service.Services, frontendFS embed.FS) *handler.Handler {
	handl := handler.NewHandler(
		svc.Auth, svc.Dimensions, svc.Invitations, svc.Materials, svc.Groups, svc.Organizations, svc.Orders, svc.Profile, svc.Protocols, svc.Reports, svc.Samples, svc.Standards,
		a, a.log, a.cfg.Auth.RefreshTokenTTL,
	)
	handl.SetFrontendFS(frontendFS)
	return handl
}

// runServer запускает HTTP-сервер и обрабатывает сигналы завершения.
func (a *App) runServer(h *handler.Handler) error {
	a.log.Info("Starting server", zap.String("address", a.cfg.Server.Host+":"+a.cfg.Server.Port))
	server := server.NewServer(a.cfg.Server, h.Init())

	go func() {
		if err := server.Start(); err != nil {
			a.log.Fatal("Stop server", zap.Error(err))
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	<-exit

	if err := server.Shutdown(context.Background()); err != nil {
		a.log.Fatal("Failed to shutdown server", zap.Error(err))
	}

	a.log.Info("Shutting down")
	return nil
}

// SwitchDatabase безопасно переключает подключение к новой БД с атомарностью.
func (a *App) SwitchDatabase(newPath string) error {
	a.cfgMu.Lock()
	defer a.cfgMu.Unlock()

	// Проверяем, что newPath отличается от текущего пути
	if a.cfg.DB.Path == newPath {
		a.log.Debug("database path unchanged", zap.String("path", newPath))
		return nil
	}

	oldConn := a.dbConn

	// Создаем новое подключение
	newConn, err := db.New(newPath, a.log)
	if err != nil {
		return fmt.Errorf("failed to connect to new DB: %w", err)
	}

	// Выполняем миграции на новом подключении перед переключением
	if err := a.runMigrations(newConn); err != nil {
		newConn.Close()
		return fmt.Errorf("migration on new DB failed: %w", err)
	}

	// Атомарно переключаем подключение
	a.dbConn = newConn
	a.cfg.DB.Path = newPath

	// Закрываем старое подключение после успешного переключения
	if oldConn != nil {
		if err := oldConn.Close(); err != nil {
			a.log.Error("Failed to close old DB", zap.Error(err))
		}
	}

	a.log.Info("Database switched successfully", zap.String("path", newPath))
	return nil
}

// IsReady возвращает статус готовности приложения.
func (a *App) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ready
}

// runMigrations выполняет миграции БД.
func (a *App) runMigrations(dbConn *sqlx.DB) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execDir := filepath.Dir(execPath)
	paths := []string{
		filepath.Join("internal", "db", "migration_sqlite"),
		filepath.Join(execDir, "internal", "db", "migration_sqlite"),
		filepath.Join(execDir, "migration_sqlite"),
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			a.log.Info("🔄 Running migrations", zap.String("path", p))
			if err := db.RunMigrations(dbConn, p, a.log); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
			return nil
		}
	}
	a.log.Warn("Migration directory not found - skipping")
	return nil
}

// GetDBPath возвращает текущий путь к БД.
func (a *App) GetDBPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg.DB.Path
}
