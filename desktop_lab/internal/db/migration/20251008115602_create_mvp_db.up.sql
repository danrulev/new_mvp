-- Включаем поддержку внешних ключей
PRAGMA foreign_keys = ON;

-- ============================================================================
-- 1. СПРАВОЧНИКИ И БАЗОВЫЕ СУЩНОСТИ
-- ============================================================================

-- Материалы (Асфальтобетон, Бетон, Грунт и т.д.)
CREATE TABLE IF NOT EXISTS materials (
    id TEXT PRIMARY KEY, -- UUID
    name TEXT NOT NULL UNIQUE,
    code TEXT, -- Краткий код, например 'AB', 'PB'
    created_at TEXT DEFAULT (datetime('now'))
);

-- ГОСТы и другие стандарты
CREATE TABLE IF NOT EXISTS standards (
    id TEXT PRIMARY KEY, -- UUID
    material_id TEXT NOT NULL,
    name TEXT NOT NULL, -- Например: "ГОСТ 9128-2013"
    description TEXT,
    valid_from DATE,
    valid_to DATE,
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    UNIQUE (material_id, name)
);
CREATE INDEX IF NOT EXISTS idx_standards_material ON standards(material_id);

-- Измерения контекста (Dimensions)
-- Описывает, от чего зависят нормы: "Климатическая зона", "Марка смеси", "Тип слоя"
CREATE TABLE IF NOT EXISTS context_dimensions (
    id TEXT PRIMARY KEY, -- UUID
    standard_id TEXT NOT NULL,
    key_name TEXT NOT NULL, -- Например: 'climate_zone', 'mix_grade', 'layer_type'
    label TEXT NOT NULL, -- Человекочитаемое: "Дорожно-климатическая зона"
    data_type TEXT NOT NULL DEFAULT 'enum', -- 'enum', 'number', 'text'
    possible_values TEXT, -- JSON массив допустимых значений: ["I", "II", "III"] или null
    FOREIGN KEY (standard_id) REFERENCES standards(id) ON DELETE CASCADE,
    UNIQUE (standard_id, key_name)
);
CREATE INDEX IF NOT EXISTS idx_dims_standard ON context_dimensions(standard_id);

-- ============================================================================
-- 2. МЕТОДЫ И НОРМАТИВЫ
-- ============================================================================

-- Методы испытаний (Шаблоны)
CREATE TABLE IF NOT EXISTS test_methods (
    id TEXT PRIMARY KEY, -- UUID
    standard_id TEXT NOT NULL,
    code TEXT, -- Уникальный код метода внутри стандарта, например 'COMP_50'
    name TEXT NOT NULL, -- "Предел прочности при сжатии при 50°C"
    description TEXT,
    formula_expr TEXT, -- Выражение для расчета, например: "(F / A)"
    unit TEXT NOT NULL, -- "МПа", "%", "мм"
    result_type TEXT NOT NULL DEFAULT 'scalar', -- 'scalar' (одно число), 'complex' (набор чисел)
    is_mandatory INTEGER DEFAULT 1, -- Обязательный ли метод для этого стандарта
    FOREIGN KEY (standard_id) REFERENCES standards(id) ON DELETE CASCADE,
    UNIQUE (standard_id, code)
);
CREATE INDEX IF NOT EXISTS idx_methods_standard ON test_methods(standard_id);

-- Параметры ввода для метода (Переменные формулы)
CREATE TABLE IF NOT EXISTS method_inputs (
    id TEXT PRIMARY KEY,
    method_id TEXT NOT NULL,
    param_key TEXT NOT NULL, -- Переменная в формуле, например 'F', 'A'
    label TEXT NOT NULL,
    unit TEXT,
    input_type TEXT DEFAULT 'number', -- 'number', 'select', 'text'
    is_required INTEGER DEFAULT 1,
    FOREIGN KEY (method_id) REFERENCES test_methods(id) ON DELETE CASCADE,
    UNIQUE (method_id, param_key)
);
CREATE INDEX IF NOT EXISTS idx_inputs_method ON method_inputs(method_id);

