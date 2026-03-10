-- Включаем поддержку внешних ключей для SQLite
PRAGMA foreign_keys = ON;

-- 1. Материалы
CREATE TABLE IF NOT EXISTS materials (
    id TEXT PRIMARY KEY, -- Храним UUID как строку
    name TEXT NOT NULL UNIQUE
);

-- 2. ГОСТы
-- Имя ГОСТа уникально только в рамках одного материала
CREATE TABLE IF NOT EXISTS gos_ts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    material_id TEXT NOT NULL,
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    UNIQUE (material_id, name)
);
CREATE INDEX IF NOT EXISTS idx_gos_ts_material_id ON gos_ts(material_id);

-- 3. Методы испытаний
-- Имя метода уникально только в рамках одного ГОСТа
CREATE TABLE IF NOT EXISTS methods (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    gost_id TEXT NOT NULL,
    formula TEXT,
    is_calculated INTEGER NOT NULL DEFAULT 0, -- 0 = false, 1 = true
    min_value REAL,
    max_value REAL,
    unit TEXT,
    FOREIGN KEY (gost_id) REFERENCES gos_ts(id) ON DELETE CASCADE,
    UNIQUE (gost_id, name)
);
CREATE INDEX IF NOT EXISTS idx_methods_gost_id ON methods(gost_id);

-- 4. Параметры методов
CREATE TABLE IF NOT EXISTS method_parameters (
    id TEXT PRIMARY KEY,
    method_id TEXT NOT NULL,
    key TEXT NOT NULL,
    label TEXT NOT NULL,
    unit TEXT,
    is_input INTEGER NOT NULL DEFAULT 1,
    is_required INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (method_id) REFERENCES methods(id) ON DELETE CASCADE,
    UNIQUE (method_id, key)
);
CREATE INDEX IF NOT EXISTS idx_method_parameters_method_id ON method_parameters(method_id);

-- 5. Пробы (образцы)
CREATE TABLE IF NOT EXISTS samples (
    id TEXT PRIMARY KEY,
    number TEXT NOT NULL,
    material_id TEXT NOT NULL,
    collection_place TEXT NOT NULL,
    note TEXT,
    material_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')), -- ISO8601 строка
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE RESTRICT
);
CREATE INDEX IF NOT EXISTS idx_samples_material_id ON samples(material_id);

-- 6. Группы экспериментов
CREATE TABLE IF NOT EXISTS experiment_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE, -- Имя группы глобально уникально (или можно сделать unique(material_id, name))
    material_id TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE RESTRICT
);

-- 7. Связь: проба ↔ группа (Many-to-Many)
CREATE TABLE IF NOT EXISTS sample_group_links (
    sample_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    PRIMARY KEY (sample_id, group_id),
    FOREIGN KEY (sample_id) REFERENCES samples(id) ON DELETE CASCADE,
    FOREIGN KEY (group_id) REFERENCES experiment_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_sample_group_links_group_id ON sample_group_links(group_id);

-- 8. Протоколы испытаний
CREATE TABLE IF NOT EXISTS protocols (
    id TEXT PRIMARY KEY,
    sample_id TEXT NOT NULL,
    group_id TEXT, -- Может быть NULL, если протокол не в группе
    lab_name TEXT NOT NULL,
    operator TEXT NOT NULL,
    project TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (sample_id) REFERENCES samples(id) ON DELETE RESTRICT,
    FOREIGN KEY (group_id) REFERENCES experiment_groups(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_protocols_sample_id ON protocols(sample_id);
CREATE INDEX IF NOT EXISTS idx_protocols_group_id ON protocols(group_id);

-- 9. Результаты по методам в рамках протокола
CREATE TABLE IF NOT EXISTS protocol_method_results (
    id TEXT PRIMARY KEY,
    protocol_id TEXT NOT NULL,
    method_id TEXT NOT NULL,
    value REAL NOT NULL,
    data TEXT, -- JSON с входными параметрами
    is_compliant INTEGER NOT NULL, -- 0 или 1
    deviation TEXT,
    note TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (protocol_id) REFERENCES protocols(id) ON DELETE CASCADE,
    FOREIGN KEY (method_id) REFERENCES methods(id) ON DELETE RESTRICT,
    UNIQUE (protocol_id, method_id)
);
CREATE INDEX IF NOT EXISTS idx_results_protocol_id ON protocol_method_results(protocol_id);
CREATE INDEX IF NOT EXISTS idx_results_method_id ON protocol_method_results(method_id);