package internal

import (
	"encoding/json"
	"os"
	"strings"
)

// Synonyms stores token->expanded tokens with weights.
// Keys are normalized to lower-case (Thai has no case).
type Synonyms struct {
	// raw map from a canonical key -> list of synonym entries
	Map map[string][]SynEntry
	// reverse index: any term -> expansions (token weights)
	Rev map[string]map[string]float64
}

type SynEntry struct {
	T string  `json:"t"`
	W float64 `json:"w"`
}

// LoadSynonyms reads synonyms from json file.
// Format:
//
//	{
//	  "iphone": [{"t":"ไอโฟน","w":0.9},{"t":"โทรศัพท์","w":0.6}],
//	  "โดรน":   [{"t":"drone","w":0.9},{"t":"อากาศยานไร้คนขับ","w":0.7}]
//	}
func LoadSynonyms(path string) (*Synonyms, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string][]SynEntry
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	s := &Synonyms{
		Map: m,
		Rev: map[string]map[string]float64{},
	}

	// Build reverse map so user can type any variant and still expand.
	for k, list := range m {
		key := normSynKey(k)
		s.addReverse(key, key, 1.0)

		for _, e := range list {
			t := normSynKey(e.T)
			w := clamp01(e.W)
			if t == "" || w <= 0 {
				continue
			}

			// key -> t
			s.addReverse(key, t, w)
			// t -> key
			s.addReverse(t, key, w)

			// connect synonyms among themselves (t -> other)
			for _, e2 := range list {
				t2 := normSynKey(e2.T)
				if t2 == "" || t2 == t {
					continue
				}
				s.addReverse(t, t2, clamp01(minFloat(w, e2.W)))
			}
		}
	}
	return s, nil
}

func (s *Synonyms) addReverse(from, to string, w float64) {
	if from == "" || to == "" || w <= 0 {
		return
	}
	if _, ok := s.Rev[from]; !ok {
		s.Rev[from] = map[string]float64{}
	}
	if cur, ok := s.Rev[from][to]; !ok || w > cur {
		s.Rev[from][to] = w
	}
}

// ExpandTokens expands a token into token weights.
// If a synonym entry is a multi-word/phrase, it will be tokenized and each token gets the same weight.
func (s *Synonyms) ExpandTokens(token string) map[string]float64 {
	out := map[string]float64{}
	if s == nil {
		return out
	}
	t := normSynKey(token)
	exp, ok := s.Rev[t]
	if !ok {
		return out
	}
	for term, w := range exp {
		for _, tt := range tokenize(term) {
			tt = normSynKey(tt)
			if tt == "" {
				continue
			}
			if cur, ok := out[tt]; !ok || w > cur {
				out[tt] = w
			}
		}
	}
	return out
}

func normSynKey(s string) string {
	return strings.TrimSpace(strings.ToLower(normalizeWS(s)))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
