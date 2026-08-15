package mysql_repo

import (
	"context"
	"database/sql"
	"fmt"

	"desktop_lab/internal/models"
	"go.uber.org/zap"
)

// ProtocolTemplateRepo реализует репозиторий для шаблонов протоколов
type ProtocolTemplateRepo struct {
	db  *sql.DB
	log *zap.Logger
}

// NewProtocolTemplateRepo создает новый репозиторий шаблонов протоколов
func NewProtocolTemplateRepo(db *sqlx.DB, log *zap.Logger) *ProtocolTemplateRepo {
	return &ProtocolTemplateRepo{db: db, log: log}
}

// Create создает новый шаблон протокола
func (r *ProtocolTemplateRepo) Create(ctx context.Context, template *models.ProtocolTemplateCreate) (*models.ProtocolTemplate, error) {
	query := `
		INSERT INTO protocol_templates 
		(organization_id, test_method_id, name, description, template_html, template_css, is_active, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		template.OrganizationID,
		template.TestMethodID,
		template.Name,
		template.Description,
		template.TemplateHTML,
		template.TemplateCSS,
		template.IsActive,
		template.CreatedBy,
	)
	if err != nil {
		r.log.Error("failed to create protocol template", zap.Error(err))
		return nil, fmt.Errorf("create protocol template: %w", err)
	}

	id, _ := result.LastInsertId()

	// Возвращаем созданный шаблон
	return r.GetByID(ctx, id)
}

// GetByID получает шаблон по ID
func (r *ProtocolTemplateRepo) GetByID(ctx context.Context, id int64) (*models.ProtocolTemplate, error) {
	query := `
		SELECT id, organization_id, test_method_id, name, description, 
		       template_html, template_css, is_active, created_by, created_at, updated_at
		FROM protocol_templates
		WHERE id = ?
	`

	template := &models.ProtocolTemplate{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.OrganizationID,
		&template.TestMethodID,
		&template.Name,
		&template.Description,
		&template.TemplateHTML,
		&template.TemplateCSS,
		&template.IsActive,
		&template.CreatedBy,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get protocol template", zap.Error(err))
		return nil, fmt.Errorf("get protocol template: %w", err)
	}

	return template, nil
}

// GetByIDWithMethodName получает шаблон с именем метода
func (r *ProtocolTemplateRepo) GetByIDWithMethodName(ctx context.Context, id int64) (*models.ProtocolTemplateResponse, error) {
	query := `
		SELECT pt.id, pt.organization_id, pt.test_method_id, tm.name as method_name, 
		       pt.name, pt.description, pt.is_active, pt.created_by, u.name as created_by_name,
		       pt.created_at, pt.updated_at
		FROM protocol_templates pt
		LEFT JOIN test_methods tm ON pt.test_method_id = tm.id
		LEFT JOIN users u ON pt.created_by = u.id
		WHERE pt.id = ?
	`

	template := &models.ProtocolTemplateResponse{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.OrganizationID,
		&template.TestMethodID,
		&template.MethodName,
		&template.Name,
		&template.Description,
		&template.IsActive,
		&template.CreatedBy,
		&template.CreatedByName,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get protocol template with method name", zap.Error(err))
		return nil, fmt.Errorf("get protocol template with method name: %w", err)
	}

	return template, nil
}

// Update обновляет шаблон протокола
func (r *ProtocolTemplateRepo) Update(ctx context.Context, id int64, update *models.ProtocolTemplateUpdate) (*models.ProtocolTemplate, error) {
	// Построение динамического UPDATE запроса
	setClauses := make([]string, 0)
	args := make([]interface{}, 0)

	if update.Name != "" {
		setClauses = append(setClauses, "name = ?")
		args = append(args, update.Name)
	}
	if update.Description != "" {
		setClauses = append(setClauses, "description = ?")
		args = append(args, update.Description)
	}
	if update.TemplateHTML != "" {
		setClauses = append(setClauses, "template_html = ?")
		args = append(args, update.TemplateHTML)
	}
	if update.TemplateCSS != "" {
		setClauses = append(setClauses, "template_css = ?")
		args = append(args, update.TemplateCSS)
	}
	if update.IsActive != nil {
		setClauses = append(setClauses, "is_active = ?")
		args = append(args, *update.IsActive)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE protocol_templates 
		SET %s
		WHERE id = ?
	`, joinStrings(setClauses, ", "))

	// Пересобираем args с ID в конце
	args = append(args[:len(args)-1], id)

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.Error("failed to update protocol template", zap.Error(err))
		return nil, fmt.Errorf("update protocol template: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Delete удаляет шаблон протокола
func (r *ProtocolTemplateRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM protocol_templates WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.Error("failed to delete protocol template", zap.Error(err))
		return fmt.Errorf("delete protocol template: %w", err)
	}

	return nil
}

