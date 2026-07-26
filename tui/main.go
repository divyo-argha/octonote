package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/ncruces/zenity"
	"github.com/nottaker/octonote/core"
)

const (
	colBg       = "#09090b"
	colSurface  = "#18181b"
	colBorder   = "#3f3f46"
	colAccent   = "#6366f1"
	colAccentLt = "#818cf8"
	colMuted    = "#94a3b8"
	colText     = "#f8fafc"
	colSubtle   = "#cbd5e1"
	colWarn     = "#f59e0b"
	colSuccess  = "#10b981"
	colTabBg    = "#27272a"
	colErr      = "#ef4444"
)

var (
	styleTabInactive = lipgloss.NewStyle().
				Padding(0, 2).
				Background(lipgloss.Color(colTabBg)).
				Foreground(lipgloss.Color(colMuted)).
				Border(lipgloss.Border{
			Top: "─", Bottom: "", Left: "│", Right: "│",
			TopLeft: "╭", TopRight: "╮", BottomLeft: "├", BottomRight: "┤",
		}, true, true, false, true).
		BorderForeground(lipgloss.Color(colBorder))

	styleTabActive = lipgloss.NewStyle().
			Padding(0, 2).
			Background(lipgloss.Color(colAccent)).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Border(lipgloss.Border{
			Top: "─", Bottom: "", Left: "│", Right: "│",
			TopLeft: "╭", TopRight: "╮", BottomLeft: "├", BottomRight: "┤",
		}, true, true, false, true).
		BorderForeground(lipgloss.Color(colAccentLt))

	styleTabBar = lipgloss.NewStyle().
			Background(lipgloss.Color(colBg)).
			Padding(0, 1)

	styleContentBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(colAccent)).
				Padding(0, 1)

	styleContentBoxBlur = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(colBorder)).
				Padding(0, 1)

	styleLegend = lipgloss.NewStyle().
			Background(lipgloss.Color(colSurface)).
			Foreground(lipgloss.Color(colSubtle)).
			Padding(0, 1)

	styleKey = lipgloss.NewStyle().
			Background(lipgloss.Color(colAccent)).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 1).
			Bold(true)

	styleSaved = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colSuccess)).
			Bold(true)

	styleUnsaved = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colWarn))

	styleTitle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colAccentLt)).
			Bold(true).
			Padding(0, 1)

	styleTabCount = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colSubtle))

	styleShareCode = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color(colAccent)).
			Bold(true).
			Padding(0, 2)

	styleShareInfo = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colAccentLt))

	styleShareErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colErr)).
			Bold(true)

	styleFilePrompt = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colText))

	styleFileInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#1e1e3f")).
			Padding(0, 1)

	styleFileErr = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colErr)).
			Bold(true)
)

// ── Messages ──────────────────────────────────────────────────────────────────

type savedMsg struct{ at time.Time }
type externalStateUpdateMsg struct{}

type shareDoneMsg struct{}
type shareCodeMsg struct{ code string }
type shareErrMsg struct{ err string }
type shareReceivedMsg struct {
	title string
	st    core.State
}
type shareStartedMsg struct {
	code string
	wait func() error
}
type shareWaitResultMsg struct{ err error }

type fileOpenedMsg struct {
	path    string
	content string
}
type fileSavedMsg struct {
	path string
	at   time.Time
}
type fileErrMsg struct{ err string }

type zenityFileSelectedMsg struct{ path string }
type zenityFileSaveSelectedMsg struct{ path string }
type zenityCanceledMsg struct{ mode filePromptMode }
type zenityFailedMsg struct {
	err  error
	mode filePromptMode
}

func selectFileCmd() tea.Msg {
	path, err := zenity.SelectFile(
		zenity.Title("Open File"),
		zenity.FileFilters{
			{
				Name: "Text Files",
				Patterns: []string{
					"*.txt", "*.md", "*.html", "*.json", "*.xml",
					"*.js", "*.ts", "*.css", "*.scss", "*.less",
					"*.go", "*.py", "*.sh", "*.bat", "*.ps1",
					"*.yaml", "*.yml", "*.ini", "*.conf", "*.cfg",
					"*.csv", "*.tsv", "*.log", "*.sql",
				},
				CaseFold: true,
			},
			{
				Name:     "All Files",
				Patterns: []string{"*"},
			},
		},
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return zenityCanceledMsg{mode: filePromptOpen}
		}
		return zenityFailedMsg{err: err, mode: filePromptOpen}
	}
	return zenityFileSelectedMsg{path: path}
}

func selectFileSaveCmd() tea.Msg {
	path, err := zenity.SelectFileSave(
		zenity.Title("Save File"),
		zenity.FileFilters{
			{
				Name: "Text Files",
				Patterns: []string{
					"*.txt", "*.md", "*.html", "*.json", "*.xml",
					"*.js", "*.ts", "*.css", "*.scss", "*.less",
					"*.go", "*.py", "*.sh", "*.bat", "*.ps1",
					"*.yaml", "*.yml", "*.ini", "*.conf", "*.cfg",
					"*.csv", "*.tsv", "*.log", "*.sql",
				},
				CaseFold: true,
			},
			{
				Name:     "All Files",
				Patterns: []string{"*"},
			},
		},
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return zenityCanceledMsg{mode: filePromptSave}
		}
		return zenityFailedMsg{err: err, mode: filePromptSave}
	}
	return zenityFileSaveSelectedMsg{path: path}
}

// ── Mode enums ────────────────────────────────────────────────────────────────

type shareMode int

const (
	shareOff       shareMode = iota
	shareSending             // waiting for peer to connect
	shareReceive             // user typing the wormhole code
	shareReceiving           // receiver connecting/handshaking
)

type filePromptMode int

const (
	filePromptOff     filePromptMode = iota
	filePromptOpen                   // user typing a path to open
	filePromptSave                   // user typing a path to save-as
	filePromptConfirm                // Y/N/Esc: save before close?
)

// ── Model ─────────────────────────────────────────────────────────────────────

