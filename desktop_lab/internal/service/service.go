package service

import (
	"context"
	"desktop_lab/internal/config"
	contextkeys "desktop_lab/internal/contextKey"
	"desktop_lab/internal/models"

	"go.uber.org/zap"
)

// Logger helper - centralized logging with context support
func loggerWith(ctx context.Context, log *zap.Logger, fields ...zap.Field) *zap.Logger {
	if reqID, ok := ctx.Value(contextkeys.RequestIDKey).(string); ok && reqID != "" {
		return log.With(append([]zap.Field{zap.String("request_id", reqID)}, fields...)...)
	}
	return log.With(fields...)
}

type DimensionRepo interface {
	GetAvailableDimensions(ctx context.Context) ([]models.ContextDimension, error)
	GetDimensionByID(ctx context.Context, id string) (models.ContextDimension, error)
	GetDimensionByKey(ctx context.Context, key string) (models.ContextDimension, error)
	AddDimension(ctx context.Context, dim models.ContextDimension) error
	UpdatePossibleValues(ctx context.Context, id string, values []string) error
	DeletePossibleValues(ctx context.Context, id string) error
	DeleteDimension(ctx context.Context, id string) error
}

// MaterialRepo работает с базовыми материалами
type MaterialRepo interface {
	Create(ctx context.Context, m models.Material) error
	GetAll(ctx context.Context) ([]models.Material, error)
	GetByID(ctx context.Context, id string) (models.Material, error)
	GetByName(ctx context.Context, name string) (models.Material, error)
	GetContextDimensionsByMaterialID(ctx context.Context, materialID string) ([]models.ContextDimension, error)
	AddContextDimensionToMaterial(ctx context.Context, materialID, dimensionID string, isRequired bool) error
	DeleteContextDimensionFromMaterial(ctx context.Context, materialID, dimensionID string) error
}

type OrganizationUserRepo interface {
	Create(ctx context.Context, id string, ou models.CreateOrganizationUserRequest) error
	GetByID(ctx context.Context, id string) (models.OrganizationUser, error)
	GetByRole(ctx context.Context, organizationID, role string, limit, offset int64) ([]models.OrganizationUser, int64, error)
	List(ctx context.Context, organizationID string, limit, offset int64) ([]models.OrganizationUser, int64, error)
	UpdateUser(ctx context.Context, id string, role *string) (models.OrganizationUser, error)
	Delete(ctx context.Context, id string) error
}

type OrganizationRepo interface {
	Create(ctx context.Context, id string, org models.CreateOrganizationRequest) error
	GetByID(ctx context.Context, id string) (models.Organization, error)
	GetOrganizationByName(ctx context.Context, name string) (models.Organization, error)
	List(ctx context.Context, limit, offset int64) ([]models.Organization, int64, error)
	Update(ctx context.Context, id string, req models.UpdateOrganizationRequest) (models.Organization, error)
	Delete(ctx context.Context, id string) error
}

type OrganizationTestsRepo interface {
	Create(ctx context.Context, id string, req models.CreateOrganizationTestRequest) error
	GetOrganizationTest(ctx context.Context, id string) (models.OrganizationTest, error)
	ListOrganizationTests(ctx context.Context, limit, offset int64) ([]models.OrganizationTest, int64, error)
	Update(ctx context.Context, id string, req models.UpdateOrganizationTestRequest) (models.OrganizationTest, error)
	DeleteOrganizationTest(ctx context.Context, id string) error
}

// StandardRepo управляет стандартами, методами и нормативами
type StandardRepo interface {
	CreateFull(ctx context.Context, req models.CreateStandardRequest) (string, error)
	GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error)
	GetApplicableLimit(ctx context.Context, methodID string, contextParams map[string]string) (models.NormativeLimit, error)
	GetTestMethod(ctx context.Context, methodID string) (models.TestMethod, error)
	GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error)
	GetMethodInputs(ctx context.Context, methodID string) ([]models.MethodInput, error)
	GetStandardDimensions(ctx context.Context, standardID string) ([]models.ContextDimension, error)
	GetMethodsFullByStandardID(ctx context.Context, standardID string) (map[string]models.TestMethodFull, error)
	GetStandardFull(ctx context.Context, standardID string) (models.StandardContext, error)
	GetMethodLimits(ctx context.Context, methodID string) ([]models.NormativeLimit, error)
	GetLimitConditions(ctx context.Context, limitID string) ([]models.LimitCondition, error)
	GetLimitConditionsForMethod(ctx context.Context, methodID string) (map[string][]models.LimitCondition, error)
	LinkDimensionToStandard(ctx context.Context, standardID, dimensionID string) error
}

