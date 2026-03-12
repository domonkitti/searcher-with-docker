package internal

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

type EngineConfig struct {
	MinScore float64

	// char n-gram
	NMin int
	NMax int
	WChr float64

	// bm25
	WBm25 float64

	// dense vector / semantic
	WVec float64

	// boosts
	BoostTitleExact  float64
	BoostTitlePhrase float64
	BoostAllTokens   float64
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

	// BM25 index
	DocTokens []map[string]int
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
		WChr: 0.25,

		WBm25: 1.0,
		WVec:  1.2,

		BoostTitleExact:  2.0,
		BoostTitlePhrase: 0.8,
		BoostAllTokens:   0.5,
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
	e := &Engine{
		Docs:         docs,
		ByID:         make(map[string]int, len(docs)),
		ChrDocVecs:   make([]map[string]float64, 0, len(docs)),
		DenseDocVecs: make([][]float64, 0, len(docs)),
		DocTokens:    make([]map[string]int, 0, len(docs)),
		DocLen:       make([]int, 0, len(docs)),
		DF:           map[string]int{},
		N:            len(docs),
		Cfg:          cfg,
		Syn:          syn,
		Embedder:     embedder,
	}

	totalDL := 0

	for i, d := range docs {
		e.ByID[d.ID] = i

		lowerText := strings.ToLower(d.Text)

		// 1) char n-gram vector
		e.ChrDocVecs = append(e.ChrDocVecs, toTF(charNgrams(lowerText, cfg.NMin, cfg.NMax)))

		// 2) BM25 tokens
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
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	qNorm := strings.ToLower(normalizeWS(q))

	// BM25 query tokens
	qToks := tokenize(q)

	// token weights (supports synonym expansion)
	qWeights := map[string]float64{}
	for _, t := range qToks {
		if t == "" {
			continue
		}
		qWeights[t] = 1.0
	}

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

	// char vector for fuzzy-ish matching
	qChrVec := toTF(charNgrams(q, e.Cfg.NMin, e.Cfg.NMax))

	// dense query vector for semantic search
	var qDense []float64
	if e.Embedder != nil {
		vec, err := e.Embedder.Embed(q)
		if err == nil {
			qDense = vec
		}
	}

	type pair struct {
		i int
		s float64
	}

	ps := make([]pair, 0, len(e.Docs))

	for i, d := range e.Docs {
		score := 0.0

		// 1) BM25
		if e.Cfg.WBm25 > 0 {
			score += e.Cfg.WBm25 * e.bm25(i, qWeights)
		}

		// 2) char cosine
		if e.Cfg.WChr > 0 {
			score += e.Cfg.WChr * cosine(qChrVec, e.ChrDocVecs[i])
		}

		// 3) dense vector cosine
		if e.Cfg.WVec > 0 && len(qDense) > 0 && len(e.DenseDocVecs[i]) > 0 {
			score += e.Cfg.WVec * cosineDense(qDense, e.DenseDocVecs[i])
		}

		// 4) boosts
		titleLower := strings.ToLower(normalizeWS(d.Title))

		if titleLower == qNorm {
			score += e.Cfg.BoostTitleExact
		}

		if strings.Contains(titleLower, qNorm) {
			score += e.Cfg.BoostTitlePhrase
		}

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
		idf := math.Log(1 + (float64(e.N)-df+0.5)/(df+0.5))

		score += wq * idf * (f * (k1 + 1)) / (f + k1*denomNorm)
	}

	return score
}

/* ===================== Tokenize ===================== */

var reToken = regexp.MustCompile(`(?:[A-Za-z0-9]+|[\p{Thai}]+)`)

// ไทยยังไม่ได้ตัดคำละเอียด ใช้ char n-gram ช่วยเป็นหลัก
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