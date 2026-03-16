package data

import (
	"os"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

type KitLine struct {
	Item    string `json:"item"`
	SubItem string `json:"subItem,omitempty"`
	Unit    string `json:"unit,omitempty"`
}

type KitDetail struct {
	KitID    string    `json:"kitId"`             // internal DB id
	SourceID string    `json:"sourceId"`          // public URL id from Excel column A
	Category string    `json:"category,omitempty"`
	KitName  string    `json:"kitName"`
	Page     string    `json:"page,omitempty"`
	Order    string    `json:"order,omitempty"`
	Special  string    `json:"special,omitempty"`
	Lines    []KitLine `json:"lines"`
}

// ---------------- EXCEL UPDATE GUIDE ----------------
//
// This loader finds a header row and maps columns by header text.
// Expected headers (Thai/English) are matched loosely, but the default order is:
//
//   A: ID / Source ID
//   B: หมวด
//   C: ชื่อชุดเครื่องมือ
//   D: รายการ
//   E: รายการย่อย
//   F: หน่วย
//   G: หน้า
//   H: ลำดับ
//   I: เงื่อนไขพิเศษ
//
// -----------------------------------------------------
type kitCols struct {
	source, cat, kit, item, sub, unit, page, order, special int
	ok                                                      bool
}

func LoadKitsFromExcel(path string) ([]KitDetail, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()

	// ใช้ source_id จาก Excel เป็นกุญแจหลัก
	bySourceID := map[string]*KitDetail{}

	for _, sh := range sheets {
		rows, err := f.GetRows(sh)
		if err != nil || len(rows) == 0 {
			continue
		}

		hRowIdx, cols := findKitHeaderAndCols(rows)
		if !cols.ok {
			hRowIdx = 0
			cols = kitCols{
				source:  0, // A
				cat:     1, // B
				kit:     2, // C
				item:    3, // D
				sub:     4, // E
				unit:    5, // F
				page:    6, // G
				order:   7, // H
				special: 8, // I
				ok:      true,
			}
		}

		for i := hRowIdx + 1; i < len(rows); i++ {
			r := rows[i]
			get := func(idx int) string {
				if idx < 0 || idx >= len(r) {
					return ""
				}
				return strings.TrimSpace(r[idx])
			}

			sourceID := normalizeKey(get(cols.source))
			category := get(cols.cat)
			kitName := get(cols.kit)
			item := get(cols.item)
			sub := get(cols.sub)
			unit := get(cols.unit)
			page := get(cols.page)
			order := get(cols.order)
			special := get(cols.special)

			if sourceID == "" || kitName == "" || item == "" {
				continue
			}

			k, ok := bySourceID[sourceID]
			if !ok {
				k = &KitDetail{
					SourceID: sourceID,
					Category: strings.TrimSpace(category),
					KitName:  strings.TrimSpace(kitName),
					Page:     strings.TrimSpace(page),
					Order:    strings.TrimSpace(order),
					Special:  strings.TrimSpace(special),
					Lines:    make([]KitLine, 0, 32),
				}
				bySourceID[sourceID] = k
			}

			if k.Category == "" && strings.TrimSpace(category) != "" {
				k.Category = strings.TrimSpace(category)
			}
			if k.Page == "" && strings.TrimSpace(page) != "" {
				k.Page = strings.TrimSpace(page)
			}
			if k.Order == "" && strings.TrimSpace(order) != "" {
				k.Order = strings.TrimSpace(order)
			}
			if k.Special == "" && strings.TrimSpace(special) != "" {
				k.Special = strings.TrimSpace(special)
			}

			k.Lines = append(k.Lines, KitLine{
				Item:    strings.TrimSpace(item),
				SubItem: strings.TrimSpace(sub),
				Unit:    strings.TrimSpace(unit),
			})
		}
	}

	out := make([]KitDetail, 0, len(bySourceID))
	for _, v := range bySourceID {
		sort.SliceStable(v.Lines, func(i, j int) bool {
			if v.Lines[i].Item == v.Lines[j].Item {
				return v.Lines[i].SubItem < v.Lines[j].SubItem
			}
			return v.Lines[i].Item < v.Lines[j].Item
		})
		out = append(out, *v)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].KitName < out[j].KitName
	})

	return out, nil
}

func normalizeKey(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func findKitHeaderAndCols(rows [][]string) (headerRow int, cols kitCols) {
	limit := len(rows)
	if limit > 20 {
		limit = 20
	}

	norm := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, "\t", "")
		return strings.ToLower(s)
	}

	for r := 0; r < limit; r++ {
		row := rows[r]
		m := map[string]int{}
		for i, cell := range row {
			k := norm(cell)
			if k != "" {
				m[k] = i
			}
		}

		_, okSource := m["id"]
		if !okSource {
			_, okSource = m["sourceid"]
		}
		if !okSource {
			_, okSource = m["source_id"]
		}
		_, okKit := m["ชื่อชุดเครื่องมือ"]
		_, okItem := m["รายการ"]

		if okSource && okKit && okItem {
			return r, kitCols{
				source:  getOr(m, "id", getOr(m, "sourceid", getOr(m, "source_id", -1))),
				cat:     getOr(m, "หมวด", -1),
				kit:     getOr(m, "ชื่อชุดเครื่องมือ", -1),
				item:    getOr(m, "รายการ", -1),
				sub:     getOr(m, "รายการย่อย", -1),
				unit:    getOr(m, "หน่วย", -1),
				page:    getOr(m, "หน้า", -1),
				order:   getOr(m, "ลำดับ", -1),
				special: getOr(m, "เงื่อนไขพิเศษ", -1),
				ok:      true,
			}
		}
	}

	return 0, kitCols{ok: false}
}

func getOr(m map[string]int, k string, def int) int {
	if v, ok := m[k]; ok {
		return v
	}
	return def
}