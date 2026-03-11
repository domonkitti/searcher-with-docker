package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ---------------- EXCEL UPDATE GUIDE ----------------
//
// Main items loader expects these columns (0-index):
//
//	A: ID (source_id)
//	B: หมวด (categoryMain)
//	C: หมวดย่อย (categorySub)
//	D: กลุ่มรายการ (group)
//	E: รายการ (title)              <-- required (empty rows are skipped)
//	F: หน้า (page)
//	G: ลำดับ (row/order)
//	H: เงื่อนไขพิเศษ (special)
//	I: การใช้งบ (budgetUse)
//	J: (emergency) คอลัมน์ใหม่/ข้อความยาว (optional)
//
// -----------------------------------------------------
const (
	ColSourceID     = 0 // A ID
	ColCategoryMain = 1 // B หมวด
	ColCategorySub  = 2 // C หมวดย่อย
	ColGroup        = 3 // D กลุ่มรายการ
	ColTitle        = 4 // E รายการ
	ColPage         = 5 // F หน้า
	ColOrder        = 6 // G ลำดับ
	ColSpecial      = 7 // H เงื่อนไขพิเศษ
	ColBudgetUse    = 8 // I การใช้งบ
	ColEmergency    = 9 // J คอลัมน์ใหม่ (ยาวๆ)
)

var headerKeywords = []string{
	"id", "หมวด", "หมวดย่อย", "กลุ่มรายการ", "รายการ", "หน้า", "ลำดับ",
	"เงื่อนไข", "การใช้งบ", "ครุภัณฑ์", "อำนาจ",
	"category", "group", "title", "page", "order",
}

func looksLikeHeader(row []string) bool {
	joined := strings.ToLower(strings.TrimSpace(strings.Join(row, " ")))
	if joined == "" {
		return false
	}
	for _, k := range headerKeywords {
		if strings.Contains(joined, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

func getCol(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func defaultDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return strings.TrimSpace(s)
}

func nonEmpty(xs []string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

type ItemExcelRow struct {
	SourceID     string
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

// LoadDocsFromExcel reads ALL sheets from Excel. Each non-empty row becomes 1 doc.
func LoadDocsFromExcel(path string, titleBoost int) ([]Doc, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	docs := make([]Doc, 0, 2048)

	counter := 0

	for _, sh := range sheets {
		rows, err := f.GetRows(sh)
		if err != nil {
			continue
		}

		for rIdx, row := range rows {
			if rIdx == 0 && looksLikeHeader(row) {
				continue
			}

			title := getCol(row, ColTitle)
			if title == "" || title == "รายการ" || title == "หมวด" {
				continue
			}

			sourceID := getCol(row, ColSourceID)
			catMain := getCol(row, ColCategoryMain)
			catSub := getCol(row, ColCategorySub)
			group := getCol(row, ColGroup)
			page := getCol(row, ColPage)
			orderNo := getCol(row, ColOrder)
			special := getCol(row, ColSpecial)
			budgetUse := getCol(row, ColBudgetUse)
			emergency := getCol(row, ColEmergency)

			joined := strings.Join(nonEmpty(row), " | ")
			boosted := strings.TrimSpace(strings.Repeat(title+" ", titleBoost))
			fullText := fmt.Sprintf("%s| %s", boosted, joined)

			counter++
			id := fmt.Sprintf("%d", counter)

			meta := map[string]any{
				"source":       "excel",
				"sourceId":     strings.TrimSpace(sourceID),
				"categoryMain": defaultDash(catMain),
				"categorySub":  strings.TrimSpace(catSub),
				"group":        strings.TrimSpace(group),
				"page":         defaultDash(page),
				"row":          defaultDash(orderNo),
				"budgetUse":    strings.TrimSpace(budgetUse),
				"emergency":    strings.ReplaceAll(strings.ReplaceAll(emergency, "\r\n", "\n"), "\r", "\n"),
				"special":      strings.ReplaceAll(strings.ReplaceAll(special, "\r\n", "\n"), "\r", "\n"),
			}

			docs = append(docs, Doc{
				ID:    id,
				Title: title,
				Text:  fullText,
				Meta:  meta,
			})
		}
	}

	return docs, nil
}

func LoadItemsFromExcelFile(path string) ([]ItemExcelRow, error) {
	docs, err := LoadDocsFromExcel(path, 1)
	if err != nil {
		return nil, err
	}

	out := make([]ItemExcelRow, 0, len(docs))
	for _, d := range docs {
		meta := d.Meta
		out = append(out, ItemExcelRow{
			SourceID:     strings.TrimSpace(toStr(meta["sourceId"])),
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
