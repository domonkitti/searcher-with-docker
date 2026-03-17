package data

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
INSERT INTO kits (source_id, category, kit_name, page, order_no, special)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id`)
	if err != nil {
		return err
	}
	defer kitStmt.Close()

	lineStmt, err := tx.PrepareContext(ctx, `
INSERT INTO kit_lines (kit_id, item, sub_item, unit, line_no, linked_item_source_id)
VALUES ($1,$2,$3,$4,$5,$6)`)
	if err != nil {
		return err
	}
	defer lineStmt.Close()

	for _, k := range kits {
		if k.SourceID == "" || k.KitName == "" {
			continue
		}
		var kitDBID int64
		if err := kitStmt.QueryRowContext(ctx, k.SourceID, k.Category, k.KitName, k.Page, k.Order, k.Special).Scan(&kitDBID); err != nil {
			return err
		}
		for i, ln := range k.Lines {
			if ln.Item == "" {
				continue
			}
			if _, err := lineStmt.ExecContext(ctx, kitDBID, ln.Item, ln.SubItem, ln.Unit, i+1, nullIfBlank(ln.LinkedItemSourceID)); err != nil {
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
    COALESCE(k.source_id, ''),
    COALESCE(k.category, ''),
    COALESCE(k.kit_name, ''),
    COALESCE(k.page, ''),
    COALESCE(k.order_no, ''),
    COALESCE(k.special, ''),
    COALESCE(kl.item, ''),
    COALESCE(kl.sub_item, ''),
    COALESCE(kl.unit, ''),
    COALESCE(kl.line_no, 0),
    COALESCE(kl.linked_item_source_id, ''),
    COALESCE(i.title, '')
FROM kits k
LEFT JOIN kit_lines kl ON kl.kit_id = k.id
LEFT JOIN items i ON LOWER(i.source_id) = LOWER(kl.linked_item_source_id)
ORDER BY k.id ASC, kl.line_no ASC, kl.id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := map[int64]*KitDetail{}
	order := make([]int64, 0, 128)

	for rows.Next() {
		var kitDBID int64
		var sourceID, category, kitName, page, orderNo, special, item, subItem, unit, linkedItemSourceID, linkedItemTitle string
		var lineNo int
		if err := rows.Scan(&kitDBID, &sourceID, &category, &kitName, &page, &orderNo, &special, &item, &subItem, &unit, &lineNo, &linkedItemSourceID, &linkedItemTitle); err != nil {
			return nil, err
		}
		kd, ok := byID[kitDBID]
		if !ok {
			kd = &KitDetail{KitID: strconv.FormatInt(kitDBID, 10), SourceID: sourceID, Category: category, KitName: kitName, Page: page, Order: orderNo, Special: special, Lines: make([]KitLine, 0, 16)}
			byID[kitDBID] = kd
			order = append(order, kitDBID)
		}
		if item != "" {
			kd.Lines = append(kd.Lines, KitLine{Item: item, SubItem: subItem, Unit: unit, LinkedItemSourceID: linkedItemSourceID, LinkedItemTitle: linkedItemTitle})
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
