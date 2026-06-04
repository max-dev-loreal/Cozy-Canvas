package utils

import (
	"bytes"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

// RenderMarkdown converts Markdown text to sanitized HTML
func RenderMarkdown(md string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return md
	}

	// Use UGCPolicy as base for sanitization (handles common tags h1-h3, p, strong, em, etc.)
	p := bluemonday.UGCPolicy()
	
	// Allow specific elements if UGCPolicy is too strict
	p.AllowElements("h1", "h2", "h3", "p", "br", "hr", "blockquote", "code", "pre")
	p.AllowElements("ul", "ol", "li")
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("src", "alt").OnElements("img")
	p.AllowURLSchemes("http", "https")

	return p.Sanitize(buf.String())
}
