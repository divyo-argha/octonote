package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
)

// FormatJSON formats or minifies a JSON string. Returns error if invalid JSON.
func FormatJSON(input string, minify bool) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty JSON input")
	}

	var raw interface{}
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	var buf bytes.Buffer
	if minify {
		if err := json.Compact(&buf, []byte(input)); err != nil {
			return "", err
		}
	} else {
		encoder := json.NewEncoder(&buf)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(raw); err != nil {
			return "", err
		}
	}
	return strings.TrimRight(buf.String(), "\n"), nil
}

// TransformCase transforms input text into target case mode.
// Supported targetCase: "upper", "lower", "title", "camel", "snake", "kebab".
func TransformCase(input string, targetCase string) string {
	switch strings.ToLower(targetCase) {
	case "upper":
		return strings.ToUpper(input)
	case "lower":
		return strings.ToLower(input)
	case "title":
		return strings.Title(strings.ToLower(input))
	case "camel":
		return toCamelCase(input)
	case "snake":
		return toSnakeCase(input)
	case "kebab":
		return toKebabCase(input)
	default:
		return input
	}
}

// CalculateMetrics returns word, character, line counts and estimated reading time.
func CalculateMetrics(text string) map[string]interface{} {
	chars := len([]rune(text))
	words := len(strings.Fields(text))
	lines := 0
	if chars > 0 {
		lines = strings.Count(text, "\n") + 1
	}

	// Average reading speed: 200 words per minute
	readingMinutes := float64(words) / 200.0
	readingSeconds := int(math.Ceil(readingMinutes * 60))

	readingTimeStr := "< 1 min"
	if readingSeconds >= 60 {
		mins := readingSeconds / 60
		readingTimeStr = fmt.Sprintf("%d min read", mins)
	} else if readingSeconds > 0 {
		readingTimeStr = fmt.Sprintf("%d sec read", readingSeconds)
	}

	return map[string]interface{}{
		"words":       words,
		"chars":       chars,
		"lines":       lines,
		"readingTime": readingTimeStr,
	}
}

// GetTemplate returns predefined scratchpad starter snippets.
func GetTemplate(name string) string {
	switch strings.ToLower(name) {
	case "todo":
		return "# 📋 Task List\n\n- [ ] Task 1\n- [ ] Task 2\n- [ ] Task 3\n- [x] Completed task\n"
	case "meeting":
		return "# 🤝 Meeting Notes\n\n**Date:** " + time.Now().Format("2006-01-02") + "\n**Attendees:** \n\n## 📌 Agenda\n- \n\n## 📝 Discussion\n- \n\n## ✅ Action Items\n- [ ] \n"
	case "table":
		return "| Feature | Status | Priority |\n| :--- | :---: | ---: |\n| Auto-Save | ✅ Done | High |\n| P2P Sharing | ✅ Done | High |\n| Themes | ✅ Active | Medium |\n"
	case "json":
		return "{\n  \"appName\": \"octoNote\",\n  \"version\": \"2.1.0\",\n  \"features\": [\n    \"Multi-tab\",\n    \"Auto-save\",\n    \"AES-256 Encryption\",\n    \"P2P Share\"\n  ],\n  \"active\": true\n}"
	case "bug":
		return "# 🐛 Bug Report\n\n### Description\n\n### Steps to Reproduce\n1. \n2. \n3. \n\n### Expected Behavior\n\n### Actual Behavior\n\n### System Info\n- OS: \n- Version: \n"
	default:
		return ""
	}
}

// Helper casing converters
func getWords(s string) []string {
	var words []string
	var current strings.Builder

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if current.Len() > 0 {
				lastR := runes[i-1]
				if unicode.IsLower(lastR) && unicode.IsUpper(r) {
					words = append(words, current.String())
					current.Reset()
				} else if unicode.IsUpper(lastR) && unicode.IsUpper(r) && i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
					words = append(words, current.String())
					current.Reset()
				}
			}
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func toCamelCase(s string) string {
	words := getWords(s)
	if len(words) == 0 {
		return s
	}
	for i, w := range words {
		if i == 0 {
			words[i] = strings.ToLower(w)
		} else {
			words[i] = strings.Title(strings.ToLower(w))
		}
	}
	return strings.Join(words, "")
}

func toSnakeCase(s string) string {
	words := getWords(s)
	if len(words) == 0 {
		return s
	}
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

func toKebabCase(s string) string {
	words := getWords(s)
	if len(words) == 0 {
		return s
	}
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "-")
}
