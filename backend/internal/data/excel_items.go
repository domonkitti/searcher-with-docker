package data

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"

	"demosearch/internal/search"
)

// ---------------- EXCEL UPDATE GUIDE ----------------
//
// Main items loader expects these columns (0-index):
//
//	A: ID (source_id)
//	B: หมวด (categoryMain)
//	C: หมวดย่อย (categorySub)
//	D: กลุ่มรายการ (group)
//	E: รายการ (title)
//	F: คำบรรยาย (description)
//	G: หน้า (page)
//	H: ลำดับ (row/order)
//	I: เงื่อนไขพิเศษ (special)
//	J: การใช้งบ (budgetUse)
//	K: อำนาจอนุมัติ / ข้อความยาว (emergency) (optional)
//	L: เงื่อนไขการอนุมัติ (approvalCondition) (optional)
//
// -----------------------------------------------------
const (
	ColSourceID     = 0  // A ID
	ColCategoryMain = 1  // B หมวด
	ColCategorySub  = 2  // C หมวดย่อย
	ColGroup        = 3  // D กลุ่มรายการ
	ColTitle        = 4  // E รายการ
	ColDescription  = 5  // F คำบรรยาย
	ColPage         = 6  // G หน้า
	ColOrder        = 7  // H ลำดับ
	ColSpecial      = 8  // I เงื่อนไขพิเศษ
	ColBudgetUse    = 9  // J การใช้งบ
	ColEmergency    = 10 // K อำนาจอนุมัติ / ข้อความยาว
	ColApprovalCond = 11 // L เงื่อนไขการอนุมัติ
)

var headerKeywords = []string{
	"id", "หมวด", "หมวดย่อย", "กลุ่มรายการ", "รายการ", "คำบรรยาย", "description", "หน้า", "ลำดับ",
	"เงื่อนไข", "การใช้งบ", "ครุภัณฑ์", "อำนาจ", "อนุมัติ",
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

func normalizeMultiline(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n"), "\r", "\n")
}

type ItemExcelRow struct {
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

// LoadDocsFromExcel reads ALL sheets from Excel. Each non-empty row becomes 1 doc.
func LoadDocsFromExcel(path string, titleBoost int) ([]search.Doc, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	docs := make([]search.Doc, 0, 2048)

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
			description := normalizeMultiline(getCol(row, ColDescription))
			page := getCol(row, ColPage)
			orderNo := getCol(row, ColOrder)
			special := normalizeMultiline(getCol(row, ColSpecial))
			budgetUse := getCol(row, ColBudgetUse)
			emergency := normalizeMultiline(getCol(row, ColEmergency))
			approvalCond := normalizeMultiline(getCol(row, ColApprovalCond))

			joined := strings.Join(nonEmpty(row), " | ")
			boostedTitle := strings.TrimSpace(strings.Repeat(title+" ", titleBoost))

			fullTextParts := []string{boostedTitle}
			if strings.TrimSpace(description) != "" {
				fullTextParts = append(fullTextParts, description)
			}
			fullTextParts = append(fullTextParts, joined)
			fullText := strings.Join(fullTextParts, " | ")

			counter++
			id := fmt.Sprintf("%d", counter)

			meta := map[string]any{
				"source":            "excel",
				"sourceId":          strings.TrimSpace(sourceID),
				"categoryMain":      defaultDash(catMain),
				"categorySub":       strings.TrimSpace(catSub),
				"group":             strings.TrimSpace(group),
				"description":       description,
				"page":              defaultDash(page),
				"row":               defaultDash(orderNo),
				"budgetUse":         strings.TrimSpace(budgetUse),
				"emergency":         emergency,
				"special":           special,
				"approvalCondition": approvalCond,
			}

			docs = append(docs, search.Doc{
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
			SourceID:          strings.TrimSpace(toStr(meta["sourceId"])),
			CategoryMain:      strings.TrimSpace(toStr(meta["categoryMain"])),
			CategorySub:       strings.TrimSpace(toStr(meta["categorySub"])),
			GroupName:         strings.TrimSpace(toStr(meta["group"])),
			Title:             strings.TrimSpace(d.Title),
			Description:       strings.TrimSpace(toStr(meta["description"])),
			Page:              strings.TrimSpace(toStr(meta["page"])),
			OrderNo:           strings.TrimSpace(toStr(meta["row"])),
			Special:           strings.TrimSpace(toStr(meta["special"])),
			BudgetUse:         strings.TrimSpace(toStr(meta["budgetUse"])),
			Emergency:         strings.TrimSpace(toStr(meta["emergency"])),
			ApprovalCondition: strings.TrimSpace(toStr(meta["approvalCondition"])),
		})
	}
	return out, nil
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}