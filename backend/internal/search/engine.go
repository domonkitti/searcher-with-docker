package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
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

type Suggestion struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

type EngineConfig struct {
	MinScore float64

	// char n-gram
	NMin int
	NMax int
	WChr float64

	// exact token BM25
	WBm25 float64

	// soft Thai fragment BM25
	WBm25Soft float64

	// dense vector / semantic
	WVec float64

	// boosts
	BoostTitleExact  float64
	BoostTitlePhrase float64
	BoostAllTokens   float64

	// suggest
	SuggestPrefixBoost float64
	SuggestInfixBoost  float64
}

type Embedder interface {
	Embed(text string) ([]float64, error)
}

type HTTPEmbedder struct {
	BaseURL string
	Client  *http.Client
}

type embedRequest struct {
	Text string `json:"text"`
}

type embedResponse struct {
	Vector []float64 `json:"vector"`
}

type Engine struct {
	Docs []Doc
	ByID map[string]int

	// char vectors
	ChrDocVecs []map[string]float64

	// dense vectors
	DenseDocVecs [][]float64
	Embedder     Embedder

	// transliteration dictionary
	Translit *Transliterator

	// exact BM25 index
	DocExactTokens []map[string]int
	DocExactLen    []int
	ExactDF        map[string]int
	AvgExactDL     float64

	// soft BM25 index (Thai fragments + exact tokens)
	DocSoftTokens []map[string]float64
	DocSoftLen    []float64
	SoftDF        map[string]int
	AvgSoftDL     float64

	N int

	Cfg EngineConfig
	Syn *Synonyms
}

func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		MinScore: 0.0,

		NMin: 3,
		NMax: 6,
		WChr: 0.15,

		WBm25:     1.35,
		WBm25Soft: 0.90,
		WVec:      0.45,

		BoostTitleExact:  1.8,
		BoostTitlePhrase: 0.9,
		BoostAllTokens:   0.8,

		SuggestPrefixBoost: 5.0,
		SuggestInfixBoost:  1.5,
	}
}

// NewHTTPEmbedderFromEnv reads embedder configuration from env.
// If EMBEDDER_URL is empty, it returns nil so the engine can still work
// in lexical-only mode.
func NewHTTPEmbedderFromEnv() *HTTPEmbedder {
	baseURL := strings.TrimSpace(os.Getenv("EMBEDDER_URL"))
	if baseURL == "" {
		return nil
	}

	timeoutMs := getEnvInt("EMBEDDER_TIMEOUT_MS", 10000)

	return &HTTPEmbedder{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
	}
}

func (h *HTTPEmbedder) Embed(text string) ([]float64, error) {
	if h == nil {
		return nil, nil
	}

	payload, err := json.Marshal(embedRequest{Text: text})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, h.BaseURL+"/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embed service returned status %d", resp.StatusCode)
	}

	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	if len(out.Vector) == 0 {
		return nil, fmt.Errorf("embed service returned empty vector")
	}

	return out.Vector, nil
}

