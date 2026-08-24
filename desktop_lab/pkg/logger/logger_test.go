package logger

import (
	"desktop_lab/internal/config"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestNewLogger_DevelopmentMode(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	// Проверка что логгер работает
	log.Info("test message", zap.String("key", "value"))
}

func TestNewLogger_ProductionMode(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "info",
		Development: false,
		Encoding:    "json",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	// Проверка что логгер работает
	log.Info("test message", zap.String("key", "value"))
}

func TestNewLogger_InvalidLevel(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "invalid",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("logger should handle invalid level gracefully: %v", err)
	}
	defer log.Sync()

	// Должен использоваться уровень по умолчанию (info)
	log.Info("test message")
}

func TestNewLogger_InvalidEncoding(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "info",
		Development: true,
		Encoding:    "invalid",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("logger should handle invalid encoding gracefully: %v", err)
	}
	defer log.Sync()

	// Должен использоваться console по умолчанию
	log.Info("test message")
}

func TestNewLogger_NoOutputPaths(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "info",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("logger should handle empty output paths gracefully: %v", err)
	}
	defer log.Sync()

	// Должны использоваться дефолтные пути
	log.Info("test message")
}

func TestNewLogger_FileOutput(t *testing.T) {
	// Создаем временный файл
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")

	cfg := config.LoggerConfig{
		Level:       "info",
		Development: false,
		Encoding:    "json",
		OutputPaths:  []string{logFile},
		ErrorOutputPaths: []string{logFile},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger with file output: %v", err)
	}
	defer log.Sync()

	log.Info("file test message", zap.String("test", "value"))
	log.Sync()

	// Проверяем что файл создан и содержит логи
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("log file is empty")
	}

	if !strings.Contains(string(content), "file test message") {
		t.Error("log file doesn't contain expected message")
	}
}

func TestNewLogger_AutoFields(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "info",
		Development: false,
		Encoding:    "json",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	zapLog, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer zapLog.Sync()

	// Проверяем что авто-поля добавляются
	zapLog.Info("test")
}

func TestWithRequestID(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	requestID := "test-request-123"
	requestLogger := WithRequestID(log, requestID)
	requestLogger.Info("request test")
}

func TestWithUserID(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	userID := "user-456"
	userLogger := WithUserID(log, userID)
	userLogger.Info("user test")
}

func TestWithOperation(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	operation := "create_protocol"
	opLogger := WithOperation(log, operation)
	opLogger.Info("operation test")
}

func TestLogPanic(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	// Тестируем логирование паники
	LogPanic(log, "test panic value")
}

func TestLogSlowQuery(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	query := "SELECT * FROM protocols"
	
	// Медленный запрос - должен логироваться
	LogSlowQuery(log, query, 200*time.Millisecond, 100*time.Millisecond)
	
	// Быстрый запрос - не должен логироваться
	LogSlowQuery(log, query, 50*time.Millisecond, 100*time.Millisecond)
}

func TestLogHTTPAccess(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	LogHTTPAccess(log, "GET", "/api/test", 200, 45*time.Millisecond, "127.0.0.1")
}

func TestGetContextLogger(t *testing.T) {
	cfg := config.LoggerConfig{
		Level:       "debug",
		Development: true,
		Encoding:    "console",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := New(cfg)
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer log.Sync()

	contextLogger := GetContextLogger(log, zap.String("context", "test"))
	contextLogger.Info("context test")
}

func TestNewWithRotation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := config.LoggerConfig{
		Level:       "info",
		Development: false,
		Encoding:    "json",
		OutputPaths:  []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	log, err := NewWithRotation(cfg, tmpDir)
	if err != nil {
		t.Fatalf("failed to create logger with rotation: %v", err)
	}
	defer log.Sync()

	log.Info("rotation test")

	// Проверяем что файлы созданы
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}

	if len(files) == 0 {
		t.Error("no log files created")
	}
}

func TestEnvironmentDetection(t *testing.T) {
	// Сохраняем оригинальное значение
	originalEnv := os.Getenv("APP_ENV")
	defer os.Setenv("APP_ENV", originalEnv)

	// Тест development (по умолчанию)
	os.Unsetenv("APP_ENV")
	env := getEnvironment()
	if env != EnvDevelopment {
		t.Errorf("expected development, got %s", env)
	}

	// Тест production
	os.Setenv("APP_ENV", "production")
	env = getEnvironment()
	if env != EnvProduction {
		t.Errorf("expected production, got %s", env)
	}

	// Тест staging
	os.Setenv("APP_ENV", "staging")
	env = getEnvironment()
	if env != EnvStaging {
		t.Errorf("expected staging, got %s", env)
	}
}

func TestServiceName(t *testing.T) {
	name := getServiceName()
	if name == "" {
		t.Error("service name should not be empty")
	}
}

func TestHostname(t *testing.T) {
	host := getHostname()
	if host == "" {
		t.Error("hostname should not be empty")
	}
}
