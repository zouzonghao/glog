package utils

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"

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
	// Also remove the <!--more--> tag from the final rendered content
	htmlContent := strings.Replace(buf.String(), "<!--more-->", "", -1)
	return template.HTML(htmlContent), nil
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
	separator := "<!--more-->"
	var excerpt string

	if strings.Contains(md, separator) {
		// Use the content before the separator as the excerpt
		excerpt = strings.Split(md, separator)[0]
	} else {
		// Fallback to the full content if separator is not found
		excerpt = md
	}

	plainText := stripMarkdown(excerpt)
	// Use runes to handle multi-byte characters like Chinese
	runes := []rune(plainText)
	if len(runes) > length {
		return string(runes[:length]) + "..."
	}
	return string(runes)
}
