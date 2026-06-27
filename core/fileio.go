package core

import (
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"
)

func OpenFile(path string) (string, error) {
	path = expandHome(path)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}

	if isBinaryData(data) {
		return "", fmt.Errorf("open: %q looks like a binary file — only text files are supported", filepath.Base(path))
	}

	return string(data), nil
}

func SaveFile(path, content string) error {
	path = expandHome(path)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save: create directory %s: %w", dir, err)
	}

	tmp := path + ".octonote.tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("save: create temp file: %w", err)
	}
	if _, err := f.Write([]byte(content)); err != nil {
		f.Close()
		return fmt.Errorf("save: write temp file: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("save: sync temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("save: close temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("save: rename to %s: %w", path, err)
	}
	return nil
}

func isBinaryData(data []byte) bool {
	probe := data
	if len(probe) > 8192 {
		probe = probe[:8192]
	}

	invalidCount := 0
	total := 0

	for len(probe) > 0 {
		// Null byte → almost certainly binary.
		if probe[0] == 0 {
			return true
		}
		r, size := utf8.DecodeRune(probe)
		total++
		if r == utf8.RuneError && size == 1 {
			invalidCount++
		}
		probe = probe[size:]
	}

	if total == 0 {
		return false
	}
	return float64(invalidCount)/float64(total) > 0.30
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
