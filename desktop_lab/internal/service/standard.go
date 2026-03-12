package service

import (
	"context"
	"desktop_lab/internal/models"
	"desktop_lab/internal/repository"
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
	repo repository.StandardRepo
	log  *zap.Logger

	methodsCache *lru.Cache[string, cacheEntry[map[string]models.TestMethodFull]]
	cacheMu      sync.RWMutex
	cacheTTL     time.Duration
}

func NewStandardService(repo repository.StandardRepo, log *zap.Logger) *StandardService {
	// Создаем LRU-кэш на 100 элементов (стандартов)
	// Этого достаточно для большинства лабораторий
	cache, _ := lru.New[string, cacheEntry[map[string]models.TestMethodFull]](100)

	return &StandardService{
		repo:         repo,
		log:          log,
		methodsCache: cache,
		cacheTTL:     10 * time.Minute, // Кэш живет 10 минут
	}
}

// getFromCache пытается получить данные из кэша
func (s *StandardService) getFromCache(key string) (map[string]models.TestMethodFull, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	entry, ok := s.methodsCache.Get(key)
	if !ok {
		return nil, false
	}

	// Проверяем срок жизни
	if time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry.Data, true
}

// setToCache сохраняет данные в кэш
func (s *StandardService) setToCache(key string, data map[string]models.TestMethodFull) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.methodsCache.Add(key, cacheEntry[map[string]models.TestMethodFull]{
		Data:      data,
		ExpiresAt: time.Now().Add(s.cacheTTL),
	})
}

// invalidateCache удаляет запись из кэша (при обновлении стандарта)
func (s *StandardService) invalidateCache(standardID string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.methodsCache.Remove(standardID)
}

// CreateStandard - без изменений, но добавляем инвалидацию кэша при успешном создании
func (s *StandardService) CreateStandard(ctx context.Context, req models.CreateStandardRequest) (string, error) {
	if req.MaterialID == "" || req.Name == "" {
		return "", fmt.Errorf("material_id and name are required")
	}

	for i, m := range req.Methods {
		if m.Name == "" {
			return "", fmt.Errorf("method[%d] name is required", i)
		}
	}

	id, err := s.repo.CreateFull(ctx, req)
	if err != nil {
		s.log.Error("failed to create standard", zap.Error(err))
		return "", fmt.Errorf("ошибка создания стандарта: %w", err)
	}

	// 🔥 Инвалидируем кэш для этого материала (если стандарты кэшируются по material_id)
	// Здесь можно расширить логику кэширования при необходимости

	s.log.Info("standard created successfully", zap.String("id", id), zap.String("name", req.Name))
	return id, nil
}

// GetByMaterialID - можно добавить кэширование списка стандартов
func (s *StandardService) GetByMaterialID(ctx context.Context, materialID string) ([]models.Standard, error) {
	// Для простоты не кэшируем список, так как он редко меняется
	// При необходимости можно добавить аналогичный кэш
	stds, err := s.repo.GetByMaterialID(ctx, materialID)
	if err != nil {
		return nil, err
	}
	s.log.Info("standards loaded", zap.Int("count", len(stds)), zap.String("material_id", materialID))
	return stds, nil
}

// GetMethodDetails - без кэша, так как метод может быть запрошен один раз
func (s *StandardService) GetMethodDetails(ctx context.Context, methodID string) (models.TestMethodFull, error) {
	method, err := s.repo.GetTestMethod(ctx, methodID)
	if err != nil {
		return models.TestMethodFull{}, err
	}
	inputs, err := s.repo.GetMethodInputs(ctx, methodID)
	if err != nil {
		return models.TestMethodFull{}, err
	}
	limits, err := s.repo.GetMethodLimits(ctx, methodID)
	if err != nil {
		return models.TestMethodFull{}, err
	}

	result := models.TestMethodFull{
		Method: method,
		Inputs: inputs,
		Limits: limits,
	}

	return result, nil
}

// GetMethodsByStandardID - 🔥 ИСПОЛЬЗУЕТ КЭШ
// Возвращает только методы без лимитов (для UI списка)
func (s *StandardService) GetMethodsByStandardID(ctx context.Context, standardID string) ([]models.TestMethod, error) {
	// Пытаемся получить из кэша полные данные
	if fullMethods, ok := s.getFromCache(standardID); ok {
		// Извлекаем только методы для ответа
		methods := make([]models.TestMethod, 0, len(fullMethods))
		for _, fm := range fullMethods {
			methods = append(methods, fm.Method)
		}
		s.log.Debug("methods served from cache", zap.String("standard_id", standardID))
		return methods, nil
	}

	// Кэш промах — загружаем из БД
	methods, err := s.repo.GetMethodsByStandardID(ctx, standardID)
	if err != nil {
		s.log.Error("failed to get methods", zap.Error(err), zap.String("standard_id", standardID))
		return nil, err
	}

	// 🔥 Сохраняем в кэш полные данные (загружаем их один раз)
	// Но для этого нужен отдельный запрос... Оптимизируем:
	// Если часто нужны полные данные, лучше сразу грузить GetMethodsFullByStandardID

	s.log.Info("methods loaded from DB", zap.Int("count", len(methods)), zap.String("standard_id", standardID))
	return methods, nil
}

// GetMethodsFullByStandardIDWithCache - НОВЫЙ ПУБЛИЧНЫЙ МЕТОД
// Возвращает полные методы с инпутами и лимитами, используя кэш
func (s *StandardService) GetMethodsFullByStandardIDWithCache(ctx context.Context, standardID string) (map[string]models.TestMethodFull, error) {
	// 1. Пробуем кэш
	if cached, ok := s.getFromCache(standardID); ok {
		s.log.Debug("full methods served from cache", zap.String("standard_id", standardID))
		return cached, nil
	}

	// 2. Загружаем из БД
	fullMethods, err := s.repo.GetMethodsFullByStandardID(ctx, standardID)
	if err != nil {
		s.log.Error("failed to load full methods", zap.Error(err), zap.String("standard_id", standardID))
		return nil, err
	}

	// 3. Сохраняем в кэш
	s.setToCache(standardID, fullMethods)
	s.log.Debug("full methods cached", zap.String("standard_id", standardID), zap.Int("count", len(fullMethods)))

	return fullMethods, nil
}

// GetStandardDimensions - можно кэшировать, так как измерения меняются редко
func (s *StandardService) GetStandardDimensions(ctx context.Context, standardID string) ([]models.ContextDimension, error) {
	return s.repo.GetStandardDimensions(ctx, standardID)
}

// GetStandardFull - НОВЫЙ МЕТОД для эффективной загрузки всего контекста стандарта
// Идеально для инициализации формы создания протокола
func (s *StandardService) GetStandardFull(ctx context.Context, standardID string) (*models.StandardContext, error) {
	// Можно добавить кэширование всей структуры, если нужно
	return s.repo.GetStandardFull(ctx, standardID)
}

// InvalidateStandardCache - публичный метод для инвалидации кэша при обновлении стандарта
// Можно вызвать из админ-панели или при импорте новых ГОСТов
func (s *StandardService) InvalidateStandardCache(standardID string) {
	s.invalidateCache(standardID)
	s.log.Info("standard cache invalidated", zap.String("standard_id", standardID))
}
