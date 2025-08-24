package segmenter

import (
	"bufio"
	_ "embed"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
)

//go:embed dict/simplified.txt
var simplifiedDict []byte

//go:embed dict/stop_word.txt
var stopWords []byte

//go:embed hmm/prob_emit.go
var probEmit []byte

var ProbEmit = make(map[byte]map[rune]float64)
var seg Segmenter

func init() {
	log.Println("Loading embedded dictionary and stop words...")

	loadHmmEmit()

	seg = Segmenter{}
	seg.Dict = NewDict()
	seg.DictSep = " "

	totalFreq := loadDictFromString(&seg, string(simplifiedDict))
	recalculateTokenDistances(seg.Dict, totalFreq)
	loadStopFromString(&seg, string(stopWords))

	log.Println("Custom dictionary and stop words loaded successfully.")
}

func loadHmmEmit() {
	log.Println("Loading and parsing HMM model...")

	re := regexp.MustCompile(`'\\u([0-9a-fA-F]{4})':\s*(-[\d\.]+)`)
	scanner := bufio.NewScanner(strings.NewReader(string(probEmit)))
	var currentState byte

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "probEmit['B']") {
			currentState = 'B'
			ProbEmit[currentState] = make(map[rune]float64)
		} else if strings.Contains(line, "probEmit['M']") {
			currentState = 'M'
			ProbEmit[currentState] = make(map[rune]float64)
		} else if strings.Contains(line, "probEmit['E']") {
			currentState = 'E'
			ProbEmit[currentState] = make(map[rune]float64)
		} else if strings.Contains(line, "probEmit['S']") {
			currentState = 'S'
			ProbEmit[currentState] = make(map[rune]float64)
		}

		if currentState != 0 {
			pairs := re.FindAllStringSubmatch(line, -1)
			for _, pair := range pairs {
				runeHex, _ := strconv.ParseInt(pair[1], 16, 32)
				prob, _ := strconv.ParseFloat(pair[2], 64)
				ProbEmit[currentState][rune(runeHex)] = prob
			}
		}
	}
	log.Println("HMM model loaded successfully.")
}

func loadDictFromString(seg *Segmenter, dict string) float64 {
	scanner := bufio.NewScanner(strings.NewReader(dict))
	var totalFreq float64
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, seg.DictSep)
		if len(fields) < 1 {
			continue
		}
		word := fields[0]
		if word == "" {
			continue
		}
		freq := 1.0
		if len(fields) > 1 {
			if f, err := strconv.ParseFloat(fields[1], 64); err == nil {
				freq = f
			}
		}
		totalFreq += freq
		pos := ""
		if len(fields) > 2 {
			pos = fields[2]
		}

		tokenText := SplitTextToWords([]byte(word))
		token := NewToken(tokenText, freq, pos, 1.0) // Temp distance
		seg.Dict.AddToken(token)
	}
	return totalFreq
}

func recalculateTokenDistances(dict *Dictionary, totalFreq float64) {
	for i, token := range dict.Tokens {
		if token.freq > 0 {
			dict.Tokens[i].distance = float32(math.Log(totalFreq / token.freq))
		}
	}
}

func loadStopFromString(seg *Segmenter, stopWords string) {
	seg.StopWordMap = make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(stopWords))
	for scanner.Scan() {
		word := scanner.Text()
		if word != "" {
			seg.StopWordMap[word] = true
		}
	}
}

func FreeJieba() {
	// No-op
}

func SegmentTextForIndex(text string) string {
	log.Printf("Segmenting for index. Input: '%s'", text)
	words := seg.Cut(text, true)
	log.Printf("After seg.Cut: %v", words)
	trimmedWords := seg.Trim(words)
	log.Printf("After seg.Trim: %v", trimmedWords)
	return strings.Join(trimmedWords, " ")
}

func SegmentTextForQuery(query string) string {
	log.Printf("Segmenting for query. Input: '%s'", query)
	words := seg.Cut(query, true)
	log.Printf("After seg.Cut: %v", words)
	trimmedWords := seg.Trim(words)
	log.Printf("After seg.Trim: %v", trimmedWords)
	return strings.Join(trimmedWords, " ")
}
