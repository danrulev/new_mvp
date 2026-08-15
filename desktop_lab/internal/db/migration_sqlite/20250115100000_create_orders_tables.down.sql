-- Откат миграции для таблиц заявок

DROP TABLE IF EXISTS organization_invitations;
DROP TABLE IF EXISTS order_workflow;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;

