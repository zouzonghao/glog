package utils

import (
	"bytes"
	"html/template"
	"regexp"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var mdRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

func RenderMarkdown(md string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := mdRenderer.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
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

func GenerateExcerpt(md string, length int) string {
	plainText := stripMarkdown(md)
	// Use runes to handle multi-byte characters like Chinese
	runes := []rune(plainText)
	if len(runes) > length {
		return string(runes[:length]) + "..."
	}
	return string(runes)
}
