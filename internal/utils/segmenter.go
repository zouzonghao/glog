package utils

import (
	"strings"

	"github.com/yanyiwu/gojieba"
)

var jiebaInstance *gojieba.Jieba

func init() {
	jiebaInstance = gojieba.NewJieba()
}

// SegmentTextForIndex 对要索引的文本进行分词
func SegmentTextForIndex(text string) string {
	// 使用搜索模式，返回一个字符串切片
	words := jiebaInstance.CutForSearch(text, true)
	return strings.Join(words, " ")
}

// SegmentTextForQuery 对用户查询的文本进行分词
func SegmentTextForQuery(query string) string {
	words := jiebaInstance.CutForSearch(query, true)
	// 对于 FTS 查询，我们通常希望词语之间是 AND 关系
	return strings.Join(words, " AND ")
}

// FreeJieba 在程序退出时释放 jieba 实例
func FreeJieba() {
	jiebaInstance.Free()
}