// ExperimentGroupRepo управляет группами экспериментов
type ExperimentGroupRepo interface {
	Create(ctx context.Context, g models.ExperimentGroup) error
	GetByID(ctx context.Context, id string) (models.ExperimentGroup, error)
	GetList(ctx context.Context, filter models.GroupListFilter) ([]models.ExperimentGroup, int64, error)
	AddSampleToGroup(ctx context.Context, sampleID, groupID string) error
	RemoveSampleFromGroup(ctx context.Context, sampleID string) error
	UpdateGroup(ctx context.Context, id string, g models.UpdateExperimentGroup) error
	DeleteGroup(ctx context.Context, id string) error
}

// SampleRepo управляет пробами
type SampleRepo interface {
	Create(ctx context.Context, s models.Sample) error
	GetByID(ctx context.Context, id string) (models.Sample, error)
	GetByGroupID(ctx context.Context, groupID string) ([]models.Sample, error)
	Update(ctx context.Context, s models.Sample) error
	Delete(ctx context.Context, id string) error
}

// ProtocolRepo управляет протоколами и результатами
type ProtocolRepo interface {
	CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error
	CreateWithSample(ctx context.Context, sample models.Sample, protocol models.Protocol, results []models.TestResult) error
	GetByID(ctx context.Context, id string) (models.Protocol, error)
	GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error)
	GetResultsByProtocolID(ctx context.Context, protocolID string) ([]models.TestResult, error)
	GetList(ctx context.Context, filter models.ProtocolListFilter) ([]models.Protocol, int64, error)
	GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateProtocol(ctx context.Context, id string, req models.UpdateProtocolRequest) error
	DeleteProtocol(ctx context.Context, id string) error
}

type UserRepo interface {
	Create(ctx context.Context, user models.User) error
	GetByID(ctx context.Context, id string) (models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Credential(ctx context.Context, email string) (string, string, error)
	Update(ctx context.Context, id string, user models.UpdateUserRequest) (models.User, error)
	Delete(ctx context.Context, id string) error
	ListActive(ctx context.Context) ([]models.User, error)
}

type TokenRepo interface {
	Create(ctx context.Context, token models.Token) error
	TokenByID(ctx context.Context, id string) (models.Token, error)
	Delete(ctx context.Context, id string) error
}

type Services struct {
	Auth          *AuthService
	Dimensions    *DimensionService
	Groups        *ExperimentGroupService
	Invitations   *InvitationService
	Materials     *MaterialService
	Organizations *OrganizationService
	Profile       *ProfileService
	Standards     *StandardService
	Protocols     *ProtocolService
	Samples       *SampleService
	Reports       *ReportService
	Orders        *OrderService
}

func NewServices(
	matRepo MaterialRepo,
	stdRepo StandardRepo,
	protRepo ProtocolRepo,
	sampRepo SampleRepo,
	groupRepo ExperimentGroupRepo,
	tokenRepo TokenRepo,
	userRepo UserRepo,
	dimRepo DimensionRepo,
	orgRepo OrganizationRepo,
	orgTestsRepo OrganizationTestsRepo,
	orgUserRepo OrganizationUserRepo,
	orderRepo OrderRepo,
	invRepo InvitationRepo,

	fontDir string,
	templatesDir string,
	wkhtmltopdfWindows []byte,
	cfg config.Config,
	log *zap.Logger,
) *Services {
	auth := NewAuthService(userRepo, tokenRepo, cfg.Auth, log)
	material := NewMaterialService(matRepo, log)
	standards := NewStandardService(stdRepo, log)
	sample := NewSampleService(sampRepo, log)
	group := NewExperimentGroupService(groupRepo, log)
	protocol := NewProtocolService(protRepo, sampRepo, stdRepo, groupRepo, matRepo, log)
	profile := NewProfileService(userRepo, log)
	report := NewReportService(protocol, material, fontDir, templatesDir, wkhtmltopdfWindows, log)
	dimension := NewDimensionService(dimRepo, log)
	organization := NewOrganizationService(orgRepo, orgUserRepo, orgTestsRepo, log)
	orders := NewOrderService(orderRepo, orgTestsRepo, log)
	invitations := NewInvitationService(invRepo, orgRepo, userRepo, log)
	return &Services{
		Auth:          auth,
		Materials:     material,
		Organizations: organization,
		Standards:     standards,
		Profile:       profile,
		Protocols:     protocol,
		Groups:        group,
		Samples:       sample,
		Reports:       report,
		Dimensions:    dimension,
		Orders:        orders,
		Invitations:   invitations,
	}
}
