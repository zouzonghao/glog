package segmenter

import (
	"reflect"
	"testing"
)

// TestDictionarySegmentation tests basic segmentation based on the dictionary.
func TestDictionarySegmentation(t *testing.T) {
	if seg.Dict == nil {
		t.Fatal("Segmenter 'seg' was not initialized correctly: Dictionary is nil")
	}
	if seg.StopWordMap == nil {
		t.Fatal("Segmenter 'seg' was not initialized correctly: StopWordMap is nil")
	}

	text := "北京天安门"
	expected := []string{"北京", "天安门"}

	result := seg.Cut(text)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Dictionary segmentation failed for '%s'.\nExpected: %v\nGot: %v", text, expected, result)
	}
}

// TestSegmentTextForIndex tests the application-level API.
func TestSegmentTextForIndex(t *testing.T) {
	text := "这是一个用于测试的句子"
	expected := "一个 用于 测试 句子"

	result := SegmentTextForIndex(text)

	if result != expected {
		t.Errorf("SegmentTextForIndex failed.\nExpected: '%s'\nGot: '%s'", expected, result)
	}
}

// TestEnglishSegmentation tests that English words are not split into letters.
func TestEnglishSegmentation(t *testing.T) {
	text := "this is a test"
	expected := []string{"this", "is", "a", "test"}
	result := seg.Trim(seg.Cut(text))
	t.Logf("Input: '%s'", text)
	t.Logf("Output: %v", result)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("English segmentation failed for '%s'.\nExpected: %v\nGot: %v", text, expected, result)
	}
}

// TestMixedSegmentation tests that English words and Chinese words are segmented correctly.
func TestMixedSegmentation(t *testing.T) {
	text := "Go语言是Google开发的"

	result := seg.Trim(seg.Cut(text))
	t.Logf("Input: '%s'", text)
	t.Logf("Output: %v", result)

	// Note: The segmentation of "开发的" might differ, so we check for the presence of key parts.
	foundGo := false
	foundGoogle := false
	foundLang := false
	for _, word := range result {
		if word == "Go" {
			foundGo = true
		}
		if word == "Google" {
			foundGoogle = true
		}
		if word == "语言" {
			foundLang = true
		}
	}

	if !foundGo || !foundGoogle || !foundLang {
		t.Errorf("Mixed segmentation failed for '%s'.\nExpected parts 'Go', 'Google', '语言' to be present.\nGot: %v", text, result)
	}
}
