CREATE TABLE IF NOT EXISTS items (
    id BIGSERIAL PRIMARY KEY,
    source_id TEXT UNIQUE,
    category_main TEXT,
    category_sub TEXT,
    group_name TEXT,
    title TEXT NOT NULL,
    description TEXT,
    page TEXT,
    order_no TEXT,
    special TEXT,
    budget_use TEXT,
    emergency TEXT,
    approval_condition TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_items_source_id
ON items(source_id);

CREATE TABLE IF NOT EXISTS kits (
    id BIGSERIAL PRIMARY KEY,
    source_id TEXT UNIQUE,
    category TEXT,
    kit_name TEXT NOT NULL,
    page TEXT,
    order_no TEXT,
    special TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_kits_source_id
ON kits(source_id);

CREATE TABLE IF NOT EXISTS kit_lines (
    id BIGSERIAL PRIMARY KEY,
    kit_id BIGINT NOT NULL REFERENCES kits(id) ON DELETE CASCADE,
    item TEXT NOT NULL,
    sub_item TEXT,
    unit TEXT,
    line_no INT NOT NULL DEFAULT 0,
    linked_item_source_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_kit_lines_linked_item_source_id
ON kit_lines(linked_item_source_id);

CREATE TABLE IF NOT EXISTS item_links (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    real_link TEXT NOT NULL,
    display_line TEXT NOT NULL,
    line_no INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_item_links_item_id
ON item_links(item_id);
