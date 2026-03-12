package data

import (
	"fmt"
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
	KitID    string    `json:"kitId"` // ✅ URL-safe ID (counter)
	Category string    `json:"category,omitempty"`
	KitName  string    `json:"kitName"`
	Page     string    `json:"page,omitempty"`
	Order    string    `json:"order,omitempty"`   // เก็บไว้ก่อน
	Special  string    `json:"special,omitempty"` // เก็บไว้ก่อน
	Lines    []KitLine `json:"lines"`
}

// ---------------- EXCEL UPDATE GUIDE ----------------
//
// This loader finds a header row and maps columns by header text.
// Expected headers (Thai/English) are matched loosely, but the default order is:
//
//   หมวด | ชื่อชุดเครื่องมือ | รายการ | รายการย่อย | หน่วย | หน้า | ลำดับ | เงื่อนไขพิเศษ
//
// ✅ You can add NEW rows freely — no code change needed.
// ⚠️ If you rename header words drastically, update findKitHeaderAndCols() keywords.
// -----------------------------------------------------
type kitCols struct {
	cat, kit, item, sub, unit, page, order, special int
	ok                                              bool
}

// Excel columns: หมวด | ชื่อชุดเครื่องมือ | รายการ | รายการย่อย | หน่วย | หน้า | ลำดับ | เงื่อนไขพิเศษ
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

	// ✅ ทำแบบ doc: id เป็น counter แต่ต้อง "คงเดิมภายในรันเดียว"
	kitNameToID := map[string]string{}
	byID := map[string]*KitDetail{}
	counter := 0

	for _, sh := range sheets {
		rows, err := f.GetRows(sh)
		if err != nil || len(rows) == 0 {
			continue
		}

		hRowIdx, cols := findKitHeaderAndCols(rows)
		if !cols.ok {
			hRowIdx = 0
			cols = kitCols{cat: 0, kit: 1, item: 2, sub: 3, unit: 4, page: 5, order: 6, special: 7, ok: true}
		}

		for i := hRowIdx + 1; i < len(rows); i++ {
			r := rows[i]
			get := func(idx int) string {
				if idx < 0 || idx >= len(r) {
					return ""
				}
				return strings.TrimSpace(r[idx])
			}

			category := get(cols.cat)
			kitName := get(cols.kit)
			item := get(cols.item)
			sub := get(cols.sub)
			unit := get(cols.unit)
			page := get(cols.page)
			order := get(cols.order)
			special := get(cols.special)

			if kitName == "" || item == "" {
				continue
			}

			// ✅ normalize key กันช่องว่างแฝง/NBSP จาก Excel
			key := normalizeKey(kitName)

			kitID, ok := kitNameToID[key]
			if !ok {
				counter++
				kitID = fmt.Sprintf("%d", counter) // ✅ URL-safe เหมือน doc
				kitNameToID[key] = kitID

				byID[kitID] = &KitDetail{
					KitID:    kitID,
					Category: strings.TrimSpace(category),
					KitName:  strings.TrimSpace(kitName),
					Page:     strings.TrimSpace(page),
					Order:    strings.TrimSpace(order),
					Special:  strings.TrimSpace(special),
					Lines:    make([]KitLine, 0, 32),
				}
			}

			k := byID[kitID]

			// เติม metadata ถ้าเจอค่าที่ไม่ว่าง
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

	// output
	out := make([]KitDetail, 0, len(byID))
	for _, v := range byID {
		sort.SliceStable(v.Lines, func(i, j int) bool {
			if v.Lines[i].Item == v.Lines[j].Item {
				return v.Lines[i].SubItem < v.Lines[j].SubItem
			}
			return v.Lines[i].Item < v.Lines[j].Item
		})
		out = append(out, *v)
	}

	// เรียงชุดตามชื่อ
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].KitName < out[j].KitName
	})

	return out, nil
}

func normalizeKey(s string) string {
	s = strings.ReplaceAll(s, "\u00a0", " ") // NBSP
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	// รวม whitespace ติด ๆ กันให้เป็นช่องเดียว
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
		return s
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

		_, okKit := m["ชื่อชุดเครื่องมือ"]
		_, okItem := m["รายการ"]
		if okKit && okItem {
			return r, kitCols{
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
