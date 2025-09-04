package segmenter

import (
	"bytes"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/vcaesar/cedar"
)

// Data Structures
type Segment struct {
	start int
	end   int
	token *Token
}
type Text []byte
type Token struct {
	text     []Text
	freq     float64
	pos      string
	distance float32
}
type Dictionary struct {
	trie        *cedar.Cedar
	maxTokenLen int
	Tokens      []Token
	totalFreq   float64
}
type Segmenter struct {
	Dict        *Dictionary
	DictSep     string
	AlphaNum    bool
	Alpha       bool
	Num         bool
	StopWordMap map[string]bool
}
type route struct {
	distance float32
	index    int
}

// Methods
func (s *Segment) Start() int    { return s.start }
func (s *Segment) End() int      { return s.end }
func (s *Segment) Token() *Token { return s.token }

func NewToken(text []Text, freq float64, pos string, totalFreq float64) Token {
	dist := float32(0.0)
	if freq > 0 {
		dist = float32(math.Log(totalFreq / freq))
	}
	return Token{text: text, freq: freq, pos: pos, distance: dist}
}
func (token *Token) Text() string { return textSliceToString(token.text) }

func NewDict() *Dictionary { return &Dictionary{trie: cedar.New()} }
func (dict *Dictionary) AddToken(token Token) {
	bytes := textSliceToBytes(token.text)
	_, err := dict.trie.Get(bytes)
	if err == nil {
		return
	}
	dict.trie.Insert(bytes, len(dict.Tokens))
	dict.Tokens = append(dict.Tokens, token)
	dict.totalFreq += token.freq
	if len(token.text) > dict.maxTokenLen {
		dict.maxTokenLen = len(token.text)
	}
}
func (dict *Dictionary) LookupTokens(words []Text, tokens []*Token) (num int) {
	var id int
	for _, word := range words {
		id, _ = dict.trie.Jump(word, id)
		val, err := dict.trie.Value(id)
		if err == nil {
			tokens[num] = &dict.Tokens[val]
			num++
		}
	}
	return
}
func (dict *Dictionary) Find(word []byte) (*Token, bool) {
	id, err := dict.trie.Jump(word, 0)
	if err != nil {
		return nil, false
	}
	val, err := dict.trie.Value(id)
	if err != nil {
		return nil, id != 0
	}
	return &dict.Tokens[val], true
}

func (seg *Segmenter) Cut(str string, hmm ...bool) []string {
	// HMM logic removed, always use cutDAGNoHMM equivalent
	return seg.cut(str)
}

// --- DAG ---

func isAlphanum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func (seg *Segmenter) cut(text string) []string {
	if text == "" {
		return []string{}
	}

	runes := []rune(text)
	// Pre-allocate with a heuristic: average word length of 3
	segments := make([]string, 0, len(runes)/3)

	if len(runes) == 0 {
		return []string{}
	}

	start := 0
	inAlphanum := isAlphanum(runes[0])

	for i := 1; i < len(runes); i++ {
		currentIsAlphanum := isAlphanum(runes[i])
		if currentIsAlphanum != inAlphanum {
			// State changed, process the segment before this point
			segmentRunes := runes[start:i]
			if inAlphanum {
				segments = append(segments, string(segmentRunes))
			} else {
				hanSegments := seg.cutDAG(segmentRunes)
				segments = append(segments, hanSegments...)
			}
			// Update state for the new segment
			start = i
			inAlphanum = currentIsAlphanum
		}
	}

	// Process the last segment
	lastSegmentRunes := runes[start:]
	if inAlphanum {
		segments = append(segments, string(lastSegmentRunes))
	} else {
		hanSegments := seg.cutDAG(lastSegmentRunes)
		segments = append(segments, hanSegments...)
	}

	return segments
}