type model struct {
	storage  *core.Storage
	state    core.State
	textareas []textarea.Model
	width    int
	height   int
	lastSaved time.Time
	dirty    bool
	quitting bool

	// preview state
	previewMode      bool
	previewScrollRow int

	// share state
	shareMode   shareMode
	shareCode   string
	shareInput  string
	shareErr    string
	shareCancel context.CancelFunc

	// file I/O state
	fileMode         filePromptMode
	fileInput        string // typed path or Y/N
	fileErr          string
	filePendingClose bool // waiting for save before closing tab
	fileSubmitting   bool // async op in flight; block further edits
}

func initialModel(s *core.Storage, st core.State) model {
	tas := make([]textarea.Model, len(st.Tabs))
	for i, tab := range st.Tabs {
		tas[i] = newTextArea()
		tas[i].SetValue(tab.Body)
	}
	m := model{
		storage:   s,
		state:     st,
		textareas: tas,
		lastSaved: time.Now(),
	}
	if m.state.ActiveIndex < len(m.textareas) {
		m.textareas[m.state.ActiveIndex].Focus()
	}
	return m
}

func newTextArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Start typing…"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#1e1e3f"))
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(colText))
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(colMuted))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colMuted))
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colBorder))
	return ta
}

func (m model) Init() tea.Cmd { return textarea.Blink }

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	// ── Async results ─────────────────────────────────────────────────────────

	case shareDoneMsg:
		m.shareMode = shareOff
		m.shareCode = ""
		m.shareErr = ""

	case shareCodeMsg:
		m.shareCode = msg.code
		m.shareErr = ""

	case shareStartedMsg:
		m.shareCode = msg.code
		m.shareErr = ""
		return m, func() tea.Msg { return shareWaitResultMsg{err: msg.wait()} }

	case shareWaitResultMsg:
		if m.shareMode != shareSending {
			// User cancelled — ignore.
			return m, nil
		}
		m.shareMode = shareOff
		m.shareCode = ""
		if msg.err != nil {
			m.shareErr = msg.err.Error()
		}

	case shareErrMsg:
		m.shareMode = shareOff
		m.shareCode = ""
		m.shareErr = msg.err

	case shareReceivedMsg:
		m.state = msg.st
		m.shareMode = shareOff
		m.shareInput = ""
		m.shareErr = ""
		m.previewMode = false
		tas := make([]textarea.Model, len(m.state.Tabs))
		for i, tab := range m.state.Tabs {
			tas[i] = newTextArea()
			tas[i].SetValue(tab.Body)
		}
		m.textareas = tas
		if m.state.ActiveIndex < len(m.textareas) {
			m.textareas[m.state.ActiveIndex].Focus()
		}
		m = m.resizeTextAreas()

	case zenityFileSelectedMsg:
		m.fileInput = msg.path
		m.fileSubmitting = true
		cmds = append(cmds, func() tea.Msg {
			content, err := core.OpenFile(msg.path)
			if err != nil {
				return fileErrMsg{err: err.Error()}
			}
			return fileOpenedMsg{path: msg.path, content: content}
		})

	case zenityFileSaveSelectedMsg:
		m.fileInput = msg.path
		m.fileSubmitting = true
		content := m.textareas[m.state.ActiveIndex].Value()
		cmds = append(cmds, func() tea.Msg {
			if err := core.SaveFile(msg.path, content); err != nil {
				return fileErrMsg{err: err.Error()}
			}
			return fileSavedMsg{path: msg.path, at: time.Now()}
		})

	case zenityCanceledMsg:
		m.fileMode = filePromptOff
		m.fileInput = ""
		m.fileSubmitting = false
		m.filePendingClose = false

	case zenityFailedMsg:
		m.fileSubmitting = false
		m.fileErr = fmt.Sprintf("System dialog error: %v. Please enter path manually.", msg.err)

	case fileOpenedMsg:
		m.fileMode = filePromptOff
		m.fileInput = ""
		m.fileErr = ""
		m.fileSubmitting = false
		m = m.loadFileIntoTab(msg.path, msg.content)
		m.triggerSave()

	case fileSavedMsg:
		m.fileMode = filePromptOff
		m.fileInput = ""
		m.fileErr = ""
		m.fileSubmitting = false
		idx := m.state.ActiveIndex
		m.state.Tabs[idx].FilePath = msg.path
		m.state.Tabs[idx].FileIsDirty = false
		m.lastSaved = msg.at
		m.dirty = false
		m.triggerSave()
		if m.filePendingClose {
			m.filePendingClose = false
			m = m.closeTab()
			m.triggerSave()
		}

	case fileErrMsg:
		m.fileErr = msg.err
		m.fileMode = filePromptOff
		m.fileInput = ""
		m.fileSubmitting = false
		m.filePendingClose = false

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m = m.resizeTextAreas()

	case externalStateUpdateMsg:
		st, err := m.storage.Load()
		if err == nil {
			m.state = st
			tas := make([]textarea.Model, len(m.state.Tabs))
			for i, tab := range m.state.Tabs {
				tas[i] = newTextArea()
				tas[i].SetValue(tab.Body)
				
				// Restore cursor to the start of the saved cursor line
				lines := strings.Split(tab.Body, "\n")
				pos := 0
				limit := tab.CursorLine
				if limit >= len(lines) {
					limit = len(lines) - 1
				}
				if limit < 0 {
					limit = 0
				}
				for l := 0; l < limit; l++ {
					pos += len(lines[l]) + 1
				}
				tas[i].SetCursor(pos)
			}
			m.textareas = tas
			if m.state.ActiveIndex < len(m.textareas) {
				if !m.previewMode {
					m.textareas[m.state.ActiveIndex].Focus()
				} else {
					m.textareas[m.state.ActiveIndex].Blur()
				}
			}
			m = m.clampScroll()
			m = m.resizeTextAreas()
		}
		return m, nil

	case savedMsg:
		m.lastSaved = msg.at
		m.dirty = false

	// ── Key handling ──────────────────────────────────────────────────────────
	case tea.KeyMsg:
		return m.handleKey(msg, cmds)
	}

	// Propagate non-key messages (blink tick, etc.) to active textarea.
	if _, ok := msg.(tea.KeyMsg); !ok {
		idx := m.state.ActiveIndex
		if idx < len(m.textareas) {
			updated, cmd := m.textareas[idx].Update(msg)
			m.textareas[idx] = updated
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKey is the single entry point for all keyboard input.
// It is extracted to keep Update clean and to avoid break/fallthrough confusion.
func (m model) handleKey(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	// Clear stale errors on any key.
	m.fileErr = ""
	m.shareErr = ""

	// ── File close-confirm prompt (Y / N / Esc) ───────────────────────────────
	if m.fileMode == filePromptConfirm {
		switch strings.ToLower(msg.String()) {
		case "y":
			idx := m.state.ActiveIndex
			path := m.state.Tabs[idx].FilePath
			content := m.textareas[idx].Value()
			m.filePendingClose = true
			m.fileMode = filePromptOff
			cmds = append(cmds, func() tea.Msg {
				if err := core.SaveFile(path, content); err != nil {
					return fileErrMsg{err: err.Error()}
				}
				return fileSavedMsg{path: path, at: time.Now()}
			})
		case "n":
			m.fileMode = filePromptOff
			m.filePendingClose = false
			m = m.closeTab()
			m.triggerSave()
		default:
			if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC {
				m.fileMode = filePromptOff
				m.filePendingClose = false
			}
		}
		return m, tea.Batch(cmds...)
	}

	// ── File path prompts (Open / Save-as) ───────────────────────────────────
	if m.fileMode == filePromptOpen || m.fileMode == filePromptSave {
		// If we're waiting for the async op (fileSubmitting), ignore all keys
		// except Escape so the user can't double-submit or corrupt state.
		if m.fileSubmitting {
			if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC {
				m.fileMode = filePromptOff
				m.fileInput = ""
				m.fileSubmitting = false
				m.filePendingClose = false
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.Type {
		case tea.KeyEscape, tea.KeyCtrlC:
			m.fileMode = filePromptOff
			m.fileInput = ""
			m.filePendingClose = false

		case tea.KeyEnter:
			path := strings.TrimSpace(m.fileInput)
			if path == "" {
				return m, tea.Batch(cmds...)
			}
			mode := m.fileMode
			content := m.textareas[m.state.ActiveIndex].Value()
			m.fileSubmitting = true
			cmds = append(cmds, func() tea.Msg {
				if mode == filePromptOpen {
					content, err := core.OpenFile(path)
					if err != nil {
						return fileErrMsg{err: err.Error()}
					}
					return fileOpenedMsg{path: path, content: content}
				}
				// Save-as
				if err := core.SaveFile(path, content); err != nil {
					return fileErrMsg{err: err.Error()}
				}
				return fileSavedMsg{path: path, at: time.Now()}
			})
			// fileMode stays set; fileSubmitting prevents further edits until
			// the async result resets everything.

		case tea.KeyBackspace, tea.KeyCtrlH:
			// Ctrl+H is the ASCII backspace (0x08) sent by many terminals.
			if len(m.fileInput) > 0 {
				runes := []rune(m.fileInput)
				m.fileInput = string(runes[:len(runes)-1])
			}

		case tea.KeyCtrlW:
			// Delete the last word (like readline's Ctrl+W).
			m.fileInput = deleteLastWord(m.fileInput)

		case tea.KeyCtrlU:
			// Clear the entire input line (like readline's Ctrl+U).
			m.fileInput = ""

		default:
			// Accept all printable runes including / . ~ - _
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.fileInput += msg.String()
			}
		}
		return m, tea.Batch(cmds...)
	}

	// ── Wormhole receive mode: user typing a share code ───────────────────────
	if m.shareMode == shareReceiving {
		if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC {
			if m.shareCancel != nil {
				m.shareCancel()
			}
			m.shareMode = shareOff
			m.shareInput = ""
		}
		return m, tea.Batch(cmds...)
	}
	if m.shareMode == shareReceive {
		switch msg.Type {
		case tea.KeyEscape, tea.KeyCtrlC:
			if m.shareCancel != nil {
				m.shareCancel()
			}
			m.shareMode = shareOff
			m.shareInput = ""
		case tea.KeyEnter:
			code := strings.TrimSpace(m.shareInput)
			if code != "" {
				ctx, cancel := context.WithCancel(context.Background())
				m.shareCancel = cancel
				m.shareMode = shareReceiving
				s := m.storage
				st := m.state
				cmds = append(cmds, func() tea.Msg {
					res, err := core.ShareReceive(ctx, code)
					if err != nil {
						if ctx.Err() != nil {
							return nil
						}
						return shareErrMsg{err: err.Error()}
					}
					newTab := core.Tab{
						ID:        fmt.Sprintf("%x", time.Now().UnixNano()),
						Title:     res.TabTitle,
						Body:      res.Body,
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
					}
					st.Tabs = append(st.Tabs, newTab)
					st.ActiveIndex = len(st.Tabs) - 1
					s.Save(st)
					return shareReceivedMsg{title: res.TabTitle, st: st}
				})
			}
		case tea.KeyBackspace:
			if len(m.shareInput) > 0 {
				m.shareInput = m.shareInput[:len(m.shareInput)-1]
			}
		case tea.KeyRunes:
			m.shareInput += msg.String()
		}
		return m, tea.Batch(cmds...)
	}

	// ── Normal mode ───────────────────────────────────────────────────────────

	if m.previewMode {
		switch msg.Type {
		case tea.KeyCtrlP:
			m.previewMode = false
			idx := m.state.ActiveIndex
			if idx < len(m.textareas) {
				m.textareas[idx].Focus()
			}
			return m, nil

		case tea.KeyUp, tea.KeyRunes:
			if msg.Type == tea.KeyUp || msg.String() == "k" || msg.Type == tea.KeyCtrlY {
				m.previewScrollRow--
				m = m.clampScroll()
				return m, nil
			}
			if msg.String() == "j" || msg.Type == tea.KeyCtrlE {
				m.previewScrollRow++
				m = m.clampScroll()
				return m, nil
			}
			return m, nil

		case tea.KeyDown:
			m.previewScrollRow++
			m = m.clampScroll()
			return m, nil

		case tea.KeyPgUp:
			m.previewScrollRow -= m.getContentHeight()
			m = m.clampScroll()
			return m, nil

		case tea.KeyPgDown, tea.KeySpace:
			m.previewScrollRow += m.getContentHeight()
			m = m.clampScroll()
			return m, nil

		case tea.KeyHome:
			m.previewScrollRow = 0
			return m, nil

		case tea.KeyEnd:
			m.previewScrollRow = m.getActiveTabLinesCount() - m.getContentHeight()
			m = m.clampScroll()
			return m, nil

		case tea.KeyCtrlC:
			m.syncSaveNow()
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlRight, tea.KeyCtrlF, tea.KeyTab:
			m = m.switchTab((m.state.ActiveIndex + 1) % len(m.state.Tabs))
			m.previewScrollRow = 0
			return m, nil

		case tea.KeyCtrlLeft, tea.KeyCtrlB:
			idx := m.state.ActiveIndex - 1
			if idx < 0 {
				idx = len(m.state.Tabs) - 1
			}
			m = m.switchTab(idx)
			m.previewScrollRow = 0
			return m, nil

		case tea.KeyBackspace, tea.KeyDelete, tea.KeyEnter:
			return m, nil
		}
	}

	switch msg.Type {

	case tea.KeyCtrlP:
		m.previewMode = true
		m.previewScrollRow = 0
		idx := m.state.ActiveIndex
		if idx < len(m.textareas) {
			m.textareas[idx].Blur()
		}
		return m, nil

	case tea.KeyCtrlC:
		// Cancel sharing if active; otherwise quit.
		if m.shareMode == shareSending && m.shareCancel != nil {
			m.shareCancel()
			m.shareMode = shareOff
			m.shareCode = ""
			return m, nil
		}
		m.syncSaveNow()
		m.quitting = true
		return m, tea.Quit

	case tea.KeyCtrlRight, tea.KeyCtrlF:
		m = m.switchTab((m.state.ActiveIndex + 1) % len(m.state.Tabs))

	case tea.KeyCtrlLeft, tea.KeyCtrlB:
		idx := m.state.ActiveIndex - 1
		if idx < 0 {
			idx = len(m.state.Tabs) - 1
		}
		m = m.switchTab(idx)

	case tea.KeyTab:
		m = m.switchTab((m.state.ActiveIndex + 1) % len(m.state.Tabs))

	case tea.KeyCtrlN, tea.KeyF5:
		m = m.newTab()
		m.triggerSave()

	// Ctrl+X or F4 → close tab (Ctrl+W is intercepted by macOS Terminal.app)
	case tea.KeyCtrlX, tea.KeyF4:
		m, cmds = m.handleClose(cmds)

	// Ctrl+O or F3 → open file
	case tea.KeyCtrlO, tea.KeyF3:
		m.fileMode = filePromptOpen
		m.fileInput = ""
		m.fileSubmitting = true
		cmds = append(cmds, selectFileCmd)

	// Ctrl+S or F2 → save to disk
	case tea.KeyCtrlS, tea.KeyF2:
		m, cmds = m.handleSave(cmds)

	// Ctrl+T → share active tab via wormhole (T for Transfer)
	case tea.KeyCtrlT:
		cmds = m.doShare(cmds)

	// Ctrl+R → receive from wormhole
	case tea.KeyCtrlR:
		m.shareMode = shareReceive
		m.shareInput = ""

	default:
		// All other keys (printable runes, arrows, etc.) go to the active textarea.
		idx := m.state.ActiveIndex
		updated, cmd := m.textareas[idx].Update(msg)
		m.textareas[idx] = updated
		cmds = append(cmds, cmd)
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace ||
			msg.Type == tea.KeyDelete || msg.Type == tea.KeyEnter {
			m.syncTabBody(idx)
		}
	}

	return m, tea.Batch(cmds...)
}

// ── File / share helpers ──────────────────────────────────────────────────────

// handleClose implements Ctrl+X (close tab).
// Prompts before closing if there are unsaved-to-disk changes.
func (m model) handleClose(cmds []tea.Cmd) (model, []tea.Cmd) {
	idx := m.state.ActiveIndex
	tab := m.state.Tabs[idx]
	content := m.textareas[idx].Value()

	if tab.FilePath != "" && tab.FileIsDirty {
		m.fileMode = filePromptConfirm
		m.filePendingClose = true
		return m, cmds
	}

	if tab.FilePath == "" && strings.TrimSpace(content) != "" {
		// New tab with content — ask where to save before closing using native dialog.
		m.fileMode = filePromptSave
		m.fileInput = ""
		m.filePendingClose = true
		m.fileSubmitting = true
		return m, append(cmds, selectFileSaveCmd)
	}

	m = m.closeTab()
	m.triggerSave()
	return m, cmds
}

// handleSave implements Ctrl+S.
func (m model) handleSave(cmds []tea.Cmd) (model, []tea.Cmd) {
	idx := m.state.ActiveIndex
	path := m.state.Tabs[idx].FilePath
	content := m.textareas[idx].Value()

	if path != "" {
		// Known file → overwrite silently.
		cmds = append(cmds, func() tea.Msg {
			if err := core.SaveFile(path, content); err != nil {
				return fileErrMsg{err: err.Error()}
			}
			return fileSavedMsg{path: path, at: time.Now()}
		})
	} else {
		// New tab → prompt for destination path using native system dialog.
		m.fileMode = filePromptSave
		m.fileInput = ""
		m.fileSubmitting = true
		cmds = append(cmds, selectFileSaveCmd)
	}
	return m, cmds
}

// doShare starts a Magic Wormhole send (Ctrl+T).
func (m model) doShare(cmds []tea.Cmd) []tea.Cmd {
	if m.shareMode == shareSending {
		return cmds
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.shareCancel = cancel
	m.shareMode = shareSending
	m.shareCode = "connecting…"
	m.shareErr = ""
	tab := m.state.Tabs[m.state.ActiveIndex]
	cmds = append(cmds, func() tea.Msg {
		code, wait, err := core.ShareSend(ctx, tab, "")
		if err != nil {
			return shareErrMsg{err: err.Error()}
		}
		return shareStartedMsg{code: code, wait: wait}
	})
	return cmds
}

// syncTabBody copies the textarea value back to the state and marks dirty.
func (m *model) syncTabBody(idx int) {
	m.state.Tabs[idx].Body = m.textareas[idx].Value()
	m.state.Tabs[idx].CursorLine = m.textareas[idx].Line()
	m.state.Tabs[idx].UpdatedAt = time.Now()
	if m.state.Tabs[idx].FilePath != "" {
		m.state.Tabs[idx].FileIsDirty = true
	}
	m.dirty = true
	m.triggerSave()
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.quitting {
		return styleTitle.Render("✦ octonote — bye! 👋") + "\n"
	}
	if m.width == 0 {
		return "Loading…"
	}

	var b strings.Builder
	title := styleTitle.Render("✦ octonote")
	tabCount := styleTabCount.Render(fmt.Sprintf(" %d tab(s)", len(m.state.Tabs)))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, title, tabCount))
	b.WriteString("\n")
	b.WriteString(m.renderTabBar())
	b.WriteString("\n")
	b.WriteString(m.renderContent())
	b.WriteString("\n")
	b.WriteString(m.renderLegend())
	return b.String()
}

func (m model) renderTabBar() string {
	numTabs := len(m.state.Tabs)
	if numTabs == 0 {
		return ""
	}

	usableW := m.width - 4
	if usableW < 10 {
		usableW = 10
	}

	// 1. Determine dynamic limits based on the number of tabs
	var activeLabelLen, inactiveLabelLen, padding int
	switch {
	case numTabs <= 3:
		activeLabelLen = 16
		inactiveLabelLen = 12
		padding = 2
	case numTabs <= 6:
		activeLabelLen = 14
		inactiveLabelLen = 8
		padding = 1
	default:
		activeLabelLen = 12
		inactiveLabelLen = 5
		padding = 1
	}

	// Helper to calculate if tab is unsaved
	isTabUnsaved := func(tabIdx int) bool {
		if tabIdx >= len(m.textareas) {
			return false
		}
		tab := m.state.Tabs[tabIdx]
		return (tab.FilePath == "" && strings.TrimSpace(m.textareas[tabIdx].Value()) != "") ||
			(tab.FilePath != "" && tab.FileIsDirty)
	}

	// 2. Pre-calculate the outer width of each tab if it were rendered
	tabWidths := make([]int, numTabs)
	for i := range m.state.Tabs {
		unsavedLen := 0
		if isTabUnsaved(i) {
			unsavedLen = 2 // "● "
		}
		prefixLen := len(fmt.Sprintf(" %d: ", i+1))
		labelLen := inactiveLabelLen
		if i == m.state.ActiveIndex {
			labelLen = activeLabelLen
		}
		titleLen := utf8.RuneCountInString(m.state.Tabs[i].Title)
		if titleLen < labelLen {
			labelLen = titleLen
		}
		// Border (2) + Padding (2 * padding) + prefixLen + unsavedLen + labelLen
		tabWidths[i] = 2 + (2 * padding) + prefixLen + unsavedLen + labelLen
	}

	// 3. Find the sliding window [start, end] centered around the active tab
	start := m.state.ActiveIndex
	end := m.state.ActiveIndex
	currentWidth := tabWidths[m.state.ActiveIndex]
	indicatorWidth := 3 // " ◀ " and " ▶ " take 3 chars each

	for {
		expanded := false

		// Try expanding left
		if start > 0 {
			nextW := tabWidths[start-1]
			leftIndicatorSpace := 0
			if start-1 > 0 {
				leftIndicatorSpace = indicatorWidth
			}
			rightIndicatorSpace := 0
			if end < numTabs-1 {
				rightIndicatorSpace = indicatorWidth
			}

			if currentWidth+nextW+leftIndicatorSpace+rightIndicatorSpace <= usableW {
				start--
				currentWidth += nextW
				expanded = true
			}
		}

		// Try expanding right
		if end < numTabs-1 {
			nextW := tabWidths[end+1]
			leftIndicatorSpace := 0
			if start > 0 {
				leftIndicatorSpace = indicatorWidth
			}
			rightIndicatorSpace := 0
			if end+1 < numTabs-1 {
				rightIndicatorSpace = indicatorWidth
			}

			if currentWidth+nextW+leftIndicatorSpace+rightIndicatorSpace <= usableW {
				end++
				currentWidth += nextW
				expanded = true
			}
		}

		if !expanded {
			break
		}
	}

	// 4. Render only the tabs in the sliding window
	tabs := make([]string, 0, numTabs)
	styleIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color(colAccentLt)).
		Background(lipgloss.Color(colBg)).
		Padding(0, 1).
		Bold(true)

	if start > 0 {
		tabs = append(tabs, styleIndicator.Render("◀"))
	}

	for i := start; i <= end; i++ {
		tab := m.state.Tabs[i]
		
		// Determine label length limit
		limit := inactiveLabelLen
		if i == m.state.ActiveIndex {
			limit = activeLabelLen
		}
		label := truncate(tab.Title, limit)
		if isTabUnsaved(i) {
			label = "● " + label
		}

		var style lipgloss.Style
		if i == m.state.ActiveIndex {
			style = styleTabActive.Padding(0, padding)
			tabs = append(tabs, style.Render(fmt.Sprintf(" %d: %s ", i+1, label)))
		} else {
			style = styleTabInactive.Padding(0, padding)
			tabs = append(tabs, style.Render(fmt.Sprintf(" %d: %s ", i+1, label)))
		}
	}

	if end < numTabs-1 {
		tabs = append(tabs, styleIndicator.Render("▶"))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)
	return styleTabBar.Width(m.width).Render(row)
}

func (m model) renderContent() string {
	idx := m.state.ActiveIndex
	if idx >= len(m.textareas) {
		return ""
	}
	contentH := m.getContentHeight()
	contentW := m.width - 4
	m.textareas[idx].SetWidth(contentW)
	m.textareas[idx].SetHeight(contentH)

	var box lipgloss.Style
	if m.textareas[idx].Focused() || m.previewMode {
		box = styleContentBox
	} else {
		box = styleContentBoxBlur
	}

	if m.previewMode {
		markdownText := RenderMarkdown(m.textareas[idx].Value())
		lines := strings.Split(markdownText, "\n")
		
		// Clamp previewScrollRow locally for safe slice indexing
		linesCount := len(lines)
		maxScroll := linesCount - contentH
		if maxScroll < 0 {
			maxScroll = 0
		}
		scrollRow := m.previewScrollRow
		if scrollRow > maxScroll {
			scrollRow = maxScroll
		}
		if scrollRow < 0 {
			scrollRow = 0
		}

		end := scrollRow + contentH
		if end > len(lines) {
			end = len(lines)
		}
		visibleLines := lines[scrollRow:end]
		for len(visibleLines) < contentH {
			visibleLines = append(visibleLines, "")
		}
		previewBody := strings.Join(visibleLines, "\n")
		return box.Width(m.width - 2).Render(previewBody)
	}

	return box.Width(m.width - 2).Render(m.textareas[idx].View())
}

func (m model) renderLegend() string {
	// File I/O error banner.
	if m.fileErr != "" {
		return styleLegend.Width(m.width).Render(styleFileErr.Render("✗ " + m.fileErr))
	}

	// Close-confirm prompt.
	if m.fileMode == filePromptConfirm {
		msg := styleFileErr.Render("Unsaved changes!") +
			styleFilePrompt.Render("  Save before closing?  ") +
			styleKey.Render("Y") + " save  " +
			styleKey.Render("N") + " discard  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(msg)
	}

	// Open-file prompt.
	if m.fileMode == filePromptOpen {
		if m.fileSubmitting {
			statusText := "Opening system file picker…"
			if m.fileInput != "" {
				statusText = "Opening " + m.fileInput + " …"
			}
			return styleLegend.Width(m.width).Render(
				styleFilePrompt.Render(statusText),
			)
		}
		input := styleFileInput.Render(m.fileInput + "▌")
		prompt := styleFilePrompt.Render("Open: ") + input +
			styleFilePrompt.Render("  ") + styleKey.Render("↵") +
			styleFilePrompt.Render(" open  ") + styleKey.Render("^U") + " clear  " +
			styleKey.Render("^W") + " del-word  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}

	// Save-as prompt.
	if m.fileMode == filePromptSave {
		if m.fileSubmitting {
			statusText := "Opening system save dialog…"
			if m.fileInput != "" {
				statusText = "Saving " + m.fileInput + " …"
			}
			return styleLegend.Width(m.width).Render(
				styleFilePrompt.Render(statusText),
			)
		}
		input := styleFileInput.Render(m.fileInput + "▌")
		prompt := styleFilePrompt.Render("Save as: ") + input +
			styleFilePrompt.Render("  ") + styleKey.Render("↵") +
			styleFilePrompt.Render(" save  ") + styleKey.Render("^U") + " clear  " +
			styleKey.Render("^W") + " del-word  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}

	// Share overlays.
	if m.shareMode == shareSending {
		var status string
		if m.shareCode == "connecting…" {
			status = styleShareInfo.Render("opening wormhole…")
		} else {
			status = "share code: " + styleShareCode.Render(m.shareCode) +
				styleShareInfo.Render("  waiting for peer…  ") +
				styleKey.Render("^C") + " cancel"
		}
		return styleLegend.Width(m.width).Render(status)
	}
	if m.shareMode == shareReceive {
		input := styleShareCode.Render("_" + m.shareInput + "_")
		prompt := styleShareInfo.Render("enter code: ") + input +
			styleShareInfo.Render("  then ") + styleKey.Render("↵") +
			styleShareInfo.Render(" to connect  ") + styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}
	if m.shareMode == shareReceiving {
		return styleLegend.Width(m.width).Render(
			styleShareInfo.Render("connecting to peer…  ") + styleKey.Render("Esc") + " cancel",
		)
	}
	if m.shareErr != "" {
		return styleLegend.Width(m.width).Render(styleShareErr.Render("share error: " + m.shareErr))
	}

	// Normal legend.
	var shortcuts []struct{ key, desc string }
	if m.previewMode {
		shortcuts = []struct{ key, desc string }{
			{"^P", "edit"},
			{"↑/↓", "scroll"},
			{"PgUp/Dn", "page"},
			{"Home/End", "jump"},
			{"^→/←", "switch"},
			{"Tab", "cycle"},
			{"^C", "quit"},
		}
	} else {
		shortcuts = []struct{ key, desc string }{
			{"^P", "preview"},
			{"^N/F5", "new"},
			{"^X/F4", "close"},
			{"^O/F3", "open"},
			{"^S/F2", "save"},
			{"^T", "share"},
			{"^R", "receive"},
			{"^→/←", "switch"},
			{"Tab", "cycle"},
			{"^C", "quit"},
		}
	}
	var parts []string
	for _, s := range shortcuts {
		parts = append(parts, styleKey.Render(s.key)+" "+s.desc)
	}

	// Right-side save status and word count.
	idx := m.state.ActiveIndex
	tab := m.state.Tabs[idx]
	var saveStatus string
	unsaved := (tab.FilePath == "" && strings.TrimSpace(m.textareas[idx].Value()) != "") ||
		(tab.FilePath != "" && tab.FileIsDirty)

	words := len(strings.Fields(m.textareas[idx].Value()))
	wordStr := fmt.Sprintf("%d word", words)
	if words != 1 {
		wordStr += "s"
	}

	switch {
	case tab.FilePath != "" && tab.FileIsDirty:
		saveStatus = styleUnsaved.Render(fmt.Sprintf("%s │ ● %s (unsaved - ^S to save)", wordStr, filepath.Base(tab.FilePath)))
	case tab.FilePath != "" && !tab.FileIsDirty:
		saveStatus = styleSaved.Render(fmt.Sprintf("%s │ ✓ %s (saved)", wordStr, filepath.Base(tab.FilePath)))
	case tab.FilePath == "" && unsaved:
		saveStatus = styleUnsaved.Render(fmt.Sprintf("%s │ ● %s (unsaved to disk - ^S to save)", wordStr, tab.Title))
	default:
		saveStatus = styleSaved.Render(fmt.Sprintf("%s │ ✓ %s (auto-saved %s)", wordStr, tab.Title, m.lastSaved.Format("15:04:05")))
	}

	usableWidth := m.width - 2 // styleLegend has Padding(0, 1)
	if usableWidth < 1 {
		usableWidth = 1
	}

	var lines []string
	currentLine := ""
	for i, part := range parts {
		partLen := visibleLen(part)
		if currentLine == "" {
			currentLine = part
		} else {
			if visibleLen(currentLine)+2+partLen > usableWidth {
				lines = append(lines, currentLine)
				currentLine = part
			} else {
				currentLine += "  " + part
			}
		}
		if i == len(parts)-1 {
			saveLen := visibleLen(saveStatus)
			if visibleLen(currentLine)+1+saveLen > usableWidth {
				lines = append(lines, currentLine)
				currentLine = ""
			}
			gap := usableWidth - visibleLen(currentLine) - saveLen
			if gap < 0 {
				gap = 0
			}
			currentLine += strings.Repeat(" ", gap) + saveStatus
			lines = append(lines, currentLine)
		}
	}

	return styleLegend.Width(m.width).Render(strings.Join(lines, "\n"))
}

// ── Tab helpers ───────────────────────────────────────────────────────────────

func (m model) switchTab(idx int) model {
	if idx < 0 || idx >= len(m.state.Tabs) {
		return m
	}
	m.textareas[m.state.ActiveIndex].Blur()
	m.state.ActiveIndex = idx
	if !m.previewMode {
		m.textareas[idx].Focus()
	}
	m.triggerSave()
	return m
}

func (m model) newTab() model {
	m.previewMode = false
	title := fmt.Sprintf("tab %d", len(m.state.Tabs)+1)
	tab := core.NewTab(title)
	m.state.Tabs = append(m.state.Tabs, tab)
	ta := newTextArea()
	ta.Focus()
	m.textareas = append(m.textareas, ta)
	m.textareas[m.state.ActiveIndex].Blur()
	m.state.ActiveIndex = len(m.state.Tabs) - 1
	return m
}

func (m model) closeTab() model {
	m.previewMode = false
	if len(m.state.Tabs) <= 1 {
		m.textareas[0].Reset()
		m.state.Tabs[0].Body = ""
		m.state.Tabs[0].FilePath = ""
		m.state.Tabs[0].FileIsDirty = false
		m.state.Tabs[0].UpdatedAt = time.Now()
		m.textareas[0].Focus()
		return m
	}
	idx := m.state.ActiveIndex
	m.state.Tabs = append(m.state.Tabs[:idx], m.state.Tabs[idx+1:]...)
	m.textareas = append(m.textareas[:idx], m.textareas[idx+1:]...)
	if m.state.ActiveIndex >= len(m.state.Tabs) {
		m.state.ActiveIndex = len(m.state.Tabs) - 1
	}
	m.textareas[m.state.ActiveIndex].Focus()
	return m
}

// loadFileIntoTab puts file content into the current tab (if empty/new) or a new tab.
func (m model) loadFileIntoTab(path, content string) model {
	m.previewMode = false
	idx := m.state.ActiveIndex
	if strings.TrimSpace(m.textareas[idx].Value()) == "" && m.state.Tabs[idx].FilePath == "" {
		// Reuse current tab.
		m.state.Tabs[idx].Title = filepath.Base(path)
		m.state.Tabs[idx].Body = content
		m.state.Tabs[idx].FilePath = path
		m.state.Tabs[idx].FileIsDirty = false
		m.state.Tabs[idx].UpdatedAt = time.Now()
		m.textareas[idx].SetValue(content)
		m.textareas[idx].Focus()
	} else {
		// Open in a new tab.
		tab := core.NewTab(filepath.Base(path))
		tab.Body = content
		tab.FilePath = path
		m.state.Tabs = append(m.state.Tabs, tab)
		ta := newTextArea()
		ta.SetValue(content)
		ta.Focus()
		m.textareas = append(m.textareas, ta)
		m.textareas[m.state.ActiveIndex].Blur()
		m.state.ActiveIndex = len(m.state.Tabs) - 1
	}
	return m
}

func (m *model) triggerSave()  { m.storage.Save(m.state) }
func (m *model) syncSaveNow() { m.storage.Save(m.state) }

func (m model) resizeTextAreas() model {
	contentH := m.height - 8
	if contentH < 4 {
		contentH = 4
	}
	contentW := m.width - 4
	for i := range m.textareas {
		m.textareas[i].SetWidth(contentW)
		m.textareas[i].SetHeight(contentH)
	}
	return m
}

func (m model) getContentHeight() int {
	contentH := m.height - 8
	if contentH < 4 {
		contentH = 4
	}
	return contentH
}

func (m model) getActiveTabLinesCount() int {
	idx := m.state.ActiveIndex
	if idx >= len(m.textareas) {
		return 0
	}
	return len(strings.Split(m.textareas[idx].Value(), "\n"))
}

func (m model) clampScroll() model {
	contentH := m.getContentHeight()
	linesCount := m.getActiveTabLinesCount()
	maxScroll := linesCount - contentH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.previewScrollRow > maxScroll {
		m.previewScrollRow = maxScroll
	}
	if m.previewScrollRow < 0 {
		m.previewScrollRow = 0
	}
	return m
}

// ── Utilities ─────────────────────────────────────────────────────────────────

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max-1]) + "…"
}

