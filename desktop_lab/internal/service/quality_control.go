package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"desktop_lab/internal/models"
	"desktop_lab/internal/repository/mysql_repo"
	"go.uber.org/zap"
)

// QualityControlService сервис контроля качества
type QualityControlService struct {
	auditRepo       *mysql_repo.AuditRepo
	versionRepo     *mysql_repo.ProtocolVersionRepo
	templateRepo    *mysql_repo.ProtocolTemplateRepo
	protocolRepo    *mysql_repo.ProtocolRepo // Для получения текущего протокола
	userRepo        *mysql_repo.UserRepo
	log             *zap.Logger
}

// NewQualityControlService создает новый сервис контроля качества
func NewQualityControlService(
	auditRepo *mysql_repo.AuditRepo,
	versionRepo *mysql_repo.ProtocolVersionRepo,
	templateRepo *mysql_repo.ProtocolTemplateRepo,
	protocolRepo *mysql_repo.ProtocolRepo,
	userRepo *mysql_repo.UserRepo,
	log *zap.Logger,
) *QualityControlService {
	return &QualityControlService{
		auditRepo:    auditRepo,
		versionRepo:  versionRepo,
		templateRepo: templateRepo,
		protocolRepo: protocolRepo,
		userRepo:     userRepo,
		log:          log,
	}
}

// ============================================================================
// AUDIT LOGS
// ============================================================================

// CreateAuditLog создает запись в журнале аудита
func (s *QualityControlService) CreateAuditLog(ctx context.Context, audit *models.AuditLog) error {
	// Автоматически заполняем время если не указано
	if audit.CreatedAt.IsZero() {
		audit.CreatedAt = time.Now()
	}

	return s.auditRepo.Create(ctx, audit)
}

// GetAuditLogs получает список записей аудита с фильтрацией
func (s *QualityControlService) GetAuditLogs(ctx context.Context, filter models.AuditLogFilter) ([]*models.AuditLog, int, error) {
	// Устанавливаем лимиты по умолчанию
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	return s.auditRepo.List(ctx, filter)
}

// GetAuditLogByID получает запись аудита по ID
func (s *QualityControlService) GetAuditLogByID(ctx context.Context, id int64) (*models.AuditLog, error) {
	return s.auditRepo.GetByID(ctx, id)
}

