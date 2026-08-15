-- Миграция для таблиц контроля качества: аудит, версии протоколов, шаблоны протоколов

-- Таблица аудита действий пользователей
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    user_name VARCHAR(255), -- Денормализация для истории
    action VARCHAR(50) NOT NULL CHECK (action IN ('create', 'update', 'delete', 'view', 'print', 'export', 'status_change')),
    resource_type VARCHAR(100) NOT NULL, -- e.g., 'protocol', 'order', 'sample'
    resource_id BIGINT NOT NULL,
    old_values JSONB, -- Предыдущее состояние
    new_values JSONB, -- Новое состояние
    ip_address VARCHAR(45), -- IPv4 или IPv6
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для быстрого поиска по логам
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource_type_id ON audit_logs(resource_type, resource_id);

-- Комментарий к таблице
COMMENT ON TABLE audit_logs IS 'Журнал аудита действий пользователей';
COMMENT ON COLUMN audit_logs.action IS 'Тип действия: create, update, delete, view, print, export, status_change';
COMMENT ON COLUMN audit_logs.old_values IS 'JSONB с предыдущим состоянием ресурса';
COMMENT ON COLUMN audit_logs.new_values IS 'JSONB с новым состоянием ресурса';

-- Таблица версий протоколов испытаний
CREATE TABLE IF NOT EXISTS protocol_versions (
    id BIGSERIAL PRIMARY KEY,
    protocol_id BIGINT NOT NULL REFERENCES protocols(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL CHECK (version_number > 0),
    content_snapshot JSONB NOT NULL, -- Полное состояние протокола на момент версии
    pdf_snapshot BYTEA, -- Бинарный PDF срез (опционально)
    changed_by BIGINT NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    changed_by_name VARCHAR(255), -- Денормализация имени
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    comment TEXT, -- Причина изменений
    is_current BOOLEAN DEFAULT FALSE -- Флаг текущей версии
);

-- Уникальный индекс: один протокол + одна версия = одна запись
CREATE UNIQUE INDEX IF NOT EXISTS idx_protocol_versions_unique 
ON protocol_versions(protocol_id, version_number);

-- Индекс для быстрого поиска текущей версии
CREATE INDEX IF NOT EXISTS idx_protocol_versions_current 
ON protocol_versions(protocol_id, is_current) WHERE is_current = TRUE;

-- Индекс для поиска по дате изменения
CREATE INDEX IF NOT EXISTS idx_protocol_versions_changed_at 
ON protocol_versions(changed_at DESC);

-- Комментарий к таблице
COMMENT ON TABLE protocol_versions IS 'История версий протоколов испытаний';
COMMENT ON COLUMN protocol_versions.content_snapshot IS 'JSONB с полным состоянием протокола на момент создания версии';
COMMENT ON COLUMN protocol_versions.pdf_snapshot IS 'Бинарные данные PDF файла на момент версии';
COMMENT ON COLUMN protocol_versions.is_current IS 'Флаг указывающий на текущую активную версию';

-- Таблица шаблонов протоколов
CREATE TABLE IF NOT EXISTS protocol_templates (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    test_method_id BIGINT NOT NULL REFERENCES test_methods(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT,
    template_html TEXT NOT NULL, -- HTML шаблон с переменными {{.Variable}}
    template_css TEXT, -- CSS стили для шаблона
    is_active BOOLEAN DEFAULT TRUE,
    created_by BIGINT NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для шаблонов
CREATE INDEX IF NOT EXISTS idx_protocol_templates_org ON protocol_templates(organization_id);
CREATE INDEX IF NOT EXISTS idx_protocol_templates_method ON protocol_templates(test_method_id);
CREATE INDEX IF NOT EXISTS idx_protocol_templates_active ON protocol_templates(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_protocol_templates_org_method 
ON protocol_templates(organization_id, test_method_id) WHERE is_active = TRUE;

-- Триггер для обновления updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_protocol_templates_updated_at ON protocol_templates;
CREATE TRIGGER trigger_protocol_templates_updated_at
    BEFORE UPDATE ON protocol_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Комментарий к таблице
COMMENT ON TABLE protocol_templates IS 'Шаблоны протоколов испытаний для генерации отчетов';
COMMENT ON COLUMN protocol_templates.template_html IS 'HTML шаблон с использованием синтаксиса Go templates ({{.Variable}})';

-- Вставка данных для аудита (типы действий)
-- Типы действий уже определены в CHECK constraint

-- Пример начальных данных для шаблонов (опционально)
-- INSERT INTO protocol_templates (organization_id, test_method_id, name, description, template_html, is_active, created_by)
-- VALUES (1, 1, 'Стандартный шаблон бетона', 'Шаблон для испытаний бетонных образцов', '<html>...</html>', TRUE, 1);