// ListByOrganization получает список шаблонов организации
func (r *ProtocolTemplateRepo) ListByOrganization(ctx context.Context, orgID int64, activeOnly bool) ([]*models.ProtocolTemplateResponse, error) {
	query := `
		SELECT pt.id, pt.organization_id, pt.test_method_id, tm.name as method_name,
		       pt.name, pt.description, pt.is_active, pt.created_by, u.name as created_by_name,
		       pt.created_at, pt.updated_at
		FROM protocol_templates pt
		LEFT JOIN test_methods tm ON pt.test_method_id = tm.id
		LEFT JOIN users u ON pt.created_by = u.id
		WHERE pt.organization_id = ?
	`

	if activeOnly {
		query += " AND pt.is_active = 1"
	}
	query += " ORDER BY pt.name ASC"

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		r.log.Error("failed to list protocol templates", zap.Error(err))
		return nil, fmt.Errorf("list protocol templates: %w", err)
	}
	defer rows.Close()

	templates := make([]*models.ProtocolTemplateResponse, 0)
	for rows.Next() {
		template := &models.ProtocolTemplateResponse{}
		var methodName, createdByByName sql.NullString

		err := rows.Scan(
			&template.ID,
			&template.OrganizationID,
			&template.TestMethodID,
			&methodName,
			&template.Name,
			&template.Description,
			&template.IsActive,
			&template.CreatedBy,
			&createdByByName,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			r.log.Error("failed to scan protocol template", zap.Error(err))
			return nil, fmt.Errorf("scan protocol template: %w", err)
		}

		if methodName.Valid {
			template.MethodName = methodName.String
		}
		if createdByByName.Valid {
			template.CreatedByName = createdByByName.String
		}

		templates = append(templates, template)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("protocol templates iteration error", zap.Error(err))
		return nil, fmt.Errorf("protocol templates iteration: %w", err)
	}

	return templates, nil
}

// ListByTestMethod получает шаблоны для конкретного метода испытаний
func (r *ProtocolTemplateRepo) ListByTestMethod(ctx context.Context, testMethodID int64, activeOnly bool) ([]*models.ProtocolTemplate, error) {
	query := `
		SELECT id, organization_id, test_method_id, name, description, 
		       template_html, template_css, is_active, created_by, created_at, updated_at
		FROM protocol_templates
		WHERE test_method_id = ?
	`

	if activeOnly {
		query += " AND is_active = 1"
	}
	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, testMethodID)
	if err != nil {
		r.log.Error("failed to list templates by test method", zap.Error(err))
		return nil, fmt.Errorf("list templates by test method: %w", err)
	}
	defer rows.Close()

	templates := make([]*models.ProtocolTemplate, 0)
	for rows.Next() {
		template := &models.ProtocolTemplate{}
		err := rows.Scan(
			&template.ID,
			&template.OrganizationID,
			&template.TestMethodID,
			&template.Name,
			&template.Description,
			&template.TemplateHTML,
			&template.TemplateCSS,
			&template.IsActive,
			&template.CreatedBy,
			&template.CreatedAt,
			&template.UpdatedAt,
		)
		if err != nil {
			r.log.Error("failed to scan protocol template", zap.Error(err))
			return nil, fmt.Errorf("scan protocol template: %w", err)
		}
		templates = append(templates, template)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("protocol templates iteration error", zap.Error(err))
		return nil, fmt.Errorf("protocol templates iteration: %w", err)
	}

	return templates, nil
}

// GetActiveByOrgAndMethod получает активный шаблон для организации и метода
func (r *ProtocolTemplateRepo) GetActiveByOrgAndMethod(ctx context.Context, orgID, testMethodID int64) (*models.ProtocolTemplate, error) {
	query := `
		SELECT id, organization_id, test_method_id, name, description, 
		       template_html, template_css, is_active, created_by, created_at, updated_at
		FROM protocol_templates
		WHERE organization_id = ? AND test_method_id = ? AND is_active = 1
		LIMIT 1
	`

	template := &models.ProtocolTemplate{}
	err := r.db.QueryRowContext(ctx, query, orgID, testMethodID).Scan(
		&template.ID,
		&template.OrganizationID,
		&template.TestMethodID,
		&template.Name,
		&template.Description,
		&template.TemplateHTML,
		&template.TemplateCSS,
		&template.IsActive,
		&template.CreatedBy,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get active template by org and method", zap.Error(err))
		return nil, fmt.Errorf("get active template by org and method: %w", err)
	}

	return template, nil
}

// SetAsActive устанавливает шаблон как активный (и деактивирует другие для того же метода)
func (r *ProtocolTemplateRepo) SetAsActive(ctx context.Context, id int64) error {
	// Сначала получаем organization_id и test_method_id шаблона
	template, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if template == nil {
		return fmt.Errorf("template not found")
	}

	// Деактивируем все шаблоны для этой организации и метода
	deactivateQuery := `
		UPDATE protocol_templates 
		SET is_active = 0, updated_at = CURRENT_TIMESTAMP
		WHERE organization_id = ? AND test_method_id = ?
	`
	_, err = r.db.ExecContext(ctx, deactivateQuery, template.OrganizationID, template.TestMethodID)
	if err != nil {
		r.log.Error("failed to deactivate other templates", zap.Error(err))
		return fmt.Errorf("deactivate other templates: %w", err)
	}

	// Активируем целевой шаблон
	activateQuery := `
		UPDATE protocol_templates 
		SET is_active = 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err = r.db.ExecContext(ctx, activateQuery, id)
	if err != nil {
		r.log.Error("failed to activate template", zap.Error(err))
		return fmt.Errorf("activate template: %w", err)
	}

	return nil
}

// Helper функция для объединения строк