func visibleLen(s string) int {
	inEscape := false
	count := 0
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		count++
	}
	return count
}

// deleteLastWord removes the last whitespace-delimited word from s,
// matching readline's Ctrl+W behaviour.
func deleteLastWord(s string) string {
	runes := []rune(strings.TrimRight(s, " \t"))
	// walk backwards over the last word
	i := len(runes) - 1
	for i >= 0 && runes[i] != ' ' && runes[i] != '/' && runes[i] != '\\' {
		i--
	}
	if i < 0 {
		return ""
	}
	return string(runes[:i+1])
}

// ── Main ──────────────────────────────────────────────────────────────────────

var version = "1.4.1"

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "-v" || arg == "--version" || arg == "-version" {
			fmt.Printf("octonote v%s\n", version)
			os.Exit(0)
		}
		if arg == "--update" || arg == "-update" {
			updateCommand()
		}
		if arg == "-h" || arg == "--help" || arg == "-help" {
			fmt.Printf("octonote v%s - Lightweight multi-tab auto-saving terminal scratchpad\n\n", version)
			fmt.Println("Usage:")
			fmt.Println("  octonote                   Open the scratchpad")
			fmt.Println("  octonote -v, --version     Print the version")
			fmt.Println("  octonote --update          Check for and install updates")
			fmt.Println("  octonote -h, --help        Show this help message")
			os.Exit(0)
		}
	}

	s, err := core.NewStorage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "octonote: %v\n", err)
		os.Exit(1)
	}
	defer s.Close()

	st, err := s.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "octonote: load state: %v\n", err)
		os.Exit(1)
	}

	m := initialModel(s, st)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go s.Watch(ctx, func() {
		p.Send(externalStateUpdateMsg{})
	})

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "octonote: %v\n", err)
		os.Exit(1)
	}
}

