package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleHeader1 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a5b4fc")).
			Bold(true).
			Underline(true)

	styleHeader2 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#38bdf8")).
			Bold(true)

	styleHeader3 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#818cf8")).
			Bold(true)

	styleBullet = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a78bfa")).
			Bold(true)

	styleTodoUnchecked = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94a3b8")).
				Bold(true)

	styleTodoChecked = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#34d399")).
				Bold(true)

	styleQuote = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#cbd5e1")).
			Italic(true)

	styleQuoteBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6366f1")).
			Bold(true)

	styleRule = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3f3f46"))

	styleCodeHeader = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a5b4fc")).
			Background(lipgloss.Color("#181826")).
			Bold(true).
			Padding(0, 1)

	styleCodeBlock = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#c084fc")).
			Background(lipgloss.Color("#181826")).
			Padding(0, 1)

	styleInlineCode = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f472b6")).
			Background(lipgloss.Color("#27273a")).
			Padding(0, 1)

	styleBold = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff"))

	styleItalic = lipgloss.NewStyle().
			Italic(true).
			Foreground(lipgloss.Color("#e2e8f0"))

	styleLink = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60a5fa")).
			Underline(true)

	styleUnderline = lipgloss.NewStyle().
			Underline(true)

	styleStrikethrough = lipgloss.NewStyle().
				Strikethrough(true).
				Foreground(lipgloss.Color("#94a3b8"))

	styleHighlight = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#facc15")).
			Padding(0, 1)
)

var (
	reBold       = regexp.MustCompile(`\*\*(.*?)\*\*`)
	reItalic     = regexp.MustCompile(`\*([^*]+)\*`)
	reInlineCode = regexp.MustCompile("`([^`]+)`")
	reLink       = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

	// HTML Tag Regexes
	reH1HTML        = regexp.MustCompile(`(?i)^\s*<h1[^>]*>(.*?)</h1>\s*$`)
	reH2HTML        = regexp.MustCompile(`(?i)^\s*<h2[^>]*>(.*?)</h2>\s*$`)
	reH3HTML        = regexp.MustCompile(`(?i)^\s*<h3[^>]*>(.*?)</h3>\s*$`)
	reH46HTML       = regexp.MustCompile(`(?i)^\s*<h[4-6][^>]*>(.*?)</h[4-6]>\s*$`)
	reHRHTML        = regexp.MustCompile(`(?i)^\s*<hr\s*/?>\s*$`)
	reLiHTML        = regexp.MustCompile(`(?i)^\s*<li[^>]*>(.*?)</li>\s*$`)
	reBoldHTML      = regexp.MustCompile(`(?i)<(?:b|strong)[^>]*>(.*?)</(?:b|strong)>`)
	reItalicHTML    = regexp.MustCompile(`(?i)<(?:i|em)[^>]*>(.*?)</(?:i|em)>`)
	reUnderlineHTML = regexp.MustCompile(`(?i)<u[^>]*>(.*?)</u>`)
	reStrikeHTML    = regexp.MustCompile(`(?i)<(?:s|del|strike)[^>]*>(.*?)</(?:s|del|strike)>`)
	reCodeHTML      = regexp.MustCompile(`(?i)<(?:code|kbd)[^>]*>(.*?)</(?:code|kbd)>`)
	reMarkHTML      = regexp.MustCompile(`(?i)<mark[^>]*>(.*?)</mark>`)
	reLinkHTML      = regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']*)["'][^>]*>(.*?)</a>`)
	reBrHTML        = regexp.MustCompile(`(?i)<br\s*/?>`)
	reStripHTML     = regexp.MustCompile(`(?i)</?(?:div|p|details|summary|span|font|table|thead|tbody|tr|td|th|section|article|figure|figcaption|center)[^>]*>`)
)

// RenderMarkdown parses the input plain text line-by-line and applies lipgloss styles
// to headers, bullet lists, checkboxes, blockquotes, inline formatting, and code blocks.
func RenderMarkdown(input string) string {
	lines := strings.Split(input, "\n")
	rendered := make([]string, len(lines))
	inCodeBlock := false
	codeLang := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 1. Code Blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				codeLang = strings.TrimPrefix(trimmed, "```")
				if codeLang == "" {
					codeLang = "code"
				}
				rendered[i] = styleCodeHeader.Render(fmt.Sprintf("┌── %s ", codeLang))
			} else {
				rendered[i] = styleCodeHeader.Render("└──")
			}
			continue
		}

		if inCodeBlock {
			rendered[i] = styleCodeBlock.Render("│ " + line)
			continue
		}

		// 2. Horizontal Rules (---, ***, ___ or <hr>, <hr/>)
		if ((trimmed == "---" || trimmed == "***" || trimmed == "___") && len(trimmed) >= 3) || reHRHTML.MatchString(trimmed) {
			rendered[i] = styleRule.Render("────────────────────────────────────────────────────────────")
			continue
		}

		// 3. Headers (# or <h1>, <h2>, <h3>)
		if strings.HasPrefix(line, "# ") {
			rendered[i] = styleHeader1.Render("󰉫 " + line[2:])
			continue
		}
		if m := reH1HTML.FindStringSubmatch(trimmed); len(m) > 1 {
			rendered[i] = styleHeader1.Render("󰉫 " + styleSpanFormatting(m[1]))
			continue
		}
		if strings.HasPrefix(line, "## ") {
			rendered[i] = styleHeader2.Render("󰉬 " + line[3:])
			continue
		}
		if m := reH2HTML.FindStringSubmatch(trimmed); len(m) > 1 {
			rendered[i] = styleHeader2.Render("󰉬 " + styleSpanFormatting(m[1]))
			continue
		}
		if strings.HasPrefix(line, "### ") {
			rendered[i] = styleHeader3.Render("󰉭 " + line[4:])
			continue
		}
		if m := reH3HTML.FindStringSubmatch(trimmed); len(m) > 1 {
			rendered[i] = styleHeader3.Render("󰉭 " + styleSpanFormatting(m[1]))
			continue
		}
		if m := reH46HTML.FindStringSubmatch(trimmed); len(m) > 1 {
			rendered[i] = styleHeader3.Render("󰉭 " + styleSpanFormatting(m[1]))
			continue
		}

		// 4. Task list items (- [ ] / - [x])
		if strings.HasPrefix(trimmed, "- [ ] ") || strings.HasPrefix(trimmed, "* [ ] ") {
			idx := strings.Index(line, "[ ] ")
			indent := line[:idx-2]
			rendered[i] = indent + styleTodoUnchecked.Render("☐ ") + styleSpanFormatting(line[idx+4:])
			continue
		}
		if strings.HasPrefix(trimmed, "- [x] ") || strings.HasPrefix(trimmed, "- [X] ") ||
			strings.HasPrefix(trimmed, "* [x] ") || strings.HasPrefix(trimmed, "* [X] ") {
			idx := strings.Index(line, "[")
			indent := line[:idx-2]
			rendered[i] = indent + styleTodoChecked.Render("☑ ") + styleSpanFormatting(line[idx+4:])
			continue
		}

		// 5. Bullet lists (- item, * item, or <li>item</li>)
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
		if m := reLiHTML.FindStringSubmatch(trimmed); len(m) > 1 {
			rendered[i] = styleBullet.Render("• ") + styleSpanFormatting(m[1])
			continue
		}

		// 6. Blockquotes (> quote)
		if strings.HasPrefix(trimmed, "> ") {
			idx := strings.Index(line, "> ")
			indent := line[:idx]
			rendered[i] = indent + styleQuoteBar.Render("▎ ") + styleQuote.Render(styleSpanFormatting(line[idx+2:]))
			continue
		}

		// 7. Regular line span formatting
		rendered[i] = styleSpanFormatting(line)
	}

	return strings.Join(rendered, "\n")
}