// LogProtocolAction логирует действие с протоколом
func (s *QualityControlService) LogProtocolAction(ctx context.Context, userID int64, userName string, action models.ActionType, protocolID int64, oldValues, newValues interface{}, ipAddress, userAgent string) error {
	audit := &models.AuditLog{
		UserID:       userID,
		UserName:     userName,
		Action:       action,
		ResourceType: "protocol",
		ResourceID:   protocolID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if oldValues != nil {
		data, err := json.Marshal(oldValues)
		if err != nil {
			s.log.Error("failed to marshal old values", zap.Error(err))
			return fmt.Errorf("marshal old values: %w", err)
		}
		audit.OldValues = data
	}

	if newValues != nil {
		data, err := json.Marshal(newValues)
		if err != nil {
			s.log.Error("failed to marshal new values", zap.Error(err))
			return fmt.Errorf("marshal new values: %w", err)
		}
		audit.NewValues = data
	}

	return s.CreateAuditLog(ctx, audit)
}

// ============================================================================
// PROTOCOL VERSIONS
// ============================================================================

// CreateProtocolVersion создает новую версию протокола
func (s *QualityControlService) CreateProtocolVersion(ctx context.Context, createReq *models.ProtocolVersionCreate) (*models.ProtocolVersion, error) {
	// Валидация
	if createReq.ProtocolID <= 0 {
		return nil, fmt.Errorf("invalid protocol_id")
	}
	if createReq.ChangedBy <= 0 {
		return nil, fmt.Errorf("invalid changed_by")
	}
	if len(createReq.ContentJSON) == 0 {
		return nil, fmt.Errorf("content_json is required")
	}

	// Проверяем, существует ли протокол (опционально, можно убрать если не критично)
	// protocol, err := s.protocolRepo.GetByID(ctx, strconv.FormatInt(createReq.ProtocolID, 10))
	// if err != nil {
	// 	return nil, fmt.Errorf("get protocol: %w", err)
	// }

	// Создаем версию
	version, err := s.versionRepo.Create(ctx, createReq)
	if err != nil {
		return nil, err
	}

	// Логируем создание версии
	_ = s.LogProtocolAction(ctx, createReq.ChangedBy, createReq.ChangedByName, models.ActionUpdate, createReq.ProtocolID, nil, map[string]interface{}{
		"version_number": version.VersionNumber,
		"comment":        createReq.Comment,
	}, "", "")

	s.log.Info("protocol version created",
		zap.Int64("protocol_id", createReq.ProtocolID),
		zap.Int("version_number", version.VersionNumber),
		zap.Int64("changed_by", createReq.ChangedBy))

	return version, nil
}

// GetProtocolVersions получает все версии протокола
func (s *QualityControlService) GetProtocolVersions(ctx context.Context, protocolID int64) ([]*models.ProtocolVersion, error) {
	if protocolID <= 0 {
		return nil, fmt.Errorf("invalid protocol_id")
	}

	return s.versionRepo.ListByProtocolID(ctx, protocolID)
}

// GetProtocolVersionByID получает версию протокола по ID
func (s *QualityControlService) GetProtocolVersionByID(ctx context.Context, versionID int64) (*models.ProtocolVersion, error) {
	if versionID <= 0 {
		return nil, fmt.Errorf("invalid version_id")
	}

	return s.versionRepo.GetByID(ctx, versionID)
}

// GetCurrentProtocolVersion получает текущую версию протокола
func (s *QualityControlService) GetCurrentProtocolVersion(ctx context.Context, protocolID int64) (*models.ProtocolVersion, error) {
	if protocolID <= 0 {
		return nil, fmt.Errorf("invalid protocol_id")
	}

	return s.versionRepo.GetCurrentVersion(ctx, protocolID)
}

// GetProtocolVersionByNumber получает версию протокола по номеру
func (s *QualityControlService) GetProtocolVersionByNumber(ctx context.Context, protocolID int64, versionNumber int) (*models.ProtocolVersion, error) {
	if protocolID <= 0 || versionNumber <= 0 {
		return nil, fmt.Errorf("invalid protocol_id or version_number")
	}

	return s.versionRepo.GetByVersionNumber(ctx, protocolID, versionNumber)
}

// GetProtocolVersionContent получает распарсенный контент версии
func (s *QualityControlService) GetProtocolVersionContent(ctx context.Context, versionID int64) (interface{}, error) {
	if versionID <= 0 {
		return nil, fmt.Errorf("invalid version_id")
	}

	return s.versionRepo.GetContentSnapshot(ctx, versionID)
}

// GetProtocolVersionPDF получает PDF версии
func (s *QualityControlService) GetProtocolVersionPDF(ctx context.Context, versionID int64) ([]byte, error) {
	if versionID <= 0 {
		return nil, fmt.Errorf("invalid version_id")
	}

	hasPDF, err := s.versionRepo.HasPDF(ctx, versionID)
	if err != nil {
		return nil, err
	}
	if !hasPDF {
		return nil, nil
	}

	return s.versionRepo.GetPDFSnapshot(ctx, versionID)
}

// DeleteProtocolVersion удаляет версию протокола (не текущую)
func (s *QualityControlService) DeleteProtocolVersion(ctx context.Context, versionID int64, deletedBy int64) error {
	if versionID <= 0 {
		return fmt.Errorf("invalid version_id")
	}

	version, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil {
		return err
	}
	if version == nil {
		return fmt.Errorf("version not found")
	}
	if version.IsCurrent {
		return fmt.Errorf("cannot delete current version")
	}

	err = s.versionRepo.Delete(ctx, versionID)
	if err != nil {
		return err
	}

	// Логируем удаление
	_ = s.LogProtocolAction(ctx, deletedBy, "", models.ActionDelete, version.ProtocolID, map[string]interface{}{
		"version_id":     versionID,
		"version_number": version.VersionNumber,
	}, nil, "", "")

	s.log.Info("protocol version deleted",
		zap.Int64("version_id", versionID),
		zap.Int64("protocol_id", version.ProtocolID))

	return nil
}

// ============================================================================
// PROTOCOL TEMPLATES
// ============================================================================

// CreateProtocolTemplate создает новый шаблон протокола
func (s *QualityControlService) CreateProtocolTemplate(ctx context.Context, createReq *models.ProtocolTemplateCreate) (*models.ProtocolTemplate, error) {
	// Валидация
	if createReq.OrganizationID <= 0 {
		return nil, fmt.Errorf("invalid organization_id")
	}
	if createReq.TestMethodID <= 0 {
		return nil, fmt.Errorf("invalid test_method_id")
	}
	if createReq.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if createReq.TemplateHTML == "" {
		return nil, fmt.Errorf("template_html is required")
	}
	if createReq.CreatedBy <= 0 {
		return nil, fmt.Errorf("invalid created_by")
	}

	// Проверяем, существует ли метод испытаний
	// method, err := s.testMethodRepo.GetByID(ctx, createReq.TestMethodID)
	// if err != nil {
	// 	return nil, fmt.Errorf("get test method: %w", err)
	// }
	// if method == nil {
	// 	return nil, fmt.Errorf("test method not found")
	// }

	template, err := s.templateRepo.Create(ctx, createReq)
	if err != nil {
		return nil, err
	}

	s.log.Info("protocol template created",
		zap.Int64("template_id", template.ID),
		zap.Int64("organization_id", createReq.OrganizationID),
		zap.Int64("test_method_id", createReq.TestMethodID))

	return template, nil
}

// GetProtocolTemplate получает шаблон по ID
func (s *QualityControlService) GetProtocolTemplate(ctx context.Context, id int64) (*models.ProtocolTemplateResponse, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid template_id")
	}

	return s.templateRepo.GetByIDWithMethodName(ctx, id)
}

