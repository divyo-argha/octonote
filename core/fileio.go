package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// dangerousPaths lists path prefixes that are never safe to read or write.
// This is a defence-in-depth measure; the primary protection is
// filepath.Clean + EvalSymlinks stripping ".." sequences.
var dangerousPaths = []string{
	"/proc/",
	"/sys/",
	"/dev/",
	"/etc/passwd",
	"/etc/shadow",
	"/etc/sudoers",
}

// validatePath resolves the path, expands "~", cleans ".." sequences, and
// checks that it does not point into restricted locations.
// It returns the cleaned absolute path, or an error if the path is rejected.
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Expand ~ shorthand.
	path = expandHome(path)

	// filepath.Clean removes ".." sequences and normalises separators.
	path = filepath.Clean(path)

	// The path must now be absolute.
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path must be absolute (got %q)", path)
	}

	// Check for rejected path prefixes.
	lower := strings.ToLower(path)
	for _, bad := range dangerousPaths {
		if strings.HasPrefix(lower, bad) || lower == strings.TrimSuffix(bad, "/") {
			return "", fmt.Errorf("access to %q is restricted", path)
		}
	}

	return path, nil
}

// OpenFile reads a local text file and returns its content.
// Returns a non-nil error if the path is invalid, restricted, or the file is binary.
func OpenFile(path string) (string, error) {
	safe, err := validatePath(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}

	// Resolve symlinks after cleaning to prevent symlink-based traversal.
	// We intentionally do NOT fail if EvalSymlinks fails (e.g. path doesn't
	// exist yet) — the read below will surface any real error.
	if resolved, err2 := filepath.EvalSymlinks(safe); err2 == nil {
		// Re-check the resolved path.
		if _, err3 := validatePath(resolved); err3 != nil {
			return "", fmt.Errorf("open: symlink %q resolves to restricted location: %w", path, err3)
		}
		safe = resolved
	}

	data, err := os.ReadFile(safe)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}

	if isBinaryData(data) {
		return "", fmt.Errorf("open: %q looks like a binary file — only text files are supported", filepath.Base(safe))
	}

	return string(data), nil
}

// SaveFile writes content to path atomically (write-to-temp, rename).
// Path traversal and restricted-location checks are applied before writing.
func SaveFile(path, content string) error {
	safe, err := validatePath(path)
	if err != nil {
		return fmt.Errorf("save: %w", err)
	}

	dir := filepath.Dir(safe)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("save: create directory %s: %w", dir, err)
	}

	tmp := safe + ".octonote.tmp"
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

	if err := os.Rename(tmp, safe); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("save: rename to %s: %w", safe, err)
	}
	return nil
}

// isBinaryData returns true if data is likely binary (not a text file).
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

// expandHome replaces a leading "~" with the current user's home directory.
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
