BEGIN;

-- 累计扫码次数：用于区分首次/再次扫码（防伪提示）
ALTER TABLE trace_code ADD COLUMN IF NOT EXISTS scan_count INT NOT NULL DEFAULT 0;

COMMIT;
