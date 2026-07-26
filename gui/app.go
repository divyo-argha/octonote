package main

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nottaker/octonote/core"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// shareSession holds the cancel function for an in-flight share operation.
type shareSession struct {
	cancel context.CancelFunc
}

type App struct {
	ctx      context.Context
	storage  *core.Storage
	state    core.State
	mu       sync.Mutex
	isLocked bool // true if waiting for decryption password


	// share session — only one active at a time
	shareMu  sync.Mutex
	shareSes *shareSession
}

func NewApp(storage *core.Storage) *App {
	return &App{storage: storage}
}

func (a *App) GetState() core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state
}

func (a *App) SaveTab(index int, body string, cursorLine int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.state.Tabs) {
		return
	}
	// Enforce max body size.
	if len(body) > 10*1024*1024 {
		body = body[:10*1024*1024]
	}
	a.state.Tabs[index].Body = body
	a.state.Tabs[index].CursorLine = cursorLine
	a.state.Tabs[index].UpdatedAt = time.Now()
	a.storage.Save(a.state)
}

func (a *App) NewTab() core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	title := fmt.Sprintf("tab %d", len(a.state.Tabs)+1)
	tab := core.NewTab(title)
	a.state.Tabs = append(a.state.Tabs, tab)
	a.state.ActiveIndex = len(a.state.Tabs) - 1
	a.storage.Save(a.state)
	return a.state
}

func (a *App) CloseTab(index int) core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.state.Tabs) {
		return a.state
	}
	// Do not allow closing pinned tabs.
	if a.state.Tabs[index].Pinned {
		return a.state
	}
	if len(a.state.Tabs) == 1 {
		a.state.Tabs[0].Body = ""
		a.state.Tabs[0].UpdatedAt = time.Now()
		a.storage.Save(a.state)
		return a.state
	}
	a.state.Tabs = append(a.state.Tabs[:index], a.state.Tabs[index+1:]...)
	if a.state.ActiveIndex >= len(a.state.Tabs) {
		a.state.ActiveIndex = len(a.state.Tabs) - 1
	}
	a.storage.Save(a.state)
	return a.state
}

func (a *App) SetActiveTab(index int) core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.state.Tabs) {
		return a.state
	}
	a.state.ActiveIndex = index
	a.storage.Save(a.state)
	return a.state
}

func (a *App) RenameTab(index int, title string) core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.state.Tabs) {
		return a.state
	}
	title = core.SanitiseTitle(title)
	if title == "" {
		title = fmt.Sprintf("tab %d", index+1)
	}

	oldPath := a.state.Tabs[index].FilePath
	if oldPath != "" {
		dir := filepath.Dir(oldPath)
		ext := filepath.Ext(oldPath)
		newTitle := title
		if ext != "" && !strings.HasSuffix(strings.ToLower(newTitle), strings.ToLower(ext)) {
			newTitle += ext
		}
		newPath := filepath.Join(dir, newTitle)
		if oldPath != newPath {
			_ = os.Rename(oldPath, newPath)
			a.state.Tabs[index].FilePath = newPath
			title = filepath.Base(newPath)
		}
	}

	a.state.Tabs[index].Title = title
	a.state.Tabs[index].UpdatedAt = time.Now()
	a.storage.Save(a.state)
	return a.state
}

// ── Encryption API ────────────────────────────────────────────────────────────

// IsLocked returns false (zero password friction).
func (a *App) IsLocked() bool {
	return false
}

// UnlockState attempts to decrypt the storage. Returns error string if failed.
func (a *App) UnlockState(password string) string {
	if !a.storage.SetPassword(password) {
		return "Incorrect password"
	}

	st, err := a.storage.Load()
	if err != nil {
		return err.Error()
	}

	a.mu.Lock()
	a.isLocked = false
	a.state = st
	a.mu.Unlock()

	a.emitStateChange()
	return ""
}

// EnableEncryption sets a new password and encrypts the file on next save.
func (a *App) EnableEncryption(password string) string {
	if len(password) < 4 {
		return "Password too short"
	}
	a.storage.SetPassword(password)
	a.storage.SetEncrypted(true)
	
	a.mu.Lock()
	a.storage.Save(a.state)
	a.mu.Unlock()
	return ""
}

// DisableEncryption removes the password and saves the file in plaintext.
func (a *App) DisableEncryption() string {
	a.storage.SetPassword("")
	a.storage.SetEncrypted(false)
	
	a.mu.Lock()
	a.storage.Save(a.state)
	a.mu.Unlock()
	return ""
}

