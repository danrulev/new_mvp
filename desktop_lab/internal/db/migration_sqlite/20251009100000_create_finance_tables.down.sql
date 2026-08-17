-- ============================================================================
-- ФИНАНСОВЫЙ МОДУЛЬ: Откат миграции
-- ============================================================================

DROP TABLE IF EXISTS payment_gateways;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS price_list_items;
DROP TABLE IF EXISTS price_lists;
