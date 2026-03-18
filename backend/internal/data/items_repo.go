package data

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"demosearch/internal/search"
)

type Item struct {
	ID                int64
	SourceID          string
	CategoryMain      string
	CategorySub       string
	GroupName         string
	Title             string
	Description       string
	Page              string
	OrderNo           string
	Special           string
	BudgetUse         string
	Emergency         string
	ApprovalCondition string
}

type ItemLink struct {
	ID          int64
	ItemID      int64
	RealLink    string
	DisplayLine string
	LineNo      int
}

func LoadItemsFromExcel(path string) ([]Item, error) {
	rows, err := LoadItemsFromExcelFile(path)
	if err != nil {
		return nil, err
	}

	out := make([]Item, 0, len(rows))
	for _, r := range rows {
		out = append(out, Item{
			SourceID:          strings.TrimSpace(r.SourceID),
			CategoryMain:      strings.TrimSpace(r.CategoryMain),
			CategorySub:       strings.TrimSpace(r.CategorySub),
			GroupName:         strings.TrimSpace(r.GroupName),
			Title:             strings.TrimSpace(r.Title),
			Description:       strings.TrimSpace(r.Description),
			Page:              strings.TrimSpace(r.Page),
			OrderNo:           strings.TrimSpace(r.OrderNo),
			Special:           strings.TrimSpace(r.Special),
			BudgetUse:         strings.TrimSpace(r.BudgetUse),
			Emergency:         strings.TrimSpace(r.Emergency),
			ApprovalCondition: strings.TrimSpace(r.ApprovalCondition),
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

	if _, err := tx.ExecContext(ctx, `DELETE FROM item_links`); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM items`); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO items (
    source_id, category_main, category_sub, group_name, title, description, page, order_no, special, budget_use, emergency, approval_condition
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, it := range items {
		if strings.TrimSpace(it.Title) == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx,
			nullIfBlank(it.SourceID),
			nullIfBlank(it.CategoryMain),
			nullIfBlank(it.CategorySub),
			nullIfBlank(it.GroupName),
			it.Title,
			nullIfBlank(it.Description),
			nullIfBlank(it.Page),
			nullIfBlank(it.OrderNo),
			nullIfBlank(it.Special),
			nullIfBlank(it.BudgetUse),
			nullIfBlank(it.Emergency),
			nullIfBlank(it.ApprovalCondition),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func ReplaceAllItemLinks(ctx context.Context, db *sql.DB, links []ItemLinkRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM item_links`); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO item_links (item_id, real_link, display_line, line_no)
SELECT i.id, $2, $3, $4
FROM items i
WHERE i.source_id = $1`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, link := range links {
		if strings.TrimSpace(link.SourceID) == "" || strings.TrimSpace(link.RealLink) == "" || strings.TrimSpace(link.DisplayLine) == "" {
			continue
		}

		res, err := stmt.ExecContext(ctx,
			strings.TrimSpace(link.SourceID),
			strings.TrimSpace(link.RealLink),
			strings.TrimSpace(link.DisplayLine),
			link.LineNo,
		)
		if err != nil {
			return err
		}

		affected, _ := res.RowsAffected()
		if affected == 0 {
			return fmt.Errorf("not found item source_id=%s for link", link.SourceID)
		}
	}

	return tx.Commit()
}

func LoadAllItems(ctx context.Context, db *sql.DB) ([]Item, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id,
       COALESCE(source_id, ''),
       COALESCE(category_main, ''),
       COALESCE(category_sub, ''),
       COALESCE(group_name, ''),
       COALESCE(title, ''),
       COALESCE(description, ''),
       COALESCE(page, ''),
       COALESCE(order_no, ''),
       COALESCE(special, ''),
       COALESCE(budget_use, ''),
       COALESCE(emergency, ''),
       COALESCE(approval_condition, '')
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
			&it.SourceID,
			&it.CategoryMain,
			&it.CategorySub,
			&it.GroupName,
			&it.Title,
			&it.Description,
			&it.Page,
			&it.OrderNo,
			&it.Special,
			&it.BudgetUse,
			&it.Emergency,
			&it.ApprovalCondition,
		); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func LoadAllItemLinksMap(ctx context.Context, db *sql.DB) (map[int64][]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `
SELECT item_id, real_link, display_line, line_no
FROM item_links
ORDER BY item_id ASC, line_no ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][]map[string]any)
	for rows.Next() {
		var itemID int64
		var realLink, displayLine string
		var lineNo int
		if err := rows.Scan(&itemID, &realLink, &displayLine, &lineNo); err != nil {
			return nil, err
		}
		out[itemID] = append(out[itemID], map[string]any{
			"realLink":    realLink,
			"displayLine": displayLine,
			"lineNo":      lineNo,
		})
	}
	return out, rows.Err()
}

func LoadDocsFromDB(ctx context.Context, db *sql.DB, titleBoost int) ([]search.Doc, error) {
	items, err := LoadAllItems(ctx, db)
	if err != nil {
		return nil, err
	}

	linksMap, err := LoadAllItemLinksMap(ctx, db)
	if err != nil {
		return nil, err
	}

	docs := make([]search.Doc, 0, len(items))
	for _, it := range items {
		boostedTitle := strings.TrimSpace(strings.Repeat(it.Title+" ", titleBoost))

		// Search text intentionally stays narrow: title + description only.
		// Extra business fields still go into Meta for rendering / filtering,
		// but should not outrank the main item title during retrieval.
		textParts := []string{
			boostedTitle,
			strings.TrimSpace(it.Description),
		}

		docs = append(docs, search.Doc{
			ID:    strconv.FormatInt(it.ID, 10),
			Title: it.Title,
			Text:  strings.Join(textParts, " | "),
			Meta: map[string]any{
				"source":            "postgres",
				"sourceId":          strings.TrimSpace(it.SourceID),
				"categoryMain":      defaultDash(it.CategoryMain),
				"categorySub":       strings.TrimSpace(it.CategorySub),
				"group":             strings.TrimSpace(it.GroupName),
				"description":       strings.ReplaceAll(strings.ReplaceAll(it.Description, "\r\n", "\n"), "\r", "\n"),
				"page":              defaultDash(it.Page),
				"row":               defaultDash(it.OrderNo),
				"budgetUse":         strings.TrimSpace(it.BudgetUse),
				"emergency":         strings.ReplaceAll(strings.ReplaceAll(it.Emergency, "\r\n", "\n"), "\r", "\n"),
				"special":           strings.ReplaceAll(strings.ReplaceAll(it.Special, "\r\n", "\n"), "\r", "\n"),
				"approvalCondition": strings.ReplaceAll(strings.ReplaceAll(it.ApprovalCondition, "\r\n", "\n"), "\r", "\n"),
				"links":             linksMap[it.ID],
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

func SaveUploadedItemLinksExcelAndImport(ctx context.Context, db *sql.DB, tempFilePath string) (int, error) {
	links, err := LoadItemLinksFromExcel(tempFilePath)
	if err != nil {
		return 0, err
	}
	if err := ReplaceAllItemLinks(ctx, db, links); err != nil {
		return 0, err
	}
	return len(links), nil
}

func MustBuildDocsOrEmpty(ctx context.Context, db *sql.DB, titleBoost int) []search.Doc {
	docs, err := LoadDocsFromDB(ctx, db, titleBoost)
	if err != nil {
		fmt.Println("load docs from db error:", err)
		return []search.Doc{}
	}
	return docs
}

func nullIfBlank(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}