func NewEngine(docs []Doc, cfg EngineConfig, syn *Synonyms, embedder Embedder) *Engine {
	var translit *Transliterator
	translitPath := strings.TrimSpace(os.Getenv("TRANSLIT_TSV"))
	if translitPath != "" {
		if tr, err := LoadTransliteratorTSV(translitPath); err == nil {
			translit = tr
		}
	}

	e := &Engine{
		Docs:           docs,
		ByID:           make(map[string]int, len(docs)),
		ChrDocVecs:     make([]map[string]float64, 0, len(docs)),
		DenseDocVecs:   make([][]float64, 0, len(docs)),
		DocExactTokens: make([]map[string]int, 0, len(docs)),
		DocExactLen:    make([]int, 0, len(docs)),
		ExactDF:        map[string]int{},
		DocSoftTokens:  make([]map[string]float64, 0, len(docs)),
		DocSoftLen:     make([]float64, 0, len(docs)),
		SoftDF:         map[string]int{},
		N:              len(docs),
		Cfg:            cfg,
		Syn:            syn,
		Embedder:       embedder,
		Translit:       translit,
	}

	totalExactDL := 0
	totalSoftDL := 0.0

	for i, d := range docs {
		e.ByID[d.ID] = i
		lowerText := strings.ToLower(d.Text)

		// 1) char n-gram vector
		e.ChrDocVecs = append(e.ChrDocVecs, toTF(charNgrams(lowerText, cfg.NMin, cfg.NMax)))

		// 2) analyzed tokens
		az := analyzeDocText(lowerText)

		exactTF := map[string]int{}
		exactSeen := map[string]bool{}
		for _, t := range az.Exact {
			exactTF[t]++
			if !exactSeen[t] {
				e.ExactDF[t]++
				exactSeen[t] = true
			}
		}
		e.DocExactTokens = append(e.DocExactTokens, exactTF)
		e.DocExactLen = append(e.DocExactLen, len(az.Exact))
		totalExactDL += len(az.Exact)

		softTF := map[string]float64{}
		softSeen := map[string]bool{}
		for t, w := range az.Soft {
			softTF[t] = w
			if !softSeen[t] {
				e.SoftDF[t]++
				softSeen[t] = true
			}
		}
		e.DocSoftTokens = append(e.DocSoftTokens, softTF)
		softLen := sumFloatMap(softTF)
		e.DocSoftLen = append(e.DocSoftLen, softLen)
		totalSoftDL += softLen

		// 3) dense vector
		if embedder != nil {
			vec, err := embedder.Embed(lowerText)
			if err != nil {
				vec = nil
			}
			e.DenseDocVecs = append(e.DenseDocVecs, vec)
		} else {
			e.DenseDocVecs = append(e.DenseDocVecs, nil)
		}
	}

	if e.N > 0 {
		e.AvgExactDL = float64(totalExactDL) / float64(e.N)
		e.AvgSoftDL = totalSoftDL / float64(e.N)
	} else {
		e.AvgExactDL = 1
		e.AvgSoftDL = 1
	}
	if e.AvgExactDL <= 0 {
		e.AvgExactDL = 1
	}
	if e.AvgSoftDL <= 0 {
		e.AvgSoftDL = 1
	}

	return e
}

func (e *Engine) GetByID(id string) (Doc, bool) {
	if i, ok := e.ByID[id]; ok {
		return e.Docs[i], true
	}
	return Doc{}, false
}
// ---------------ส่วนของการ เซฺิร์ชและวิเคราะห์ข้อความ----------------

