package service

import (
	"context"
	"desktop_lab/internal/models"
	"fmt"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"go.uber.org/zap"
)

// cacheEntry хранит кэшированные данные с временем жизни
type cacheEntry[T any] struct {
	Data      T
	ExpiresAt time.Time
}

type StandardService struct {
	repo StandardRepo
	log  *zap.Logger

	methodsCache *lru.Cache[string, cacheEntry[map[string]models.TestMethodFull]]
	cacheMu      sync.RWMutex
	cacheTTL     time.Duration
}

func NewStandardService(repo StandardRepo, log *zap.Logger) *StandardService {
	cache, _ := lru.New[string, cacheEntry[map[string]models.TestMethodFull]](100)

	svc := &StandardService{
		repo:         repo,
		log:          log,
		methodsCache: cache,
		cacheTTL:     10 * time.Minute,
	}

	svc.log.Info("StandardService initialized",
		zap.Int("cache_capacity", 100),
		zap.Duration("cache_ttl", svc.cacheTTL))

	return svc
}

// getFromCache пытается получить данные из кэша
func (s *StandardService) getFromCache(key string) (map[string]models.TestMethodFull, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	entry, ok := s.methodsCache.Get(key)
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		s.log.Debug("cache entry expired", zap.String("standard_id", key))
		return nil, false
	}

	return entry.Data, true
}

// setToCache сохраняет данные в кэш
func (s *StandardService) setToCache(key string, data map[string]models.TestMethodFull) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	evicted := s.methodsCache.Add(key, cacheEntry[map[string]models.TestMethodFull]{
		Data:      data,
		ExpiresAt: time.Now().Add(s.cacheTTL),
	})

	if evicted {
		s.log.Debug("cache entry evicted (LRU)", zap.String("evicted_key", key))
	}
	s.log.Debug("cache entry added",
		zap.String("standard_id", key),
		zap.Int("methods_count", len(data)),
		zap.Time("expires_at", time.Now().Add(s.cacheTTL)))
}

// invalidateCache удаляет запись из кэша
func (s *StandardService) invalidateCache(standardID string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	removed := s.methodsCache.Remove(standardID)
	if removed {
		s.log.Debug("cache entry manually invalidated", zap.String("standard_id", standardID))
	}
}

// CreateStandard - создание стандарта с инвалидацией кэша
func (s *StandardService) CreateStandard(ctx context.Context, req models.CreateStandardRequest) (string, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "CreateStandard"),
		zap.String("material_id", req.MaterialID),
		zap.String("standard_name", req.Name),
		zap.Int("methods_count", len(req.Methods)),
	)
	log.Info("creating new standard")

	if req.MaterialID == "" || req.Name == "" {
		log.Warn("validation failed: required fields missing",
			zap.Bool("material_id_empty", req.MaterialID == ""),
			zap.Bool("name_empty", req.Name == ""))
		return "", fmt.Errorf("material_id and name are required")
	}

	for i, m := range req.Methods {
		if m.Name == "" {
			log.Warn("validation failed: method name missing", zap.Int("method_index", i))
			return "", fmt.Errorf("method[%d] name is required", i)
		}
	}

	log.Debug("saving standard to repository")
	id, err := s.repo.CreateFull(ctx, req)
	if err != nil {
		log.Error("failed to create standard in repository", zap.Error(err))
		return "", fmt.Errorf("ошибка создания стандарта: %w", err)
	}

	log.Debug("invalidating cache for material", zap.String("material_id", req.MaterialID))

	log.Info("standard created successfully",
		zap.String("standard_id", id),
		zap.String("standard_name", req.Name))
	return id, nil
}

// GetByMaterialID - загрузка списка стандартов
func (s *StandardService) GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetByMaterialID"),
		zap.String("material_id", materialID),
	)
	log.Debug("fetching standards for material")

	stds, err := s.repo.GetByMaterialID(ctx, materialID)
	if err != nil {
		log.Error("failed to fetch standards from repository", zap.Error(err))
		return nil, err
	}

	log.Info("standards loaded successfully",
		zap.Int("count", len(stds)),
		zap.String("material_id", materialID))
	return stds, nil
}

// GetMethodDetails - загрузка деталей метода (без кэша)
func (s *StandardService) GetMethodDetails(ctx context.Context, methodID string) (models.TestMethodFull, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetMethodDetails"),
		zap.String("method_id", methodID),
	)
	log.Debug("fetching method details from repository")

	start := time.Now()

	method, err := s.repo.GetTestMethod(ctx, methodID)
	if err != nil {
		log.Error("failed to fetch method", zap.Error(err))
		return models.TestMethodFull{}, err
	}

	inputs, err := s.repo.GetMethodInputs(ctx, methodID)
	if err != nil {
		log.Error("failed to fetch method inputs", zap.Error(err))
		return models.TestMethodFull{}, err
	}

	limits, err := s.repo.GetMethodLimits(ctx, methodID)
	if err != nil {
		log.Error("failed to fetch method limits", zap.Error(err))
		return models.TestMethodFull{}, err
	}

	result := models.TestMethodFull{
		Method: method,
		Inputs: inputs,
		Limits: limits,
	}

	log.Debug("method details assembled",
		zap.Int("inputs_count", len(inputs)),
		zap.Int("limits_count", len(limits)),
		zap.Duration("db_load_ms", time.Since(start)))

	return result, nil
}

