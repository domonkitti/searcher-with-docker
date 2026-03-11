package internal

import (
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

type ItemLinkRow struct {
	SourceID    string
	RealLink    string
	DisplayLine string
	LineNo      int
}

func LoadItemLinksFromExcel(path string) ([]ItemLinkRow, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	sheets := f.GetSheetList()
	out := make([]ItemLinkRow, 0, 1024)
	lineNo := 0

	for _, sh := range sheets {
		rows, err := f.GetRows(sh)
		if err != nil {
			continue
		}

		for i, row := range rows {
			if i == 0 && looksLikeItemLinksHeader(row) {
				continue
			}

			sourceID := strings.TrimSpace(getCol(row, 0))
			realLink := strings.TrimSpace(getCol(row, 1))
			displayLine := strings.TrimSpace(getCol(row, 2))

			if sourceID == "" && realLink == "" && displayLine == "" {
				continue
			}
			if sourceID == "" || realLink == "" || displayLine == "" {
				continue
			}

			lineNo++
			out = append(out, ItemLinkRow{
				SourceID:    sourceID,
				RealLink:    realLink,
				DisplayLine: displayLine,
				LineNo:      lineNo,
			})
		}
	}

	return out, nil
}

func looksLikeItemLinksHeader(row []string) bool {
	joined := strings.ToLower(strings.TrimSpace(strings.Join(row, " ")))
	if joined == "" {
		return false
	}
	keywords := []string{"id", "reallink", "displayline", "display", "url", "link"}
	for _, k := range keywords {
		if strings.Contains(joined, k) {
			return true
		}
	}
	return false
}
