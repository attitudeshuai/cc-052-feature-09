BEGIN;

-- 包装单位字典（件/盒/箱…），溯源码与包装单位的对应关系以此为准
CREATE TABLE IF NOT EXISTS package_unit (
    code       VARCHAR(16) PRIMARY KEY,
    name       VARCHAR(32) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO package_unit (code, name, sort_order) VALUES
    ('bag',  '袋', 10),
    ('box',  '盒', 20),
    ('case', '箱', 30)
ON CONFLICT (code) DO NOTHING;

-- 每个溯源码对应一个包装单位
ALTER TABLE trace_code ADD COLUMN IF NOT EXISTS package_unit_code VARCHAR(16)
    REFERENCES package_unit(code);

-- 历史码（002 之前生成的）统一补默认包装单位，保证「一条码↔一包装单位」完整对应
UPDATE trace_code
   SET package_unit_code = 'bag'
 WHERE package_unit_code IS NULL;

ALTER TABLE trace_code ALTER COLUMN package_unit_code SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_trace_code_package_unit ON trace_code(package_unit_code);

-- 每次扫码都留痕：据此区分首次/再次并统计真实扫码次数；
-- region 只来自扫码端显式上报的行政区划码，不接受 IP / 转发头推断
CREATE TABLE IF NOT EXISTS trace_scan (
    id            BIGSERIAL PRIMARY KEY,
    trace_code_id BIGINT      NOT NULL REFERENCES trace_code(id) ON DELETE CASCADE,
    scanned_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    region_code   VARCHAR(6),
    source        VARCHAR(16) NOT NULL DEFAULT 'consumer',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_trace_scan_code ON trace_scan(trace_code_id);
CREATE INDEX IF NOT EXISTS idx_trace_scan_code_time ON trace_scan(trace_code_id, scanned_at);

COMMIT;
