package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ---------------- EXCEL UPDATE GUIDE ----------------
//
// This loader expects these columns (0-index):
//   A: หมวด (categoryMain)
//   B: หมวดย่อย (categorySub)
//   C: กลุ่มรายการ (group)
//   D: รายการ (title)              <-- required (empty rows are skipped)
//   E: หน้า (page)
//   F: ลำดับ (row/order)
//   G: เงื่อนไขพิเศษ (special)
//   H: การใช้งบ (budgetUse)
//   I: (emergency) คอลัมน์ใหม่/ข้อความยาว (optional)
//
// ✅ You can add NEW rows freely (append at the bottom) — no code change needed.
// ⚠️ If you INSERT/REORDER columns, update the Col* constants below to match.
// -----------------------------------------------------
// Column mapping for data.xlsx (0-index): A=0, B=1, ...
const (
	ColCategoryMain = 0 // A หมวด
	ColCategorySub  = 1 // B หมวดย่อย
	ColGroup        = 2 // C กลุ่มรายการ
	ColTitle        = 3 // D รายการ
	ColPage         = 4 // E หน้า
	ColOrder        = 5 // F ลำดับ
	ColSpecial      = 6 // G เงื่อนไขพิเศษ
	ColBudgetUse    = 7 // H การใช้งบ
	ColEmergency    = 8 // I คอลัมน์ใหม่ (ยาวๆ)
)

var headerKeywords = []string{
	"หมวด", "หมวดย่อย", "กลุ่มรายการ", "รายการ", "หน้า", "ลำดับ",
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
			// Skip header if first row looks like header
			if rIdx == 0 && looksLikeHeader(row) {
				continue
			}

			title := getCol(row, ColTitle)
			if title == "" || title == "รายการ" || title == "หมวด" {
				continue
			}

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
			id := fmt.Sprintf("%d", counter) // ✅ URL-safe ID

			meta := map[string]any{
				"source":       "excel",
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