// IsEncryptionEnabled returns whether the file is currently set to encrypt.
func (a *App) IsEncryptionEnabled() bool {
	return a.storage.IsEncryptedFile()
}

// ── New v2.0 API methods ───────────────────────────────────────────────────────

// ReorderTabs moves the tab at fromIndex to toIndex, shifting intervening tabs.
// Returns the new state after persisting.
func (a *App) ReorderTabs(fromIndex, toIndex int) core.State {
	a.mu.Lock()
	defer a.mu.Unlock()

	tabs := a.state.Tabs
	if fromIndex < 0 || fromIndex >= len(tabs) || toIndex < 0 || toIndex >= len(tabs) || fromIndex == toIndex {
		return a.state
	}

	// Remove the tab from its current position.
	moved := tabs[fromIndex]
	newTabs := make([]core.Tab, 0, len(tabs))
	newTabs = append(newTabs, tabs[:fromIndex]...)
	newTabs = append(newTabs, tabs[fromIndex+1:]...)

	// Insert at the target position.
	result := make([]core.Tab, 0, len(tabs))
	result = append(result, newTabs[:toIndex]...)
	result = append(result, moved)
	result = append(result, newTabs[toIndex:]...)
	a.state.Tabs = result

	// Adjust active index to follow the moved tab.
	switch {
	case a.state.ActiveIndex == fromIndex:
		a.state.ActiveIndex = toIndex
	case fromIndex < a.state.ActiveIndex && toIndex >= a.state.ActiveIndex:
		a.state.ActiveIndex--
	case fromIndex > a.state.ActiveIndex && toIndex <= a.state.ActiveIndex:
		a.state.ActiveIndex++
	}

	a.storage.Save(a.state)
	return a.state
}

// PinTab sets the Pinned field on a tab and re-persists state.
func (a *App) PinTab(index int, pinned bool) core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.state.Tabs) {
		return a.state
	}
	a.state.Tabs[index].Pinned = pinned
	a.state.Tabs[index].UpdatedAt = time.Now()
	a.storage.Save(a.state)
	return a.state
}

// DuplicateTab creates a new tab with the same content as the tab at index.
func (a *App) DuplicateTab(index int) core.State {
	a.mu.Lock()
	defer a.mu.Unlock()
	if index < 0 || index >= len(a.state.Tabs) {
		return a.state
	}
	src := a.state.Tabs[index]
	title := core.SanitiseTitle(src.Title + " (copy)")
	dup := core.NewTab(title)
	dup.Body = src.Body
	a.state.Tabs = append(a.state.Tabs, dup)
	a.state.ActiveIndex = len(a.state.Tabs) - 1
	a.storage.Save(a.state)
	return a.state
}

// GetStorageDir returns the directory where state.json lives.
func (a *App) GetStorageDir() string {
	return a.storage.Dir()
}

// GetLastError returns the last storage error as a string (empty if none).
func (a *App) GetLastError() string {
	if err := a.storage.LastError(); err != nil {
		return err.Error()
	}
	return ""
}

// GetSystemInfo returns diagnostic information for the About / Settings panel.
func (a *App) GetSystemInfo() map[string]interface{} {
	a.mu.Lock()
	tabCount := len(a.state.Tabs)
	a.mu.Unlock()

	dir := a.storage.Dir()
	stateFile := filepath.Join(dir, "state.json")
	var stateSize int64
	if info, err := os.Stat(stateFile); err == nil {
		stateSize = info.Size()
	}

	return map[string]interface{}{
		"storageDir":    dir,
		"stateFileSize": stateSize,
		"tabCount":      tabCount,
		"buildVersion":  "2.0.0",
	}
}

