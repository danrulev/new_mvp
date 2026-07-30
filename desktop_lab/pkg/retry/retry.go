// Package retry предоставляет механизмы повторных попыток для операций с БД и другими ресурсами.
package retry

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Config конфигурирует стратегию повторных попыток.
type Config struct {
	MaxRetries   int           // Максимальное количество попыток
	InitialDelay time.Duration // Начальная задержка перед первой повторной попыткой
	Multiplier   float64       // Множитель задержки (exponential backoff)
	MaxDelay     time.Duration // Максимальная задержка между попытками
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() Config {
	return Config{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		Multiplier:   2.0,
		MaxDelay:     2 * time.Second,
	}
}

// ErrMaxRetriesExceeded ошибка, указывающая на превышение максимального количества попыток.
var ErrMaxRetriesExceeded = errors.New("max retries exceeded")

// Operation функция, которую нужно выполнить с повторными попытками.
// Возвращает результат и ошибку. Если ошибка nil, операция считается успешной.
// Для Go 1.19 используется interface{} вместо дженериков.
type Operation func() (interface{}, error)

// Do выполняет операцию с повторными попытками согласно конфигурации.
// Использует exponential backoff стратегию для задержек между попытками.
// Контекст может быть использован для отмены операции.
func Do(ctx context.Context, cfg Config, op Operation) (interface{}, error) {
	var (
		result interface{}
		err    error
		delay  = cfg.InitialDelay
	)

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Проверяем контекст перед каждой попыткой
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		result, err = op()
		if err == nil {
			return result, nil
		}

		// Если это последняя попытка, возвращаем ошибку сразу
		if attempt >= cfg.MaxRetries {
			return result, fmt.Errorf("%w after %d attempts: %v", ErrMaxRetriesExceeded, cfg.MaxRetries+1, err)
		}

		// Ждем перед следующей попыткой
		select {
		case <-ctx.Done():
			return result, fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
		case <-time.After(delay):
		}

		// Увеличиваем задержку для следующей попытки (exponential backoff)
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return result, err
}

// DoVoid выполняет операцию без возврата значения с повторными попытками.
// Удобно для операций записи/обновления/удаления.
func DoVoid(ctx context.Context, cfg Config, op func() error) error {
	_, err := Do(ctx, cfg, func() (struct{}, error) {
		return struct{}{}, op()
	})
	return err
}

// IsRetryable определяет, является ли ошибка потенциально временной.
// Может быть расширено для конкретных типов ошибок БД.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// SQLite busy/locked ошибки
	retryableErrors := []string{
		"database is locked",
		"database is busy",
		"locking protocol",
		"SQLITE_BUSY",
		"SQLITE_LOCKED",
		"context deadline exceeded",
		"connection refused",
		"temporary failure",
	}

	for _, re := range retryableErrors {
		if contains(errStr, re) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