func (e *Engine) Search(query string, k int) []SearchResult {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	qNorm := strings.ToLower(normalizeWS(q))
	az := analyzeQueryText(q, e.Syn, e.Translit)
	fmt.Printf("QUERY=%q exact=%v soft=%v\n", q, az.Exact, az.Soft)

	var qDense []float64
	if e.Embedder != nil {
		vec, err := e.Embedder.Embed(q)
		if err == nil {
			qDense = vec
		}
	}

	qChrVec := toTF(charNgrams(q, e.Cfg.NMin, e.Cfg.NMax))

	wBm25 := e.Cfg.WBm25
	wBm25Soft := e.Cfg.WBm25Soft
	wChr := e.Cfg.WChr
	wVec := e.Cfg.WVec

	// ลด soft/vec ลงบ้างสำหรับ query สั้น แต่ไม่กดแรงเกิน
	if len(az.Exact) <= 2 {
		wBm25Soft *= 0.75
		wChr *= 0.85
		wVec *= 0.80
	}

	type pair struct {
		i int
		s float64
	}
	ps := make([]pair, 0, len(e.Docs))

	for i, d := range e.Docs {
		score := 0.0

		if wBm25 > 0 {
			score += wBm25 * e.bm25Exact(i, az.Exact)
		}
		if wBm25Soft > 0 {
			score += wBm25Soft * e.bm25Soft(i, az.Soft)
		}
		if wChr > 0 {
			score += wChr * cosine(qChrVec, e.ChrDocVecs[i])
		}
		if wVec > 0 && len(qDense) > 0 && len(e.DenseDocVecs[i]) > 0 {
			score += wVec * cosineDense(qDense, e.DenseDocVecs[i])
		}

		titleLower := strings.ToLower(normalizeWS(d.Title))

		// exact phrase ช่วยได้ แต่ไม่แรงเวอร์
		if titleLower == qNorm {
			score += e.Cfg.BoostTitleExact
		}
		if strings.Contains(titleLower, qNorm) {
			score += e.Cfg.BoostTitlePhrase
		}

		// ดึงนิสัย autocomplete เดิมกลับมาแบบเบา ๆ
		if strings.HasPrefix(titleLower, qNorm) {
			score += 2.5
		}

		for tok, w := range az.Soft {
			if tok == "" {
				continue
			}
			if strings.Contains(titleLower, tok) {
				score += 0.20 * w
			}
		}

		// exact token match ใช้เป็น bonus อย่างเดียว
		// ห้ามใช้เป็น penalty เพราะจะฆ่าพวก "ตัดต้นไม้"
		if len(az.Exact) > 0 {
			matchedTerms := countMatchedExactTokens(titleLower, az.Exact)
			allInTitle := matchedTerms == len(az.Exact)

			if allInTitle {
				score += e.Cfg.BoostAllTokens
				if len(az.Exact) >= 2 {
					score += 0.8
					score += tokenSpanBoost(titleLower, az.Exact)
				}
			}
		}

		ps = append(ps, pair{i: i, s: score})
	}

	sort.Slice(ps, func(i, j int) bool {
		if ps[i].s == ps[j].s {
			return e.Docs[ps[i].i].Title < e.Docs[ps[j].i].Title
		}
		return ps[i].s > ps[j].s
	})

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
// search helper
func countMatchedExactTokens(title string, toks []string) int {
	matched := 0
	for _, t := range toks {
		if t == "" {
			continue
		}
		if strings.Contains(title, t) {
			matched++
		}
	}
	return matched
}

func tokenSpanBoost(title string, toks []string) float64 {
	pos := make([]int, 0, len(toks))
	for _, t := range toks {
		if t == "" {
			continue
		}
		i := strings.Index(title, t)
		if i < 0 {
			return 0
		}
		pos = append(pos, i)
	}

	if len(pos) < 2 {
		return 0
	}

	minPos, maxPos := pos[0], pos[0]
	for _, p := range pos[1:] {
		if p < minPos {
			minPos = p
		}
		if p > maxPos {
			maxPos = p
		}
	}

	span := maxPos - minPos
	switch {
	case span <= 8:
		return 1.2
	case span <= 16:
		return 0.7
	case span <= 28:
		return 0.3
	default:
		return 0
	}
}

func isThaiCombiningMark(r rune) bool {
	switch {
	case r == 0x0E31:
		return true
	case r >= 0x0E34 && r <= 0x0E3A:
		return true
	case r >= 0x0E47 && r <= 0x0E4E:
		return true
	default:
		return false
	}
}

func looksUsableThaiTerm(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	rs := []rune(s)
	if len(rs) < 3 {
		return false
	}
	if isThaiCombiningMark(rs[0]) {
		return false
	}
	return true
}

func buildCoverageTerms(qNorm string, exact []string, soft map[string]float64) []string {
	seen := map[string]bool{}
	out := make([]string, 0, 6)

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}

	// exact เดิมใส่ก่อน
	for _, t := range exact {
		add(t)
	}

	// ถ้า exact มีคำเดียว และ query ไม่มีเว้นวรรค
	// ให้ดึงคำ prefix/suffix ที่ดูเป็นคำไทยจริงจาก soft มาเพิ่ม
	if len(exact) == 1 && !strings.Contains(qNorm, " ") {
		var bestPrefix string
		var bestSuffix string

		for tok, w := range soft {
			if w < 0.30 {
				continue
			}
			if !looksUsableThaiTerm(tok) {
				continue
			}

			if strings.HasPrefix(qNorm, tok) {
				if bestPrefix == "" || len([]rune(tok)) < len([]rune(bestPrefix)) {
					bestPrefix = tok
				}
			}
			if strings.HasSuffix(qNorm, tok) {
				if bestSuffix == "" || len([]rune(tok)) < len([]rune(bestSuffix)) {
					bestSuffix = tok
				}
			}
		}

		add(bestPrefix)
		add(bestSuffix)
	}

	return out
}

