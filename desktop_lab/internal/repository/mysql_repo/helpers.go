package mysql_repo

import (
	"fmt"
	"time"
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
