package segmenter

import (
	"bufio"
	"math"
	"strconv"
	"strings"
)

var seg Segmenter

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

func SegmentTextForIndex(text string) string {
	words := seg.Cut(text)
	trimmedWords := seg.Trim(words)
	return strings.Join(trimmedWords, " ")
}

func SegmentTextForQuery(query string) string {
	words := seg.Cut(query)
	trimmedWords := seg.Trim(words)
	return strings.Join(trimmedWords, " ")
}
