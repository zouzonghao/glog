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
	// Keep the <!--more--> tag in the final rendered content
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
