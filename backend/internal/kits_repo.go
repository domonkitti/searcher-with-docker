package internal

import (
    "context"
    "database/sql"
    "strconv"
)

func ReplaceAllKits(ctx context.Context, db *sql.DB, kits []KitDetail) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    if _, err := tx.ExecContext(ctx, `DELETE FROM kit_lines`); err != nil {
        return err
    }
    if _, err := tx.ExecContext(ctx, `DELETE FROM kits`); err != nil {
        return err
    }

    kitStmt, err := tx.PrepareContext(ctx, `
INSERT INTO kits (category, kit_name, page, order_no, special)
VALUES ($1,$2,$3,$4,$5)
RETURNING id`)
    if err != nil {
        return err
    }
    defer kitStmt.Close()

    lineStmt, err := tx.PrepareContext(ctx, `
INSERT INTO kit_lines (kit_id, item, sub_item, unit, line_no)
VALUES ($1,$2,$3,$4,$5)`)
    if err != nil {
        return err
    }
    defer lineStmt.Close()

    for _, k := range kits {
        if k.KitName == "" {
            continue
        }
        var kitDBID int64
        if err := kitStmt.QueryRowContext(ctx,
            k.Category,
            k.KitName,
            k.Page,
            k.Order,
            k.Special,
        ).Scan(&kitDBID); err != nil {
            return err
        }

        for i, ln := range k.Lines {
            if ln.Item == "" {
                continue
            }
            if _, err := lineStmt.ExecContext(ctx,
                kitDBID,
                ln.Item,
                ln.SubItem,
                ln.Unit,
                i+1,
            ); err != nil {
                return err
            }
        }
    }

    return tx.Commit()
}

func LoadKitDetailsFromDB(ctx context.Context, db *sql.DB) ([]KitDetail, error) {
    rows, err := db.QueryContext(ctx, `
SELECT
    k.id,
    k.category,
    k.kit_name,
    k.page,
    k.order_no,
    k.special,
    kl.item,
    kl.sub_item,
    kl.unit,
    kl.line_no
FROM kits k
LEFT JOIN kit_lines kl ON kl.kit_id = k.id
ORDER BY k.id ASC, kl.line_no ASC, kl.id ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    byID := map[int64]*KitDetail{}
    order := make([]int64, 0, 128)

    for rows.Next() {
        var (
            kitDBID                                 int64
            category, kitName, page, orderNo       sql.NullString
            special, item, subItem, unit           sql.NullString
            lineNo                                 sql.NullInt64
        )
        if err := rows.Scan(
            &kitDBID,
            &category,
            &kitName,
            &page,
            &orderNo,
            &special,
            &item,
            &subItem,
            &unit,
            &lineNo,
        ); err != nil {
            return nil, err
        }

        kd, ok := byID[kitDBID]
        if !ok {
            kd = &KitDetail{
                KitID:    strconv.FormatInt(kitDBID, 10),
                Category: category.String,
                KitName:  kitName.String,
                Page:     page.String,
                Order:    orderNo.String,
                Special:  special.String,
                Lines:    make([]KitLine, 0, 16),
            }
            byID[kitDBID] = kd
            order = append(order, kitDBID)
        }

        if item.Valid && item.String != "" {
            kd.Lines = append(kd.Lines, KitLine{
                Item:    item.String,
                SubItem: subItem.String,
                Unit:    unit.String,
            })
        }
    }
    if err := rows.Err(); err != nil {
        return nil, err
    }

    out := make([]KitDetail, 0, len(order))
    for _, id := range order {
        out = append(out, *byID[id])
    }
    return out, nil
}

func SaveUploadedKitsExcelAndImport(ctx context.Context, db *sql.DB, tempFilePath string) (int, error) {
    kits, err := LoadKitsFromExcel(tempFilePath)
    if err != nil {
        return 0, err
    }
    if err := ReplaceAllKits(ctx, db, kits); err != nil {
        return 0, err
    }
    return len(kits), nil
}
