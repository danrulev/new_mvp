package mysql_repo

import (
	"context"
	"database/sql"
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

	// Рекомендуется также передавать created_at и updated_at, если они не имеют DEFAULT CURRENT_TIMESTAMP в БД
	_, err := r.db.ExecContext(ctx,
		"INSERT INTO users (id, name, email, password, role) VALUES (?, ?, ?, ?, ?)",
		user.ID, user.Name, user.Email, user.Password, user.Role,
	)
	if err != nil {
		log.Error("failed to create user", zap.Error(err))
		return err
	}
	log.Info("User created successfully")
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (models.User, error) {
	log := logQuery(ctx, r.log, "SELECT", "users", zap.String("id", id))
	log.Debug("fetching user by ID")

	var user models.User
	// ДОБАВЛЕНО: AND deleted_at IS NULL
	err := r.db.QueryRowContext(ctx, "SELECT id, name, email, role FROM users WHERE id = ? AND deleted_at IS NULL", id).Scan(
		&user.ID, &user.Name, &user.Email, &user.Role)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, fmt.Errorf("user not found")
		}
		return models.User{}, err
	}

	log.Debug("User fetched successfully", zap.String("id", id))
	return user, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	log := r.log.With(zap.String("method", "GetByEmail"), zap.String("email", email))
	log.Debug("fetching user by email")

	var user models.User
	// ДОБАВЛЕНО: AND deleted_at IS NULL
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, name, email, password, role, created_at, updated_at, deleted_at 
		 FROM users WHERE email = ? AND deleted_at IS NULL`, email).
		StructScan(&user)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("get user by email: user not found")
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) GetByName(ctx context.Context, name string) (*models.User, error) {
	log := r.log.With(zap.String("method", "GetByName"), zap.String("name", name))
	log.Debug("fetching user by name")

	var user models.User
	// ДОБАВЛЕНО: AND deleted_at IS NULL
	err := r.db.QueryRowxContext(ctx,
		`SELECT id, name, email, password, role, created_at, updated_at, deleted_at 
		 FROM users WHERE name = ? AND deleted_at IS NULL`, name).
		StructScan(&user)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("get user by name: user not found")
		}
		return nil, fmt.Errorf("get user by name: %w", err)
	}

	return &user, nil
}

func (r *UserRepo) Credential(ctx context.Context, email string) (string, string, error) {
	log := logQuery(ctx, r.log, "SELECT", "users", zap.String("email", email))
	log.Debug("fetching user credentials")

	var id, password string
	// ИСПРАВЛЕНО: deleted_at = NULL -> deleted_at IS NULL
	err := r.db.QueryRowContext(ctx, "SELECT id, password FROM users WHERE email = ? AND deleted_at IS NULL", email).Scan(&id, &password)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("user not found")
		}
		return "", "", err
	}

	log.Debug("User credentials fetched successfully", zap.String("email", email))
	return id, password, nil
}

func (r *UserRepo) List(ctx context.Context, limit, offset int64) ([]models.User, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "users", zap.Int64("limit", limit), zap.Int64("offset", offset))
	log.Debug("fetching paginated users list")

	var total int64
	// ДОБАВЛЕНО: WHERE deleted_at IS NULL для корректного подсчета
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []models.User{}, 0, nil // Лучше возвращать пустой срез, а не nil
	}

	// ДОБАВЛЕНО: WHERE deleted_at IS NULL
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, email, name, role, created_at, updated_at, deleted_at 
		 FROM users 
		 WHERE deleted_at IS NULL
		 ORDER BY created_at DESC 
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User

		// РЕКОМЕНДАЦИЯ: Если models.User имеет поля CreatedAt, UpdatedAt, DeletedAt типа *time.Time,
		// можно использовать rows.StructScan(&u) и избавиться от ручного парсинга строк.
		// Ниже оставлен ваш вариант с исправленной обработкой ошибок.

		var createdAt, updatedAt, deletedAt sql.NullString // Безопаснее использовать sql.NullString

		err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, 0, err
		}

		if createdAt.Valid {
			if t, err := helperParseTime(createdAt.String); err == nil {
				u.CreatedAt = t
			} else {
				r.log.Warn("failed to parse created_at", zap.String("val", createdAt.String), zap.Error(err))
			}
		}

		if updatedAt.Valid {
			if t, err := helperParseTime(updatedAt.String); err == nil {
				u.UpdatedAt = t
			} else {
				r.log.Warn("failed to parse updated_at", zap.String("val", updatedAt.String), zap.Error(err))
			}
		}

		if deletedAt.Valid {
			if t, err := helperParseTime(deletedAt.String); err == nil {
				u.DeletedAt = &t // или u.DeletedAt = t, зависит от вашей модели
			} else {
				r.log.Warn("failed to parse deleted_at", zap.String("val", deletedAt.String), zap.Error(err))
			}
		}

		users = append(users, u)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	log.Debug("users fetched", zap.Int64("count", total))
	return users, total, nil
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
	}
	if user.Name != nil {
		userUpdateFields = append(userUpdateFields, "name = ?")
		userUpdateValues = append(userUpdateValues, *user.Name)
	}
	if user.Password != nil {
		userUpdateFields = append(userUpdateFields, "password = ?")
		userUpdateValues = append(userUpdateValues, *user.Password)
	}
	if user.Role != nil {
		userUpdateFields = append(userUpdateFields, "role = ?")
		userUpdateValues = append(userUpdateValues, *user.Role)
	}

	if len(userUpdateFields) == 0 {
		log.Debug("no fields to update")
		return r.GetByID(ctx, id)
	}

	// ИСПРАВЛЕНО: Используем time.Now().UTC() для консистентности
	userUpdateFields = append(userUpdateFields, "updated_at = ?")
	userUpdateValues = append(userUpdateValues, time.Now().UTC())

	// ИСПРАВЛЕНО: id добавляется только один раз в самом конце
	userUpdateValues = append(userUpdateValues, id)

	query := fmt.Sprintf(`UPDATE users SET %s WHERE id = ? AND deleted_at IS NULL`, strings.Join(userUpdateFields, ", "))

	_, err := r.db.ExecContext(ctx, query, userUpdateValues...)
	if err != nil {
		return models.User{}, fmt.Errorf("failed to update user: %w", err)
	}

	log.Info("User updated successfully", zap.String("id", id))
	return r.GetByID(ctx, id)
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "users", zap.String("id", id))
	log.Debug("deleting user")

	// ИСПРАВЛЕНО: Синтаксис MySQL (NOW()) вместо SQLite (datetime('now')), убрана лишняя скобка
	_, err := r.db.ExecContext(ctx, "UPDATE users SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.Info("User deleted successfully", zap.String("id", id))
	return nil
}
