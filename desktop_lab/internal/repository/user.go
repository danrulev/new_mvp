package repository

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type UserRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewUserRepo(db *sqlx.DB, log *zap.Logger) *UserRepo {
	return &UserRepo{db: db, log: log}
}

func (r *UserRepo) Create(ctx context.Context, user models.User) error {
	log := logQuery(ctx, r.log, "INSERT", "users",
		zap.String("id", user.ID), zap.String("name", user.Name), zap.String("email", user.Email), zap.String("role", user.Role),
	)
	log.Info("starting creating user")
	_, err := r.db.ExecContext(ctx, "INSERT INTO users (id, name, email, password, role) VALUES (?, ?, ?, ?, ?)",
		user.ID, user.Name, user.Email, user.Password, user.Role)

	log.Info("User created successfully")
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (models.User, error) {
	log := logQuery(ctx, r.log, "SELECT", "users", zap.String("id", id))
	log.Debug("fetching user by ID")

	var user models.User
	err := r.db.QueryRowContext(ctx, "SELECT id, name, email, role FROM users WHERE id = ?", id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		return models.User{}, err
	}

	log.Debug("User fetched successfully", zap.String("id", id))
	return user, err
}

func (r *UserRepo) Credential(ctx context.Context, email string) (string, string, error) {
	log := logQuery(ctx, r.log, "SELECT", "users", zap.String("email", email))
	log.Debug("fetching user by email")

	var id, password string
	err := r.db.QueryRowContext(ctx, "SELECT id, password FROM users WHERE email = ?", email).Scan(&id, &password)
	if err != nil {
		return "", "", err
	}

	log.Debug("User fetched successfully", zap.String("email", email))
	return id, password, err
}

func (r *UserRepo) Update(ctx context.Context, id string, user models.UpdateUserRequest) (models.User, error) {
	log := logQuery(ctx, r.log, "UPDATE", "users", zap.String("id", id))
	log.Debug("updating user")

	var (
		userUpdateFields []string
		userUpdateValues []interface{}
	)

	if user.Email != nil {
		userUpdateFields = append(userUpdateFields, "email = ?")
		userUpdateValues = append(userUpdateValues, *user.Email)
		log.Debug("email updated", zap.String("email", *user.Email))
	}

	if user.Name != nil {
		userUpdateFields = append(userUpdateFields, "name = ?")
		userUpdateValues = append(userUpdateValues, *user.Name)
		log.Debug("name updated", zap.String("name", *user.Name))
	}

	if user.Password != nil {
		userUpdateFields = append(userUpdateFields, "password = ?")
		userUpdateValues = append(userUpdateValues, *user.Password)
		log.Debug("password updated")
	}

	if user.Role != nil {
		userUpdateFields = append(userUpdateFields, "role = ?")
		userUpdateValues = append(userUpdateValues, *user.Role)
		log.Debug("role updated", zap.String("role", *user.Role))
	}

	if len(userUpdateFields) == 0 {
		log.Debug("no fields to update")
		return models.User{}, nil
	}

	userUpdateFields = append(userUpdateFields, "updated_at = ?")
	userUpdateValues = append(userUpdateValues, time.Now().Format(timeLayout))

	query := fmt.Sprintf(`UPDATE users SET %v WHERE id = ?`, strings.Join(userUpdateFields, ", "))
	_, err := r.db.ExecContext(ctx, query, append(userUpdateValues, id)...)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to update user: %w", err)
	}

	log.Info("User updated successfully", zap.String("id", id), zap.String("fields_updated", strings.Join(userUpdateFields, ", ")))
	return r.GetByID(ctx, id)
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "users", zap.String("id", id))
	log.Debug("deleting user")

	_, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.Info("User deleted successfully", zap.String("id", id))
	return nil
}
