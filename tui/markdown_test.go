package main

import (
	"strings"
	"testing"
)

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "Header markdown",
			input:    "# Hello World",
			contains: "Hello World",
		},
		{
			name:     "Header HTML tag",
			input:    "<h1>Hello World</h1>",
			contains: "Hello World",
		},
		{
			name:     "Header HTML h2 tag",
			input:    "<h2>Sub Header</h2>",
			contains: "Sub Header",
		},
		{
			name:     "Bold HTML tag",
			input:    "This is <b>bold</b> and <strong>strong</strong> text",
			contains: "bold",
		},
		{
			name:     "Underline HTML tag",
			input:    "Here is <u>underlined</u> text",
			contains: "underlined",
		},
		{
			name:     "Kbd and code HTML tag",
			input:    "Press <kbd>Ctrl+C</kbd> or <code>code</code>",
			contains: "Ctrl+C",
		},
		{
			name:     "Mark highlight HTML tag",
			input:    "Important <mark>highlight</mark>",
			contains: "highlight",
		},
		{
			name:     "HTML horizontal rule",
			input:    "<hr />",
			contains: "────────",
		},
		{
			name:     "HTML bullet item",
			input:    "<li>Item One</li>",
			contains: "Item One",
		},
		{
			name:     "HTML link tag",
			input:    `<a href="https://example.com">Example Site</a>`,
			contains: "Example Site",
		},
		{
			name:     "HTML container strip",
			input:    "<details><summary>Summary</summary><div>Content inside</div></details>",
			contains: "Summary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := RenderMarkdown(tt.input)
			if !strings.Contains(res, tt.contains) {
				t.Errorf("RenderMarkdown(%q) expected to contain %q, got %q", tt.input, tt.contains, res)
			}
		})
	}
}
