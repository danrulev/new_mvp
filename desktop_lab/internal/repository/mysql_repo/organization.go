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

type OrganizationRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewOrganizationRepo(db *sqlx.DB, log *zap.Logger) *OrganizationRepo {
	return &OrganizationRepo{db: db, log: log}
}

func (r *OrganizationRepo) Create(ctx context.Context, id string, org models.CreateOrganizationRequest) error {
	log := logQuery(ctx, r.log, "INSERT", "organizations",
		zap.String("id", id), zap.String("name", org.Name), zap.String("email", org.Email), zap.String("phone", org.Phone), zap.String("address", org.Address),
	)
	log.Debug("creating new organization")

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// ИСПРАВЛЕНО: Идиоматичный и безопасный способ отката транзакции.
	// Если Commit() пройдет успешно, Rollback() просто вернет ошибку, которую мы игнорируем.
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		"INSERT INTO organizations (id, name, email, phone, address) VALUES (?, ?, ?, ?, ?)",
		id, org.Name, org.Email, org.Phone, org.Address,
	)
	if err != nil {
		return models.MakeError(err, models.ErrFailedToCreate, "organization")
	}

	// ВНИМАНИЕ: Убедитесь, что передача `id` дважды здесь корректна.
	// Обычно здесь должен быть userID создателя: (user_id, organization_id, role)
	_, err = tx.ExecContext(ctx,
		"INSERT INTO organization_users (id, organization_id, role) VALUES (?, ?, ?)",
		id, id, "super_admin",
	)
	if err != nil {
		return models.MakeError(err, models.ErrFailedToCreate, "organization_user_link")
	}

	if err = tx.Commit(); err != nil {
		log.Error("transaction commit failed", zap.Error(err))
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug("successfully created new organization")
	return nil
}

func (r *OrganizationRepo) GetByID(ctx context.Context, id string) (models.Organization, error) {
	log := logQuery(ctx, r.log, "SELECT", "organizations", zap.String("id", id))
	log.Debug("fetching organization by ID")

	var org models.Organization
	// ДОБАВЛЕНО: AND deleted_at IS NULL
	err := r.db.GetContext(ctx, &org, "SELECT * FROM organizations WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Organization{}, fmt.Errorf("organization not found")
		}
		return models.Organization{}, err
	}

	log.Debug("organization retrieved successfully")
	return org, nil
}

func (r *OrganizationRepo) GetOrganizationByName(ctx context.Context, name string) (models.Organization, error) {
	log := logQuery(ctx, r.log, "SELECT", "organizations", zap.String("name", name))
	log.Debug("fetching organization by name")

	var org models.Organization
	// ДОБАВЛЕНО: AND deleted_at IS NULL
	err := r.db.GetContext(ctx, &org, "SELECT * FROM organizations WHERE name = ? AND deleted_at IS NULL", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.Organization{}, fmt.Errorf("organization not found")
		}
		return models.Organization{}, err
	}

	log.Debug("organization retrieved successfully")
	return org, nil
}

func (r *OrganizationRepo) List(ctx context.Context, limit, offset int64) ([]models.Organization, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "organizations", zap.Int64("limit", limit), zap.Int64("offset", offset))
	log.Debug("fetching paginated organizations list")

	var total int64
	// ДОБАВЛЕНО: WHERE deleted_at IS NULL
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// ИСПРАВЛЕНО: Возвращаем пустой срез вместо ошибки для пагинации
	if total == 0 {
		return []models.Organization{}, 0, nil
	}

	// ДОБАВЛЕНО: WHERE deleted_at IS NULL
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email, phone, address, created_at, updated_at, deleted_at
         FROM organizations 
         WHERE deleted_at IS NULL
         ORDER BY created_at DESC 
         LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []models.Organization
	for rows.Next() {
		var org models.Organization

		// ИСПОЛЬЗУЕМ sql.NullString для безопасной работы с NULL значениями дат
		var createdAt, updatedAt, deletedAt sql.NullString

		err := rows.Scan(&org.ID, &org.Name, &org.Email, &org.Phone, &org.Address, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, 0, err
		}

		if createdAt.Valid {
			if t, err := helperParseTime(createdAt.String); err == nil {
				org.CreatedAt = t
			} else {
				r.log.Warn("failed to parse created_at", zap.String("val", createdAt.String), zap.Error(err))
			}
		}

		if updatedAt.Valid {
			if t, err := helperParseTime(updatedAt.String); err == nil {
				org.UpdatedAt = t
			} else {
				r.log.Warn("failed to parse updated_at", zap.String("val", updatedAt.String), zap.Error(err))
			}
		}

		if deletedAt.Valid {
			if t, err := helperParseTime(deletedAt.String); err == nil {
				// Предполагается, что DeletedAt в модели - это *time.Time или time.Time
				// Если time.Time, то нужно проверить, как ваша модель это принимает
				org.DeletedAt = t
			} else {
				r.log.Warn("failed to parse deleted_at", zap.String("val", deletedAt.String), zap.Error(err))
			}
		}

		orgs = append(orgs, org)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	log.Debug("organizations fetched", zap.Int64("count", total))
	return orgs, total, nil
}

func (r *OrganizationRepo) Update(ctx context.Context, id string, req models.UpdateOrganizationRequest) (models.Organization, error) {
	log := logQuery(ctx, r.log, "UPDATE", "organizations", zap.String("id", id))
	log.Debug("updating organization")

	var (
		orgUpdateFields []string
		orgUpdateValues []interface{}
	)

	if req.Email != nil {
		orgUpdateFields = append(orgUpdateFields, "email = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Email)
	}
	if req.Name != nil {
		orgUpdateFields = append(orgUpdateFields, "name = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Name)
	}
	if req.Phone != nil {
		orgUpdateFields = append(orgUpdateFields, "phone = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Phone)
	}
	if req.Address != nil {
		orgUpdateFields = append(orgUpdateFields, "address = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Address)
	}

	if len(orgUpdateFields) == 0 {
		log.Debug("no fields to update")
		return r.GetByID(ctx, id)
	}

	// ИСПРАВЛЕНО: Используем UTC время для консистентности с БД
	orgUpdateFields = append(orgUpdateFields, "updated_at = ?")
	orgUpdateValues = append(orgUpdateValues, time.Now().UTC())
	orgUpdateValues = append(orgUpdateValues, id)

	query := fmt.Sprintf(`UPDATE organizations SET %s WHERE id = ? AND deleted_at IS NULL`, strings.Join(orgUpdateFields, ", "))

	_, err := r.db.ExecContext(ctx, query, orgUpdateValues...)
	if err != nil {
		return models.Organization{}, fmt.Errorf("failed to update organization: %w", err)
	}

	log.Info("Organization updated successfully", zap.String("id", id))
	return r.GetByID(ctx, id)
}

func (r *OrganizationRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "organizations", zap.String("id", id))
	log.Debug("deleting organization")

	// ИСПРАВЛЕНО: Синтаксис MySQL (NOW()) и добавлена закрывающая скобка
	_, err := r.db.ExecContext(ctx, "UPDATE organizations SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	log.Debug("organization deleted successfully")
	return nil
}
