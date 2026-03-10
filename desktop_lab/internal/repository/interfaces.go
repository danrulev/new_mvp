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
	// CreateFull создает стандарт со всеми вложенными сущностями (транзакция)
	CreateFull(ctx context.Context, req models.CreateStandardRequest) (string, error)

	// GetByMaterialID возвращает стандарты с загруженными методами и лимитами
	GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error)

	// GetApplicableLimit находит подходящий лимит для метода и контекста пробы
	// Это ключевая функция для валидации
	GetApplicableLimit(ctx context.Context, methodID string, contextParams map[string]string) (*models.NormativeLimit, error)

	// GetMethodWithInputs загружает метод вместе с параметрами ввода
	GetMethodWithInputs(ctx context.Context, methodID string) (*models.TestMethod, error)

	GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error)
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
	GetByID(ctx context.Context, id string) (*models.Sample, error)
}

// ProtocolRepo управляет протоколами и результатами
type ProtocolRepo interface {
	// CreateFull создает протокол, привязывает к пробе и сохраняет результаты (транзакция)
	CreateFull(ctx context.Context, protocol models.Protocol, results []models.TestResult) error

	// GetByID загружает протокол с результатами и данными пробы
	GetByID(ctx context.Context, id string) (*models.Protocol, error)

	// GetByGroupID загружает список протоколов группы (без полных результатов, только мета)
	GetByGroupID(ctx context.Context, groupID string) ([]models.Protocol, error)

	// GetResultsByProtocolID загружает результаты конкретного протокола
	GetResultsByProtocolID(ctx context.Context, protocolID string) ([]models.TestResult, error)
	GetList(ctx context.Context, limit, offset int64) ([]models.Protocol, int64, error)
}
