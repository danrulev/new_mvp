package repository

import (
	"context"
	"desktop_lab/internal/models"
)

// MaterialRepo работает с базовыми материалами
type MaterialRepo interface {
	Create(ctx context.Context, m models.Material) error
	GetAll(ctx context.Context) ([]models.Material, error)
	GetByID(ctx context.Context, id string) (models.Material, error)
	GetByName(ctx context.Context, name string) (models.Material, error)
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
}

// ExperimentGroupRepo управляет группами экспериментов
type ExperimentGroupRepo interface {
	Create(ctx context.Context, g models.ExperimentGroup) error
	GetByID(ctx context.Context, id string) (models.ExperimentGroup, error)
	GetList(ctx context.Context, limit, offset int64) ([]models.ExperimentGroup, int64, error)
}

// SampleRepo управляет пробами
type SampleRepo interface {
	Create(ctx context.Context, s models.Sample) error
	GetByID(ctx context.Context, id string) (models.Sample, error)
	GetByGroupID(ctx context.Context, groupID string) ([]models.Sample, error)
}

// ProtocolRepo управляет протоколами и результатами
type ProtocolRepo interface {
	CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error
	GetByID(ctx context.Context, id string) (models.Protocol, error)
	GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error)
	GetResultsByProtocolID(ctx context.Context, protocolID string) ([]models.TestResult, error)
	GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error)
	GetProtocolFull(ctx context.Context, id string) (models.ProtocolFull, error)
}
