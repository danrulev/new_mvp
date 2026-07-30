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

type App struct {
	dbConn  *sqlx.DB
	cfgMu   sync.RWMutex
	cfg     *config.Config
	log     *zap.Logger
	fontDir string
	mu      sync.RWMutex
	ready   bool
}

// ============================================================================
// ИНИЦИАЛИЗАЦИЯ
// ============================================================================
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

	if a.cfg.DB.AllowSelection {
		selectedPath, ok, err := dlgs.File(
			"Выберите файл базы данных",
			"*.db *.sqlite",
			false, // ← важно: false для выбора файла
		)
		if err != nil {
			return fmt.Errorf("file dialog error: %w", err)
		}
		if !ok {
			a.log.Info("Database selection cancelled by user")
			return nil // или верните ошибку, если отмена недопустима
		}

		if _, statErr := os.Stat(selectedPath); os.IsNotExist(statErr) {
			a.log.Warn("Selected database file does not exist, will create new",
				zap.String("path", selectedPath))
		}

		cfg.DB.Path = selectedPath
		a.log.Info("Database selected", zap.String("path", selectedPath))
	}

	// 3. Шрифты
	fontDir, err := font.ExtractFonts(fontFS)
	if err != nil {
		return fmt.Errorf("failed to extract fonts: %w", err)
	}
	a.fontDir = fontDir

	// 4. БД
	dbConn, err := db.New(cfg.DB.Path, a.log)
	if err != nil {
		return fmt.Errorf("db connection failed: %w", err)
	}
	a.dbConn = dbConn

	// 5. Миграции
	if err := a.runMigrations(dbConn); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	repos := repository.NewRepository(dbConn, a.log)

	svc := service.NewServices(
		repos.Material, repos.Standard, repos.Protocol, repos.Sample, repos.Group, repos.Token, repos.User, repos.Dimension,
		a.fontDir, "templates", wkhtmltopdfWindows, *a.cfg, a.log,
	)

	if err := data.SeedData(svc, a.log); err != nil {
		a.log.Error("Seed failed", zap.Error(err))
	} else {
		a.log.Info("Data seed completed")
	}

	handl := handler.NewHandler(
		svc.Auth, svc.Dimensions, svc.Materials, svc.Groups, svc.Protocols, svc.Reports, svc.Samples, svc.Standards,
		a, a.log, a.cfg.Auth.RefreshTokenTTL,
	)

	handl.SetFrontendFS(frontendFS)

	a.ready = true
	a.log.Info("Application initialized successfully")

	a.log.Info("Starting server", zap.String("address", cfg.Server.Host+":"+cfg.Server.Port))
	server := server.NewServer(cfg.Server, handl.Init())

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

// Метод для безопасной смены БД
func (a *App) SwitchDatabase(newPath string) error {
	a.cfgMu.Lock()
	defer a.cfgMu.Unlock()

	if a.dbConn != nil {
		if err := a.dbConn.Close(); err != nil {
			a.log.Error("Failed to close old DB", zap.Error(err))
		}
	}

	a.cfg.DB.Path = newPath

	newConn, err := db.New(newPath, a.log)
	if err != nil {
		return fmt.Errorf("failed to connect to new DB: %w", err)
	}
	a.dbConn = newConn

	if err := a.runMigrations(newConn); err != nil {
		return fmt.Errorf("migration on new DB failed: %w", err)
	}

	a.log.Info("Database switched successfully", zap.String("path", newPath))
	return nil
}

func (a *App) IsReady() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.ready
}

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

func (a *App) GetDBPath() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.cfg.DB.Path
}
