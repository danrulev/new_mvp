package mysql_repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// timeLayout - единый формат времени для всех репозиториев (UTC)
const timeLayout = "2006-01-02 15:04:05"

// nullString возвращает sql.NullString для строк (пустая строка = NULL)
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// nullFloat64 возвращает sql.NullFloat64 для *float64 (nil = NULL)
func nullFloat64(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

// parseTime парсит время из строки в формате UTC и возвращает локальное время
func parseTime(raw interface{}) (time.Time, error) {
	if raw == nil {
		return time.Time{}, nil
	}

	str, ok := raw.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("expected string for time, got %T", raw)
	}

	if str == "" {
		return time.Time{}, nil
	}

	t, err := time.ParseInLocation(timeLayout, str, time.UTC)
	if err != nil {
		return time.Time{}, err
	}

	return t.Local(), nil
}

// checkContext проверяет контекст на отмену перед выполнением операции
func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// repoLogger создает логгер с полями репозитория и операции
type repoLogger struct {
	log    *zap.Logger
	repo   string
	ctx    context.Context
}

func newRepoLogger(ctx context.Context, log *zap.Logger, repoName string) *repoLogger {
	requestID, _ := ctx.Value("request_id").(string)
	
	var logger *zap.Logger
	if requestID != "" {
		logger = log.With(
			zap.String("request_id", requestID),
			zap.String("repo", repoName),
		)
	} else {
		logger = log.With(zap.String("repo", repoName))
	}
	
	return &repoLogger{
		log:  logger,
		repo: repoName,
		ctx:  ctx,
	}
}

func (rl *repoLogger) debug(msg string, fields ...zap.Field) {
	rl.log.Debug(msg, fields...)
}

func (rl *repoLogger) info(msg string, fields ...zap.Field) {
	rl.log.Info(msg, fields...)
}

func (rl *repoLogger) warn(msg string, fields ...zap.Field) {
	rl.log.Warn(msg, fields...)
}

func (rl *repoLogger) error(msg string, fields ...zap.Field) {
	rl.log.Error(msg, fields...)
}

func (rl *repoLogger) with(fields ...zap.Field) *repoLogger {
	return &repoLogger{
		log:  rl.log.With(fields...),
		repo: rl.repo,
		ctx:  rl.ctx,
	}
}
