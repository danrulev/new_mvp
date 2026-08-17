-- ============================================================================
-- ФИНАНСОВЫЙ МОДУЛЬ: Прайс-листы, Счета, Платежи
-- ============================================================================

-- Прайс-листы организаций
CREATE TABLE IF NOT EXISTS price_lists (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    name TEXT NOT NULL,
    is_default INTEGER DEFAULT 0,
    valid_from DATE,
    valid_to DATE,
    currency TEXT NOT NULL DEFAULT 'RUB',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_price_lists_org ON price_lists(organization_id);
CREATE INDEX IF NOT EXISTS idx_price_lists_default ON price_lists(is_default);

-- Позиции прайс-листа
CREATE TABLE IF NOT EXISTS price_list_items (
    id TEXT PRIMARY KEY,
    price_list_id TEXT NOT NULL,
    test_method_id TEXT NOT NULL,
    base_price REAL NOT NULL,
    discount_percent REAL DEFAULT 0,
    min_quantity INTEGER DEFAULT 1,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (price_list_id) REFERENCES price_lists(id) ON DELETE CASCADE,
    FOREIGN KEY (test_method_id) REFERENCES test_methods(id) ON DELETE CASCADE,
    UNIQUE (price_list_id, test_method_id)
);
CREATE INDEX IF NOT EXISTS idx_price_list_items_pl ON price_list_items(price_list_id);
CREATE INDEX IF NOT EXISTS idx_price_list_items_tm ON price_list_items(test_method_id);

-- Счета на оплату
CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    order_id TEXT NOT NULL,
    invoice_number TEXT NOT NULL UNIQUE,
    customer_id TEXT NOT NULL,
    billing_address TEXT DEFAULT '{}', -- JSON
    subtotal REAL NOT NULL DEFAULT 0,
    tax REAL NOT NULL DEFAULT 0,
    discount REAL NOT NULL DEFAULT 0,
    total REAL NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'draft', -- 'draft', 'sent', 'paid', 'overdue', 'cancelled'
    due_date DATE,
    paid_at TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (customer_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_invoices_order ON invoices(order_id);
CREATE INDEX IF NOT EXISTS idx_invoices_customer ON invoices(customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status);
CREATE INDEX IF NOT EXISTS idx_invoices_number ON invoices(invoice_number);

-- Позиции счета
CREATE TABLE IF NOT EXISTS invoice_items (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    order_item_id TEXT,
    description TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    unit_price REAL NOT NULL,
    subtotal REAL NOT NULL,
    created_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE,
    FOREIGN KEY (order_item_id) REFERENCES order_items(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_invoice_items_inv ON invoice_items(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_oi ON invoice_items(order_item_id);

-- Платежи
CREATE TABLE IF NOT EXISTS payments (
    id TEXT PRIMARY KEY,
    invoice_id TEXT NOT NULL,
    amount REAL NOT NULL,
    payment_method TEXT NOT NULL, -- 'card', 'bank_transfer', 'cash', 'online'
    transaction_id TEXT,
    payment_date TEXT NOT NULL,
    metadata TEXT DEFAULT '{}', -- JSON данные от платежной системы
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending', 'completed', 'failed', 'refunded'
    created_at TEXT DEFAULT (datetime('now')),
    
    FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_payments_invoice ON payments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_transaction ON payments(transaction_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);

-- Платежные шлюзы
CREATE TABLE IF NOT EXISTS payment_gateways (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL UNIQUE, -- 'stripe', 'cloudpayments', 'tinkoff'
    api_keys TEXT NOT NULL, -- Encrypted JSON
    webhook_url TEXT,
    is_active INTEGER DEFAULT 0,
    config TEXT DEFAULT '{}', -- JSON конфигурация
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_payment_gateways_active ON payment_gateways(is_active);
