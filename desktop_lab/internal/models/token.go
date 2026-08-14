package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// TimeString - кастомный тип для работы с временем из MySQL
type TimeString time.Time

func (t *TimeString) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string for time, got %T", value)
	}

	if str == "" {
		return nil
	}

	// Парсим время в формате MySQL
	parsed, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		return err
	}

	*t = TimeString(parsed)
	return nil
}

func (t TimeString) Value() (driver.Value, error) {
	return time.Time(t).Format("2006-01-02 15:04:05"), nil
}

// ToTime конвертирует TimeString в time.Time
func (t TimeString) ToTime() time.Time {
	return time.Time(t)
}

type Token struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	Role      Role       `db:"role"`
	ExpiresAt TimeString `db:"expires_at"` // было time.Time
	CreatedAt TimeString `db:"created_at"` // было time.Time
}

type TokenResponse struct {
	AccessToken  string
	RefreshToken string
}

func NewTimeString(t time.Time) TimeString {
	return TimeString(t)
}
