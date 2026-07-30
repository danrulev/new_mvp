package mysql_repo

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// timeLayout - единый формат времени для всех репозиториев (UTC)
const timeLayout = "2006-01-02 15:04:05"

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
