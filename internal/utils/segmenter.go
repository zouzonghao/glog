package utils

import (
	_ "embed"
	"log"
	"strings"

	"github.com/go-ego/gse"
)

//go:embed dict/simplified.txt
var simplifiedDict string

//go:embed dict/stop_word.txt
var stopWords string

var seg gse.Segmenter

func init() {
	log.Println("Loading embedded dictionary and stop words...")
	var err error
	// Use gse.NewEmbed to load the main dictionary from the embedded string,
	// while also enabling the default English tokenizer.
	seg, err = gse.NewEmbed("zh,"+simplifiedDict, "en")
	if err != nil {
		log.Fatalf("Failed to create segmenter with embedded dictionary: %v", err)
	}

	// Load custom stop words from the embedded string.
	err = seg.LoadStopEmbed(stopWords)
	if err != nil {
		log.Fatalf("Failed to load embedded stop words: %v", err)
	}
	log.Println("Custom dictionary and stop words loaded successfully.")
}

// FreeJieba is no longer needed with gse as it's pure Go.
// We keep the function to avoid breaking changes in other parts of the code,
// but it will do nothing.
func FreeJieba() {
	// No-op
}

// SegmentTextForIndex segments text for indexing purposes.
func SegmentTextForIndex(text string) string {
	// 1. Cut text into words.
	words := seg.Cut(text, true)

	// 2. Trim stop words using the loaded list.
	trimmedWords := seg.Trim(words)

	// 3. Join words for the FTS index and add debug logging.
	result := strings.Join(trimmedWords, " ")
	log.Printf("[Debug Index] Original text snippet: \"%.100s...\" -> Segmented: \"%s\"", text, result)

	return result
}

// SegmentTextForQuery segments text for search query purposes.
func SegmentTextForQuery(query string) string {
	// 1. Cut query into words.
	words := seg.Cut(query, true)

	// 2. Trim stop words using the loaded list.
	trimmedWords := seg.Trim(words)

	// 3. Join words for the FTS query and add debug logging.
	result := strings.Join(trimmedWords, " ")
	log.Printf("[Debug Query] Original: \"%s\" -> Segmented: \"%s\"", query, result)

	return result
}

// filterStopWords is no longer needed as the stop word dictionary now handles punctuation.
