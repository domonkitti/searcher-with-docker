package data

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type KitLine struct {
	Item    string `json:"item"`
	SubItem string `json:"subItem,omitempty"`
	Unit    string `json:"unit,omitempty"`
}

type KitDetail struct {
	KitID    string    `json:"kitId"` // internal DB id
	SourceID string    `json:"sourceId"`
	Category string    `json:"category,omitempty"`
	KitName  string    `json:"kitName"`
	Page     string    `json:"page,omitempty"`
	Order    string    `json:"order,omitempty"`
	Special  string    `json:"special,omitempty"`
	Lines    []KitLine `json:"lines"`
}

type kitCols struct {
	source, cat, kit, item, sub, unit, page, order, special int
	ok                                                      bool
}

// ใช้เก็บหน้าทั้งหมดของแต่ละ KIT ชั่วคราวตอน import
type kitAgg struct {
	detail KitDetail
	pages  []string
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
	bySourceID := map[string]*kitAgg{}

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
			page := strings.TrimSpace(get(cols.page))
			order := get(cols.order)
			special := get(cols.special)

			if sourceID == "" || kitName == "" || item == "" {
				continue
			}

			agg, ok := bySourceID[sourceID]
			if !ok {
				agg = &kitAgg{
					detail: KitDetail{
						SourceID: sourceID,
						Category: strings.TrimSpace(category),
						KitName:  strings.TrimSpace(kitName),
						Order:    strings.TrimSpace(order),
						Special:  strings.TrimSpace(special),
						Lines:    make([]KitLine, 0, 32),
					},
					pages: make([]string, 0, 8),
				}
				bySourceID[sourceID] = agg
			}

			k := &agg.detail

			if k.Category == "" && strings.TrimSpace(category) != "" {
				k.Category = strings.TrimSpace(category)
			}
			if k.Order == "" && strings.TrimSpace(order) != "" {
				k.Order = strings.TrimSpace(order)
			}
			if k.Special == "" && strings.TrimSpace(special) != "" {
				k.Special = strings.TrimSpace(special)
			}

			if page != "" {
				agg.pages = append(agg.pages, page)
			}

			k.Lines = append(k.Lines, KitLine{
				Item:    strings.TrimSpace(item),
				SubItem: strings.TrimSpace(sub),
				Unit:    strings.TrimSpace(unit),
			})
		}
	}

	out := make([]KitDetail, 0, len(bySourceID))
	for _, agg := range bySourceID {
		agg.detail.Page = summarizePageRange(agg.pages)

		sort.SliceStable(agg.detail.Lines, func(i, j int) bool {
			if agg.detail.Lines[i].Item == agg.detail.Lines[j].Item {
				return agg.detail.Lines[i].SubItem < agg.detail.Lines[j].SubItem
			}
			return agg.detail.Lines[i].Item < agg.detail.Lines[j].Item
		})

		out = append(out, agg.detail)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].KitName < out[j].KitName
	})

	return out, nil
}

func summarizePageRange(pages []string) string {
	if len(pages) == 0 {
		return ""
	}

	seen := map[string]bool{}
	cleaned := make([]string, 0, len(pages))
	nums := make([]int, 0, len(pages))
	allNumeric := true

	for _, p := range pages {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		cleaned = append(cleaned, p)

		n, err := strconv.Atoi(p)
		if err != nil {
			allNumeric = false
			continue
		}
		nums = append(nums, n)
	}

	if len(cleaned) == 0 {
		return ""
	}

	// ถ้าเป็นตัวเลขทั้งหมด ให้สรุปเป็น min-max
	if allNumeric && len(nums) == len(cleaned) {
		sort.Ints(nums)
		minPage := nums[0]
		maxPage := nums[len(nums)-1]

		if minPage == maxPage {
			return strconv.Itoa(minPage)
		}
		return strconv.Itoa(minPage) + " - " + strconv.Itoa(maxPage)
	}

	// ถ้าไม่ใช่ตัวเลขล้วน เอา fallback เป็นค่าที่เจอเรียงตามลำดับ
	if len(cleaned) == 1 {
		return cleaned[0]
	}
	return cleaned[0] + " - " + cleaned[len(cleaned)-1]
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

	return 0, kitCols{}
}

func getOr(m map[string]int, key string, fallback int) int {
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}