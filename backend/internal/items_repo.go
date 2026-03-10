package internal

import (
    "context"
    "database/sql"
    "fmt"
    "strconv"
    "strings"
)

type Item struct {
    ID           int64
    CategoryMain string
    CategorySub  string
    GroupName    string
    Title        string
    Page         string
    OrderNo      string
    Special      string
    BudgetUse    string
    Emergency    string
}

func LoadItemsFromExcel(path string) ([]Item, error) {
    docs, err := LoadDocsFromExcel(path, 1)
    if err != nil {
        return nil, err
    }

    out := make([]Item, 0, len(docs))
    for _, d := range docs {
        meta := d.Meta
        out = append(out, Item{
            CategoryMain: strings.TrimSpace(toStr(meta["categoryMain"])),
            CategorySub:  strings.TrimSpace(toStr(meta["categorySub"])),
            GroupName:    strings.TrimSpace(toStr(meta["group"])),
            Title:        strings.TrimSpace(d.Title),
            Page:         strings.TrimSpace(toStr(meta["page"])),
            OrderNo:      strings.TrimSpace(toStr(meta["row"])),
            Special:      strings.TrimSpace(toStr(meta["special"])),
            BudgetUse:    strings.TrimSpace(toStr(meta["budgetUse"])),
            Emergency:    strings.TrimSpace(toStr(meta["emergency"])),
        })
    }
    return out, nil
}

func ReplaceAllItems(ctx context.Context, db *sql.DB, items []Item) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.ExecContext(ctx, `DELETE FROM items`); err != nil {
        return err
    }

    stmt, err := tx.PrepareContext(ctx, `
INSERT INTO items (
    category_main, category_sub, group_name, title, page, order_no, special, budget_use, emergency
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`)
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, it := range items {
        if strings.TrimSpace(it.Title) == "" {
            continue
        }
        if _, err := stmt.ExecContext(ctx,
            it.CategoryMain,
            it.CategorySub,
            it.GroupName,
            it.Title,
            it.Page,
            it.OrderNo,
            it.Special,
            it.BudgetUse,
            it.Emergency,
        ); err != nil {
            return err
        }
    }

    return tx.Commit()
}

func LoadAllItems(ctx context.Context, db *sql.DB) ([]Item, error) {
    rows, err := db.QueryContext(ctx, `
SELECT id, category_main, category_sub, group_name, title, page, order_no, special, budget_use, emergency
FROM items
ORDER BY id ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    out := make([]Item, 0, 2048)
    for rows.Next() {
        var it Item
        if err := rows.Scan(
            &it.ID,
            &it.CategoryMain,
            &it.CategorySub,
            &it.GroupName,
            &it.Title,
            &it.Page,
            &it.OrderNo,
            &it.Special,
            &it.BudgetUse,
            &it.Emergency,
        ); err != nil {
            return nil, err
        }
        out = append(out, it)
    }
    return out, rows.Err()
}

func LoadDocsFromDB(ctx context.Context, db *sql.DB, titleBoost int) ([]Doc, error) {
    items, err := LoadAllItems(ctx, db)
    if err != nil {
        return nil, err
    }

    docs := make([]Doc, 0, len(items))
    for _, it := range items {
        boosted := strings.TrimSpace(strings.Repeat(it.Title+" ", titleBoost))
        docs = append(docs, Doc{
            ID:    strconv.FormatInt(it.ID, 10),
            Title: it.Title,
            Text:  boosted,
            Meta: map[string]any{
                "source":       "postgres",
                "categoryMain": defaultDash(it.CategoryMain),
                "categorySub":  strings.TrimSpace(it.CategorySub),
                "group":        strings.TrimSpace(it.GroupName),
                "page":         defaultDash(it.Page),
                "row":          defaultDash(it.OrderNo),
                "budgetUse":    strings.TrimSpace(it.BudgetUse),
                "emergency":    strings.ReplaceAll(strings.ReplaceAll(it.Emergency, "\r\n", "\n"), "\r", "\n"),
                "special":      strings.ReplaceAll(strings.ReplaceAll(it.Special, "\r\n", "\n"), "\r", "\n"),
            },
        })
    }
    return docs, nil
}

func SaveUploadedExcelAndImport(ctx context.Context, db *sql.DB, tempFilePath string) (int, error) {
    items, err := LoadItemsFromExcel(tempFilePath)
    if err != nil {
        return 0, err
    }
    if err := ReplaceAllItems(ctx, db, items); err != nil {
        return 0, err
    }
    return len(items), nil
}

func MustBuildDocsOrEmpty(ctx context.Context, db *sql.DB, titleBoost int) []Doc {
    docs, err := LoadDocsFromDB(ctx, db, titleBoost)
    if err != nil {
        fmt.Println("load docs from db error:", err)
        return []Doc{}
    }
    return docs
}
