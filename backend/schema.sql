CREATE TABLE IF NOT EXISTS items (
    id BIGSERIAL PRIMARY KEY,
    category_main TEXT,
    category_sub TEXT,
    group_name TEXT,
    title TEXT NOT NULL,
    page TEXT,
    order_no TEXT,
    special TEXT,
    budget_use TEXT,
    emergency TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS kits (
    id BIGSERIAL PRIMARY KEY,
    category TEXT,
    kit_name TEXT NOT NULL,
    page TEXT,
    order_no TEXT,
    special TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS kit_lines (
    id BIGSERIAL PRIMARY KEY,
    kit_id BIGINT NOT NULL REFERENCES kits(id) ON DELETE CASCADE,
    item TEXT NOT NULL,
    sub_item TEXT,
    unit TEXT,
    line_no INT NOT NULL DEFAULT 0
);
