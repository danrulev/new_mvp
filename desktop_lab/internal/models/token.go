package models

import "time"

type Token struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

type TokenResponse struct {
	AccessToken  string
	RefreshToken string
}
