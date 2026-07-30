package models

import "time"

type User struct {
	ID        string     `json:"id" db:"id"`
	Name      string     `json:"name" db:"name"`
	Email     string     `json:"email" db:"email" validate:"required,email"`
	Password  string     `json:"password" db:"password" validate:"required"`
	Role      string     `json:"role" db:"role"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at"`
}

type CreateUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Role     string `json:"role" validate:"required"`
}

type SignInRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type UpdateUserRequest struct {
	Name     *string `json:"name" db:"name"`
	Email    *string `json:"email" db:"email" validate:"email"`
	Password *string `json:"password" db:"password"`
	Role     *string `json:"role" db:"role"`
}
