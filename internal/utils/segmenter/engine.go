package segmenter

import (
	"bytes"
	"log"
	"math"
	"regexp"
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
type jumper struct {
	minDistance float32
	token       *Token
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
	useHMM := true
	if len(hmm) > 0 && !hmm[0] {
		useHMM = false
	}
	if useHMM {
		return seg.cutDAG(str)
	}
	return seg.cutDAGNoHMM(str)
}

func (seg *Segmenter) segmentWords(text []Text) []Segment {
	jumpers := make([]jumper, len(text))
	tokens := make([]*Token, seg.Dict.maxTokenLen)
	for current := 0; current < len(text); current++ {
		var baseDistance float32
		if current > 0 {
			baseDistance = jumpers[current-1].minDistance
		}
		tx := text[current:minInt(current+seg.Dict.maxTokenLen, len(text))]
		numTokens := seg.Dict.LookupTokens(tx, tokens)
		for iToken := 0; iToken < numTokens; iToken++ {
			location := current + len(tokens[iToken].text) - 1
			updateJumper(&jumpers[location], baseDistance, tokens[iToken])
		}
		if numTokens == 0 || len(tokens[0].text) > 1 {
			updateJumper(&jumpers[current], baseDistance,
				&Token{text: []Text{text[current]}, freq: 1, distance: 32, pos: "x"})
		}
	}
	numSeg := 0
	for index := len(text) - 1; index >= 0; {
		location := index - len(jumpers[index].token.text) + 1
		numSeg++
		index = location - 1
	}
	outputSegments := make([]Segment, numSeg)
	for index := len(text) - 1; index >= 0; {
		location := index - len(jumpers[index].token.text) + 1
		numSeg--
		outputSegments[numSeg].token = jumpers[index].token
		index = location - 1
	}
	bytePosition := 0
	for iSeg := 0; iSeg < len(outputSegments); iSeg++ {
		outputSegments[iSeg].start = bytePosition
		bytePosition += textSliceByteLen(outputSegments[iSeg].token.text)
		outputSegments[iSeg].end = bytePosition
	}
	return outputSegments
}

func updateJumper(jumper *jumper, baseDistance float32, token *Token) {
	newDistance := baseDistance + token.distance
	if jumper.minDistance == 0 || jumper.minDistance > newDistance {
		jumper.minDistance = newDistance
		jumper.token = token
	}
}

// --- DAG ---
var reEng = regexp.MustCompile(`[[:alnum:]]`)

func (seg *Segmenter) cutDAG(text string) []string {
	return seg.cut(text, true)
}

func (seg *Segmenter) cutDAGNoHMM(text string) []string {
	return seg.cut(text, false)
}

func (seg *Segmenter) cut(text string, hmm bool) []string {
	var (
		segments []string
		runes    = []rune(text)
	)

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
			routeMap[idx] = route{distance: 10000.0 + routeMap[idx+1].distance, index: idx}
		}
	}

	idx := 0
	for idx < len(runes) {
		ridx := routeMap[idx].index
		word := string(runes[idx : ridx+1])

		// Check if HMM should be used
		if hmm && len(word) == 1 {
			// Find the range of consecutive single-character words
			end := idx + 1
			for end < len(runes) {
				nextRidx := routeMap[end].index
				if nextRidx == end { // It's a single-character word
					end++
				} else {
					break
				}
			}

			// If there's a sequence of single-character words, run Viterbi on them
			if end > idx+1 {
				log.Printf("DAG found consecutive single characters. Invoking HMM for: '%s'", string(runes[idx:end]))
				hmmSeg := viterbi(runes[idx:end])
				if len(hmmSeg) > 0 {
					log.Printf("HMM Result: %v", hmmSeg)
					segments = append(segments, hmmSeg...)
					idx = end
					continue
				}
			}
		}

		log.Printf("DAG Result: '%s'", word)
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

// --- HMM ---
var (
	PrevStatus = map[rune][]rune{
		'B': {'E', 'S'},
		'M': {'M', 'B'},
		'E': {'B', 'M'},
		'S': {'S', 'E'},
	}
	startP = map[rune]float64{
		'B': -0.262686603532,
		'E': -3.14e+100,
		'M': -3.14e+100,
		'S': -1.46526333986,
	}
	transP = map[rune]map[rune]float64{
		'B': {'E': -0.653333431965, 'M': -0.765455118334},
		'E': {'B': -0.933489733133, 'S': -0.800739298181},
		'M': {'E': -0.68013402119, 'M': -0.733423709737},
		'S': {'B': -0.94485843981, 'S': -0.781857883616},
	}
)

func viterbi(runes []rune) []string {
	var (
		V      = make([]map[rune]float64, len(runes))
		path   = make(map[rune][]rune)
		result []string
	)

	for _, y := range []rune{'B', 'M', 'E', 'S'} {
		var (
			prob float64
			ok   bool
		)
		if prob, ok = ProbEmit[byte(y)][runes[0]]; !ok {
			prob = -3.14e+100
		}
		V[0] = make(map[rune]float64)
		V[0][y] = startP[y] + prob
		path[y] = []rune{y}
	}

	for t := 1; t < len(runes); t++ {
		newpath := make(map[rune][]rune)
		V[t] = make(map[rune]float64)

		for _, y := range []rune{'B', 'M', 'E', 'S'} {
			var (
				prob float64
				ok   bool
			)
			if prob, ok = ProbEmit[byte(y)][runes[t]]; !ok {
				prob = -3.14e+100
			}

			maxProb := -3.14e+100
			var maxState rune
			for _, y0 := range PrevStatus[y] {
				p := V[t-1][y0] + transP[y0][y] + prob
				if p > maxProb {
					maxProb = p
					maxState = y0
				}
			}
			V[t][y] = maxProb
			tmp := make([]rune, len(path[maxState]))
			copy(tmp, path[maxState])
			newpath[y] = append(tmp, y)
		}
		path = newpath
	}

	maxProb := -3.14e+100
	var maxState rune
	for _, y := range []rune{'E', 'S'} {
		if V[len(runes)-1][y] > maxProb {
			maxProb = V[len(runes)-1][y]
			maxState = y
		}
	}

	states := path[maxState]
	var (
		begin int
		word  string
	)
	for i, char := range states {
		switch char {
		case 'B':
			begin = i
		case 'E':
			word = string(runes[begin : i+1])
			result = append(result, word)

		case 'S':
			word = string(runes[i : i+1])
			result = append(result, word)

		}
	}

	return result
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

func FilterSymbol(text string) (new string) {
	for _, value := range text {
		if !unicode.IsSymbol(value) &&
			!unicode.IsSpace(value) && !unicode.IsPunct(value) {
			new += string(value)
		}
	}

	return
}
