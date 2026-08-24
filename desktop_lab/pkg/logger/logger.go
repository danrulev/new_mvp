package logger

import (
	"desktop_lab/internal/config"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// AppVersion устанавливается при сборке через ldflags
var AppVersion = "dev"

// Environment определяет окружение приложения
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// LoggerConfig расширенная конфигурация для production
type LoggerConfig struct {
	Level             string   `mapstructure:"level" validate:"required"`
	Development       bool     `mapstructure:"development"`
	DisableCaller     bool     `mapstructure:"disable_caller"`
	DisableStacktrace bool     `mapstructure:"disable_stacktrace"`
	Encoding          string   `mapstructure:"encoding" validate:"required"`
	OutputPaths       []string `mapstructure:"output_paths"`
	ErrorOutputPaths  []string `mapstructure:"error_output_paths"`
	
	// Production настройки
	Environment        string `mapstructure:"environment"`
	ServiceName        string `mapstructure:"service_name"`
	EnableSampling     bool   `mapstructure:"enable_sampling"`
	SamplingInitial    int    `mapstructure:"sampling_initial"`
	SamplingThereafter int    `mapstructure:"sampling_thereafter"`
	MaxFileSize        int    `mapstructure:"max_file_size_mb"`
	MaxBackups         int    `mapstructure:"max_backups"`
	MaxAge             int    `mapstructure:"max_age_days"`
	Compress           bool   `mapstructure:"compress_logs"`
}

// New создает production-ready logger с расширенными возможностями
func New(cfg config.LoggerConfig) (*zap.Logger, error) {
	var configErrors []error

	// Парсинг уровня логирования
	level := zap.NewAtomicLevel()
	err := level.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		configErrors = append(configErrors, fmt.Errorf("invalid log level '%s': %w", cfg.Level, err))
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	// Определение кодировщика
	encoding := strings.ToLower(cfg.Encoding)
	if encoding != "json" && encoding != "console" {
		configErrors = append(configErrors, fmt.Errorf("invalid encoding '%s': must be 'json' or 'console'", cfg.Encoding))
		encoding = "console"
	}

	// Настройка путей вывода
	outputPaths, err := setUpOutput(cfg.OutputPaths)
	if err != nil {
		configErrors = append(configErrors, err)
		outputPaths = []string{"stderr"}
	}

	errorOutputPaths, err := setUpOutput(cfg.ErrorOutputPaths)
	if err != nil {
		configErrors = append(configErrors, err)
		errorOutputPaths = []string{"stderr"}
	}

	// Конфигурация энкодера
	encoderConfig := newEncoderConfig(cfg.Development, encoding)

	// Создание базовой конфигурации Zap
	zapConfig := zap.Config{
		Level:             level,
		Development:       cfg.Development,
		DisableCaller:     cfg.DisableCaller,
		DisableStacktrace: cfg.DisableStacktrace,
		Encoding:          encoding,
		EncoderConfig:     encoderConfig,
		OutputPaths:       outputPaths,
		ErrorOutputPaths:  errorOutputPaths,
	}

	// Production оптимизации
	if !cfg.Development {
		// Включаем сэмплинг для production
		zapConfig.Sampling = &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
		
		// В production всегда используем JSON формат и выключаем caller для производительности
		if encoding == "console" {
			zapConfig.Encoding = "json"
			zapConfig.EncoderConfig = newEncoderConfig(false, "json")
		}
		zapConfig.DisableCaller = true
	}

	// Сборка логгера
	zapLogger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	// Логирование ошибок конфигурации
	for _, err := range configErrors {
		zapLogger.Warn("logger config error", zap.Error(err))
	}

	// Добавление полей по умолчанию
	defaultFields := []zap.Field{
		zap.String("service", getServiceName()),
		zap.String("version", AppVersion),
		zap.String("environment", string(getEnvironment())),
		zap.String("hostname", getHostname()),
	}
	
	zapLogger = zapLogger.With(defaultFields...)

	return zapLogger, nil
}

// newEncoderConfig создает конфигурацию энкодера в зависимости от режима
func newEncoderConfig(isDevelopment bool, encoding string) zapcore.EncoderConfig {
	if isDevelopment && encoding == "console" {
		return zap.NewDevelopmentEncoderConfig()
	}

	// Production encoder config
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "timestamp"
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.LevelKey = "level"
	cfg.NameKey = "logger"
	cfg.CallerKey = "caller"
	cfg.FunctionKey = ""
	cfg.MessageKey = "message"
	cfg.StacktraceKey = "stacktrace"
	cfg.LineEnding = zapcore.DefaultLineEnding
	cfg.EncodeLevel = zapcore.LowercaseLevelEncoder
	cfg.EncodeDuration = zapcore.MillisDurationEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	
	return cfg
}

// NewWithRotation создает logger с ротацией файлов
func NewWithRotation(cfg config.LoggerConfig, logDir string) (*zap.Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Создаем файлы для логов
	timestamp := time.Now().Format("20060102_150405")
	infoLogPath := filepath.Join(logDir, fmt.Sprintf("app_%s.log", timestamp))
	errorLogPath := filepath.Join(logDir, fmt.Sprintf("error_%s.log", timestamp))

	// Используем lumberjack для ротации (если доступен)
	// Для простоты пока используем стандартные файлы
	outputPaths := []string{infoLogPath, "stdout"}
	errorOutputPaths := []string{errorLogPath, "stderr"}

	cfg.OutputPaths = outputPaths
	cfg.ErrorOutputPaths = errorOutputPaths

	return New(cfg)
}

// GetContextLogger извлекает logger из контекста или создает новый
func GetContextLogger(base *zap.Logger, fields ...zap.Field) *zap.Logger {
	return base.With(fields...)
}

// WithRequestID добавляет request_id к logger
func WithRequestID(logger *zap.Logger, requestID string) *zap.Logger {
	return logger.With(zap.String("request_id", requestID))
}

// WithUserID добавляет user_id к logger
func WithUserID(logger *zap.Logger, userID string) *zap.Logger {
	return logger.With(zap.String("user_id", userID))
}

// WithOperation добавляет operation к logger
func WithOperation(logger *zap.Logger, operation string) *zap.Logger {
	return logger.With(zap.String("operation", operation))
}

// Helper функции для различных типов логирования

// LogPanic логирует панику и добавляет stack trace
func LogPanic(logger *zap.Logger, r interface{}) {
	logger.Error("panic recovered",
		zap.Any("panic_value", r),
		zap.Stack("stack"),
	)
}

// LogSlowQuery логирует медленные SQL запросы
func LogSlowQuery(logger *zap.Logger, query string, duration time.Duration, threshold time.Duration) {
	if duration > threshold {
		logger.Warn("slow query detected",
			zap.String("query", query),
			zap.Duration("duration_ms", duration),
			zap.Duration("threshold_ms", threshold),
		)
	}
}

// LogHTTPAccess логирует HTTP запросы в формате access log
func LogHTTPAccess(logger *zap.Logger, method, path string, statusCode int, latency time.Duration, clientIP string) {
	logger.Info("http_access",
		zap.String("method", method),
		zap.String("path", path),
		zap.Int("status", statusCode),
		zap.Duration("latency_ms", latency),
		zap.String("client_ip", clientIP),
	)
}

// Вспомогательные функции

func getServiceName() string {
	executable, err := os.Executable()
	if err != nil {
		return "desktop-lab"
	}
	return filepath.Base(executable)
}

func getEnvironment() Environment {
	env := os.Getenv("APP_ENV")
	if env == "" {
		return EnvDevelopment
	}
	return Environment(env)
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func setUpOutput(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no output paths provided")
	}

	var (
		result []string
		issues []string
	)

	seen := make(map[string]bool)

	for _, path := range paths {
		if seen[path] {
			continue
		}

		seen[path] = true

		switch path {
		case "stdout", "stderr":
			result = append(result, path)
		default:
			// Создаем директорию если не существует
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				issues = append(issues, fmt.Sprintf("cannot create directory for '%s': %v", path, err))
				continue
			}
			
			f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				issues = append(issues, fmt.Sprintf("cannot open '%s': %v", path, err))
				continue
			}
			f.Close()
			result = append(result, path)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid output paths after processing")
	}

	if len(issues) > 0 {
		return result, fmt.Errorf("some paths had issues: %s", strings.Join(issues, "; "))
	}

	return result, nil
}