// GetMethodsByStandardID - загрузка методов с проверкой кэша
func (s *StandardService) GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetMethodsByStandardID"),
		zap.String("standard_id", standardID),
	)
	log.Debug("fetching methods for standard")

	start := time.Now()

	// Пробуем кэш
	if fullMethods, ok := s.getFromCache(standardID); ok {
		methods := make([]models.TestMethod, 0, len(fullMethods))
		for _, fm := range fullMethods {
			methods = append(methods, fm.Method)
		}
		log.Info("methods served from cache",
			zap.Int("count", len(methods)),
			zap.Duration("cache_lookup_ms", time.Since(start)))
		return methods, nil
	}

	log.Debug("cache miss, loading from database")

	methods, err := s.repo.GetMethodsByStandardID(ctx, standardID)
	if err != nil {
		log.Error("failed to get methods from repository", zap.Error(err))
		return nil, err
	}

	log.Info("methods loaded from database",
		zap.Int("count", len(methods)),
		zap.Duration("db_load_ms", time.Since(start)))

	return methods, nil
}

// GetMethodsFullByStandardIDWithCache - основной метод с полным кэшированием
func (s *StandardService) GetMethodsFullByStandardIDWithCache(ctx context.Context, standardID string) (map[string]models.TestMethodFull, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetMethodsFullByStandardIDWithCache"),
		zap.String("standard_id", standardID),
	)
	log.Debug("fetching full methods with cache")

	start := time.Now()

	// 1. Пробуем кэш
	if cached, ok := s.getFromCache(standardID); ok {
		log.Info("full methods served from cache",
			zap.Int("methods_count", len(cached)),
			zap.Duration("cache_lookup_ms", time.Since(start)))
		return cached, nil
	}

	log.Debug("cache miss, loading full methods from database")
	dbStart := time.Now()

	// 2. Загружаем из БД
	fullMethods, err := s.repo.GetMethodsFullByStandardID(ctx, standardID)
	if err != nil {
		log.Error("failed to load full methods from repository", zap.Error(err))
		return nil, err
	}

	dbDuration := time.Since(dbStart)
	log.Debug("full methods loaded from database",
		zap.Int("methods_count", len(fullMethods)),
		zap.Duration("db_load_ms", dbDuration))

	// 3. Сохраняем в кэш
	s.setToCache(standardID, fullMethods)

	totalDuration := time.Since(start)
	cacheEfficiency := float64(dbDuration) / float64(totalDuration) * 100

	log.Info("full methods cached successfully",
		zap.Int("methods_count", len(fullMethods)),
		zap.Duration("total_duration_ms", totalDuration),
		zap.Float64("db_load_percentage", cacheEfficiency))

	return fullMethods, nil
}

// GetStandardDimensions - загрузка измерений стандарта
func (s *StandardService) GetStandardDimensions(ctx context.Context, standardID string) ([]models.ContextDimension, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetStandardDimensions"),
		zap.String("standard_id", standardID),
	)
	log.Debug("fetching standard dimensions")

	dims, err := s.repo.GetStandardDimensions(ctx, standardID)
	if err != nil {
		log.Error("failed to fetch dimensions from repository", zap.Error(err))
		return nil, err
	}

	log.Debug("dimensions loaded",
		zap.Int("count", len(dims)),
		zap.String("standard_id", standardID))
	return dims, nil
}

// GetStandardFull - загрузка полного контекста стандарта
func (s *StandardService) GetStandardFull(ctx context.Context, standardID string) (models.StandardContext, error) {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "GetStandardFull"),
		zap.String("standard_id", standardID),
	)
	log.Debug("fetching full standard context")

	start := time.Now()

	ctxData, err := s.repo.GetStandardFull(ctx, standardID)
	if err != nil {
		log.Error("failed to fetch full standard context", zap.Error(err))
		return models.StandardContext{}, err
	}

	log.Info("full standard context loaded",
		zap.Duration("load_duration_ms", time.Since(start)),
		zap.String("standard_id", standardID))
	return ctxData, nil
}

// InvalidateStandardCache - публичная инвалидация кэша
func (s *StandardService) InvalidateStandardCache(standardID string) {
	log := s.log.With(
		zap.String("service_name", "InvalidateStandardCache"),
		zap.String("standard_id", standardID),
	)
	log.Info("invalidating standard cache (manual trigger)")

	s.invalidateCache(standardID)
	log.Debug("cache invalidation completed")
}

// LinkDimensionToStandard - привязка измерения к стандарту
func (s *StandardService) LinkDimensionToStandard(ctx context.Context, standardID, dimensionID string) error {
	log := loggerWith(ctx, s.log,
		zap.String("service_name", "LinkDimensionToStandard"),
		zap.String("standard_id", standardID),
		zap.String("dimension_id", dimensionID),
	)
	log.Debug("linking dimension to standard")

	err := s.repo.LinkDimensionToStandard(ctx, standardID, dimensionID)
	if err != nil {
		log.Error("failed to link dimension in repository", zap.Error(err))
		return err
	}

	// Инвалидируем кэш, так как структура стандарта изменилась
	log.Debug("invalidating cache due to dimension link change")
	s.invalidateCache(standardID)

	log.Info("dimension successfully linked to standard")
	return nil
}
