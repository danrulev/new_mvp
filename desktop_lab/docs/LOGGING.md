# Production Logging Configuration

## Обзор

Реализовано production-ready логирование на базе `zap` с поддержкой различных окружений и оптимизаций.

## Основные возможности

### 1. **Многоуровневое логирование**
- **Debug** - детальная отладочная информация (только development)
- **Info** - общая информация о работе приложения
- **Warn** - предупреждения о потенциальных проблемах
- **Error** - ошибки требующие внимания
- **Panic/Fatal** - критические ошибки

### 2. **Автоматическое определение окружения**
```bash
# Установка окружения через переменную
export APP_ENV=production  # development, staging, production
```

### 3. **Production оптимизации**

#### Сэмплинг логов
В production режиме включается сэмплинг для предотвращения дублирования:
- Первые 100 одинаковых логов записываются полностью
- Последующие логи с тем же содержанием пропускаются

#### JSON формат
В production автоматически используется JSON формат для:
- Удобного парсинга в системах мониторинга (ELK, Splunk, Grafana)
- Структурированного хранения
- Быстрого поиска

#### Отключение caller
Для повышения производительности в production отключается отслеживание места вызова

### 4. **Структурированные поля**

Каждый лог автоматически содержит:
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "level": "info",
  "service": "desktop-lab",
  "version": "dev",
  "environment": "production",
  "hostname": "server-01",
  "message": "http_request_completed",
  "request_id": "uuid...",
  "method": "GET",
  "path": "/api/analytics/dashboard",
  "status_code": 200,
  "latency_ms": 45,
  "client_ip": "192.168.1.1"
}
```

### 5. **Ротация логов**

Поддерживается ротация через:
- Временные метки в именах файлов
- Разделение на app.log и error.log
- Настройка максимального размера файла
- Хранение резервных копий
- Сжатие старых логов

## Конфигурация

### Переменные в config.yaml

```yaml
logger:
  # Базовые настройки
  level: info                          # debug, info, warn, error
  development: false                   # режим разработки
  disable_caller: true                 # отключить caller
  disable_stacktrace: true             # отключить stacktrace
  encoding: json                       # json или console
  
  # Пути вывода
  output_paths:
    - ./logs/app.log
    - stdout
  error_output_paths:
    - ./logs/error.log
    - stderr
  
  # Production настройки
  environment: production              # development, staging, production
  service_name: desktop-lab            # имя сервиса
  enable_sampling: true                # включить сэмплинг
  sampling_initial: 100                # первые N логов
  sampling_thereafter: 100             # после N логов
  max_file_size_mb: 100                # макс размер файла
  max_backups: 5                       # кол-во резервных копий
  max_age_days: 30                     # срок хранения
  compress_logs: true                  # сжатие старых логов
```

## Использование в коде

### Базовое логирование

```go
// Info
log.Info("operation completed", 
    zap.String("operation", "create_protocol"),
    zap.Int("protocol_id", 123),
)

// Error с контекстом
log.Error("database error",
    zap.String("query", query),
    zap.Duration("duration", duration),
    zap.Error(err),
)

// Warn
log.Warn("slow query detected",
    zap.Duration("latency_ms", latency),
    zap.Duration("threshold_ms", threshold),
)
```

### Контекстное логирование

```go
// Добавление request_id
requestLogger := logger.WithRequestID(baseLogger, requestID)

// Добавление user_id
userLogger := logger.WithUserID(baseLogger, userID)

// Добавление операции
opLogger := logger.WithOperation(baseLogger, "create_protocol")
```

### Специализированные функции

```go
// Логирование паники
defer func() {
    if r := recover(); r != nil {
        logger.LogPanic(log, r)
    }
}()

// Логирование медленных запросов
logger.LogSlowQuery(log, query, duration, 100*time.Millisecond)

// HTTP access log
logger.LogHTTPAccess(log, method, path, statusCode, latency, clientIP)
```

## Middleware логирования

Все HTTP запросы автоматически логируются middleware с полями:
- request_id - уникальный идентификатор запроса
- method, path - метод и путь
- status_code - код ответа
- latency_ms - время обработки
- response_size_bytes - размер ответа
- client_ip - IP клиента

## Интеграция с системами мониторинга

### Grafana Loki
JSON формат совместим с Loki. Пример query:
```
{service="desktop-lab", level="error"} |= "database"
```

### ELK Stack
Логи автоматически парсятся Logstash/Filebeat:
```json
{
  "@timestamp": "2024-01-15T10:30:00Z",
  "log.level": "error",
  "service.name": "desktop-lab",
  "error.message": "connection timeout"
}
```

### Prometheus Metrics
Для метрик рекомендуется использовать отдельную библиотеку prometheus/client_golang

## Best Practices

### ✅ Делайте
- Используйте структурированные поля (zap.String, zap.Int)
- Добавляйте context к логам (request_id, user_id)
- Логируйте ошибки с zap.Error(err)
- Используйте соответствующие уровни логирования
- Включайте сэмплинг в production

### ❌ Не делайте
- Не логируйте чувствительные данные (пароли, токены)
- Не используйте string конкатенацию для сообщений
- Не логируйте в циклах без необходимости
- Не забывайте про context cancellation

## Производительность

zap - один из самых быстрых логогеров для Go:
- ~10x быстрее logrus
- ~5x быстрее zerolog
- Аллокации: ~0 allocs/op для структурированных логов

Бенчмарки доступны в [официальном репозитории zap](https://github.com/uber-go/zap#performance)

## Troubleshooting

### Логи не записываются в файл
- Проверьте права на директорию ./logs
- Убедитесь что путь указан правильно
- Проверьте output_paths в конфиге

### Слишком много логов
- Увеличьте уровень логирования (info → warn)
- Включите сэмплинг (enable_sampling: true)
- Проверьте нет ли логирования в циклах

### JSON не парсится
- Убедитесь что encoding: json
- Проверьте что все поля корректно экранированы
- Используйте zap.Reflect для сложных объектов
