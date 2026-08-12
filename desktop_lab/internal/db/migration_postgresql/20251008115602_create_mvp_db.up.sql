-- migrations/000001_init_schema.up.sql


-- Тип-перечисление для ролей (типобезопасность вместо CHECK)
CREATE TYPE user_role AS ENUM ('admin', 'user', 'moderator');
CREATE TYPE org_role AS ENUM ('admin', 'editor', 'viewer');
CREATE TYPE test_status AS ENUM ('draft', 'completed', 'approved', 'archived');
CREATE TYPE limit_type AS ENUM ('min', 'max', 'range', 'discrete');
CREATE TYPE input_type AS ENUM ('number', 'select', 'text', 'boolean');


-- ============================================================================
-- 1. СПРАВОЧНИКИ И БАЗОВЫЕ СУЩНОСТИ
-- ============================================================================

-- Материалы
CREATE TABLE materials (
    id VARCHAR(36) PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT materials_name_not_empty CHECK (char_length(name) > 0)
);

CREATE INDEX idx_materials_code ON materials(code) WHERE code IS NOT NULL;

-- ГОСТы и стандарты
CREATE TABLE standards (
    id VARCHAR(36) PRIMARY KEY,
    material_id VARCHAR(36) NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    valid_from DATE,
    valid_to DATE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    UNIQUE (material_id, name),
    CONSTRAINT standards_valid_range CHECK (valid_from IS NULL OR valid_to IS NULL OR valid_to >= valid_from)
);

CREATE INDEX idx_standards_material ON standards(material_id);
CREATE INDEX idx_standards_validity ON standards(valid_from, valid_to) WHERE valid_from IS NOT NULL;

-- Измерения контекста (Dimensions)
CREATE TABLE context_dimensions (
    id VARCHAR(36) PRIMARY KEY,
    key_name TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL,
    data_type TEXT NOT NULL DEFAULT 'enum' CHECK (data_type IN ('enum', 'number', 'text', 'boolean')),
    possible_values JSONB,  -- Нативный JSONB для гибких значений
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT context_dims_key_not_empty CHECK (char_length(key_name) > 0)
);

-- Связь материалов с измерениями
CREATE TABLE material_context_dims (
    id VARCHAR(36) PRIMARY KEY,
    material_id VARCHAR(36) NOT NULL,
    dimension_id VARCHAR(36) NOT NULL,
    is_required BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE CASCADE,
    FOREIGN KEY (dimension_id) REFERENCES context_dimensions(id) ON DELETE CASCADE,
    UNIQUE (material_id, dimension_id)
);

CREATE INDEX idx_mat_ctx_mat ON material_context_dims(material_id);
CREATE INDEX idx_mat_ctx_dim ON material_context_dims(dimension_id);

-- Связь стандартов с измерениями
CREATE TABLE standard_context_dims (
    id VARCHAR(36) PRIMARY KEY,
    standard_id VARCHAR(36) NOT NULL,
    dimension_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (standard_id) REFERENCES standards(id) ON DELETE CASCADE,
    FOREIGN KEY (dimension_id) REFERENCES context_dimensions(id) ON DELETE CASCADE,
    UNIQUE (standard_id, dimension_id)
);

CREATE INDEX idx_std_ctx_std ON standard_context_dims(standard_id);


-- ============================================================================
-- 2. МЕТОДЫ И НОРМАТИВЫ
-- ============================================================================

-- Методы испытаний
CREATE TABLE test_methods (
    id VARCHAR(36) PRIMARY KEY,
    standard_id VARCHAR(36) NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    formula_expr TEXT,
    unit TEXT NOT NULL,
    result_type TEXT NOT NULL DEFAULT 'scalar' CHECK (result_type IN ('scalar', 'complex', 'boolean')),
    is_mandatory BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (standard_id) REFERENCES standards(id) ON DELETE CASCADE,
    UNIQUE (standard_id, code),
    CONSTRAINT methods_name_not_empty CHECK (char_length(name) > 0)
);

CREATE INDEX idx_methods_standard ON test_methods(standard_id);
CREATE INDEX idx_methods_code ON test_methods(code);

-- Параметры ввода для метода
CREATE TABLE method_inputs (
    id VARCHAR(36) PRIMARY KEY,
    method_id VARCHAR(36) NOT NULL,
    param_key TEXT NOT NULL,
    label TEXT NOT NULL,
    unit TEXT,
    input_type input_type DEFAULT 'number',  -- Используем ENUM-тип
    is_required BOOLEAN DEFAULT true,
    default_value JSONB,  -- Значение по умолчанию (для number: 0, для select: "option1")
    validation_rules JSONB,  -- {"min": 0, "max": 100, "pattern": "^\\d+$"}
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (method_id) REFERENCES test_methods(id) ON DELETE CASCADE,
    UNIQUE (method_id, param_key)
);

CREATE INDEX idx_inputs_method ON method_inputs(method_id);