-- Нормативные требования (Лимиты)
-- Здесь хранятся конкретные цифры (мин/макс) для конкретных условий
CREATE TABLE IF NOT EXISTS normative_limits (
    id TEXT PRIMARY KEY,
    method_id TEXT NOT NULL,
    limit_type TEXT NOT NULL, -- 'min', 'max', 'range', 'discrete'
    min_value REAL,
    max_value REAL,
    discrete_values TEXT, -- JSON массив для дискретных значений
    note TEXT, -- Ссылка на таблицу в ГОСТе, комментарий
    priority INTEGER DEFAULT 0, -- Приоритет, если условия пересекаются
    FOREIGN KEY (method_id) REFERENCES test_methods(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_limits_method ON normative_limits(method_id);

-- Условия применения норматива (Фильтры)
-- Связывает лимит с конкретными значениями измерений (Dimensions)
-- Пример: Limit_ID=1 требует Climate_Zone='I' AND Mix_Grade='I'
CREATE TABLE IF NOT EXISTS limit_conditions (
    id TEXT PRIMARY KEY,
    limit_id TEXT NOT NULL,
    dimension_key TEXT NOT NULL, -- Должен совпадать с key_name в context_dimensions
    condition_operator TEXT DEFAULT '=', -- '=', 'IN', '!='
    expected_value TEXT NOT NULL, -- Значение, которое должно быть у пробы
    FOREIGN KEY (limit_id) REFERENCES normative_limits(id) ON DELETE CASCADE,
    UNIQUE (limit_id, dimension_key)
);
CREATE INDEX IF NOT EXISTS idx_conditions_limit ON limit_conditions(limit_id);

-- ============================================================================
-- 3. ЭКСПЕРИМЕНТАЛЬНЫЕ ДАННЫЕ (ПРОТОКОЛЫ)
-- ============================================================================

-- Группы экспериментов (Объекты, Участки дорог)
CREATE TABLE IF NOT EXISTS experiment_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    material_id TEXT NOT NULL,
    project_name TEXT NOT NULL,
    location TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE RESTRICT
);

-- Пробы (Образцы)
-- Здесь фиксируется КОНТЕКСТ пробы, необходимый для подбора норм
CREATE TABLE IF NOT EXISTS samples (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL,
    sample_number TEXT NOT NULL, -- Номер пробы в журнале
    collection_date DATE,
    
    -- Контекстные параметры пробы (значения Dimensions)
    -- Хранятся как JSON для гибкости: {"climate_zone": "II", "mix_grade": "I", "layer": "upper"}
    context_params TEXT NOT NULL DEFAULT '{}', 
    
    note TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (group_id) REFERENCES experiment_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_samples_group ON samples(group_id);
-- Индекс для поиска проб по параметрам (потребуется функциональный индекс или поиск по JSON в коде)

-- Протоколы испытаний (Заголовок протокола)
CREATE TABLE IF NOT EXISTS protocols (
    id TEXT PRIMARY KEY,
    sample_id TEXT NOT NULL,
    protocol_number TEXT, -- Номер документа
    lab_name TEXT,
    operator_name TEXT,
    test_date DATE,
    status TEXT DEFAULT 'draft', -- 'draft', 'completed', 'approved'
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (sample_id) REFERENCES samples(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_protocols_sample ON protocols(sample_id);

-- Результаты испытаний (Строки протокола)
CREATE TABLE IF NOT EXISTS test_results (
    id TEXT PRIMARY KEY,
    protocol_id TEXT NOT NULL,
    method_id TEXT NOT NULL,
    
    -- Входные данные (JSON): {"F": 1500, "A": 28.3}
    input_data TEXT NOT NULL DEFAULT '{}', 
    
    -- Расчетное значение
    calculated_value REAL, 
    
    -- Результат валидации
    applied_limit_id TEXT, -- Какой именно лимит был применен
    is_compliant INTEGER, -- 1 = Норма, 0 = Не норма, NULL = Не проверено
    deviation_msg TEXT, -- Сообщение об отклонении
    
    note TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (protocol_id) REFERENCES protocols(id) ON DELETE CASCADE,
    FOREIGN KEY (method_id) REFERENCES test_methods(id) ON DELETE RESTRICT,
    FOREIGN KEY (applied_limit_id) REFERENCES normative_limits(id) ON DELETE SET NULL,
    UNIQUE (protocol_id, method_id)
);
CREATE INDEX IF NOT EXISTS idx_results_protocol ON test_results(protocol_id);
CREATE INDEX IF NOT EXISTS idx_results_method ON test_results(method_id);

-- ============================================================================
-- 4. ДОПОЛНИТЕЛЬНЫЕ ТАБЛИЦЫ ДЛЯ ОТЧЕТОВ И НАСТРОЕК
-- ============================================================================

-- Шаблоны отчетов (опционально, если будете хранить конфиги PDF в БД)
CREATE TABLE IF NOT EXISTS report_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    material_id TEXT, -- null для глобального
    template_content TEXT, -- HTML/Jinja2 шаблон или путь к файлу
    is_default INTEGER DEFAULT 0
);

-- Таблица для хранения справочников значений (если не хотите хардкодить в JSON)
-- Например, список всех регионов для климатических зон
CREATE TABLE IF NOT EXISTS reference_data (
    id TEXT PRIMARY KEY,
    category TEXT NOT NULL, -- 'climate_zones', 'mix_grades', 'regions'
    key_val TEXT NOT NULL,
    label_val TEXT NOT NULL,
    metadata TEXT, -- Доп. инфо в JSON
    UNIQUE (category, key_val)
);
CREATE INDEX IF NOT EXISTS idx_ref_category ON reference_data(category);