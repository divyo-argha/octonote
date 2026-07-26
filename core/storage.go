package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	currentStateVersion = 2
	maxTitleLen         = 128
	maxBodyLen          = 10 * 1024 * 1024 // 10 MB soft limit
)

type Tab struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	CursorLine  int       `json:"cursor_line"`
	CursorCol   int       `json:"cursor_col,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	FilePath    string    `json:"file_path,omitempty"`     // absolute path to on-disk file, empty if unsaved
	FileIsDirty bool      `json:"file_is_dirty,omitempty"` // true when body differs from last file save
	Pinned      bool      `json:"pinned,omitempty"`        // if true, tab is locked to front and cannot be closed accidentally
}

type State struct {
	Tabs        []Tab `json:"tabs"`
	ActiveIndex int   `json:"active_index"`
	Version     int   `json:"version"`
}

type Storage struct {
	mu            sync.Mutex
	dir           string
	file          string
	ch            chan State
	lastErr       error
	done          chan struct{}
	lastWriteTime time.Time
}

func StateDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("octonote: cannot determine config dir: %w", err)
	}
	return filepath.Join(base, "octonote"), nil
}

func NewStorage() (*Storage, error) {
	dir, err := StateDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("octonote: cannot create config dir %s: %w", dir, err)
	}

	s := &Storage{
		dir:  dir,
		file: filepath.Join(dir, "state.json"),
		ch:   make(chan State, 64),
		done: make(chan struct{}),
	}

	go s.syncWriter()
	return s, nil
}

func (s *Storage) Load() (State, error) {
	data, err := os.ReadFile(s.file)
	if os.IsNotExist(err) {
		return defaultState(), nil
	}
	if err != nil {
		return State{}, fmt.Errorf("octonote: read state: %w", err)
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return defaultState(), nil
	}

	// Reject files claiming an impossible future version.
	if st.Version > currentStateVersion+10 {
		return defaultState(), fmt.Errorf("octonote: state version %d is too new (expected <= %d)", st.Version, currentStateVersion)
	}

	st = MigrateState(st)

	if len(st.Tabs) == 0 {
		st = defaultState()
	}
	if st.ActiveIndex >= len(st.Tabs) || st.ActiveIndex < 0 {
		st.ActiveIndex = 0
	}
	return st, nil
}

func (s *Storage) Save(st State) {
	select {
	case s.ch <- st:
	default:
		select {
		case <-s.ch:
		default:
		}
		select {
		case s.ch <- st:
		default:
		}
	}
}

func (s *Storage) Close() {
	close(s.ch)
	<-s.done
}

func (s *Storage) LastError() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastErr
}

func (s *Storage) Dir() string { return s.dir }

func (s *Storage) syncWriter() {
	defer close(s.done)

	for st := range s.ch {
		drained := st
		for {
			select {
			case newer, ok := <-s.ch:
				if !ok {
					s.atomicWrite(drained)
					return
				}
				drained = newer
			default:
				goto write
			}
		}
	write:
		s.atomicWrite(drained)
	}
}

func (s *Storage) atomicWrite(st State) {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		s.setErr(err)
		return
	}

	tmp := s.file + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		s.setErr(err)
		return
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		s.setErr(err)
		return
	}
	if err := f.Sync(); err != nil {
		f.Close()
		s.setErr(err)
		return
	}
	if err := f.Close(); err != nil {
		s.setErr(err)
		return
	}

	s.mu.Lock()
	s.lastWriteTime = time.Now().Add(5 * time.Second)
	s.mu.Unlock()

	if err := os.Rename(tmp, s.file); err != nil {
		s.setErr(err)
		return
	}

	// Capture the file's exact modTime to avoid reload loops from our own writes.
	if info, err := os.Stat(s.file); err == nil {
		s.mu.Lock()
		s.lastWriteTime = info.ModTime()
		s.mu.Unlock()
	}

	s.setErr(nil)
}

func (s *Storage) setErr(err error) {
	s.mu.Lock()
	s.lastErr = err
	s.mu.Unlock()
}

// Watch blocks and polls state.json every 500ms. It calls onChange
// when an external modification is detected (i.e. not written by this process).
func (s *Storage) Watch(ctx context.Context, onChange func()) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastMod time.Time
	if info, err := os.Stat(s.file); err == nil {
		lastMod = info.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(s.file)
			if err != nil {
				continue
			}
			mod := info.ModTime()
			if mod.After(lastMod) {
				s.mu.Lock()
				isOurWrite := !mod.After(s.lastWriteTime)
				s.mu.Unlock()

				lastMod = mod
				if !isOurWrite {
					onChange()
				}
			}
		}
	}
}

// MigrateState upgrades a loaded State to the current schema version.
// It is idempotent and safe to call on any version.
func MigrateState(st State) State {
	// v1 → v2: no structural changes, just ensure version field is set.
	if st.Version < 2 {
		st.Version = currentStateVersion
	}
	// Sanitise all tab fields on every load.
	for i := range st.Tabs {
		st.Tabs[i].Title = SanitiseTitle(st.Tabs[i].Title)
		st.Tabs[i].Body  = stripNUL(st.Tabs[i].Body)
		if st.Tabs[i].ID == "" {
			st.Tabs[i].ID = generateID()
		}
	}
	return st
}

// SanitiseTitle strips control characters from a tab title and truncates to maxTitleLen.
func SanitiseTitle(title string) string {
	title = strings.TrimSpace(title)
	title = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1 // drop control chars including NUL
		}
		return r
	}, title)
	runes := []rune(title)
	if len(runes) > maxTitleLen {
		title = string(runes[:maxTitleLen])
	}
	return title
}

// stripNUL removes NUL bytes from a string.
func stripNUL(s string) string {
	return strings.ReplaceAll(s, "\x00", "")
}

func defaultState() State {
	now := time.Now()
	return State{
		Version:     currentStateVersion,
		ActiveIndex: 0,
		Tabs: []Tab{
			{
				ID:        generateID(),
				Title:     "scratch",
				Body:      "",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}
}

func NewTab(title string) Tab {
	now := time.Now()
	title = SanitiseTitle(title)
	if title == "" {
		title = "scratch"
	}
	return Tab{
		ID:        generateID(),
		Title:     title,
		Body:      "",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func generateID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
