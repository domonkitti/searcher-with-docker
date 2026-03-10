package internal

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type Doc struct {
	ID    string         `json:"id"`
	Title string         `json:"title"`
	Text  string         `json:"-"`
	Meta  map[string]any `json:"meta"`
}

type SearchResult struct {
	ID    string         `json:"id"`
	Title string         `json:"title"`
	Score float64        `json:"score"`
	Meta  map[string]any `json:"meta"`
}

type EngineConfig struct {
	MinScore float64

	// char n-gram (ช่วยภาษาไทย/พิมพ์ไม่เต็ม)
	NMin int
	NMax int
	WChr float64

	// bm25 (คำตรง/keyword)
	WBm25 float64

	// boosts
	BoostTitleExact  float64
	BoostTitlePhrase float64
	BoostAllTokens   float64
}

type Engine struct {
	Docs []Doc
	ByID map[string]int

	// char vectors
	ChrDocVecs []map[string]float64

	// BM25 index
	DocTokens []map[string]int // tf per doc
	DocLen    []int
	DF        map[string]int
	N         int
	AvgDL     float64

	Cfg EngineConfig

	Syn *Synonyms
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MinScore: 0.0,

		NMin: 3,
		NMax: 6,
		WChr: 0.35,

		WBm25: 1.0,

		BoostTitleExact:  2.5,
		BoostTitlePhrase: 1.0,
		BoostAllTokens:   0.6,
	}
}

func NewEngine(docs []Doc, cfg EngineConfig, syn *Synonyms) *Engine {
	e := &Engine{
		Docs:       docs,
		ByID:       make(map[string]int, len(docs)),
		ChrDocVecs: make([]map[string]float64, 0, len(docs)),
		DocTokens:  make([]map[string]int, 0, len(docs)),
		DocLen:     make([]int, 0, len(docs)),
		DF:         map[string]int{},
		N:          len(docs),
		Cfg:        cfg,
	}

	totalDL := 0

	for i, d := range docs {
		e.ByID[d.ID] = i

		// ✅ ทำให้ index ไม่สนตัวเล็ก/ตัวใหญ่
		lowerText := strings.ToLower(d.Text)

		// ---------- char n-gram vector ----------
		e.ChrDocVecs = append(e.ChrDocVecs, toTF(charNgrams(lowerText, cfg.NMin, cfg.NMax)))

		// ---------- BM25 tokens ----------
		toks := tokenize(lowerText)

		tf := map[string]int{}
		seen := map[string]bool{}
		for _, t := range toks {
			tf[t]++
			if !seen[t] {
				e.DF[t]++
				seen[t] = true
			}
		}

		e.DocTokens = append(e.DocTokens, tf)
		e.DocLen = append(e.DocLen, len(toks))
		totalDL += len(toks)
	}

	if e.N > 0 {
		e.AvgDL = float64(totalDL) / float64(e.N)
	} else {
		e.AvgDL = 1
	}

	return e
}

func (e *Engine) GetByID(id string) (Doc, bool) {
	if i, ok := e.ByID[id]; ok {
		return e.Docs[i], true
	}
	return Doc{}, false
}