-- Нормативные требования (Лимиты)
CREATE TABLE normative_limits (
    id VARCHAR(36) PRIMARY KEY,
    method_id VARCHAR(36) NOT NULL,
    limit_type limit_type NOT NULL,  -- ENUM-тип
    min_value NUMERIC,  -- NUMERIC для точных расчетов
    max_value NUMERIC,
    discrete_values JSONB,  -- Для дискретных значений: ["A", "B", "C"]
    note TEXT,
    priority INTEGER DEFAULT 0 CHECK (priority >= 0),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (method_id) REFERENCES test_methods(id) ON DELETE CASCADE,
    CONSTRAINT limits_range_valid CHECK (
        (limit_type != 'range') OR 
        (min_value IS NOT NULL AND max_value IS NOT NULL AND max_value >= min_value)
    )
);

CREATE INDEX idx_limits_method ON normative_limits(method_id);
CREATE INDEX idx_limits_priority ON normative_limits(method_id, priority DESC);

-- Условия применения норматива (Фильтры)
CREATE TABLE limit_conditions (
    id VARCHAR(36) PRIMARY KEY,
    limit_id VARCHAR(36) NOT NULL,
    dimension_key TEXT NOT NULL,
    condition_operator TEXT DEFAULT '=' CHECK (condition_operator IN ('=', '!=', 'IN', '>', '<', '>=' , '<=')),
    expected_value TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (limit_id) REFERENCES normative_limits(id) ON DELETE CASCADE,
    UNIQUE (limit_id, dimension_key)
);

CREATE INDEX idx_conditions_limit ON limit_conditions(limit_id);
CREATE INDEX idx_conditions_lookup ON limit_conditions(dimension_key, expected_value);


-- ============================================================================
-- 3. ЭКСПЕРИМЕНТАЛЬНЫЕ ДАННЫЕ (ПРОТОКОЛЫ)
-- ============================================================================

-- Группы экспериментов
CREATE TABLE experiment_groups (
    id VARCHAR(36) PRIMARY KEY,
    name TEXT NOT NULL,
    material_id VARCHAR(36) NOT NULL,
    project_name TEXT NOT NULL,
    location TEXT,
    metadata JSONB DEFAULT '{}',  -- Дополнительные поля
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(36),  -- Ссылка на users.id
    
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE RESTRICT,
    CONSTRAINT groups_name_not_empty CHECK (char_length(name) > 0)
);

CREATE INDEX idx_groups_material ON experiment_groups(material_id);
CREATE INDEX idx_groups_created_by ON experiment_groups(created_by);

-- Пробы (Образцы)
CREATE TABLE samples (
    id VARCHAR(36) PRIMARY KEY,
    group_id VARCHAR(36),
    material_id VARCHAR(36) NOT NULL,
    sample_number TEXT NOT NULL,
    collection_date TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    collection_place TEXT,
    
    -- Контекстные параметры пробы: нативный JSONB с GIN-индексом
    context_params JSONB NOT NULL DEFAULT '{}',
    photo_url TEXT,
    length_mm NUMERIC,
    width_mm NUMERIC,
    height_mm NUMERIC,
    shape TEXT,
    weight_grams NUMERIC,
    color TEXT,
    batch_number TEXT,
    manufacturer TEXT,
    note TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(36),
    
    FOREIGN KEY (group_id) REFERENCES experiment_groups(id) ON DELETE SET NULL,
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    
    UNIQUE (group_id, sample_number) WHERE group_id IS NOT NULL
);

-- 🔥 Критически важные индексы для JSONB
CREATE INDEX idx_samples_group ON samples(group_id);
CREATE INDEX idx_samples_material ON samples(material_id);
CREATE INDEX idx_samples_context_gin ON samples USING GIN (context_params);  -- Поиск по любому ключу
CREATE INDEX idx_samples_collection_date ON samples(collection_date);

-- Протоколы испытаний
CREATE TABLE protocols (
    id VARCHAR(36) PRIMARY KEY,
    sample_id VARCHAR(36) NOT NULL,
    protocol_number TEXT,
    lab_name TEXT,
    operator_name TEXT,
    test_date TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    status test_status DEFAULT 'draft',  -- ENUM-тип
    note TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (sample_id) REFERENCES samples(id) ON DELETE CASCADE,
    CONSTRAINT protocols_number_unique UNIQUE (sample_id, protocol_number) WHERE protocol_number IS NOT NULL
);

CREATE INDEX idx_protocols_sample ON protocols(sample_id);
CREATE INDEX idx_protocols_status ON protocols(status) WHERE status != 'draft';
CREATE INDEX idx_protocols_test_date ON protocols(test_date);

-- Результаты испытаний
CREATE TABLE test_results (
    id VARCHAR(36) PRIMARY KEY,
    protocol_id VARCHAR(36) NOT NULL,
    method_id VARCHAR(36) NOT NULL,
    
    -- Входные данные: JSONB для гибкости + валидация на уровне приложения
    input_data JSONB NOT NULL DEFAULT '{}',
    
    -- Расчетное значение: NUMERIC для точности
    calculated_value NUMERIC,
    calculated_unit TEXT,
    
    -- Результат валидации
    applied_limit_id VARCHAR(36),
    is_compliant BOOLEAN,  -- NULL = не проверено
    deviation_msg TEXT,
    
    note TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (protocol_id) REFERENCES protocols(id) ON DELETE CASCADE,
    FOREIGN KEY (method_id) REFERENCES test_methods(id) ON DELETE RESTRICT,
    FOREIGN KEY (applied_limit_id) REFERENCES normative_limits(id) ON DELETE SET NULL,
    
    UNIQUE (protocol_id, method_id)
);

