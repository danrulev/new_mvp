// Package circuitbreaker реализует паттерн Circuit Breaker для защиты от сбоев внешних зависимостей.
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State представляет состояние circuit breaker.
type State int

const (
	// StateClosed - нормальное состояние, запросы выполняются.
	StateClosed State = iota
	// StateOpen - цепь разомкнута, запросы блокируются.
	StateOpen
	// StateHalfOpen - тестовое состояние, разрешается один пробный запрос.
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen ошибка, возвращаемая когда цепь разомкнута.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// Config конфигурирует circuit breaker.
type Config struct {
	MaxFailures   int           // Максимальное количество ошибок перед размыканием цепи
	Timeout       time.Duration // Время, через которое цепь перейдет в half-open состояние
	HalfOpenCalls int           // Количество успешных вызовов в half-open для закрытия цепи
}

// DefaultConfig возвращает конфигурацию по умолчанию.
func DefaultConfig() Config {
	return Config{
		MaxFailures:   5,
		Timeout:       30 * time.Second,
		HalfOpenCalls: 1,
	}
}

// CircuitBreaker реализует паттерн Circuit Breaker.
type CircuitBreaker struct {
	mu            sync.RWMutex
	state         State
	failures      int
	successes     int
	lastFailureAt time.Time
	config        Config
}

// New создает новый CircuitBreaker с заданной конфигурацией.
func New(cfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:  StateClosed,
		config: cfg,
	}
}

// Execute выполняет функцию с защитой circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	if cb.shouldAllowRequest() {
		cb.mu.Unlock()

		err := fn()

		cb.mu.Lock()
		if err != nil {
			cb.recordFailure()
		} else {
			cb.recordSuccess()
		}
		cb.mu.Unlock()

		return err
	}

	cb.mu.Unlock()
	return ErrCircuitOpen
}

// ExecuteWithResult выполняет функцию с возвратом значения и защитой circuit breaker.
// Для Go 1.19 используется interface{} вместо дженериков.
func (cb *CircuitBreaker) ExecuteWithResult(fn func() (interface{}, error)) (interface{}, error) {
	cb.mu.Lock()
	if !cb.shouldAllowRequest() {
		cb.mu.Unlock()
		return nil, ErrCircuitOpen
	}
	cb.mu.Unlock()

	result, err := fn()

	cb.mu.Lock()
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}
	cb.mu.Unlock()

	return result, err
}

// shouldAllowRequest определяет, можно ли выполнить запрос.
func (cb *CircuitBreaker) shouldAllowRequest() bool {
	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Проверяем, истек ли таймаут
		if time.Since(cb.lastFailureAt) > cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordFailure регистрирует ошибку и при необходимости размыкает цепь.
func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailureAt = time.Now()

	if cb.state == StateHalfOpen {
		// В half-open состоянии любая ошибка снова размыкает цепь
		cb.state = StateOpen
		cb.failures = cb.config.MaxFailures
	} else if cb.failures >= cb.config.MaxFailures {
		cb.state = StateOpen
	}
}

// recordSuccess регистрирует успешное выполнение.
func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.config.HalfOpenCalls {
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
		}
	} else if cb.state == StateClosed {
		// Сбрасываем счетчик ошибок после успешного выполнения в закрытом состоянии
		cb.failures = 0
	}
}

// GetState возвращает текущее состояние circuit breaker.
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset сбрасывает circuit breaker в начальное состояние.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastFailureAt = time.Time{}
}

// GetStats возвращает статистику circuit breaker.
func (cb *CircuitBreaker) GetStats() (failures int, successes int, state State) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures, cb.successes, cb.state
}
