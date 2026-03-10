package internal

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "time"

    _ "github.com/lib/pq"
)

func OpenDB() (*sql.DB, error) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }

    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }

    db.SetMaxOpenConns(10)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(30 * time.Minute)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        _ = db.Close()
        return nil, err
    }

    return db, nil
}

func EnsureSchema(db *sql.DB) error {
    const q = `
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
`
    _, err := db.Exec(q)
    return err
}
