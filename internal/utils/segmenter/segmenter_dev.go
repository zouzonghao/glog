//go:build !release

package segmenter

import (
	"log"
	"os"
)

func init() {
	log.Println("Running in debug mode, loading dictionaries from filesystem...")

	seg = Segmenter{}
	seg.Dict = NewDict()
	seg.DictSep = " "

	// Try loading from project root (for go run)
	dictBytes, err := os.ReadFile("internal/utils/segmenter/dict/simplified.txt")
	if err != nil {
		// If it fails, try loading from package directory (for go test)
		dictBytes, err = os.ReadFile("dict/simplified.txt")
		if err != nil {
			log.Fatalf("Failed to read simplified.txt in dev mode from both project root and package dir: %v", err)
		}
	}

	stopBytes, err := os.ReadFile("internal/utils/segmenter/dict/stop_word.txt")
	if err != nil {
		stopBytes, err = os.ReadFile("dict/stop_word.txt")
		if err != nil {
			log.Fatalf("Failed to read stop_word.txt in dev mode from both project root and package dir: %v", err)
		}
	}

	totalFreq := loadDictFromString(&seg, string(dictBytes))
	recalculateTokenDistances(seg.Dict, totalFreq)
	loadStopFromString(&seg, string(stopBytes))

	log.Println("Dictionaries loaded successfully from filesystem.")
}
