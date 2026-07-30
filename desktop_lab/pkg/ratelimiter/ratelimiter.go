// Package ratelimiter реализует ограничение частоты запросов (rate limiting).
package ratelimiter

import (
	"sync"
	"time"
)

// Limiter реализует алгоритм token bucket для rate limiting.
type Limiter struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillRate   float64 // токенов в секунду
	lastRefillAt time.Time
}

// New создает новый лимитер с заданными параметрами.
// maxTokens - максимальное количество токенов (burst size)
// refillRate - скорость пополнения токенов в секунду
func New(maxTokens float64, refillRate float64) *Limiter {
	return &Limiter{
		tokens:       maxTokens,
		maxTokens:    maxTokens,
		refillRate:   refillRate,
		lastRefillAt: time.Now(),
	}
}

// Allow проверяет, разрешен ли запрос в текущий момент.
// Возвращает true если есть доступные токены, false в противном случае.
func (l *Limiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()

	if l.tokens >= 1 {
		l.tokens--
		return true
	}

	return false
}

// Wait блокирует выполнение до тех пор, пока не станет доступен токен.
// Максимальное время ожидания - timeout.
func (l *Limiter) Wait(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for {
		if l.Allow() {
			return true
		}

		if time.Now().After(deadline) {
			return false
		}

		// Ждем короткое время перед следующей попыткой
		time.Sleep(10 * time.Millisecond)
	}
}

// refill пополняет токены на основе прошедшего времени.
func (l *Limiter) refill() {
	now := time.Now()
	elapsed := now.Sub(l.lastRefillAt).Seconds()

	l.tokens += elapsed * l.refillRate
	if l.tokens > l.maxTokens {
		l.tokens = l.maxTokens
	}

	l.lastRefillAt = now
}

// GetTokens возвращает текущее количество доступных токенов.
func (l *Limiter) GetTokens() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.refill()
	return l.tokens
}

// Reset сбрасывает лимитер в начальное состояние.
func (l *Limiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.tokens = l.maxTokens
	l.lastRefillAt = time.Now()
}

// RateLimiter управляет множеством лимитеров для разных ключей (например, IP адресов).
type RateLimiter struct {
	mu         sync.RWMutex
	limiters   map[string]*Limiter
	maxTokens  float64
	refillRate float64
}

// NewRateLimiter создает менеджер лимитеров.
func NewRateLimiter(maxTokens float64, refillRate float64) *RateLimiter {
	return &RateLimiter{
		limiters:   make(map[string]*Limiter),
		maxTokens:  maxTokens,
		refillRate: refillRate,
	}
}

// GetLimiter возвращает или создает лимитер для указанного ключа.
func (rl *RateLimiter) GetLimiter(key string) *Limiter {
	// Сначала пытаемся получить без блокировки (для производительности)
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	// Если не нашли, создаем новый с блокировкой
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Проверяем еще раз после получения блокировки
	if limiter, exists = rl.limiters[key]; exists {
		return limiter
	}

	limiter = New(rl.maxTokens, rl.refillRate)
	rl.limiters[key] = limiter
	return limiter
}

// Allow проверяет, разрешен ли запрос для данного ключа.
func (rl *RateLimiter) Allow(key string) bool {
	return rl.GetLimiter(key).Allow()
}

// Remove удаляет лимитер для указанного ключа.
func (rl *RateLimiter) Remove(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, key)
}

// Cleanup удаляет неактивные лимитеры.
func (rl *RateLimiter) Cleanup(threshold time.Duration) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	count := 0
	for key, limiter := range rl.limiters {
		limiter.mu.Lock()
		if time.Since(limiter.lastRefillAt) > threshold {
			delete(rl.limiters, key)
			count++
		}
		limiter.mu.Unlock()
	}

	return count
}