func countMatchedTerms(title string, toks []string) int {
	matched := 0
	for _, t := range toks {
		if t == "" {
			continue
		}
		if strings.Contains(title, t) {
			matched++
		}
	}
	return matched
}

func tokenSpanBoost(title string, toks []string) float64 {
	pos := make([]int, 0, len(toks))
	for _, t := range toks {
		if t == "" {
			continue
		}
		i := strings.Index(title, t)
		if i < 0 {
			return 0
		}
		pos = append(pos, i)
	}

	if len(pos) < 2 {
		return 0
	}

	minPos, maxPos := pos[0], pos[0]
	for _, p := range pos[1:] {
		if p < minPos {
			minPos = p
		}
		if p > maxPos {
			maxPos = p
		}
	}

	span := maxPos - minPos
	switch {
	case span <= 8:
		return 1.2
	case span <= 16:
		return 0.7
	case span <= 28:
		return 0.3
	default:
		return 0
	}
}
func (e *Engine) Suggest(query string, k int) []Suggestion {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	if k <= 0 {
		k = 8
	}

	results := e.Search(q, k*3)

	seen := map[string]bool{}
	out := make([]Suggestion, 0, k)

	for _, r := range results {
		title := strings.TrimSpace(r.Title)
		if title == "" || seen[title] {
			continue
		}
		seen[title] = true

		out = append(out, Suggestion{
			Text:  title,
			Score: r.Score,
		})

		if len(out) >= k {
			break
		}
	}

	return out
}

/* ===================== BM25 ===================== */

func (e *Engine) bm25Exact(docIdx int, qTokens []string) float64 {
	k1 := 1.2
	b := 0.75
	if len(qTokens) == 0 {
		return 0
	}

	tf := e.DocExactTokens[docIdx]
	dl := float64(e.DocExactLen[docIdx])
	denomNorm := (1 - b) + b*(dl/e.AvgExactDL)

	score := 0.0
	seen := map[string]bool{}
	for _, qt := range qTokens {
		if qt == "" || seen[qt] {
			continue
		}
		seen[qt] = true
		f := float64(tf[qt])
		if f == 0 {
			continue
		}
		df := float64(e.ExactDF[qt])
		idf := math.Log(1 + (float64(e.N)-df+0.5)/(df+0.5))
		score += idf * (f * (k1 + 1)) / (f + k1*denomNorm)
	}
	return score
}

func (e *Engine) bm25Soft(docIdx int, qWeights map[string]float64) float64 {
	k1 := 1.2
	b := 0.75
	if len(qWeights) == 0 {
		return 0
	}

	tf := e.DocSoftTokens[docIdx]
	dl := e.DocSoftLen[docIdx]
	denomNorm := (1 - b) + b*(dl/e.AvgSoftDL)

	score := 0.0
	for qt, wq := range qWeights {
		if qt == "" || wq <= 0 {
			continue
		}
		f := tf[qt]
		if f == 0 {
			continue
		}
		df := float64(e.SoftDF[qt])
		idf := math.Log(1 + (float64(e.N)-df+0.5)/(df+0.5))
		score += wq * idf * (f * (k1 + 1)) / (f + k1*denomNorm)
	}
	return score
}

/* ===================== Tokenize ===================== */

var reToken = regexp.MustCompile(`(?:[A-Za-z0-9]+|[\p{Thai}]+)`)

// tokenize is preserved for legacy helpers / synonym expansion callers.
func tokenize(s string) []string {
	return tokenizeExact(s)
}

/* ===================== Char n-gram ===================== */

func charNgrams(text string, nMin, nMax int) []string {
	text = strings.TrimSpace(normalizeWS(text))
	if text == "" {
		return nil
	}

	L := utf8.RuneCountInString(text)

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

func cosineDense(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}

	var dot float64
	var na float64
	var nb float64

	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}

	if na == 0 || nb == 0 {
		return 0
	}

	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

/* ===================== Helpers ===================== */

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func sumFloatMap(m map[string]float64) float64 {
	total := 0.0
	for _, v := range m {
		total += v
	}
	return total
}
