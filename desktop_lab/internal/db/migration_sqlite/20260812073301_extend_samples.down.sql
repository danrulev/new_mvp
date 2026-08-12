-- Rollback migration for extending samples table
-- SQLite does not support DROP COLUMN in older versions, so we recreate the table

-- Create temporary table with old structure
CREATE TABLE IF NOT EXISTS samples_old (
    id TEXT PRIMARY KEY,
    group_id TEXT,
    material_id TEXT NOT NULL,
    sample_number TEXT NOT NULL,
    collection_date TEXT DEFAULT (datetime('now')),
    collection_place TEXT,
    context_params TEXT NOT NULL DEFAULT '{}', 
    note TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

-- Copy data from new table to old table (excluding new columns)
INSERT INTO samples_old (id, group_id, material_id, sample_number, collection_date, collection_place, context_params, note, created_at, updated_at)
SELECT id, group_id, material_id, sample_number, collection_date, collection_place, context_params, note, created_at, updated_at
FROM samples;

-- Drop the new table
DROP TABLE IF EXISTS samples;

-- Rename old table to samples
ALTER TABLE samples_old RENAME TO samples;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_samples_group ON samples(group_id);
