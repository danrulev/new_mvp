-- Rollback migration for extending samples table (PostgreSQL)

ALTER TABLE samples 
    DROP COLUMN IF EXISTS photo_url,
    DROP COLUMN IF EXISTS length_mm,
    DROP COLUMN IF EXISTS width_mm,
    DROP COLUMN IF EXISTS height_mm,
    DROP COLUMN IF EXISTS shape,
    DROP COLUMN IF EXISTS weight_grams,
    DROP COLUMN IF EXISTS color,
    DROP COLUMN IF EXISTS batch_number,
    DROP COLUMN IF EXISTS manufacturer;

DROP INDEX IF EXISTS idx_samples_batch;
