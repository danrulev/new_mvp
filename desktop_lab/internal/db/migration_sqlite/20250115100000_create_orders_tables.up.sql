-- Включаем поддержку внешних ключей
PRAGMA foreign_keys = ON;

-- ============================================================================
-- ТАБЛИЦЫ ДЛЯ УПРАВЛЕНИЯ ЗАЯВКАМИ (ORDERS)
-- ============================================================================

-- Заявки на исследования
CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    created_by TEXT NOT NULL,
    assigned_to TEXT,
    status TEXT NOT NULL DEFAULT 'new', -- new, approved, rejected, assigned, in_progress, on_hold, completed, cancelled
    priority TEXT NOT NULL DEFAULT 'normal', -- normal, high, urgent
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    internal_comment TEXT,
    external_comment TEXT,
    total_amount REAL NOT NULL DEFAULT 0.0,
    currency TEXT NOT NULL DEFAULT 'RUB',
    client_name TEXT NOT NULL,
    client_email TEXT NOT NULL,
    client_phone TEXT,
    sample_location TEXT,
    due_date TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    completed_at TEXT,
    
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_orders_created_by ON orders(created_by);
CREATE INDEX IF NOT EXISTS idx_orders_assigned_to ON orders(assigned_to);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_priority ON orders(priority);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);

-- Позиции заявки (конкретные исследования)
CREATE TABLE IF NOT EXISTS order_items (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    test_method_id TEXT NOT NULL,
    test_method_name TEXT NOT NULL,
    assigned_to TEXT,
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL DEFAULT 0.0,
    subtotal REAL NOT NULL DEFAULT 0.0,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, assigned, in_work, completed, cancelled
    sample_required INTEGER DEFAULT 0,
    sample_notes TEXT,
    sample_count INTEGER DEFAULT 0,
    sample_delivered INTEGER DEFAULT 0,
    sample_received_at TEXT,
    internal_notes TEXT,
    result_notes TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    completed_at TEXT,
    
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (test_method_id) REFERENCES test_methods(id) ON DELETE RESTRICT,
    FOREIGN KEY (assigned_to) REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_method ON order_items(test_method_id);
CREATE INDEX IF NOT EXISTS idx_order_items_status ON order_items(status);
CREATE INDEX IF NOT EXISTS idx_order_items_assigned_to ON order_items(assigned_to);

-- История изменения статусов заявок (workflow)
CREATE TABLE IF NOT EXISTS order_workflow (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL,
    comment TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_order_workflow_order ON order_workflow(order_id);
CREATE INDEX IF NOT EXISTS idx_order_workflow_created_at ON order_workflow(created_at);



