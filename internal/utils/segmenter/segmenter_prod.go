//go:build release

package segmenter

import (
	_ "embed"
	"log"
)

//go:embed dict/simplified.txt
var simplifiedDict []byte

//go:embed dict/stop_word.txt
var stopWords []byte

func init() {
	log.Println("Running in release mode, loading embedded dictionaries...")

	seg = Segmenter{}
	seg.Dict = NewDict()
	seg.DictSep = " "

	totalFreq := loadDictFromString(&seg, string(simplifiedDict))
	recalculateTokenDistances(seg.Dict, totalFreq)
	loadStopFromString(&seg, string(stopWords))

	log.Println("Embedded dictionaries loaded successfully.")
}
