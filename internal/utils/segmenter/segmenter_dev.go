//go:build !release

package segmenter

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Load loads the dictionaries from the filesystem for development mode.
func Load() {
	log.Println("Running in debug mode, loading dictionaries from filesystem...")

	// --- Robust path finding ---
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Failed to get current file path")
	}
	dictDir := filepath.Join(filepath.Dir(filename), "dict")
	dictPath := filepath.Join(dictDir, "simplified.txt")
	stopPath := filepath.Join(dictDir, "stop_word.txt")
	// --- End of robust path finding ---

	seg = Segmenter{}
	seg.Dict = NewDict()
	seg.DictSep = " "

	dictBytes, err := os.ReadFile(dictPath)
	if err != nil {
		log.Fatalf("Failed to read simplified.txt in dev mode: %v", err)
	}

	stopBytes, err := os.ReadFile(stopPath)
	if err != nil {
		log.Fatalf("Failed to read stop_word.txt in dev mode: %v", err)
	}

	totalFreq := loadDictFromString(&seg, string(dictBytes))
	recalculateTokenDistances(seg.Dict, totalFreq)
	loadStopFromString(&seg, string(stopBytes))

	log.Println("Dictionaries loaded successfully from filesystem.")
}
