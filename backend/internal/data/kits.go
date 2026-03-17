package data

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

type KitLine struct {
	Item               string `json:"item"`
	SubItem            string `json:"subItem,omitempty"`
	Unit               string `json:"unit,omitempty"`
	LinkedItemSourceID string `json:"linkedItemSourceId,omitempty"`
	LinkedItemTitle    string `json:"linkedItemTitle,omitempty"`
}

type KitDetail struct {
	KitID    string    `json:"kitId"`
	SourceID string    `json:"sourceId"`
	Category string    `json:"category,omitempty"`
	KitName  string    `json:"kitName"`
	Page     string    `json:"page,omitempty"`
	Order    string    `json:"order,omitempty"`
	Special  string    `json:"special,omitempty"`
	Lines    []KitLine `json:"lines"`
}

type kitCols struct {
	source, cat, kit, item, sub, unit, page, order, special, linkedItemSource int
	ok                                                                        bool
}

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
	bySourceID := map[string]*kitAgg{}

	for _, sh := range sheets {
		rows, err := f.GetRows(sh)
		if err != nil || len(rows) == 0 {
			continue
		}

		hRowIdx, cols := findKitHeaderAndCols(rows)
		if !cols.ok {
			hRowIdx = 0
			cols = kitCols{source: 0, cat: 1, kit: 2, item: 3, sub: 4, unit: 5, page: 6, order: 7, special: 8, linkedItemSource: 9, ok: true}
		}

		var lastSourceID, lastCategory, lastKitName, lastOrder, lastSpecial string

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
			linkedItemSourceID := normalizeKey(get(cols.linkedItemSource))

			if sourceID != "" {
				lastSourceID = sourceID
			} else {
				sourceID = lastSourceID
			}
			if category != "" {
				lastCategory = category
			} else {
				category = lastCategory
			}
			if kitName != "" {
				lastKitName = kitName
			} else {
				kitName = lastKitName
			}
			if order != "" {
				lastOrder = order
			} else {
				order = lastOrder
			}
			if special != "" {
				lastSpecial = special
			} else {
				special = lastSpecial
			}

			if sourceID == "" || kitName == "" || strings.TrimSpace(item) == "" {
				continue
			}

			agg, ok := bySourceID[sourceID]
			if !ok {
				agg = &kitAgg{detail: KitDetail{SourceID: sourceID, Category: strings.TrimSpace(category), KitName: strings.TrimSpace(kitName), Order: strings.TrimSpace(order), Special: strings.TrimSpace(special), Lines: make([]KitLine, 0, 32)}, pages: make([]string, 0, 8)}
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

			k.Lines = append(k.Lines, KitLine{Item: strings.TrimSpace(item), SubItem: strings.TrimSpace(sub), Unit: strings.TrimSpace(unit), LinkedItemSourceID: linkedItemSourceID})
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
	if allNumeric && len(nums) == len(cleaned) {
		sort.Ints(nums)
		minPage := nums[0]
		maxPage := nums[len(nums)-1]
		if minPage == maxPage {
			return strconv.Itoa(minPage)
		}
		return strconv.Itoa(minPage) + " - " + strconv.Itoa(maxPage)
	}
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
			linkedIdx := getOr(m, "linkeditemsourceid", getOr(m, "itemsourceid", getOr(m, "รหัสรายการ", 9)))
			return r, kitCols{source: getOr(m, "id", getOr(m, "sourceid", getOr(m, "source_id", -1))), cat: getOr(m, "หมวด", -1), kit: getOr(m, "ชื่อชุดเครื่องมือ", -1), item: getOr(m, "รายการ", -1), sub: getOr(m, "รายการย่อย", getOr(m, "รายละเอียด", -1)), unit: getOr(m, "หน่วย", -1), page: getOr(m, "หน้า", -1), order: getOr(m, "ลำดับ", -1), special: getOr(m, "หมายเหตุ", -1), linkedItemSource: linkedIdx, ok: true}
		}
	}
	return 0, kitCols{}
}

func getOr(m map[string]int, key string, def int) int {
	if v, ok := m[key]; ok {
		return v
	}
	return def
}