// UpdateProtocolTemplate обновляет шаблон протокола
func (s *QualityControlService) UpdateProtocolTemplate(ctx context.Context, id int64, update *models.ProtocolTemplateUpdate) (*models.ProtocolTemplate, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid template_id")
	}

	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("template not found")
	}

	return s.templateRepo.Update(ctx, id, update)
}

// DeleteProtocolTemplate удаляет шаблон протокола
func (s *QualityControlService) DeleteProtocolTemplate(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("invalid template_id")
	}

	existing, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("template not found")
	}

	return s.templateRepo.Delete(ctx, id)
}

// GetProtocolTemplatesByOrganization получает шаблоны организации
func (s *QualityControlService) GetProtocolTemplatesByOrganization(ctx context.Context, orgID int64, activeOnly bool) ([]*models.ProtocolTemplateResponse, error) {
	if orgID <= 0 {
		return nil, fmt.Errorf("invalid organization_id")
	}

	return s.templateRepo.ListByOrganization(ctx, orgID, activeOnly)
}

// GetProtocolTemplatesByTestMethod получает шаблоны для метода испытаний
func (s *QualityControlService) GetProtocolTemplatesByTestMethod(ctx context.Context, testMethodID int64, activeOnly bool) ([]*models.ProtocolTemplate, error) {
	if testMethodID <= 0 {
		return nil, fmt.Errorf("invalid test_method_id")
	}

	return s.templateRepo.ListByTestMethod(ctx, testMethodID, activeOnly)
}

// GetActiveProtocolTemplate получает активный шаблон для организации и метода
func (s *QualityControlService) GetActiveProtocolTemplate(ctx context.Context, orgID, testMethodID int64) (*models.ProtocolTemplate, error) {
	if orgID <= 0 || testMethodID <= 0 {
		return nil, fmt.Errorf("invalid organization_id or test_method_id")
	}

	return s.templateRepo.GetActiveByOrgAndMethod(ctx, orgID, testMethodID)
}

// SetProtocolTemplateAsActive устанавливает шаблон как активный
func (s *QualityControlService) SetProtocolTemplateAsActive(ctx context.Context, templateID int64) error {
	if templateID <= 0 {
		return fmt.Errorf("invalid template_id")
	}

	return s.templateRepo.SetAsActive(ctx, templateID)
}

// RenderProtocolTemplate рендерит шаблон с данными (базовая реализация)
func (s *QualityControlService) RenderProtocolTemplate(ctx context.Context, templateID int64, data interface{}) (string, error) {
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		return "", err
	}
	if template == nil {
		return "", fmt.Errorf("template not found")
	}

	// Здесь будет логика рендеринга Go templates
	// Пока возвращаем сырой HTML
	return template.TemplateHTML, nil
}