// ExportAllTabs writes each tab as a .md file into dir.
// Returns a non-empty error string on failure, or "" on success.
func (a *App) ExportAllTabs(dir string) string {
	// Basic path validation: dir must be absolute and must exist.
	if !filepath.IsAbs(dir) {
		return "export: directory path must be absolute"
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return fmt.Sprintf("export: %s is not a valid directory", dir)
	}

	a.mu.Lock()
	tabs := make([]core.Tab, len(a.state.Tabs))
	copy(tabs, a.state.Tabs)
	a.mu.Unlock()

	timestamp := time.Now().Format("20060102-150405")
	zipPath := filepath.Join(dir, "octonote-export-"+timestamp+".zip")
	f, err := os.Create(zipPath)
	if err != nil {
		return fmt.Sprintf("export: cannot create zip: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for i, tab := range tabs {
		// Sanitise filename: replace slashes/colons, limit length.
		name := sanitiseFilename(tab.Title)
		if name == "" {
			name = fmt.Sprintf("tab-%d", i+1)
		}
		w, err := zw.Create(name + ".md")
		if err != nil {
			continue
		}
		_, _ = fmt.Fprintf(w, "# %s\n\n%s\n", tab.Title, tab.Body)
	}

	return ""
}

// sanitiseFilename removes characters that are invalid in cross-platform filenames.
func sanitiseFilename(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', '\x00':
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) > 64 {
		result = result[:64]
	}
	return result
}

func (a *App) emitStateChange() {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "state:changed", a.state)
}

// ── Share feature ────────────────────────────────────────────────────────────

// ShareSend opens a Magic Wormhole for the currently active tab.
// senderLabel is an optional name the sender provides (e.g. "Alice") so the
// receiver can see who shared the content. Pass an empty string to omit.
// It immediately emits "share:code" with the generated code, then waits
// for the peer to connect. On success it emits "share:done", on error
// it emits "share:error".
func (a *App) ShareSend(senderLabel string) {
	a.mu.Lock()
	idx := a.state.ActiveIndex
	var tab core.Tab
	if idx >= 0 && idx < len(a.state.Tabs) {
		tab = a.state.Tabs[idx]
	}
	a.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	a.shareMu.Lock()
	if a.shareSes != nil {
		a.shareSes.cancel() // cancel any previous session
	}
	a.shareSes = &shareSession{cancel: cancel}
	a.shareMu.Unlock()

	go func() {
		defer cancel()
		code, wait, err := core.ShareSend(ctx, tab, senderLabel)
		if err != nil {
			runtime.EventsEmit(a.ctx, "share:error", err.Error())
			return
		}
		runtime.EventsEmit(a.ctx, "share:code", code)
		if err := wait(); err != nil {
			// Context cancelled = user hit cancel — emit nothing.
			if ctx.Err() == nil {
				runtime.EventsEmit(a.ctx, "share:error", err.Error())
			}
			return
		}
		runtime.EventsEmit(a.ctx, "share:done", nil)
	}()
}

