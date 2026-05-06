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

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type App struct {
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

	// 5. Миграции
	if err := a.runMigrations(dbConn); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// 6. Репозитории и Сервисы
	matRepo := repository.NewMaterialRepo(dbConn, a.log)
	stdRepo := repository.NewStandardRepo(dbConn, a.log)
	protRepo := repository.NewProtocolRepo(dbConn, a.log)
	sampRepo := repository.NewSampleRepo(dbConn, a.log)
	groupRepo := repository.NewExperimentGroupRepo(dbConn, a.log)
	dimRepo := repository.NewDimensionRepo(dbConn, a.log)

	svc := service.NewServices(
		matRepo, stdRepo, protRepo, sampRepo, groupRepo, dimRepo,
		a.fontDir, "templates", wkhtmltopdfWindows, a.log,
	)

	if err := data.SeedData(svc, a.log); err != nil {
		a.log.Error("Seed failed", zap.Error(err))
		// Не прерываем запуск, но логируем ошибку
	} else {
		a.log.Info("Data seed completed")
	}

	handl := handler.NewHandler(svc.Dimensions, svc.Materials, svc.Groups, svc.Protocols, svc.Reports, svc.Samples, svc.Standards, a.log)

	handl.SetFrontendFS(frontendFS)

	a.ready = true
	a.log.Info("Application initialized successfully")

	server := server.NewServer(cfg.Server, handl.Init())

	go func() {
		if err := server.Start(); err != nil {
			a.log.Fatal("Failed to start server", zap.Error(err))
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
		filepath.Join("internal", "db", "migration"),
		filepath.Join(execDir, "internal", "db", "migration"),
		filepath.Join(execDir, "migration"),
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