func updateCommand() {
	fmt.Println("Checking for updates...")
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/divyo-argha/octonote/releases/latest", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", "octonote-updater")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking for updates: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "GitHub API returned status: %s\n", resp.Status)
		os.Exit(1)
	}

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing update info: %v\n", err)
		os.Exit(1)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	if !isNewerVersion(latest, version) {
		fmt.Printf("octonote is already up-to-date (v%s).\n", version)
		os.Exit(0)
	}

	fmt.Printf("\nA new version of octonote is available: v%s -> v%s\n", version, latest)

	execPath, err := os.Executable()
	if err == nil {
		// Detect npm installation
		if strings.Contains(execPath, "node_modules") || strings.Contains(execPath, "npm") {
			fmt.Println("To update, please run:")
			fmt.Println("  npm install -g octonote@latest")
			os.Exit(0)
		}
	}

	fmt.Print("Would you like to download and install this update? (y/N): ")
	var answer string
	fmt.Scanln(&answer)
	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" {
		fmt.Println("Update cancelled.")
		os.Exit(0)
	}

	// Standalone binary self-update
	fmt.Println("Downloading update...")
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		fmt.Fprintf(os.Stderr, "Unsupported architecture for self-update: %s. Please update manually.\n", runtime.GOARCH)
		os.Exit(1)
	}

	var osName string
	switch runtime.GOOS {
	case "darwin":
		osName = "darwin"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "windows"
	default:
		fmt.Fprintf(os.Stderr, "Unsupported OS for self-update: %s. Please update manually.\n", runtime.GOOS)
		os.Exit(1)
	}

	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	binaryName := fmt.Sprintf("octonote-%s-%s%s", osName, arch, ext)
	downloadURL := fmt.Sprintf("https://github.com/divyo-argha/octonote/releases/download/v%s/%s", latest, binaryName)

	resp, err = http.Get(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading binary: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Failed to download update binary from URL: %s (Status: %s)\n", downloadURL, resp.Status)
		os.Exit(1)
	}

	tmpPath := execPath + ".tmp"
	out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		os.Exit(1)
	}

	_, err = io.Copy(out, resp.Body)
	out.Close() // Close file write handle
	if err != nil {
		os.Remove(tmpPath)
		fmt.Fprintf(os.Stderr, "Error saving binary: %v\n", err)
		os.Exit(1)
	}

	// Rename dance
	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)
	err = os.Rename(execPath, oldPath)
	if err != nil {
		// Try direct overwrite (Unix)
		err = os.Rename(tmpPath, execPath)
		if err != nil {
			os.Remove(tmpPath)
			fmt.Fprintf(os.Stderr, "Error replacing binary: %v\n", err)
			os.Exit(1)
		}
	} else {
		err = os.Rename(tmpPath, execPath)
		if err != nil {
			_ = os.Rename(oldPath, execPath) // restore
			os.Remove(tmpPath)
			fmt.Fprintf(os.Stderr, "Error replacing binary: %v\n", err)
			os.Exit(1)
		}
		_ = os.Remove(oldPath)
	}

	fmt.Println("✓ Successfully updated octonote!")
	os.Exit(0)
}

func isNewerVersion(latest, current string) bool {
	lParts := strings.Split(strings.TrimPrefix(latest, "v"), ".")
	cParts := strings.Split(strings.TrimPrefix(current, "v"), ".")
	for i := 0; i < len(lParts) && i < len(cParts); i++ {
		var lVal, cVal int
		fmt.Sscanf(lParts[i], "%d", &lVal)
		fmt.Sscanf(cParts[i], "%d", &cVal)
		if lVal > cVal {
			return true
		}
		if lVal < cVal {
			return false
		}
	}
	return len(lParts) > len(cParts)
}

