package search

import (
	"bufio"
	"os"
	"strings"
)

type Transliterator struct {
	TH2EN map[string][]string
	EN2TH map[string][]string
}

func LoadTransliteratorTSV(path string) (*Transliterator, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tr := &Transliterator{
		TH2EN: map[string][]string{},
		EN2TH: map[string][]string{},
	}

	sc := bufio.NewScanner(f)
	first := true

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		// ข้าม header เช่น: th	en	check
		if first {
			first = false
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "th\t") || lower == "th" {
				continue
			}
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		th := normalizeTranslitToken(parts[0])
		en := normalizeTranslitToken(parts[1])

		if th == "" || en == "" {
			continue
		}

		tr.TH2EN[th] = appendUniqueString(tr.TH2EN[th], en)
		tr.EN2TH[en] = appendUniqueString(tr.EN2TH[en], th)
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}

	return tr, nil
}

func normalizeTranslitToken(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = normalizeWS(s)
	return s
}

func appendUniqueString(xs []string, v string) []string {
	for _, x := range xs {
		if x == v {
			return xs
		}
	}
	return append(xs, v)
}
