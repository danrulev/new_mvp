-- Миграция для таблиц контроля качества: аудит, версии протоколов, шаблоны протоколов (SQLite версия)

-- Таблица аудита действий пользователей
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    user_name TEXT, -- Денормализация для истории
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete', 'view', 'print', 'export', 'status_change')),
    resource_type TEXT NOT NULL, -- e.g., 'protocol', 'order', 'sample'
    resource_id INTEGER NOT NULL,
    old_values TEXT, -- JSON предыдущее состояние
    new_values TEXT, -- JSON новое состояние
    ip_address TEXT, -- IP адрес
    user_agent TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для быстрого поиска по логам
CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Таблица версий протоколов испытаний
CREATE TABLE IF NOT EXISTS protocol_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    protocol_id INTEGER NOT NULL REFERENCES protocols(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL CHECK (version_number > 0),
    content_snapshot TEXT NOT NULL, -- JSON полное состояние протокола
    pdf_snapshot BLOB, -- Бинарный PDF срез (опционально)
    changed_by INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    changed_by_name TEXT, -- Денормализация имени
    changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    comment TEXT, -- Причина изменений
    is_current INTEGER DEFAULT 0 -- Флаг текущей версии (0/1)
);

-- Уникальный индекс: один протокол + одна версия = одна запись
CREATE UNIQUE INDEX IF NOT EXISTS idx_protocol_versions_unique 
ON protocol_versions(protocol_id, version_number);

-- Индекс для быстрого поиска текущей версии
CREATE INDEX IF NOT EXISTS idx_protocol_versions_current 
ON protocol_versions(protocol_id, is_current) WHERE is_current = 1;

-- Индекс для поиска по дате изменения
CREATE INDEX IF NOT EXISTS idx_protocol_versions_changed_at 
ON protocol_versions(changed_at DESC);

-- Таблица шаблонов протоколов
CREATE TABLE IF NOT EXISTS protocol_templates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    test_method_id INTEGER NOT NULL REFERENCES test_methods(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    template_html TEXT NOT NULL, -- HTML шаблон с переменными {{.Variable}}
    template_css TEXT, -- CSS стили для шаблона
    is_active INTEGER DEFAULT 1,
    created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Индексы для шаблонов
CREATE INDEX IF NOT EXISTS idx_protocol_templates_org ON protocol_templates(organization_id);
CREATE INDEX IF NOT EXISTS idx_protocol_templates_method ON protocol_templates(test_method_id);
CREATE INDEX IF NOT EXISTS idx_protocol_templates_active ON protocol_templates(is_active) WHERE is_active = 1;
CREATE INDEX IF NOT EXISTS idx_protocol_templates_org_method 
ON protocol_templates(organization_id, test_method_id) WHERE is_active = 1;

-- Триггер для обновления updated_at в SQLite
DROP TRIGGER IF EXISTS trigger_protocol_templates_updated_at;
CREATE TRIGGER trigger_protocol_templates_updated_at
    AFTER UPDATE ON protocol_templates
    FOR EACH ROW
BEGIN
    UPDATE protocol_templates SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
