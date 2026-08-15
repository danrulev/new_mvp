package mysql_repo

import (
	"context"
	"database/sql"
	"desktop_lab/internal/models"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// ProtocolVersionRepo реализует репозиторий для версий протоколов
type ProtocolVersionRepo struct {
	db  *sqlx.DB
	log *zap.Logger
}

// NewProtocolVersionRepo создает новый репозиторий версий протоколов
func NewProtocolVersionRepo(db *sqlx.DB, log *zap.Logger) *ProtocolVersionRepo {
	return &ProtocolVersionRepo{db: db, log: log}
}

// Create создает новую версию протокола
func (r *ProtocolVersionRepo) Create(ctx context.Context, version *models.ProtocolVersionCreate) (*models.ProtocolVersion, error) {
	// Получаем текущий максимальный номер версии
	maxVersionQuery := `SELECT COALESCE(MAX(version_number), 0) FROM protocol_versions WHERE protocol_id = ?`
	var maxVersion int
	err := r.db.QueryRowContext(ctx, maxVersionQuery, version.ProtocolID).Scan(&maxVersion)
	if err != nil {
		r.log.Error("failed to get max version", zap.Error(err))
		return nil, fmt.Errorf("get max version: %w", err)
	}

	newVersionNumber := maxVersion + 1

	// Сбрасываем флаг is_current у всех предыдущих версий
	resetCurrentQuery := `UPDATE protocol_versions SET is_current = 0 WHERE protocol_id = ?`
	_, err = r.db.ExecContext(ctx, resetCurrentQuery, version.ProtocolID)
	if err != nil {
		r.log.Error("failed to reset current versions", zap.Error(err))
		return nil, fmt.Errorf("reset current versions: %w", err)
	}

	// Создаем новую версию
	insertQuery := `
		INSERT INTO protocol_versions 
		(protocol_id, version_number, content_snapshot, pdf_snapshot, changed_by, changed_by_name, comment, is_current)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1)
	`

	result, err := r.db.ExecContext(ctx, insertQuery,
		version.ProtocolID,
		newVersionNumber,
		version.ContentJSON,
		version.PDFFile,
		version.ChangedBy,
		version.ChangedByName,
		version.Comment,
	)
	if err != nil {
		r.log.Error("failed to create protocol version", zap.Error(err))
		return nil, fmt.Errorf("create protocol version: %w", err)
	}

	id, _ := result.LastInsertId()

	// Возвращаем созданную версию
	return r.GetByID(ctx, id)
}

// GetByID получает версию протокола по ID
func (r *ProtocolVersionRepo) GetByID(ctx context.Context, id int64) (*models.ProtocolVersion, error) {
	query := `
		SELECT id, protocol_id, version_number, content_snapshot, pdf_snapshot, 
		       changed_by, changed_by_name, changed_at, comment, is_current
		FROM protocol_versions
		WHERE id = ?
	`

	version := &models.ProtocolVersion{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&version.ID,
		&version.ProtocolID,
		&version.VersionNumber,
		&version.ContentSnapshot,
		&version.PDFSnapshot,
		&version.ChangedBy,
		&version.ChangedByName,
		&version.ChangedAt,
		&version.Comment,
		&version.IsCurrent,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get protocol version", zap.Error(err))
		return nil, fmt.Errorf("get protocol version: %w", err)
	}

	return version, nil
}

// ListByProtocolID получает все версии протокола
func (r *ProtocolVersionRepo) ListByProtocolID(ctx context.Context, protocolID int64) ([]*models.ProtocolVersion, error) {
	query := `
		SELECT id, protocol_id, version_number, content_snapshot, pdf_snapshot, 
		       changed_by, changed_by_name, changed_at, comment, is_current
		FROM protocol_versions
		WHERE protocol_id = ?
		ORDER BY version_number DESC
	`

	rows, err := r.db.QueryContext(ctx, query, protocolID)
	if err != nil {
		r.log.Error("failed to list protocol versions", zap.Error(err))
		return nil, fmt.Errorf("list protocol versions: %w", err)
	}
	defer rows.Close()

	versions := make([]*models.ProtocolVersion, 0)
	for rows.Next() {
		version := &models.ProtocolVersion{}
		err := rows.Scan(
			&version.ID,
			&version.ProtocolID,
			&version.VersionNumber,
			&version.ContentSnapshot,
			&version.PDFSnapshot,
			&version.ChangedBy,
			&version.ChangedByName,
			&version.ChangedAt,
			&version.Comment,
			&version.IsCurrent,
		)
		if err != nil {
			r.log.Error("failed to scan protocol version", zap.Error(err))
			return nil, fmt.Errorf("scan protocol version: %w", err)
		}
		versions = append(versions, version)
	}

	if err = rows.Err(); err != nil {
		r.log.Error("protocol versions iteration error", zap.Error(err))
		return nil, fmt.Errorf("protocol versions iteration: %w", err)
	}

	return versions, nil
}

// GetCurrentVersion получает текущую активную версию протокола
func (r *ProtocolVersionRepo) GetCurrentVersion(ctx context.Context, protocolID int64) (*models.ProtocolVersion, error) {
	query := `
		SELECT id, protocol_id, version_number, content_snapshot, pdf_snapshot, 
		       changed_by, changed_by_name, changed_at, comment, is_current
		FROM protocol_versions
		WHERE protocol_id = ? AND is_current = 1
		LIMIT 1
	`

	version := &models.ProtocolVersion{}
	err := r.db.QueryRowContext(ctx, query, protocolID).Scan(
		&version.ID,
		&version.ProtocolID,
		&version.VersionNumber,
		&version.ContentSnapshot,
		&version.PDFSnapshot,
		&version.ChangedBy,
		&version.ChangedByName,
		&version.ChangedAt,
		&version.Comment,
		&version.IsCurrent,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get current protocol version", zap.Error(err))
		return nil, fmt.Errorf("get current protocol version: %w", err)
	}

	return version, nil
}

// GetByVersionNumber получает версию протокола по номеру
func (r *ProtocolVersionRepo) GetByVersionNumber(ctx context.Context, protocolID int64, versionNumber int) (*models.ProtocolVersion, error) {
	query := `
		SELECT id, protocol_id, version_number, content_snapshot, pdf_snapshot, 
		       changed_by, changed_by_name, changed_at, comment, is_current
		FROM protocol_versions
		WHERE protocol_id = ? AND version_number = ?
	`

	version := &models.ProtocolVersion{}
	err := r.db.QueryRowContext(ctx, query, protocolID, versionNumber).Scan(
		&version.ID,
		&version.ProtocolID,
		&version.VersionNumber,
		&version.ContentSnapshot,
		&version.PDFSnapshot,
		&version.ChangedBy,
		&version.ChangedByName,
		&version.ChangedAt,
		&version.Comment,
		&version.IsCurrent,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get protocol version by number", zap.Error(err))
		return nil, fmt.Errorf("get protocol version by number: %w", err)
	}

	return version, nil
}

// Delete удаляет версию протокола (нельзя удалить текущую версию)
func (r *ProtocolVersionRepo) Delete(ctx context.Context, id int64) error {
	// Проверяем, не является ли версия текущей
	version, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if version == nil {
		return fmt.Errorf("version not found")
	}
	if version.IsCurrent {
		return fmt.Errorf("cannot delete current version")
	}

	deleteQuery := `DELETE FROM protocol_versions WHERE id = ?`
	_, err = r.db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		r.log.Error("failed to delete protocol version", zap.Error(err))
		return fmt.Errorf("delete protocol version: %w", err)
	}

	return nil
}

