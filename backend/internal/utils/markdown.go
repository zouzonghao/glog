package utils

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var md goldmark.Markdown

func init() {
	md = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			// 启用 Unsafe 选项是为了让 Markdown 表格的 align 属性能够被渲染
			// 这对于实现表格列的对齐功能是必需的。
			// 警告：这会允许在 Markdown 中使用原始 HTML，如果 Markdown 内容来自不受信任的用户，可能存在 XSS 风险。
			// 在这个博客项目中，内容是博主自己控制的，所以风险较低。
			html.WithUnsafe(),
		),
	)
}

// wrapTablesInDiv 使用 goquery 解析 HTML，并为每个 table 元素包裹一个带 class 的 div。
// 这是为了在后端处理响应式表格，避免前端 JS 操作 DOM 导致的页面闪烁。
func wrapTablesInDiv(htmlContent string) (string, error) {
	// 从字符串读取 HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", err
	}

	// 查找所有的 table 元素并用 div 包裹它们
	doc.Find("table").Each(func(i int, s *goquery.Selection) {
		s.WrapHtml(`<div class="table-wrapper"></div>`)
	})

	// 将修改后的 HTML 写回字符串
	newHtml, err := doc.Html()
	if err != nil {
		return "", err
	}

	return newHtml, nil
}

// RenderMarkdown 将 markdown 字符串转换为处理过的 HTML 模板。
func RenderMarkdown(mdStr string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(mdStr), &buf); err != nil {
		return "", err
	}

	// 在返回之前，先对生成的 HTML 进行处理，为表格添加包裹 div
	processedHtml, err := wrapTablesInDiv(buf.String())
	if err != nil {
		// 如果处理失败，可以选择返回原始 HTML 或错误
		// 这里我们选择返回错误，以便于调试
		return "", err
	}

	return template.HTML(processedHtml), nil
}

// stripMarkdown removes markdown formatting for excerpt generation.
func stripMarkdown(md string) string {
	// 1. Remove Markdown images and links
	re := regexp.MustCompile(`(\[!\[.*?\]\(.*?\)\])|(\[.*?\]\(.*?\))`)
	md = re.ReplaceAllString(md, "")
	// 2. Remove headings, bold, italics, etc.
	re = regexp.MustCompile("(?m)[*#>`~-]")
	md = re.ReplaceAllString(md, "")
	// 3. Replace newlines with spaces
	re = regexp.MustCompile(`\s+`)
	md = re.ReplaceAllString(md, " ")
	return md
}

// GenerateExcerpt creates a plain text excerpt from Markdown content based on a separator.
func GenerateExcerpt(md string, length int) string {
	// Use a regex to find the separator, allowing for optional whitespace.
	// This makes the separator detection more robust.
	separatorRegex := regexp.MustCompile(`<!--\s*more\s*-->`)
	var excerpt string

	split := separatorRegex.Split(md, 2)
	if len(split) > 1 {
		// Use the content before the separator as the excerpt
		excerpt = split[0]
	} else {
		// If separator is not found, return an empty string.
		return ""
	}

	plainText := stripMarkdown(excerpt)
	// Use runes to handle multi-byte characters like Chinese
	runes := []rune(plainText)
	if len(runes) > length {
		return string(runes[:length]) + "..."
	}
	return string(runes)
}

// ExtractFirstImageURL uses a regular expression to find the first Markdown image URL.
func ExtractFirstImageURL(md string) string {
	// Regex to find the first markdown image: ![alt text](image_url)
	re := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	matches := re.FindStringSubmatch(md)

	if len(matches) > 1 {
		// The first capturing group (index 1) is the URL
		return matches[1]
	}

	return ""
}
