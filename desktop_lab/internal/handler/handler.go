package handler

import (
	"desktop_lab/internal/service"
	"desktop_lab/pkg/ratelimiter"
	"embed"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DatabaseSwitcher определяет интерфейс для переключения БД.
type DatabaseSwitcher interface {
	SwitchDatabase(newPath string) error
	GetDBPath() string
}

// Handler содержит все обработчики HTTP-запросов.
type Handler struct {
	log             *zap.Logger
	auth            *service.AuthService
	dimension       *service.DimensionService
	invitation      *service.InvitationService
	material        *service.MaterialService
	profile         *service.ProfileService
	group           *service.ExperimentGroupService
	order           *service.OrderService
	protocol        *service.ProtocolService
	report          *service.ReportService
	sample          *service.SampleService
	standard        *service.StandardService
	organization    *service.OrganizationService
	qualityControl  *service.QualityControlService
	analytics       *service.AnalyticsService
	priceList       *service.PriceListService
	invoice         *service.InvoiceService
	payment         *service.PaymentService
	paymentGateway  *service.PaymentGatewayService
	appRef          DatabaseSwitcher
	frontendFS      embed.FS
	frontendFSReady bool
	refreshTokenTTL time.Duration
	rateLimiter     *ratelimiter.RateLimiter
}

// NewHandler создает новый экземпляр Handler.
func NewHandler(
	auth *service.AuthService,
	dimension *service.DimensionService,
	invitation *service.InvitationService,
	material *service.MaterialService,
	group *service.ExperimentGroupService,
	organization *service.OrganizationService,
	order *service.OrderService,
	profile *service.ProfileService,
	protocol *service.ProtocolService,
	report *service.ReportService,
	sample *service.SampleService,
	standard *service.StandardService,
	qualityControl *service.QualityControlService,
	analytics *service.AnalyticsService,
	priceList *service.PriceListService,
	invoice *service.InvoiceService,
	payment *service.PaymentService,
	paymentGateway *service.PaymentGatewayService,
	appRef DatabaseSwitcher,
	log *zap.Logger,
	refreshTokenTTL time.Duration,
) *Handler {
	// Создаем rate limiter: 10 запросов в секунду с burst до 20
	rateLimiter := ratelimiter.NewRateLimiter(20, 10)

	return &Handler{
		auth:            auth,
		dimension:       dimension,
		invitation:      invitation,
		material:        material,
		group:           group,
		profile:         profile,
		protocol:        protocol,
		report:          report,
		sample:          sample,
		standard:        standard,
		organization:    organization,
		order:           order,
		qualityControl:  qualityControl,
		analytics:       analytics,
		priceList:       priceList,
		invoice:         invoice,
		payment:         payment,
		paymentGateway:  paymentGateway,
		appRef:          appRef,
		log:             log,
		refreshTokenTTL: refreshTokenTTL,
		rateLimiter:     rateLimiter,
	}
}

// SetFrontendFS устанавливает файловую систему для фронтенда.
func (h *Handler) SetFrontendFS(fs embed.FS) {
	h.frontendFS = fs
	h.frontendFSReady = true
}

// Init инициализирует и настраивает HTTP-роутер.
func (h *Handler) Init() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery(), h.logging(), h.rateLimitMiddleware(h.rateLimiter))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API Routes
	api := router.Group("/api/v1")
	h.initAuthRoutes(api)
	h.initMaterialRoutes(api)
	h.initStandardRoutes(api)
	h.initSampleRoutes(api)
	h.initGroupRoutes(api)
	h.initDimensionRoutes(api)
	h.initProtocolRoutes(api)
	h.initReportRoutes(api)
	h.initDBRoutes(api)
	h.initOrganizationRoutes(api)
	h.initProfileRoutes(api)
	h.initOrderRoutes(api)
	h.initInvitationRoutes(api)
	h.initQualityControlRoutes(api)
	h.initAnalyticsRoutes(api)
	h.initFinanceRoutes(api)

	// Frontend routes (SPA)
	router.NoRoute(h.serveFrontend)

	return router
}

// serveFrontend обрабатывает маршруты фронтенда.
func (h *Handler) serveFrontend(c *gin.Context) {
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

	// Нормализуем путь
	filePath := strings.TrimPrefix(path, "/")
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := "frontend/" + filePath
	data, err := h.frontendFS.ReadFile(fullPath)
	if err == nil {
		contentType := getContentType(filePath)
		c.Header("Content-Type", contentType)
		if strings.HasSuffix(filePath, ".html") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		} else {
			c.Header("Cache-Control", "public, max-age=86400")
		}
		c.Data(http.StatusOK, contentType, data)
		return
	}

	// SPA роутинг: отдаем index.html для путей без расширения
	if !strings.Contains(filePath, ".") {
		indexData, errIndex := h.frontendFS.ReadFile("frontend/index.html")
		if errIndex == nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Data(http.StatusOK, "text/html; charset=utf-8", indexData)
			return
		}
	}

	c.Status(http.StatusNotFound)
}

// getContentType возвращает MIME-тип по расширению файла.
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
