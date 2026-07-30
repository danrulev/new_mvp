package mysql_repo

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// timeLayout - единый формат времени для всех репозиториев
const timeLayout = "2006-01-02 15:04:05"

// helperParseTime безопасно парсит время из строки (ожидается UTC в БД) и возвращает локальное время
func helperParseTime(raw interface{}) (time.Time, error) {
	if raw == nil {
		return time.Time{}, nil
	}

	str, ok := raw.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("expected string for time, got %T", raw)
	}

	return time.Parse(timeLayout, str)
}

// helperParseTimeMaterial безопасно парсит время из строки (ожидается UTC в БД) и возвращает локальное время
func helperParseTimeMaterial(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	t, err := time.ParseInLocation(timeLayout, timeStr, time.UTC)
	if err != nil {
		return time.Time{}, err
	}

	return t.Local(), nil
}

// helperParseTimeWithLog парсит время с логированием ошибок
func helperParseTimeWithLog(ctx context.Context, log *zap.Logger, raw string, fieldName, fallbackReason string) time.Time {
	if raw == "" {
		return time.Time{}
	}

	t, err := time.Parse(timeLayout, raw)
	if err != nil {
		log.Warn("failed to parse time field",
			zap.String("field", fieldName),
			zap.String("raw_value", raw),
			zap.Error(err),
			zap.String("fallback_reason", fallbackReason))
		return time.Now()
	}

	return t
}

// helperCheckContext проверяет контекст на отмену перед выполнением операции
func helperCheckContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
		return nil
	}
}
