package utils

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yanyiwu/gojieba"
)

var x *gojieba.Jieba

//go:embed dict/*
var dictFS embed.FS

func init() {
	// Create a temporary directory to write the embedded dictionaries
	tmpDir, err := os.MkdirTemp("", "glog-jieba-dict")
	if err != nil {
		log.Fatalf("Failed to create temp dir for jieba dict: %v", err)
	}

	dictNames := []string{
		"jieba.dict.utf8",
		"hmm_model.utf8",
		"user.dict.utf8",
		"idf.utf8",
		"stop_words.utf8",
	}
	dictPaths := make([]string, len(dictNames))

	for i, name := range dictNames {
		data, err := fs.ReadFile(dictFS, filepath.Join("dict", name))
		if err != nil {
			log.Fatalf("Failed to read embedded dict file %s: %v", name, err)
		}
		tmpPath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(tmpPath, data, 0644); err != nil {
			log.Fatalf("Failed to write temporary dict file %s: %v", name, err)
		}
		dictPaths[i] = tmpPath
	}

	x = gojieba.NewJieba(dictPaths...)
}

// FreeJieba releases the C++ memory used by gojieba.
func FreeJieba() {
	x.Free()
}

// SegmentTextForIndex segments text for indexing purposes (e.g., saving to database).
// It uses full mode to get all possible words.
func SegmentTextForIndex(text string) string {
	words := x.Cut(text, true)
	return strings.Join(filterStopWords(words), " ")
}

// SegmentTextForQuery segments text for search query purposes.
// It uses search mode for better precision.
func SegmentTextForQuery(query string) string {
	words := x.CutForSearch(query, true)
	return strings.Join(filterStopWords(words), " ")
}

// filterStopWords removes common stop words and single-character words.
func filterStopWords(words []string) []string {
	var result []string
	for _, word := range words {
		trimmedWord := strings.TrimSpace(word)
		// Allow single-character words, but filter out empty strings.
		if len(trimmedWord) > 0 {
			result = append(result, trimmedWord)
		}
	}
	return result
}
