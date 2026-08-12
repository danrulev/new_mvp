-- Migration to extend samples table with additional fields
-- Adds: photo_url, dimensions (length/width/height), shape, weight, color, batch_number, manufacturer

-- SQLite migration
ALTER TABLE samples ADD COLUMN photo_url TEXT;
ALTER TABLE samples ADD COLUMN length_mm REAL;
ALTER TABLE samples ADD COLUMN width_mm REAL;
ALTER TABLE samples ADD COLUMN height_mm REAL;
ALTER TABLE samples ADD COLUMN shape TEXT;
ALTER TABLE samples ADD COLUMN weight_grams REAL;
ALTER TABLE samples ADD COLUMN color TEXT;
ALTER TABLE samples ADD COLUMN batch_number TEXT;
ALTER TABLE samples ADD COLUMN manufacturer TEXT;
ALTER TABLE samples ADD COLUMN updated_at TEXT DEFAULT (datetime('now'));

-- Add index for batch_number search
CREATE INDEX IF NOT EXISTS idx_samples_batch ON samples(batch_number) WHERE batch_number IS NOT NULL;