CREATE INDEX idx_results_protocol ON test_results(protocol_id);
CREATE INDEX idx_results_method ON test_results(method_id);
CREATE INDEX idx_results_compliance ON test_results(is_compliant) WHERE is_compliant IS NOT NULL;
CREATE INDEX idx_results_input_gin ON test_results USING GIN (input_data);  -- Поиск по входным параметрам


-- ============================================================================
-- 4. ПОЛЬЗОВАТЕЛИ, ОРГАНИЗАЦИИ, БЕЗОПАСНОСТЬ
-- ============================================================================

-- Пользователи
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(name) >= 2),
    email TEXT NOT NULL UNIQUE CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    password_hash TEXT NOT NULL,  -- 🔐 bcrypt/argon2 хэш, НЕ пароль!
    role user_role NOT NULL DEFAULT 'user',  -- ENUM-тип
    is_active BOOLEAN DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,  -- Soft delete
    
    CONSTRAINT users_email_not_empty CHECK (char_length(email) > 0)
);

-- 🔥 Индексы для аутентификации и поиска
CREATE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL AND is_active = true;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted ON users(deleted_at) WHERE deleted_at IS NOT NULL;

-- Refresh tokens / сессии
CREATE TABLE tokens (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,  -- 🔐 Хэш токена, не сам токен!
    token_type TEXT NOT NULL DEFAULT 'refresh' CHECK (token_type IN ('refresh', 'api_key', 'reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT,
    ip_address INET,  -- Нативный тип для IP
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT tokens_not_expired CHECK (expires_at > created_at)
);

CREATE INDEX idx_tokens_user ON tokens(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_tokens_hash ON tokens(token_hash);
CREATE INDEX idx_tokens_expiry ON tokens(expires_at) WHERE revoked_at IS NULL;

-- Организации
CREATE TABLE organizations (
    id VARCHAR(36) PRIMARY KEY,
    name TEXT NOT NULL UNIQUE CHECK (char_length(name) >= 2),
    address TEXT,
    phone TEXT CHECK (phone ~ '^[\+]?[(]?[0-9]{1,4}[)]?[-\s\.]?[0-9]{1,4}[-\s\.]?[0-9]{1,9}$'),
    email TEXT CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'),
    metadata JSONB DEFAULT '{}',  -- Дополнительные реквизиты
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_organizations_active ON organizations(name) WHERE deleted_at IS NULL;

-- Связь пользователей с организациями (исправлено название таблицы!)
CREATE TABLE organization_users (  -- ✅ Было: ogranization_users (опечатка)
    id VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role org_role NOT NULL DEFAULT 'viewer',  -- ENUM-тип
    is_owner BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (organization_id, user_id)
);

CREATE INDEX idx_org_users_org ON organization_users(organization_id);
CREATE INDEX idx_org_users_user ON organization_users(user_id) WHERE organization_users.deleted_at IS NULL;

-- Прайс-лист организации на методы
CREATE TABLE organization_tests (
    id VARCHAR(36) PRIMARY KEY,
    organization_id VARCHAR(36) NOT NULL,
    test_method_id VARCHAR(36) NOT NULL,
    price NUMERIC(10, 2) NOT NULL CHECK (price >= 0),  -- Деньги: всегда NUMERIC!
    currency TEXT DEFAULT 'RUB' CHECK (currency IN ('RUB', 'USD', 'EUR')),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (test_method_id) REFERENCES test_methods(id) ON DELETE CASCADE,
    UNIQUE (organization_id, test_method_id) WHERE is_active = true
);

CREATE INDEX idx_org_tests_active ON organization_tests(organization_id, is_active) WHERE is_active = true;


-- Шаблоны отчетов
CREATE TABLE report_templates (
    id VARCHAR(36) PRIMARY KEY,
    name TEXT NOT NULL,
    material_id VARCHAR(36),
    template_content TEXT NOT NULL,  -- HTML/Jinja2 или путь к файлу
    template_format TEXT DEFAULT 'html' CHECK (template_format IN ('html', 'pdf', 'docx')),
    is_default BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (material_id) REFERENCES materials(id) ON DELETE SET NULL,
    UNIQUE (material_id, name) WHERE material_id IS NOT NULL AND is_default = false
);

-- Справочные данные (климатические зоны, марки и т.д.)
CREATE TABLE reference_data (
    id VARCHAR(36) PRIMARY KEY,
    category TEXT NOT NULL,
    key_val TEXT NOT NULL,
    label_val TEXT NOT NULL,
    sort_order INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE (category, key_val)
);

CREATE INDEX idx_ref_category ON reference_data(category);
CREATE INDEX idx_ref_lookup ON reference_data(category, key_val);