// ShareReceive connects to a wormhole using the user-supplied code.
// On success it imports the received content as a new tab and emits
// "share:received" with the new tab title. On error it emits "share:error".
func (a *App) ShareReceive(code string) {
	if code == "" {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.shareMu.Lock()
	if a.shareSes != nil {
		a.shareSes.cancel()
	}
	a.shareSes = &shareSession{cancel: cancel}
	a.shareMu.Unlock()

	go func() {
		defer cancel()
		result, err := core.ShareReceive(ctx, code)
		if err != nil {
			if ctx.Err() == nil {
				runtime.EventsEmit(a.ctx, "share:error", err.Error())
			}
			return
		}
		a.mu.Lock()
		newTab := core.Tab{
			ID:        fmt.Sprintf("%x", time.Now().UnixNano()),
			Title:     result.TabTitle,
			Body:      result.Body,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		a.state.Tabs = append(a.state.Tabs, newTab)
		a.state.ActiveIndex = len(a.state.Tabs) - 1
		a.storage.Save(a.state)
		a.mu.Unlock()
		runtime.EventsEmit(a.ctx, "share:received", map[string]interface{}{
			"title": result.TabTitle,
			"state": a.state,
		})
	}()
}

// ShareCancel aborts any in-flight share or receive operation.
func (a *App) ShareCancel() {
	a.shareMu.Lock()
	defer a.shareMu.Unlock()
	if a.shareSes != nil {
		a.shareSes.cancel()
		a.shareSes = nil
	}
}

// ── File I/O ─────────────────────────────────────────────────────────────────

// OpenFile reads a local file and returns its content.
// Returns an error string (non-empty) on failure.
func (a *App) OpenFile(path string) map[string]interface{} {
	content, err := core.OpenFile(path)
	if err != nil {
		return map[string]interface{}{"error": err.Error(), "content": ""}
	}
	return map[string]interface{}{"error": "", "content": content}
}

// PromptSaveFileDialog opens OS native file save dialog and saves tab content.
func (a *App) PromptSaveFileDialog(tabIndex int, content string) string {
	a.mu.Lock()
	if tabIndex < 0 || tabIndex >= len(a.state.Tabs) {
		a.mu.Unlock()
		return "invalid tab index"
	}
	tab := a.state.Tabs[tabIndex]
	a.mu.Unlock()

	defaultFilename := sanitiseFilename(tab.Title)
	if defaultFilename == "" {
		defaultFilename = "scratchpad"
	}
	if !strings.HasSuffix(defaultFilename, ".md") && !strings.HasSuffix(defaultFilename, ".txt") {
		defaultFilename += ".md"
	}

	selectedPath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save Note to Disk File",
		DefaultFilename: defaultFilename,
		Filters: []runtime.FileFilter{
			{DisplayName: "Markdown Files (*.md)", Pattern: "*.md"},
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})

	if err != nil || selectedPath == "" {
		return "cancelled"
	}

	if err := core.SaveFile(selectedPath, content); err != nil {
		return fmt.Sprintf("failed to save file: %v", err)
	}

	a.mu.Lock()
	if tabIndex >= 0 && tabIndex < len(a.state.Tabs) {
		a.state.Tabs[tabIndex].Title = filepath.Base(selectedPath)
		a.state.Tabs[tabIndex].FilePath = selectedPath
		a.state.Tabs[tabIndex].FileIsDirty = false
		a.state.Tabs[tabIndex].Body = content
		a.state.Tabs[tabIndex].UpdatedAt = time.Now()
		a.storage.Save(a.state)
	}
	a.mu.Unlock()

	return selectedPath
}

// SaveFileAs writes content to path and records the file path on the active tab.
func (a *App) SaveFileAs(tabIndex int, path, content string) string {
	if err := core.SaveFile(path, content); err != nil {
		return err.Error()
	}
	a.mu.Lock()
	if tabIndex >= 0 && tabIndex < len(a.state.Tabs) {
		a.state.Tabs[tabIndex].Title = filepath.Base(path)
		a.state.Tabs[tabIndex].FilePath = path
		a.state.Tabs[tabIndex].FileIsDirty = false
		a.state.Tabs[tabIndex].Body = content
		a.state.Tabs[tabIndex].UpdatedAt = time.Now()
		a.storage.Save(a.state)
	}
	a.mu.Unlock()
	return ""
}

// SaveCurrentFile saves content to the tab's known FilePath (must already be set).
func (a *App) SaveCurrentFile(tabIndex int, content string) string {
	a.mu.Lock()
	var path string
	if tabIndex >= 0 && tabIndex < len(a.state.Tabs) {
		path = a.state.Tabs[tabIndex].FilePath
	}
	a.mu.Unlock()

	if path == "" {
		return "no file path set — use Save As first"
	}
	if err := core.SaveFile(path, content); err != nil {
		return err.Error()
	}
	a.mu.Lock()
	if tabIndex >= 0 && tabIndex < len(a.state.Tabs) {
		a.state.Tabs[tabIndex].FileIsDirty = false
		a.state.Tabs[tabIndex].Body = content
		a.state.Tabs[tabIndex].UpdatedAt = time.Now()
		a.storage.Save(a.state)
	}
	a.mu.Unlock()
	return ""
}

// GetTabFilePath returns the on-disk file path for the given tab (empty if none).
func (a *App) GetTabFilePath(tabIndex int) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if tabIndex < 0 || tabIndex >= len(a.state.Tabs) {
		return ""
	}
	return a.state.Tabs[tabIndex].FilePath
}

// ── Developer Tools API ───────────────────────────────────────────────────────

// FormatJSON formats or minifies JSON string.
func (a *App) FormatJSON(input string, minify bool) map[string]interface{} {
	formatted, err := core.FormatJSON(input, minify)
	if err != nil {
		return map[string]interface{}{"error": err.Error(), "result": ""}
	}
	return map[string]interface{}{"error": "", "result": formatted}
}

// TransformCase transforms input text casing.
func (a *App) TransformCase(input string, targetCase string) string {
	return core.TransformCase(input, targetCase)
}

// CalculateMetrics calculates word, char, line counts and reading time.
func (a *App) CalculateMetrics(text string) map[string]interface{} {
	return core.CalculateMetrics(text)
}

// GetTemplate returns template starter text.
func (a *App) GetTemplate(name string) string {
	return core.GetTemplate(name)
}

