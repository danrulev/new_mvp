-- Откат миграции для таблиц контроля качества

-- Удаляем триггер и функцию
DROP TRIGGER IF EXISTS trigger_protocol_templates_updated_at ON protocol_templates;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Удаляем таблицы в обратном порядке
DROP TABLE IF EXISTS protocol_templates CASCADE;
DROP TABLE IF EXISTS protocol_versions CASCADE;
DROP TABLE IF EXISTS audit_logs CASCADE;
