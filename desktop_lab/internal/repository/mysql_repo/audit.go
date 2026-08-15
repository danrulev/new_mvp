package mysql_repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"desktop_lab/internal/models"
	"go.uber.org/zap"
)

// AuditRepo реализует репозиторий для аудита
type AuditRepo struct {
	db *sql.DB
	log *zap.Logger
}

// NewAuditRepo создает новый репозиторий аудита
func NewAuditRepo(db *sqlx.DB, log *zap.Logger) *AuditRepo {
	return &AuditRepo{db: db, log: log}
}

// Create создает запись аудита
func (r *AuditRepo) Create(ctx context.Context, audit *models.AuditLog) error {
	query := `
		INSERT INTO audit_logs 
		(user_id, user_name, action, resource_type, resource_id, old_values, new_values, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var oldValuesJSON, newValuesJSON []byte
	var err error

	if audit.OldValues != nil {
		oldValuesJSON, err = json.Marshal(audit.OldValues)
		if err != nil {
			r.log.Error("failed to marshal old_values", zap.Error(err))
			return fmt.Errorf("marshal old_values: %w", err)
		}
	}

	if audit.NewValues != nil {
		newValuesJSON, err = json.Marshal(audit.NewValues)
		if err != nil {
			r.log.Error("failed to marshal new_values", zap.Error(err))
			return fmt.Errorf("marshal new_values: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		audit.UserID,
		audit.UserName,
		audit.Action,
		audit.ResourceType,
		audit.ResourceID,
		oldValuesJSON,
		newValuesJSON,
		audit.IPAddress,
		audit.UserAgent,
	)

	if err != nil {
		r.log.Error("failed to create audit log", zap.Error(err))
		return fmt.Errorf("create audit log: %w", err)
	}

	return nil
}

// GetByID получает запись аудита по ID
func (r *AuditRepo) GetByID(ctx context.Context, id int64) (*models.AuditLog, error) {
	query := `
		SELECT id, user_id, user_name, action, resource_type, resource_id, 
		       old_values, new_values, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE id = ?
	`

	audit := &models.AuditLog{}
	var oldValuesJSON, newValuesJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&audit.ID,
		&audit.UserID,
		&audit.UserName,
		&audit.Action,
		&audit.ResourceType,
		&audit.ResourceID,
		&oldValuesJSON,
		&newValuesJSON,
		&audit.IPAddress,
		&audit.UserAgent,
		&audit.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get audit log", zap.Error(err))
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	// Распарсиваем JSON если нужно
	if len(oldValuesJSON) > 0 {
		audit.OldValues = oldValuesJSON
	}
	if len(newValuesJSON) > 0 {
		audit.NewValues = newValuesJSON
	}

	return audit, nil
}

// List получает список записей аудита с фильтрацией
func (r *AuditRepo) List(ctx context.Context, filter models.AuditLogFilter) ([]*models.AuditLog, int, error) {
	// Построение запроса с динамическими условиями
	baseQuery := `
		SELECT id, user_id, user_name, action, resource_type, resource_id, 
		       old_values, new_values, ip_address, user_agent, created_at
		FROM audit_logs
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	whereConditions := make([]string, 0)

	if filter.UserID != nil {
		whereConditions = append(whereConditions, "user_id = ?")
		args = append(args, *filter.UserID)
	}

	if filter.ResourceType != nil {
		whereConditions = append(whereConditions, "resource_type = ?")
		args = append(args, *filter.ResourceType)
	}

	if filter.ResourceID != nil {
		whereConditions = append(whereConditions, "resource_id = ?")
		args = append(args, *filter.ResourceID)
	}

	if filter.Action != nil {
		whereConditions = append(whereConditions, "action = ?")
		args = append(args, *filter.Action)
	}

	if filter.DateFrom != nil {
		whereConditions = append(whereConditions, "created_at >= ?")
		args = append(args, *filter.DateFrom)
	}

	if filter.DateTo != nil {
		whereConditions = append(whereConditions, "created_at <= ?")
		args = append(args, *filter.DateTo)
	}

	if len(whereConditions) > 0 {
		baseQuery += " AND " + joinStrings(whereConditions, " AND ")
	}

	// Получение общего количества
	countQuery := "SELECT COUNT(*) FROM audit_logs WHERE 1=1"
	if len(whereConditions) > 0 {
		countQuery += " AND " + joinStrings(whereConditions, " AND ")
	}

	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		r.log.Error("failed to count audit logs", zap.Error(err))
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	// Добавление сортировки и лимитов
	baseQuery += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		r.log.Error("failed to list audit logs", zap.Error(err))
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	audits := make([]*models.AuditLog, 0)
	for rows.Next() {
		audit := &models.AuditLog{}
		var oldValuesJSON, newValuesJSON []byte

		err := rows.Scan(
			&audit.ID,
			&audit.UserID,
			&audit.UserName,
			&audit.Action,
			&audit.ResourceType,
			&audit.ResourceID,
			&oldValuesJSON,
			&newValuesJSON,
			&audit.IPAddress,
			&audit.UserAgent,
			&audit.CreatedAt,
		)
		if err != nil {
			r.log.Error("failed to scan audit log", zap.Error(err))
			return nil, 0, fmt.Errorf("scan audit log: %w", err)
		}

		if len(oldValuesJSON) > 0 {
			audit.OldValues = oldValuesJSON
		}
		if len(newValuesJSON) > 0 {
			audit.NewValues = newValuesJSON
		}

		audits = append(audits, audit)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("audit logs iteration error", zap.Error(err))
		return nil, 0, fmt.Errorf("audit logs iteration: %w", err)
	}

	return audits, total, nil
}

// DeleteOld удаляет старые записи аудита (для ротации логов)
func (r *AuditRepo) DeleteOld(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < ?`

	result, err := r.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		r.log.Error("failed to delete old audit logs", zap.Error(err))
		return 0, fmt.Errorf("delete old audit logs: %w", err)
	}

	affected, _ := result.RowsAffected()
	return affected, nil
}

// Helper функция для объединения условий
