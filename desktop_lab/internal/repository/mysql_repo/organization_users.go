package mysql_repo

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type OrganizationUserRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewOrganizationUserRepo(db *sqlx.DB, log *zap.Logger) *OrganizationUserRepo {
	return &OrganizationUserRepo{db: db, log: log}
}

func (r *OrganizationUserRepo) Create(ctx context.Context, id string, ou models.CreateOrganizationUserRequest) error {
	log := logQuery(ctx, r.log, "INSERT", "organization_users",
		zap.String("id", id), zap.String("organization_id", ou.OrganizationID), zap.String("user_id", ou.UserID), zap.String("role", ou.Role),
	)

	log.Info("starting creating organization user")

	_, err := r.db.ExecContext(ctx, "INSERT INTO organization_users (id, organization_id, user_id, role) VALUES (?, ?, ?, ?)",
		id, ou.OrganizationID, ou.UserID, ou.Role)
	if err != nil {
		return err
	}

	log.Info("organization user created successfully")
	return nil
}

func (r *OrganizationUserRepo) GetByID(ctx context.Context, id string) (models.OrganizationUser, error) {
	log := logQuery(ctx, r.log, "SELECT", "organization_users",
		zap.String("id", id))

	log.Debug("fetching organization user by ID")

	var ou models.OrganizationUser
	var createdAt, updatedAt string
	row := r.db.QueryRowContext(ctx, "SELECT id, organization_id, user_id, role, created_at, updated_at FROM organization_users WHERE id = ?", id)
	err := row.Scan(&ou.ID, &ou.OrganizationID, &ou.UserID, &ou.Role, &createdAt, &updatedAt)
	if err != nil {
		return models.OrganizationUser{}, err
	}

	ou.CreatedAt, err = helperParseTime(createdAt)
	if err != nil {
		log.Warn("failed to parse created_at", zap.Error(err))
		ou.CreatedAt = time.Now()
	}
	ou.UpdatedAt, err = helperParseTime(updatedAt)
	if err != nil {
		log.Warn("failed to parse updated_at", zap.Error(err))
		ou.UpdatedAt = time.Now()
	}

	log.Debug("organization user retrieved successfully")
	return ou, nil
}

func (r *OrganizationUserRepo) GetByRole(ctx context.Context, organizationID, role string, limit, offset int64) ([]models.OrganizationUser, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "organization_users",
		zap.String("organization_id", organizationID),
		zap.String("role", role))

	log.Debug("fetching paginated organization users by role list")
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organization_users WHERE role = ? AND organization_id = ?`, role, organizationID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, organization_id, user_id, role, created_at, updated_at
         FROM organization_users 
         WHERE role = ? AND organization_id = ?
         ORDER BY created_at DESC 
         LIMIT ? OFFSET ?`,
		role, organizationID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.OrganizationUser
	for rows.Next() {
		var ou models.OrganizationUser
		var createdAt, updatedAt string

		err := rows.Scan(&ou.ID, &ou.OrganizationID, &ou.UserID, &ou.Role, &createdAt, &updatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		ou.CreatedAt, err = helperParseTime(createdAt)
		if err != nil {
			r.log.Warn("failed parse created at date", zap.Error(err))
			ou.CreatedAt = time.Now()
		}

		ou.UpdatedAt, err = helperParseTime(updatedAt)
		if err != nil {
			r.log.Warn("failed parse updated at date", zap.Error(err))
			ou.UpdatedAt = time.Now()
		}

		users = append(users, ou)
	}

	log.Debug("list organization users by role", zap.Int64("count", total))

	return users, total, nil
}

func (r *OrganizationUserRepo) List(ctx context.Context, organizationID string, limit, offset int64) ([]models.OrganizationUser, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "organization_users", zap.String("organization_id", organizationID), zap.Int64("limit", limit), zap.Int64("offset", offset))
	log.Debug("fetching paginated organization users list")

	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organization_users WHERE organization_id = ?`, organizationID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, organization_id, user_id, role, created_at, updated_at  
		 FROM organization_users 
		 ORDER BY created_at DESC 
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	var users []models.OrganizationUser
	var createdAt, updatedAt string
	for rows.Next() {
		var ou models.OrganizationUser
		err := rows.Scan(&ou.ID, &ou.OrganizationID, &ou.UserID, &ou.Role, &createdAt, &updatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		ou.CreatedAt, err = helperParseTime(createdAt)
		if err != nil {
			r.log.Warn("failed parse created at date", zap.Error(err))
			ou.CreatedAt = time.Now()
		}

		ou.UpdatedAt, err = helperParseTime(updatedAt)
		if err != nil {
			r.log.Warn("failed parse updated at date", zap.Error(err))
			ou.UpdatedAt = time.Now()
		}

		users = append(users, ou)
	}
	log.Debug("list organization users", zap.Int64("count", total))
	return users, total, nil
}

func (r *OrganizationUserRepo) UpdateUser(ctx context.Context, id string, role *string) (models.OrganizationUser, error) {
	log := logQuery(ctx, r.log, "UPDATE", "organization_users", zap.String("id", id), zap.String("role", *role))
	log.Debug("updating organization user")

	if role == nil {
		return r.GetByID(ctx, id)
	}

	_, err := r.db.ExecContext(ctx, "UPDATE organization_users SET role = ?, updated_at = (datetime('now') WHERE id = ?", role, id)
	if err != nil {
		log.Error("update failed", zap.Error(err))
		return models.OrganizationUser{}, err
	}

	log.Debug("updated organization user")
	return r.GetByID(ctx, id)
}

func (r *OrganizationUserRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "organization_users", zap.String("id", id))
	log.Debug("deleting organization user")

	_, err := r.db.ExecContext(ctx, "DELETE FROM organization_users WHERE id = ?", id)
	if err != nil {
		log.Error("delete failed", zap.Error(err))
		return err
	}

	log.Info("organization user deleted successfully")
	return nil
}
