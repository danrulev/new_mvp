package mysql_repo

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type InvitationRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

func NewInvitationRepo(db *sqlx.DB, log *zap.Logger) *InvitationRepo {
	return &InvitationRepo{db: db, log: log}
}

// Create создает новую заявку на регистрацию
func (r *InvitationRepo) Create(ctx context.Context, invitation models.RegistrationInvitation) error {
	logger := logQuery(ctx, r.log, "INSERT", "registration_invitations", zap.String("id", invitation.ID))
	logger.Info("creating new registration invitation")

	query := `INSERT INTO registration_invitations (
		id, email, name, role, status, message, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.ExecContext(ctx, query,
		invitation.ID,
		invitation.Email,
		invitation.Name,
		invitation.Role,
		invitation.Status,
		invitation.Message,
		invitation.CreatedAt.Format(time.RFC3339),
		invitation.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		logger.Error("failed to create invitation", zap.Error(err))
		return err
	}

	logger.Info("invitation created successfully")
	return nil
}

// GetByID получает заявку по ID
func (r *InvitationRepo) GetByID(ctx context.Context, id string) (models.RegistrationInvitation, error) {
	logger := logQuery(ctx, r.log, "SELECT", "registration_invitations", zap.String("id", id))
	logger.Debug("fetching invitation by ID")

	var invitation models.RegistrationInvitation
	var createdAt, updatedAt sql.NullString
	var reviewedBy sql.NullString
	var reviewedAt sql.NullString

	query := `SELECT id, email, name, role, status, message, reviewed_by, reviewed_at, created_at, updated_at
	FROM registration_invitations WHERE id = ?`

	row := r.db.QueryRowContext(ctx, query, id)
	err := row.Scan(
		&invitation.ID, &invitation.Email, &invitation.Name, &invitation.Role,
		&invitation.Status, &invitation.Message,
		&reviewedBy, &reviewedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		logger.Error("failed to fetch invitation", zap.Error(err))
		return models.RegistrationInvitation{}, err
	}

	invitation.CreatedAt, _ = parseTime(createdAt.String)
	invitation.UpdatedAt, _ = parseTime(updatedAt.String)
	
	if reviewedBy.Valid && reviewedBy.String != "" {
		invitation.ReviewedBy = &reviewedBy.String
	}
	if reviewedAt.Valid && reviewedAt.String != "" {
		t, _ := parseTime(reviewedAt.String)
		invitation.ReviewedAt = &t
	}

	logger.Debug("invitation retrieved successfully")
	return invitation, nil
}

// GetByEmail получает заявку по email
func (r *InvitationRepo) GetByEmail(ctx context.Context, email string) (models.RegistrationInvitation, error) {
	logger := logQuery(ctx, r.log, "SELECT", "registration_invitations", zap.String("email", email))
	logger.Debug("fetching invitation by email")

	var invitation models.RegistrationInvitation
	var createdAt, updatedAt sql.NullString
	var reviewedBy sql.NullString
	var reviewedAt sql.NullString

	query := `SELECT id, email, name, role, status, message, reviewed_by, reviewed_at, created_at, updated_at
	FROM registration_invitations WHERE email = ? ORDER BY created_at DESC LIMIT 1`

	row := r.db.QueryRowContext(ctx, query, email)
	err := row.Scan(
		&invitation.ID, &invitation.Email, &invitation.Name, &invitation.Role,
		&invitation.Status, &invitation.Message,
		&reviewedBy, &reviewedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug("invitation not found by email")
			return models.RegistrationInvitation{}, err
		}
		logger.Error("failed to fetch invitation by email", zap.Error(err))
		return models.RegistrationInvitation{}, err
	}

	invitation.CreatedAt, _ = parseTime(createdAt.String)
	invitation.UpdatedAt, _ = parseTime(updatedAt.String)
	
	if reviewedBy.Valid && reviewedBy.String != "" {
		invitation.ReviewedBy = &reviewedBy.String
	}
	if reviewedAt.Valid && reviewedAt.String != "" {
		t, _ := parseTime(reviewedAt.String)
		invitation.ReviewedAt = &t
	}

	logger.Debug("invitation retrieved successfully by email")
	return invitation, nil
}

// List получает список заявок с фильтрацией
func (r *InvitationRepo) List(ctx context.Context, filter models.InvitationListFilter) ([]models.RegistrationInvitation, int64, error) {
	logger := logQuery(ctx, r.log, "SELECT", "registration_invitations", zap.Any("filter", filter))
	logger.Debug("fetching invitations list")

	baseQuery := `FROM registration_invitations WHERE 1=1`
	countQuery := `SELECT COUNT(*) ` + baseQuery
	args := []interface{}{}

	if filter.Email != "" {
		baseQuery += ` AND email LIKE ?`
		args = append(args, "%"+filter.Email+"%")
	}
	if filter.Status != "" {
		baseQuery += ` AND status = ?`
		args = append(args, filter.Status)
	}

	var total int64
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		logger.Error("failed to count invitations", zap.Error(err))
		return nil, 0, err
	}

	if total == 0 {
		return nil, 0, nil
	}

	selectQuery := `SELECT id, email, name, role, status, message, reviewed_by, reviewed_at, created_at, updated_at ` +
		baseQuery + ` ORDER BY created_at DESC LIMIT ? OFFSET ?`

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		logger.Error("failed to fetch invitations", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var invitations []models.RegistrationInvitation
	for rows.Next() {
		var inv models.RegistrationInvitation
		var createdAt, updatedAt sql.NullString
		var reviewedBy sql.NullString
		var reviewedAt sql.NullString

		err := rows.Scan(
			&inv.ID, &inv.Email, &inv.Name, &inv.Role,
			&inv.Status, &inv.Message,
			&reviewedBy, &reviewedAt, &createdAt, &updatedAt,
		)
		if err != nil {
			logger.Error("failed to scan invitation", zap.Error(err))
			return nil, 0, err
		}

		inv.CreatedAt, _ = parseTime(createdAt.String)
		inv.UpdatedAt, _ = parseTime(updatedAt.String)
		
		if reviewedBy.Valid && reviewedBy.String != "" {
			inv.ReviewedBy = &reviewedBy.String
		}
		if reviewedAt.Valid && reviewedAt.String != "" {
			t, _ := parseTime(reviewedAt.String)
			inv.ReviewedAt = &t
		}

		invitations = append(invitations, inv)
	}

	logger.Debug("invitations list retrieved", zap.Int64("total", total), zap.Int("count", len(invitations)))
	return invitations, total, nil
}

// UpdateStatus обновляет статус заявки
func (r *InvitationRepo) UpdateStatus(ctx context.Context, id string, status models.InvitationStatus, reviewedBy *string, reviewedAt *time.Time, message string) error {
	logger := logQuery(ctx, r.log, "UPDATE", "registration_invitations", zap.String("id", id), zap.String("status", string(status)))
	logger.Info("updating invitation status")

	var query string
	var args []interface{}

	if reviewedBy != nil && reviewedAt != nil {
		query = `UPDATE registration_invitations 
			SET status = ?, reviewed_by = ?, reviewed_at = ?, updated_at = ? 
			WHERE id = ?`
		args = []interface{}{status, *reviewedBy, reviewedAt.Format(time.RFC3339), time.Now().Format(time.RFC3339), id}
	} else {
		query = `UPDATE registration_invitations 
			SET status = ?, updated_at = ? 
			WHERE id = ?`
		args = []interface{}{status, time.Now().Format(time.RFC3339), id}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		logger.Error("failed to update invitation status", zap.Error(err))
		return err
	}

	logger.Info("invitation status updated successfully")
	return nil
}

// Delete удаляет заявку
func (r *InvitationRepo) Delete(ctx context.Context, id string) error {
	logger := logQuery(ctx, r.log, "DELETE", "registration_invitations", zap.String("id", id))
	logger.Info("deleting invitation")

	query := `DELETE FROM registration_invitations WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		logger.Error("failed to delete invitation", zap.Error(err))
		return err
	}

	logger.Info("invitation deleted successfully")
	return nil
}