// GetContentSnapshot распарсивает JSON контент версии
func (r *ProtocolVersionRepo) GetContentSnapshot(ctx context.Context, id int64) (interface{}, error) {
	version, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if version == nil {
		return nil, fmt.Errorf("version not found")
	}

	var content interface{}
	err = json.Unmarshal(version.ContentSnapshot, &content)
	if err != nil {
		r.log.Error("failed to unmarshal content snapshot", zap.Error(err))
		return nil, fmt.Errorf("unmarshal content snapshot: %w", err)
	}

	return content, nil
}

// HasPDF проверяет, есть ли PDF у версии
func (r *ProtocolVersionRepo) HasPDF(ctx context.Context, id int64) (bool, error) {
	query := `SELECT CASE WHEN pdf_snapshot IS NOT NULL AND LENGTH(pdf_snapshot) > 0 THEN 1 ELSE 0 END 
			  FROM protocol_versions WHERE id = ?`

	var hasPDF int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&hasPDF)
	if err != nil {
		r.log.Error("failed to check PDF existence", zap.Error(err))
		return false, fmt.Errorf("check PDF existence: %w", err)
	}

	return hasPDF == 1, nil
}

// GetPDFSnapshot получает бинарные данные PDF
func (r *ProtocolVersionRepo) GetPDFSnapshot(ctx context.Context, id int64) ([]byte, error) {
	query := `SELECT pdf_snapshot FROM protocol_versions WHERE id = ?`

	var pdfData []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(&pdfData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.log.Error("failed to get PDF snapshot", zap.Error(err))
		return nil, fmt.Errorf("get PDF snapshot: %w", err)
	}

	return pdfData, nil
}