func styleSpanFormatting(line string) string {
	// Markdown inline code
	line = reInlineCode.ReplaceAllStringFunc(line, func(match string) string {
		sub := reInlineCode.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleInlineCode.Render(sub[1])
		}
		return match
	})

	// Markdown bold
	line = reBold.ReplaceAllStringFunc(line, func(match string) string {
		sub := reBold.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleBold.Render(sub[1])
		}
		return match
	})

	// Markdown italic
	line = reItalic.ReplaceAllStringFunc(line, func(match string) string {
		sub := reItalic.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleItalic.Render(sub[1])
		}
		return match
	})

	// Markdown link [title](url) -> title (url)
	line = reLink.ReplaceAllStringFunc(line, func(match string) string {
		sub := reLink.FindStringSubmatch(match)
		if len(sub) > 2 {
			return styleLink.Render(sub[1]) + " " + styleItalic.Render("("+sub[2]+")")
		}
		return match
	})

	// HTML bold <b> / <strong>
	line = reBoldHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reBoldHTML.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleBold.Render(sub[1])
		}
		return match
	})

	// HTML italic <i> / <em>
	line = reItalicHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reItalicHTML.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleItalic.Render(sub[1])
		}
		return match
	})

	// HTML underline <u>
	line = reUnderlineHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reUnderlineHTML.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleUnderline.Render(sub[1])
		}
		return match
	})

	// HTML strike <s> / <del> / <strike>
	line = reStrikeHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reStrikeHTML.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleStrikethrough.Render(sub[1])
		}
		return match
	})

	// HTML code & kbd <code> / <kbd>
	line = reCodeHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reCodeHTML.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleInlineCode.Render(sub[1])
		}
		return match
	})

	// HTML highlight <mark>
	line = reMarkHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reMarkHTML.FindStringSubmatch(match)
		if len(sub) > 1 {
			return styleHighlight.Render(sub[1])
		}
		return match
	})

	// HTML link <a href="...">title</a>
	line = reLinkHTML.ReplaceAllStringFunc(line, func(match string) string {
		sub := reLinkHTML.FindStringSubmatch(match)
		if len(sub) > 2 {
			return styleLink.Render(sub[2]) + " " + styleItalic.Render("("+sub[1]+")")
		}
		return match
	})

	// HTML line breaks <br>
	line = reBrHTML.ReplaceAllString(line, " ")

	// Strip remaining structural HTML tags (e.g. <div>, <span>, <details>) for clean terminal rendering
	line = reStripHTML.ReplaceAllString(line, "")

	return line
}
