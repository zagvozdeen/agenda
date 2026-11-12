package web

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var markdownRenderer = goldmark.New(goldmark.WithExtensions(extension.GFM))

var markdownPolicy = func() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(regexp.MustCompile("^language-[a-zA-Z0-9_-]+$")).OnElements("code")
	policy.AllowElements("input")
	policy.AllowAttrs("type").Matching(regexp.MustCompile("^checkbox$")).OnElements("input")
	policy.AllowAttrs("checked", "disabled").OnElements("input")
	return policy
}()

func renderMarkdown(value string) (template.HTML, error) {
	var output bytes.Buffer
	if err := markdownRenderer.Convert([]byte(value), &output); err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}
	return template.HTML(markdownPolicy.SanitizeBytes(output.Bytes())), nil
}
