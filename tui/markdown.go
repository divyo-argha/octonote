package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleHeader1 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#818cf8")).
			Bold(true).
			Underline(true)

	styleHeader2 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10b981")).
			Bold(true)

	styleBullet = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#818cf8")).
			Bold(true)

	styleCodeBlock = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a78bfa")).
			Background(lipgloss.Color("#1e1e2f")).
			Padding(0, 1)

	styleInlineCode = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f472b6")).
			Background(lipgloss.Color("#2e2e3f")).
			Padding(0, 1)

	styleBold = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff"))

	styleItalic = lipgloss.NewStyle().
			Italic(true)
)

var (
	reBold       = regexp.MustCompile(`\*\*(.*?)\*\*`)
	reItalic     = regexp.MustCompile(`\*(.*?)\*`)
	reInlineCode = regexp.MustCompile("`(.*?)`")
)

// RenderMarkdown parses the input plain text line-by-line and applies lipgloss styles
// to headers, bullet lists, inline bold/italic/code, and block code blocks.
func RenderMarkdown(input string) string {
	lines := strings.Split(input, "\n")
	rendered := make([]string, len(lines))
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 1. Code Blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			rendered[i] = "" // Don't show the backticks
			continue
		}

		if inCodeBlock {
			rendered[i] = styleCodeBlock.Render(line)
			continue
		}

		// 2. Headers
		if strings.HasPrefix(line, "# ") {
			rendered[i] = styleHeader1.Render(line[2:])
			continue
		}
		if strings.HasPrefix(line, "## ") {
			rendered[i] = styleHeader2.Render(line[3:])
			continue
		}

		// 3. Bullet lists
		if strings.HasPrefix(trimmed, "- ") {
			idx := strings.Index(line, "- ")
			indent := line[:idx]
			rendered[i] = indent + styleBullet.Render("• ") + styleSpanFormatting(line[idx+2:])
			continue
		}
		if strings.HasPrefix(trimmed, "* ") {
			idx := strings.Index(line, "* ")
			indent := line[:idx]
			rendered[i] = indent + styleBullet.Render("• ") + styleSpanFormatting(line[idx+2:])
			continue
		}

		// 4. Regular line span formatting
		rendered[i] = styleSpanFormatting(line)
	}

	return strings.Join(rendered, "\n")
}

func styleSpanFormatting(line string) string {
	// Simple inline code replacement
	line = reInlineCode.ReplaceAllStringFunc(line, func(match string) string {
		content := reInlineCode.FindStringSubmatch(match)[1]
		return styleInlineCode.Render(content)
	})

	// Simple bold replacement
	line = reBold.ReplaceAllStringFunc(line, func(match string) string {
		content := reBold.FindStringSubmatch(match)[1]
		return styleBold.Render(content)
	})

	// Simple italic replacement
	line = reItalic.ReplaceAllStringFunc(line, func(match string) string {
		content := reItalic.FindStringSubmatch(match)[1]
		return styleItalic.Render(content)
	})

	return line
}
