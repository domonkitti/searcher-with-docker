package search

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// analyzerOutput splits analysis into exact tokens and softer Thai fragments.
//
// exact:
//   - coarse tokens used for precision (regex tokenization)
//
// soft:
//   - exact tokens + Thai sub-fragments used to bridge compound forms
//     e.g. "ซ่อมอาคาร" <-> "ค่าซ่อมแซมอาคาร"
//
// The goal is not perfect Thai segmentation. The goal is high-recall retrieval
// without requiring a manual synonym list for every phrase.
type analyzerOutput struct {
	Exact []string
	Soft  map[string]float64
}

func analyzeDocText(text string) analyzerOutput {
	text = strings.ToLower(normalizeWS(strings.TrimSpace(text)))
	if text == "" {
		return analyzerOutput{Soft: map[string]float64{}}
	}

	exact := tokenizeExact(text)
	soft := map[string]float64{}

	for _, t := range exact {
		addWeightedToken(soft, t, 1.0)
		if isThaiOnly(t) {
			for frag, w := range thaiFragmentWeights(t) {
				addWeightedToken(soft, frag, w)
			}
		}
	}

	return analyzerOutput{
		Exact: exact,
		Soft:  soft,
	}
}

func analyzeQueryText(text string, syn *Synonyms, translit *Transliterator) analyzerOutput {
	text = strings.ToLower(normalizeWS(strings.TrimSpace(text)))
	if text == "" {
		return analyzerOutput{Soft: map[string]float64{}}
	}

	exact := tokenizeExact(text)
	soft := map[string]float64{}

	for _, t := range exact {
		addWeightedToken(soft, t, 1.0)
		if isThaiOnly(t) {
			for frag, w := range thaiFragmentWeights(t) {
				addWeightedToken(soft, frag, w)
			}
		}
	}

	if syn != nil {
		for _, t := range exact {
			for tok, w := range syn.ExpandTokens(t) {
				addWeightedToken(soft, tok, w)
				if isThaiOnly(tok) {
					for frag, fw := range thaiFragmentWeights(tok) {
						addWeightedToken(soft, frag, w*fw)
					}
				}
			}
		}
	}

	if translit != nil {
		for _, t := range exact {
			for _, alt := range translit.EN2TH[t] {
				for _, tok := range tokenizeExact(alt) {
					addWeightedToken(soft, tok, 0.95)
					if isThaiOnly(tok) {
						for frag, fw := range thaiFragmentWeights(tok) {
							addWeightedToken(soft, frag, 0.95*fw)
						}
					}
				}
			}
			for _, alt := range translit.TH2EN[t] {
				for _, tok := range tokenizeExact(alt) {
					addWeightedToken(soft, tok, 0.95)
				}
			}
		}
	}

	return analyzerOutput{
		Exact: exact,
		Soft:  soft,
	}
}

func addWeightedToken(m map[string]float64, tok string, w float64) {
	tok = strings.TrimSpace(tok)
	if tok == "" || w <= 0 {
		return
	}
	if cur, ok := m[tok]; !ok || w > cur {
		m[tok] = w
	}
}

func tokenizeExact(s string) []string {
	s = strings.ToLower(s)
	raw := reToken.FindAllString(s, -1)
	out := make([]string, 0, len(raw))
	seen := map[string]bool{}
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func isThaiOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 0x0E00 || r > 0x0E7F {
			return false
		}
	}
	return true
}

func thaiFragmentWeights(s string) map[string]float64 {
	rs := []rune(strings.TrimSpace(s))
	n := len(rs)
	if n < 4 {
		return nil
	}

	out := map[string]float64{}

	// Prefix / suffix fragments are usually more meaningful than arbitrary middle grams.
	for size := 3; size <= minInt(8, n); size++ {
		prefix := string(rs[:size])
		suffix := string(rs[n-size:])
		out[prefix] = maxFloat(out[prefix], fragWeight(size, true))
		out[suffix] = maxFloat(out[suffix], fragWeight(size, true))
	}

	// Sliding fragments add recall for compounds but use lower weights to avoid noise.
	for size := 3; size <= minInt(6, n); size++ {
		for i := 0; i+size <= n; i++ {
			frag := string(rs[i : i+size])
			out[frag] = maxFloat(out[frag], fragWeight(size, i == 0 || i+size == n))
		}
	}

	return out
}

func fragWeight(size int, edge bool) float64 {
	base := map[int]float64{
		3: 0.22,
		4: 0.40,
		5: 0.54,
		6: 0.64,
		7: 0.72,
		8: 0.78,
	}[size]
	if edge {
		base += 0.10
	}
	if base > 0.88 {
		return 0.88
	}
	return base
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func softTermsSorted(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		wi, wj := m[keys[i]], m[keys[j]]
		if wi == wj {
			li, lj := utf8.RuneCountInString(keys[i]), utf8.RuneCountInString(keys[j])
			if li == lj {
				return keys[i] < keys[j]
			}
			return li > lj
		}
		return wi > wj
	})
	return keys
}
