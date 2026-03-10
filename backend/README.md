# Backend

This backend now uses PostgreSQL as the source of truth for searchable items.
Search ranking still runs in Go memory, so `/api/search` and `/api/doc/:id` behave the same as before.

## Main flow

1. Backend starts and connects to PostgreSQL using `DATABASE_URL`.
2. It ensures the `items` table exists.
3. It loads all items from PostgreSQL and builds the in-memory search engine.
4. Admin uploads an Excel file to `POST /api/admin/import/items`.
5. Backend parses Excel, deletes old items, inserts new items, then rebuilds the search engine.

## Required files

- `kits.xlsx`
- `data/rules.json`
- `data/synonyms.json`

`items` no longer comes from a startup Excel file. It now comes from PostgreSQL.

## Important API

- `GET /api/search?q=...`
- `GET /api/doc/:id`
- `POST /api/admin/import/items` (multipart field name: `file`)
