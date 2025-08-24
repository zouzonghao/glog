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

	result := seg.Cut(text, true)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Dictionary segmentation failed for '%s'.\nExpected: %v\nGot: %v", text, expected, result)
	}
}

// TestHMMSegmentation tests the HMM model's ability to handle out-of-vocabulary words.
func TestHMMSegmentation(t *testing.T) {
	text := "小明硕士毕业于中国科学院计算所，后在日本京都大学深造"
	expected := []string{"小明", "硕士", "毕业", "于", "中国科学院", "计算所", "，", "后", "在", "日本", "京都大学", "深造"}

	result := seg.Cut(text, true)

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("HMM segmentation failed for '%s'.\nExpected: %v\nGot: %v", text, expected, result)
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
