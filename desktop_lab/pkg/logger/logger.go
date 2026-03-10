package logger

import (
	"desktop_lab/internal/config"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
)

func New(cfg config.LoggerConfig) (*zap.Logger, error) {
	var configErrors []error

	level := zap.NewAtomicLevel()
	err := level.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		configErrors = append(configErrors, fmt.Errorf("invalid log level '%s': %w", cfg.Level, err))
		level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	encoding := strings.ToLower(cfg.Encoding)
	if encoding != "json" && encoding != "console" {
		configErrors = append(configErrors, fmt.Errorf("invalid encoding '%s': must be 'json' or 'console'", cfg.Encoding))
		encoding = "console"
	}

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

	encoderConfig := zap.NewProductionEncoderConfig()
	if cfg.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

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

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	for _, err := range configErrors {
		zapLogger.Warn("logger config error", zap.Error(err))
	}

	return zapLogger, nil
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
