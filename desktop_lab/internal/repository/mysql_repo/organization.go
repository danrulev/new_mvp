package mysql_repo

import (
	"context"
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
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Error("transaction rollback failed", zap.Error(rbErr))
			} else {
				log.Debug("transaction rolled back")
			}
		}
	}()

	_, err = tx.ExecContext(ctx, "INSERT INTO organizations (id, name, email, phone, address) VALUES (?, ?, ?, ?, ?)", id, org.Name, org.Email, org.Phone, org.Address)
	if err != nil {
		return models.MakeError(err, models.ErrFailedToCreate, "organization")
	}

	if _, err = tx.ExecContext(ctx, "INSERT INTO organization_users (id, organization_id, role) VALUES (?, ?, ?)", id, id, "super_admin"); err != nil {
		return models.MakeError(err, models.ErrFailedToCreate, "organization")
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
	err := r.db.GetContext(ctx, &org, "SELECT * FROM organizations WHERE id = ?", id)
	if err != nil {
		return models.Organization{}, err
	}

	log.Debug("organization retrieved successfully")
	return org, nil
}

func (r *OrganizationRepo) GetOrganizationByName(ctx context.Context, name string) (models.Organization, error) {
	log := logQuery(ctx, r.log, "SELECT", "organizations", zap.String("name", name))
	log.Debug("fetching organization by name")

	var org models.Organization
	err := r.db.GetContext(ctx, &org, "SELECT * FROM organizations WHERE name = ?", name)
	if err != nil {
		return models.Organization{}, err
	}

	log.Debug("organization retrieved successfully")
	return org, nil
}

func (r *OrganizationRepo) List(ctx context.Context, limit, offset int64) ([]models.Organization, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "organizations", zap.Int64("limit", limit), zap.Int64("offset", offset))
	log.Debug("fetching paginated organizations list")

	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, models.ErrNotFound
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, email, phone, address, created_at, updated_at, deleted_at
         FROM organizations 
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
		var createdAt, updatedAt, deletedAt string
		err := rows.Scan(&org.ID, &org.Name, &org.Email, &org.Phone, &org.Address, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			return nil, 0, err
		}

		org.CreatedAt, err = helperParseTime(createdAt)
		if err != nil {
			r.log.Warn("failed to parse created_at", zap.String("val", createdAt), zap.Error(err))
			org.CreatedAt = time.Now() // Fallback
		}

		org.UpdatedAt, err = helperParseTime(updatedAt)
		if err != nil {
			r.log.Warn("failed to parse updated_at", zap.String("val", updatedAt), zap.Error(err))
			org.UpdatedAt = time.Now() // Fallback
		}

		org.DeletedAt, err = helperParseTime(deletedAt)
		if err != nil {
			r.log.Warn("failed to parse deleted_at", zap.String("val", deletedAt), zap.Error(err))
			org.DeletedAt = time.Now() // Fallback
		}

		orgs = append(orgs, org)
	}
	log.Debug("organizations", zap.Int64("count", total))
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
		log.Debug("email updated", zap.String("email", *req.Email))
	}

	if req.Name != nil {
		orgUpdateFields = append(orgUpdateFields, "name = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Name)
		log.Debug("name updated", zap.String("name", *req.Name))
	}

	if req.Phone != nil {
		orgUpdateFields = append(orgUpdateFields, "phone = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Phone)
		log.Debug("phone updated", zap.String("phone", *req.Phone))
	}

	if req.Address != nil {
		orgUpdateFields = append(orgUpdateFields, "address = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Address)
		log.Debug("address updated", zap.String("address", *req.Address))
	}

	if len(orgUpdateFields) == 0 {
		return r.GetByID(ctx, id)
	}

	orgUpdateFields = append(orgUpdateFields, "updated_at = ?")
	orgUpdateValues = append(orgUpdateValues, time.Now().Format(timeLayout))
	orgUpdateValues = append(orgUpdateValues, id)

	query := fmt.Sprintf(`UPDATE organizations SET %v WHERE id = ? AND deleted_at IS NULL`, strings.Join(orgUpdateFields, ", "))
	_, err := r.db.ExecContext(ctx, query, orgUpdateValues...)
	if err != nil {
		return models.Organization{}, fmt.Errorf("failed to update organization: %w", err)
	}

	log.Info("Organization updated successfully", zap.String("id", id), zap.String("fields_updated", strings.Join(orgUpdateFields, ", ")))
	return r.GetByID(ctx, id)
}

func (r *OrganizationRepo) Delete(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "organizations", zap.String("id", id))
	log.Debug("deleting organization")

	_, err := r.db.ExecContext(ctx, "UPDATE organizations SET deleted_at = (datetime('now') WHERE id = ? AND deleted_at IS NULL", id)
	if err != nil {
		return err
	}

	log.Debug("deleted organization")
	return nil
}
