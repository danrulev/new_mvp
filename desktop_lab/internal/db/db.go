package db

import (
	"context"
	"desktop_lab/pkg/retry"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// DBConfig конфигурация подключения к базе данных.
type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultDBConfig возвращает конфигурацию по умолчанию для SQLite.
func DefaultDBConfig() DBConfig {
	return DBConfig{
		MaxOpenConns:    4, // Увеличено с 1 до 4 для лучшей конкурентности
		MaxIdleConns:    4, // Увеличено с 1 до 4
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 5 * time.Minute,
	}
}

func New(dbPath string, log *zap.Logger) (*sqlx.DB, error) {
	return NewWithConfig(dbPath, log, DefaultDBConfig())
}

func NewWithConfig(dbPath string, log *zap.Logger, cfg DBConfig) (*sqlx.DB, error) {
	dsn := fmt.Sprintf("file:%s?_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL", dbPath)

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	log.Info("SQLite database connected",
		zap.String("path", dbPath),
		zap.Int("max_open_conns", cfg.MaxOpenConns),
		zap.Int("max_idle_conns", cfg.MaxIdleConns))

	return db, nil
}

// ExecWithRetry выполняет SQL запрос с повторными попытками при временных ошибках.
func ExecWithRetry(ctx context.Context, db *sqlx.DB, log *zap.Logger, query string, args ...interface{}) error {
	cfg := retry.DefaultConfig()

	return retry.DoVoid(ctx, cfg, func() error {
		_, err := db.ExecContext(ctx, query, args...)
		if err != nil && retry.IsRetryable(err) {
			log.Debug("Retrying database operation",
				zap.String("query", query),
				zap.Error(err))
			return err
		}
		return err
	})
}

// QueryWithRetry выполняет SQL запрос с возвратом результата и повторными попытками.
// Для Go 1.19 используется interface{} вместо дженериков.
func QueryWithRetry(ctx context.Context, db *sqlx.DB, log *zap.Logger, query string, args ...interface{}) ([]interface{}, error) {
	cfg := retry.DefaultConfig()

	result, err := retry.Do(ctx, cfg, func() (interface{}, error) {
		var results []interface{}
		err := db.SelectContext(ctx, &results, query, args...)
		if err != nil && retry.IsRetryable(err) {
			log.Debug("Retrying database query",
				zap.String("query", query),
				zap.Error(err))
			return nil, err
		}
		return results, err
	})
	if err != nil {
		return nil, err
	}

	// Преобразуем результат в []interface{}
	if resultSlice, ok := result.([]interface{}); ok {
		return resultSlice, nil
	}

	return []interface{}{result}, nil
}
