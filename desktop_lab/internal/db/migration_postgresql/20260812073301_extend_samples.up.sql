-- Migration to extend samples table with additional fields
-- Adds: photo_url, dimensions (length/width/height), shape, weight, color, batch_number, manufacturer

-- PostgreSQL migration
ALTER TABLE samples 
    ADD COLUMN photo_url TEXT,
    ADD COLUMN length_mm NUMERIC,
    ADD COLUMN width_mm NUMERIC,
    ADD COLUMN height_mm NUMERIC,
    ADD COLUMN shape TEXT,
    ADD COLUMN weight_grams NUMERIC,
    ADD COLUMN color TEXT,
    ADD COLUMN batch_number TEXT,
    ADD COLUMN manufacturer TEXT;

-- Add index for batch_number search
CREATE INDEX IF NOT EXISTS idx_samples_batch ON samples(batch_number) WHERE batch_number IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN samples.photo_url IS 'URL или путь к фотографии образца';
COMMENT ON COLUMN samples.length_mm IS 'Длина образца в мм';
COMMENT ON COLUMN samples.width_mm IS 'Ширина образца в мм';
COMMENT ON COLUMN samples.height_mm IS 'Высота образца в мм';
COMMENT ON COLUMN samples.shape IS 'Форма образца (куб, цилиндр, призма и т.д.)';
COMMENT ON COLUMN samples.weight_grams IS 'Вес образца в граммах';
COMMENT ON COLUMN samples.color IS 'Цвет образца';
COMMENT ON COLUMN samples.batch_number IS 'Номер партии/серии';
COMMENT ON COLUMN samples.manufacturer IS 'Производитель образца';