// cutDAG performs segmentation on a slice of runes, assuming it's non-alphanumeric text.
func (seg *Segmenter) cutDAG(runes []rune) []string {
	if len(runes) == 0 {
		return []string{}
	}
	// Pre-allocate with a heuristic: average word length of 2
	segments := make([]string, 0, len(runes)/2)

	dag := seg.getDAG(runes)

	routeMap := make(map[int]route)
	routeMap[len(runes)] = route{distance: 0.0, index: 0}

	for idx := len(runes) - 1; idx >= 0; idx-- {
		routes := make([]route, 0)
		for _, i := range dag[idx] {
			word := runes[idx : i+1]
			token, ok := seg.Dict.Find(toLower([]byte(string(word))))
			if ok && token != nil {
				routes = append(routes, route{distance: token.distance + routeMap[i+1].distance, index: i})
			}
		}

		if len(routes) > 0 {
			sort.Slice(routes, func(i, j int) bool {
				return routes[i].distance < routes[j].distance
			})
			routeMap[idx] = routes[0]
		} else {
			// No word found in dictionary, treat as a single character
			routeMap[idx] = route{distance: 10000.0 + routeMap[idx+1].distance, index: idx}
		}
	}

	idx := 0
	for idx < len(runes) {
		ridx := routeMap[idx].index
		word := string(runes[idx : ridx+1])

		segments = append(segments, word)
		idx = ridx + 1
	}

	return segments
}

func (seg *Segmenter) getDAG(runes []rune) map[int][]int {
	dag := make(map[int][]int, len(runes))
	n := len(runes)
	for k := 0; k < n; k++ {
		paths := make([]int, 0)

		_, ok := seg.Dict.Find(toLower([]byte(string(runes[k : k+1]))))
		if ok {
			paths = append(paths, k)
		}

		for i := k + 1; i < n; i++ {
			_, ok := seg.Dict.Find(toLower([]byte(string(runes[k : i+1]))))
			if ok {
				paths = append(paths, i)
			}
		}

		if len(paths) == 0 {
			paths = append(paths, k)
		}
		dag[k] = paths
	}
	return dag
}

// --- Utility Functions ---
func SplitTextToWords(text []byte) []Text {
	output := make([]Text, 0, len(text)/3)
	current := 0
	inAlphanumeric := true
	for current < len(text) {
		r, size := utf8.DecodeRune(text[current:])
		isAlphanumeric := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if isAlphanumeric != inAlphanumeric {
			output = append(output, toLower(text[:current]))
			text = text[current:]
			current = 0
		}
		current += size
		inAlphanumeric = isAlphanumeric
	}
	output = append(output, toLower(text))
	return output
}

func toLower(text []byte) []byte {
	output := make([]byte, len(text))
	copy(output, text)
	for i, b := range output {
		if b >= 'A' && b <= 'Z' {
			output[i] = b + ('a' - 'A')
		}
	}
	return output
}

func textSliceByteLen(text []Text) (length int) {
	for _, t := range text {
		length += len(t)
	}
	return
}

func textSliceToString(text []Text) string {
	var b strings.Builder
	for _, t := range text {
		b.Write(t)
	}
	return b.String()
}

func textSliceToBytes(text []Text) []byte {
	var b bytes.Buffer
	for _, t := range text {
		b.Write(t)
	}
	return b.Bytes()
}

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}

// --- Trim ---
func (seg *Segmenter) Trim(s []string) (r []string) {
	for i := 0; i < len(s); i++ {
		si := FilterSymbol(s[i])
		if _, ok := seg.StopWordMap[si]; ok {
			si = ""
		}

		if si != "" {
			r = append(r, si)
		}
	}

	return
}

func FilterSymbol(text string) string {
	var builder strings.Builder
	builder.Grow(len(text)) // Pre-allocate memory for efficiency
	for _, value := range text {
		if !unicode.IsSymbol(value) &&
			!unicode.IsSpace(value) && !unicode.IsPunct(value) {
			builder.WriteRune(value)
		}
	}
	return builder.String()
}