func (e *Engine) Search(query string, k int) []SearchResult {
	// ✅ query lowercase ตั้งแต่ต้น (ไม่สนตัวเล็ก/ตัวใหญ่)
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	qNorm := strings.ToLower(normalizeWS(q))

	// --- query tokens (BM25) ---
	qToks := tokenize(q)

	// weights for BM25 tokens (supports synonym)
	qWeights := map[string]float64{}
	for _, t := range qToks {
		if t == "" {
			continue
		}
		qWeights[t] = 1.0
	}

	// expand synonym tokens with lower weight
	if e.Syn != nil {
		for _, t := range qToks {
			exp := e.Syn.ExpandTokens(t)
			for tok, w := range exp {
				if tok == "" || w <= 0 {
					continue
				}
				if cur, ok := qWeights[tok]; !ok || w > cur {
					qWeights[tok] = w
				}
			}
		}
	}
	// --- query char vec (cosine) ---
	qChrVec := toTF(charNgrams(q, e.Cfg.NMin, e.Cfg.NMax))

	type pair struct {
		i int
		s float64
	}
	ps := make([]pair, 0, len(e.Docs))

	for i, d := range e.Docs {
		score := 0.0

		// (1) BM25 keyword score
		if e.Cfg.WBm25 > 0 {
			score += e.Cfg.WBm25 * e.bm25(i, qWeights)
		}

		// (2) Char cosine (ช่วยไทย/คำพิมพ์ไม่ครบ)
		if e.Cfg.WChr > 0 {
			score += e.Cfg.WChr * cosine(qChrVec, e.ChrDocVecs[i])
		}

		// (3) Boosts ให้ ranking สมเหตุสมผล
		titleLower := strings.ToLower(normalizeWS(d.Title))

		// exact title match
		if titleLower == qNorm {
			score += e.Cfg.BoostTitleExact
		}

		// phrase containment in title
		if strings.Contains(titleLower, qNorm) {
			score += e.Cfg.BoostTitlePhrase
		}

		// all tokens coverage in title
		if len(qToks) > 0 {
			allInTitle := true
			for _, t := range qToks {
				if t == "" {
					continue
				}
				if !strings.Contains(titleLower, t) {
					allInTitle = false
					break
				}
			}
			if allInTitle {
				score += e.Cfg.BoostAllTokens
			}
		}

		ps = append(ps, pair{i: i, s: score})
	}

	sort.Slice(ps, func(i, j int) bool { return ps[i].s > ps[j].s })

	out := make([]SearchResult, 0, k)
	for _, p := range ps {
		if p.s < e.Cfg.MinScore {
			break
		}
		d := e.Docs[p.i]
		out = append(out, SearchResult{
			ID:    d.ID,
			Title: d.Title,
			Score: math.Round(p.s*10000) / 10000,
			Meta:  d.Meta,
		})
		if len(out) >= k {
			break
		}
	}
	return out
}

/* ===================== BM25 ===================== */

func (e *Engine) bm25(docIdx int, qWeights map[string]float64) float64 {
	// BM25 params (ค่ามาตรฐาน)
	k1 := 1.2
	b := 0.75

	tf := e.DocTokens[docIdx]
	dl := float64(e.DocLen[docIdx])

	denomNorm := (1 - b) + b*(dl/e.AvgDL)

	score := 0.0
	for qt, wq := range qWeights {
		if qt == "" {
			continue
		}
		f := float64(tf[qt])
		if f == 0 {
			continue
		}
		df := float64(e.DF[qt])
		// idf แบบปลอดภัย
		idf := math.Log(1 + (float64(e.N)-df+0.5)/(df+0.5))

		score += wq * idf * (f * (k1 + 1)) / (f + k1*denomNorm)
	}
	return score
}

/* ===================== Tokenize ===================== */

var reToken = regexp.MustCompile(`(?:[A-Za-z0-9]+|[\p{Thai}]+)`)

// tokenize: ดึง token แบบง่าย (เน้นอังกฤษ/ตัวเลข)
// ไทยจะอาศัย char n-gram เป็นหลัก
func tokenize(s string) []string {
	s = strings.ToLower(s)
	raw := reToken.FindAllString(s, -1)
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

/* ===================== Char n-gram + Cosine ===================== */

func charNgrams(text string, nMin, nMax int) []string {
	text = strings.TrimSpace(normalizeWS(text))
	if text == "" {
		return nil
	}

	L := utf8.RuneCountInString(text)
	// query สั้น: Par -> 2..3 / 3..5
	if L <= 3 {
		nMin, nMax = 2, 3
	} else if L <= 6 {
		nMin, nMax = 3, 5
	}

	t := " " + text + " "
	rs := []rune(t)

	grams := make([]string, 0, len(rs))
	for n := nMin; n <= nMax; n++ {
		end := len(rs) - n + 1
		if end <= 0 {
			continue
		}
		for i := 0; i < end; i++ {
			grams = append(grams, string(rs[i:i+n]))
		}
	}
	return grams
}

func normalizeWS(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inWS := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v' {
			if !inWS {
				b.WriteRune(' ')
				inWS = true
			}
			continue
		}
		inWS = false
		b.WriteRune(r)
	}
	return b.String()
}

func toTF(grams []string) map[string]float64 {
	tf := map[string]float64{}
	for _, g := range grams {
		tf[g] += 1.0
	}
	return tf
}

func l2Norm(v map[string]float64) float64 {
	sum := 0.0
	for _, x := range v {
		sum += x * x
	}
	return math.Sqrt(sum)
}

func cosine(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	if len(a) > len(b) {
		a, b = b, a
	}
	dot := 0.0
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	na := l2Norm(a)
	nb := l2Norm(b)
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (na * nb)
}
