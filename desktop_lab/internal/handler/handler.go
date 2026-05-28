package handler

import (
	"desktop_lab/internal/service"
	"embed"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DatabaseSwitcher interface {
	SwitchDatabase(newPath string) error
	GetDBPath() string
}

type Handler struct {
	log       *zap.Logger
	auth      *service.AuthService
	dimension *service.DimensionService
	material  *service.MaterialService
	group     *service.ExperimentGroupService
	protocol  *service.ProtocolService
	report    *service.ReportService
	sample    *service.SampleService
	standard  *service.StandardService

	appRef DatabaseSwitcher

	frontendFS      embed.FS
	frontendFSReady bool

	refreshTokenTTL time.Duration
}

func NewHandler(
	dimension *service.DimensionService,
	material *service.MaterialService,
	group *service.ExperimentGroupService,
	protocol *service.ProtocolService,
	report *service.ReportService,
	sample *service.SampleService,
	standard *service.StandardService,

	appRef DatabaseSwitcher,

	log *zap.Logger,
) *Handler {
	return &Handler{
		dimension: dimension,
		material:  material,
		group:     group,
		protocol:  protocol,
		report:    report,
		sample:    sample,
		standard:  standard,
		appRef:    appRef,
		log:       log,
	}
}

func (h *Handler) SetFrontendFS(fs embed.FS) {
	h.frontendFS = fs
	h.frontendFSReady = true
}

func (h *Handler) Init() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin.Recovery(),
		h.logging(),
	)

	// 1. Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// 2. API Routes
	// Используем /api/v1 как основной префикс, чтобы соответствовать ожиданиям фронтенда
	api := router.Group("/api/v1")
	{
		h.initMaterialRoutes(api)
		h.initStandardRoutes(api)
		h.initSampleRoutes(api)
		h.initGroupRoutes(api)
		h.initDimensionRoutes(api)
		h.initProtocolRoutes(api)
		h.initReportRoutes(api)
		h.initDBRoutes(api)
	}

	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// Игнорируем API
		if strings.HasPrefix(path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}

		if !h.frontendFSReady {
			c.Status(http.StatusNotFound)
			return
		}

		// Нормализуем путь: убираем ведущий слеш
		filePath := strings.TrimPrefix(path, "/")

		// Если путь пустой (корень /), отдаем index.html
		if filePath == "" {
			filePath = "index.html"
		}

		// Пытаемся прочитать файл из embed
		// Важно: файлы в embed лежат в папке frontend/
		fullPath := "frontend/" + filePath

		data, err := h.frontendFS.ReadFile(fullPath)
		if err == nil {
			// Файл найден
			c.Header("Content-Type", getContentType(filePath))
			// Кэшируем статику, но не HTML
			if !strings.HasSuffix(filePath, ".html") {
				c.Header("Cache-Control", "public, max-age=86400")
			} else {
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			}
			c.Data(http.StatusOK, getContentType(filePath), data)
			return
		}

		// Если файл не найден и это не запрос к HTML-странице (SPA роутинг),
		// пробуем отдать index.html для поддержки клиентского роутинга
		// Но только если в пути нет точки (чтобы не ломать 404 для missing.css)
		if !strings.Contains(filePath, ".") {
			indexData, errIndex := h.frontendFS.ReadFile("frontend/index.html")
			if errIndex == nil {
				c.Header("Content-Type", "text/html; charset=utf-8")
				c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
				c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
				return
			}
		}

		// Если ничего не помогло
		c.Status(http.StatusNotFound)
	})

	return router
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(path, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}
