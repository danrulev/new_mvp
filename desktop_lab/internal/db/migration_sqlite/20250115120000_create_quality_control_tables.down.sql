-- Откат миграции для таблиц контроля качества (SQLite версия)

-- Удаляем триггер
DROP TRIGGER IF EXISTS trigger_protocol_templates_updated_at;

-- Удаляем таблицы в обратном порядке
DROP TABLE IF EXISTS protocol_templates;
DROP TABLE IF EXISTS protocol_versions;
DROP TABLE IF EXISTS audit_logs;
