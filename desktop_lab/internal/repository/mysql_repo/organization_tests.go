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

type OrganizationTestsRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewOrganizationTestsRepo(db *sqlx.DB, log *zap.Logger) *OrganizationTestsRepo {
	return &OrganizationTestsRepo{db: db, log: log}
}

func (r *OrganizationTestsRepo) Create(ctx context.Context, id string, req models.CreateOrganizationTestRequest) error {
	log := logQuery(ctx, r.log, "INSERT", "organization_tests",
		zap.String("organization_id", req.OrganizationID), zap.String("test_method_id", req.TestMethodID), zap.Float64("price", req.Price),
	)
	log.Info("creating organization test")

	_, err := r.db.ExecContext(ctx, "INSERT INTO organization_tests (id, organization_id, test_method_id, price, description) VALUES (?, ?, ?, ?, ?)",
		id, req.OrganizationID, req.TestMethodID, req.Price, req.Description)
	if err != nil {
		return err
	}

	log.Info("created organization test")
	return nil
}

func (r *OrganizationTestsRepo) GetOrganizationTest(ctx context.Context, id string) (models.OrganizationTest, error) {
	log := logQuery(ctx, r.log, "SELECT", "organization_tests",
		zap.String("organization_test_id", id))

	log.Debug("fetching organization test by ID")
	var organizationTest models.OrganizationTest
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, organization_id, description, price, test_method_id, created_at, updated_at
         FROM organization_tests
         WHERE id = ?`, id).Scan(
		&organizationTest.ID, &organizationTest.OrganizationID, &organizationTest.Description,
		&organizationTest.Price, &organizationTest.TestMethodID, &createdAt, &updatedAt,
	)
	if err != nil {
		return models.OrganizationTest{}, err
	}

	organizationTest.CreatedAt, err = helperParseTime(createdAt)
	if err != nil {
		r.log.Warn("failed parse created at date", zap.Error(err), zap.String("val", createdAt))
		organizationTest.CreatedAt = time.Now()
	}

	organizationTest.UpdatedAt, err = helperParseTime(updatedAt)
	if err != nil {
		r.log.Warn("failed parse updated at date", zap.Error(err), zap.String("val", updatedAt))
		organizationTest.UpdatedAt = time.Now()
	}

	return organizationTest, nil
}

func (r *OrganizationTestsRepo) ListOrganizationTests(ctx context.Context, limit, offset int64) ([]models.OrganizationTest, int64, error) {
	log := logQuery(ctx, r.log, "SELECT", "organization_tests", zap.Int64("limit", limit), zap.Int64("offset", offset))
	log.Debug("fetching paginated organization tests list")

	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organization_tests`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, description, organization_id, price, test_method_id, created_at, updated_at  
		 FROM organization_tests 
		 ORDER BY created_at DESC 
		 LIMIT ? OFFSET ?`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}

	defer rows.Close()

	var tests []models.OrganizationTest
	for rows.Next() {
		var ot models.OrganizationTest
		var createdAt, updatedAt string
		if err := rows.Scan(&ot.ID, &ot.Description, &ot.OrganizationID, &ot.Price, &ot.TestMethodID, &createdAt, &updatedAt); err != nil {
			return nil, 0, err
		}

		ot.CreatedAt, err = helperParseTime(createdAt)
		if err != nil {
			r.log.Warn("failed parse created at date", zap.Error(err), zap.String("val", createdAt))
			ot.CreatedAt = time.Now()
		}

		ot.UpdatedAt, err = helperParseTime(updatedAt)
		if err != nil {
			r.log.Warn("failed parse updated at date", zap.Error(err), zap.String("val", updatedAt))
			ot.UpdatedAt = time.Now()
		}

		tests = append(tests, ot)
	}

	log.Debug("organizations tests", zap.Int64("count", total))
	return tests, total, nil
}

func (r *OrganizationTestsRepo) Update(ctx context.Context, id string, req models.UpdateOrganizationTestRequest) (models.OrganizationTest, error) {
	log := logQuery(ctx, r.log, "UPDATE", "organizations_tests", zap.String("id", id))
	log.Debug("updating organization test")
	var (
		orgUpdateFields []string
		orgUpdateValues []interface{}
	)

	if req.Description != nil {
		orgUpdateFields = append(orgUpdateFields, "description = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Description)
		log.Debug("description updated", zap.String("description", *req.Description))
	}

	if req.Price != nil {
		orgUpdateFields = append(orgUpdateFields, "price = ?")
		orgUpdateValues = append(orgUpdateValues, *req.Price)
		log.Debug("price updated", zap.Float64("price", *req.Price))
	}

	if len(orgUpdateFields) == 0 {
		return r.GetOrganizationTest(ctx, id)
	}

	orgUpdateFields = append(orgUpdateFields, "updated_at = ?")
	orgUpdateValues = append(orgUpdateValues, time.Now().Format(timeLayout))
	orgUpdateValues = append(orgUpdateValues, id)
	query := fmt.Sprintf(`UPDATE organization_tests SET %v WHERE id = ? AND deleted_at IS NULL`, strings.Join(orgUpdateFields, ", "))

	_, err := r.db.ExecContext(ctx, query, orgUpdateValues...)
	if err != nil {
		return models.OrganizationTest{}, fmt.Errorf("failed to update organization test: %w", err)
	}

	log.Debug("Organization test updated successfully", zap.String("id", id))
	return r.GetOrganizationTest(ctx, id)
}

func (r *OrganizationRepo) DeleteOrganizationTest(ctx context.Context, id string) error {
	log := logQuery(ctx, r.log, "DELETE", "organization_tests", zap.String("id", id))
	log.Debug("deleting organization test")

	_, err := r.db.ExecContext(ctx, "DELETE FROM organization_tests WHERE id = ?", id)
	if err != nil {
		return err
	}

	log.Info("organization test deleted successfully")
	return nil
}
