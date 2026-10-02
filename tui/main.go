package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
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
	"golang.org/x/term"
)

var version = "2.2.0"

// ── Color Palette & Styles ───────────────────────────────────────────────────

const (
	colBg            = "#09090b"
	colSurface       = "#121217"
	colSurfaceActive = "#1c1c24"
	colBorder        = "#272732"
	colBorderFocus   = "#6366f1"
	colAccent        = "#6366f1"
	colAccentLt      = "#818cf8"
	colAccentGlow    = "#a5b4fc"
	colMuted         = "#94a3b8"
	colText          = "#f8fafc"
	colSubtle        = "#64748b"
	colWarn          = "#f59e0b"
	colSuccess       = "#10b981"
	colTabBg         = "#18181f"
	colErr           = "#ef4444"
)

var (
	styleBrand = lipgloss.NewStyle().
			Background(lipgloss.Color(colAccent)).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)

	styleVersionPill = lipgloss.NewStyle().
				Background(lipgloss.Color(colSurfaceActive)).
				Foreground(lipgloss.Color(colAccentLt)).
				Padding(0, 1)

	styleTabCountPill = lipgloss.NewStyle().
				Background(lipgloss.Color(colSurface)).
				Foreground(lipgloss.Color(colMuted)).
				Padding(0, 1)

	styleHeaderMeta = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colSubtle))

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
			BorderForeground(lipgloss.Color(colBorderFocus)).
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
			Background(lipgloss.Color(colSurfaceActive)).
			Foreground(lipgloss.Color(colAccentLt)).
			Padding(0, 1).
			Bold(true)

	styleModePill = lipgloss.NewStyle().
			Background(lipgloss.Color(colAccent)).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)

	styleSaved = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colSuccess)).
			Bold(true)

	styleUnsaved = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colWarn)).
			Bold(true)

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

	styleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colAccent)).
			Background(lipgloss.Color(colSurface)).
			Padding(1, 2)
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

// ── Mode Enums ────────────────────────────────────────────────────────────────

type shareMode int

const (
	shareOff shareMode = iota
	shareSending
	shareReceive
	shareReceiving
)

type filePromptMode int

const (
	filePromptOff filePromptMode = iota
	filePromptOpen
	filePromptSave
	filePromptConfirm
)

const (
	previewOff   = 0
	previewSplit = 1
	previewFull  = 2
)

// ── Model ─────────────────────────────────────────────────────────────────────

type model struct {
	storage   *core.Storage
	state     core.State
	textareas []textarea.Model
	width     int
	height    int
	lastSaved time.Time
	dirty     bool
	quitting  bool

	// preview state
	previewMode      int // previewOff (0), previewSplit (1), previewFull (2)
	previewScrollRow int

	// share state
	shareMode   shareMode
	shareCode   string
	shareInput  string
	shareErr    string
	shareCancel context.CancelFunc

	// file I/O state
	fileMode         filePromptMode
	fileInput        string
	fileErr          string
	filePendingClose bool
	fileSubmitting   bool

	// tab renaming state
	renameMode  bool
	renameInput string

	// find / search state
	findMode     bool
	findInput    string
	findMatches  []int
	findMatchIdx int

	// help overlay modal
	helpMode bool
}

func initialModel(s *core.Storage, st core.State) model {
	tas := make([]textarea.Model, len(st.Tabs))
	for i, tab := range st.Tabs {
		tas[i] = newTextArea()
		tas[i].SetValue(tab.Body)
	}

	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		w = 90
		h = 28
	}

	previewInit := previewOff
	if w >= 80 {
		previewInit = previewSplit
	}

	m := model{
		storage:     s,
		state:       st,
		textareas:   tas,
		width:       w,
		height:      h,
		previewMode: previewInit,
		lastSaved:   time.Now(),
	}

	if m.state.ActiveIndex < len(m.textareas) {
		m.textareas[m.state.ActiveIndex].Focus()
	}

	m = m.resizeTextAreas()
	return m
}

func newTextArea() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Start typing your scratch notes… (Ctrl+P for markdown preview, Ctrl+S to save to disk)"
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(20)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(lipgloss.Color("#181824"))
	ta.FocusedStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(colText))
	ta.BlurredStyle.Base = lipgloss.NewStyle().Foreground(lipgloss.Color(colMuted))
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colSubtle))
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color(colBorder))
	return ta
}

func (m model) Init() tea.Cmd { return textarea.Blink }

// ── Update ────────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		if msg.Width > 0 && msg.Height > 0 {
			m.width = msg.Width
			m.height = msg.Height
			m = m.resizeTextAreas()
		}

	case externalStateUpdateMsg:
		newSt, err := m.storage.Load()
		if err == nil {
			m.state = newSt
			tas := make([]textarea.Model, len(m.state.Tabs))
			for i, tab := range m.state.Tabs {
				tas[i] = newTextArea()
				tas[i].SetValue(tab.Body)
			}
			m.textareas = tas
			if m.state.ActiveIndex < len(m.textareas) {
				if m.previewMode != previewFull {
					m.textareas[m.state.ActiveIndex].Focus()
				}
			}
			m = m.resizeTextAreas()
		}
		return m, nil

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
		m.previewMode = previewOff
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
		idx := m.state.ActiveIndex
		m.state.Tabs[idx].FilePath = msg.path
		m.state.Tabs[idx].Title = filepath.Base(msg.path)
		m.state.Tabs[idx].FileIsDirty = false
		m.lastSaved = msg.at
		m.dirty = false
		m.fileMode = filePromptOff
		m.fileInput = ""
		m.fileErr = ""
		m.fileSubmitting = false
		if m.filePendingClose {
			m.filePendingClose = false
			m = m.closeTab()
		}
		m.triggerSave()

	case fileErrMsg:
		m.fileErr = msg.err
		m.fileSubmitting = false

	case savedMsg:
		m.lastSaved = msg.at
		m.dirty = false

	case tea.KeyMsg:
		return m.handleKey(msg, cmds)
	}

	// Propagate non-key messages to active textarea
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

func (m model) handleKey(msg tea.KeyMsg, cmds []tea.Cmd) (tea.Model, tea.Cmd) {
	m.fileErr = ""
	m.shareErr = ""

	// ── Help Modal Overlay ────────────────────────────────────────────────────
	if m.helpMode {
		if msg.Type == tea.KeyEscape || msg.Type == tea.KeyCtrlC || msg.String() == "q" || msg.Type == tea.KeyF1 {
			m.helpMode = false
		}
		return m, nil
	}

	// ── Tab Renaming Mode ─────────────────────────────────────────────────────
	if m.renameMode {
		switch msg.Type {
		case tea.KeyEscape, tea.KeyCtrlC:
			m.renameMode = false
			m.renameInput = ""
		case tea.KeyEnter:
			name := strings.TrimSpace(m.renameInput)
			if name != "" {
				idx := m.state.ActiveIndex
				m.state.Tabs[idx].Title = name
				m.state.Tabs[idx].UpdatedAt = time.Now()
				m.triggerSave()
			}
			m.renameMode = false
			m.renameInput = ""
		case tea.KeyBackspace, tea.KeyCtrlH:
			if len(m.renameInput) > 0 {
				runes := []rune(m.renameInput)
				m.renameInput = string(runes[:len(runes)-1])
			}
		case tea.KeyCtrlU:
			m.renameInput = ""
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.renameInput += msg.String()
			}
		}
		return m, nil
	}

	// ── Find / Search Mode ────────────────────────────────────────────────────
	if m.findMode {
		switch msg.Type {
		case tea.KeyEscape, tea.KeyCtrlC:
			m.findMode = false
			m.findInput = ""
			m.findMatches = nil
		case tea.KeyEnter:
			if len(m.findMatches) > 0 {
				m.findMatchIdx = (m.findMatchIdx + 1) % len(m.findMatches)
				idx := m.state.ActiveIndex
				m.textareas[idx].SetCursor(m.findMatches[m.findMatchIdx])
			}
		case tea.KeyBackspace, tea.KeyCtrlH:
			if len(m.findInput) > 0 {
				runes := []rune(m.findInput)
				m.findInput = string(runes[:len(runes)-1])
				m.updateFindMatches()
			}
		case tea.KeyCtrlU:
			m.findInput = ""
			m.updateFindMatches()
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.findInput += msg.String()
				m.updateFindMatches()
			}
		}
		return m, nil
	}

	// ── File Close Confirm (Y / N / Esc) ──────────────────────────────────────
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

	// ── File Path Prompts (Open / Save-As) ────────────────────────────────────
	if m.fileMode == filePromptOpen || m.fileMode == filePromptSave {
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
					c, err := core.OpenFile(path)
					if err != nil {
						return fileErrMsg{err: err.Error()}
					}
					return fileOpenedMsg{path: path, content: c}
				}
				if err := core.SaveFile(path, content); err != nil {
					return fileErrMsg{err: err.Error()}
				}
				return fileSavedMsg{path: path, at: time.Now()}
			})
		case tea.KeyBackspace, tea.KeyCtrlH:
			if len(m.fileInput) > 0 {
				runes := []rune(m.fileInput)
				m.fileInput = string(runes[:len(runes)-1])
			}
		case tea.KeyCtrlW:
			m.fileInput = deleteLastWord(m.fileInput)
		case tea.KeyCtrlU:
			m.fileInput = ""
		default:
			if msg.Type == tea.KeyRunes || msg.Type == tea.KeySpace {
				m.fileInput += msg.String()
			}
		}
		return m, tea.Batch(cmds...)
	}

	// ── Wormhole Share Handling ───────────────────────────────────────────────
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

	// ── Full Preview Mode ─────────────────────────────────────────────────────
	if m.previewMode == previewFull {
		switch msg.Type {
		case tea.KeyCtrlP, tea.KeyEsc:
			m.previewMode = previewOff
			m = m.resizeTextAreas()
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
			if msg.String() == "?" {
				m.helpMode = true
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

		case tea.KeyF1:
			m.helpMode = true
			return m, nil

		case tea.KeyCtrlC:
			m.syncSaveNow()
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlRight, tea.KeyTab:
			m = m.switchTab((m.state.ActiveIndex + 1) % len(m.state.Tabs))
			m.previewScrollRow = 0
			return m, nil

		case tea.KeyCtrlLeft, tea.KeyShiftTab:
			idx := m.state.ActiveIndex - 1
			if idx < 0 {
				idx = len(m.state.Tabs) - 1
			}
			m = m.switchTab(idx)
			m.previewScrollRow = 0
			return m, nil
		}
	}

	// ── Standard Normal Mode ──────────────────────────────────────────────────
	switch msg.Type {

	// Help overlay (F1)
	case tea.KeyF1:
		m.helpMode = true
		return m, nil

	// Markdown Preview toggle (Ctrl+P) - cycles Split Live Preview -> Full Preview -> Off
	case tea.KeyCtrlP:
		if m.previewMode == previewOff {
			if m.width >= 70 {
				m.previewMode = previewSplit
				m = m.resizeTextAreas()
				idx := m.state.ActiveIndex
				if idx < len(m.textareas) {
					m.textareas[idx].Focus()
				}
			} else {
				m.previewMode = previewFull
				m = m.resizeTextAreas()
				idx := m.state.ActiveIndex
				if idx < len(m.textareas) {
					m.textareas[idx].Blur()
				}
			}
		} else if m.previewMode == previewSplit {
			m.previewMode = previewFull
			m = m.resizeTextAreas()
			idx := m.state.ActiveIndex
			if idx < len(m.textareas) {
				m.textareas[idx].Blur()
			}
		} else {
			m.previewMode = previewOff
			m = m.resizeTextAreas()
			idx := m.state.ActiveIndex
			if idx < len(m.textareas) {
				m.textareas[idx].Focus()
			}
		}
		m.previewScrollRow = 0
		return m, nil

	// Quit (Ctrl+C)
	case tea.KeyCtrlC:
		if m.shareMode == shareSending && m.shareCancel != nil {
			m.shareCancel()
			m.shareMode = shareOff
			m.shareCode = ""
			return m, nil
		}
		m.syncSaveNow()
		m.quitting = true
		return m, tea.Quit

	// Tab switching
	case tea.KeyCtrlRight:
		m = m.switchTab((m.state.ActiveIndex + 1) % len(m.state.Tabs))
	case tea.KeyCtrlLeft:
		idx := m.state.ActiveIndex - 1
		if idx < 0 {
			idx = len(m.state.Tabs) - 1
		}
		m = m.switchTab(idx)
	case tea.KeyTab:
		m = m.switchTab((m.state.ActiveIndex + 1) % len(m.state.Tabs))
	case tea.KeyShiftTab:
		idx := m.state.ActiveIndex - 1
		if idx < 0 {
			idx = len(m.state.Tabs) - 1
		}
		m = m.switchTab(idx)

	// New Tab (Ctrl+N or F5)
	case tea.KeyCtrlN, tea.KeyF5:
		m = m.newTab()
		m.triggerSave()

	// Close Tab (Ctrl+W, Ctrl+X, or F4)
	case tea.KeyCtrlW, tea.KeyCtrlX, tea.KeyF4:
		m, cmds = m.handleClose(cmds)

	// Rename Tab (F6 or Ctrl+E)
	case tea.KeyF6, tea.KeyCtrlE:
		m.renameMode = true
		m.renameInput = m.state.Tabs[m.state.ActiveIndex].Title

	// Find in current note (Ctrl+F)
	case tea.KeyCtrlF:
		m.findMode = true
		m.findInput = ""
		m.findMatches = nil
		m.findMatchIdx = 0

	// Open File (Ctrl+O or F3)
	case tea.KeyCtrlO, tea.KeyF3:
		m.fileMode = filePromptOpen
		m.fileInput = ""
		m.fileSubmitting = true
		cmds = append(cmds, selectFileCmd)

	// Save File to Disk (Ctrl+S or F2)
	case tea.KeyCtrlS, tea.KeyF2:
		m, cmds = m.handleSave(cmds)

	// Wormhole Transfer (Ctrl+T)
	case tea.KeyCtrlT:
		cmds = m.doShare(cmds)

	// Wormhole Receive (Ctrl+R)
	case tea.KeyCtrlR:
		m.shareMode = shareReceive
		m.shareInput = ""

	default:
		// Check for Alt+1 .. Alt+9 direct tab jumps
		if msg.Alt && len(msg.Runes) > 0 {
			r := msg.Runes[0]
			if r >= '1' && r <= '9' {
				target := int(r - '1')
				if target < len(m.state.Tabs) {
					m = m.switchTab(target)
					return m, nil
				}
			}
		}

		// Textarea input
		idx := m.state.ActiveIndex
		updated, cmd := m.textareas[idx].Update(msg)
		m.textareas[idx] = updated
		cmds = append(cmds, cmd)
		if msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace ||
			msg.Type == tea.KeyDelete || msg.Type == tea.KeyEnter {
			if m.previewMode == previewOff && m.width >= 70 {
				m.previewMode = previewSplit
				m = m.resizeTextAreas()
			}
			m.syncTabBody(idx)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) updateFindMatches() {
	if m.findInput == "" {
		m.findMatches = nil
		m.findMatchIdx = 0
		return
	}
	body := strings.ToLower(m.textareas[m.state.ActiveIndex].Value())
	query := strings.ToLower(m.findInput)
	m.findMatches = nil
	m.findMatchIdx = 0

	pos := 0
	for {
		idx := strings.Index(body[pos:], query)
		if idx == -1 {
			break
		}
		m.findMatches = append(m.findMatches, pos+idx)
		pos += idx + len(query)
	}

	if len(m.findMatches) > 0 {
		m.textareas[m.state.ActiveIndex].SetCursor(m.findMatches[0])
	}
}

// ── Tab & File Operations ─────────────────────────────────────────────────────

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

func (m model) handleSave(cmds []tea.Cmd) (model, []tea.Cmd) {
	idx := m.state.ActiveIndex
	path := m.state.Tabs[idx].FilePath
	content := m.textareas[idx].Value()

	if path != "" {
		cmds = append(cmds, func() tea.Msg {
			if err := core.SaveFile(path, content); err != nil {
				return fileErrMsg{err: err.Error()}
			}
			return fileSavedMsg{path: path, at: time.Now()}
		})
	} else {
		m.fileMode = filePromptSave
		m.fileInput = ""
		m.fileSubmitting = true
		cmds = append(cmds, selectFileSaveCmd)
	}
	return m, cmds
}

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

func (m model) switchTab(idx int) model {
	if idx < 0 || idx >= len(m.state.Tabs) {
		return m
	}
	m.textareas[m.state.ActiveIndex].Blur()
	m.state.ActiveIndex = idx
	if m.previewMode != previewFull {
		m.textareas[idx].Focus()
	}
	m.triggerSave()
	return m
}

func (m model) newTab() model {
	if m.width >= 70 {
		m.previewMode = previewSplit
	} else {
		m.previewMode = previewOff
	}
	m = m.resizeTextAreas()
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
	m.previewMode = previewOff
	m = m.resizeTextAreas()
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

func (m model) loadFileIntoTab(path, content string) model {
	m.previewMode = previewOff
	m = m.resizeTextAreas()
	idx := m.state.ActiveIndex
	if strings.TrimSpace(m.textareas[idx].Value()) == "" && m.state.Tabs[idx].FilePath == "" {
		m.state.Tabs[idx].Title = filepath.Base(path)
		m.state.Tabs[idx].Body = content
		m.state.Tabs[idx].FilePath = path
		m.state.Tabs[idx].FileIsDirty = false
		m.state.Tabs[idx].UpdatedAt = time.Now()
		m.textareas[idx].SetValue(content)
		m.textareas[idx].Focus()
	} else {
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

func (m *model) triggerSave() { m.storage.Save(m.state) }
func (m *model) syncSaveNow() { m.storage.Save(m.state) }

func (m model) resizeTextAreas() model {
	contentH := m.getContentHeight()
	contentW := m.width - 4
	if m.previewMode == previewSplit && m.width >= 70 {
		contentW = (m.width - 4) / 2
	}
	if contentW < 10 {
		contentW = 10
	}
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

// ── View Rendering ────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.quitting {
		return styleBrand.Render(" ✦ octonote — saved. See you next time! 👋 ") + "\n"
	}

	if m.width <= 0 {
		m.width = 90
		m.height = 28
		m = m.resizeTextAreas()
	}

	// 1. Header Row
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// 2. Tab Bar
	b.WriteString(m.renderTabBar())
	b.WriteString("\n")

	// 3. Main Content or Help Overlay
	if m.helpMode {
		b.WriteString(m.renderHelpModal())
	} else {
		b.WriteString(m.renderContent())
	}
	b.WriteString("\n")

	// 4. Status Bar & Interactive Prompts
	b.WriteString(m.renderLegend())
	return b.String()
}

func (m model) renderHeader() string {
	idx := m.state.ActiveIndex
	ta := m.textareas[idx]
	words := len(strings.Fields(ta.Value()))
	chars := len([]rune(ta.Value()))
	line := ta.Line() + 1

	left := lipgloss.JoinHorizontal(
		lipgloss.Center,
		styleBrand.Render("✦ octonote"),
		" ",
		styleVersionPill.Render("v"+version),
		" ",
		styleTabCountPill.Render(fmt.Sprintf("%d/%d tabs", idx+1, len(m.state.Tabs))),
	)

	right := styleHeaderMeta.Render(fmt.Sprintf("Ln %d  │  %d words · %d chars  ", line, words, chars))

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
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

	var activeLabelLen, inactiveLabelLen, padding int
	switch {
	case numTabs <= 3:
		activeLabelLen = 18
		inactiveLabelLen = 14
		padding = 2
	case numTabs <= 6:
		activeLabelLen = 14
		inactiveLabelLen = 10
		padding = 1
	default:
		activeLabelLen = 12
		inactiveLabelLen = 6
		padding = 1
	}

	isTabUnsaved := func(tabIdx int) bool {
		if tabIdx >= len(m.textareas) {
			return false
		}
		tab := m.state.Tabs[tabIdx]
		return (tab.FilePath == "" && strings.TrimSpace(m.textareas[tabIdx].Value()) != "") ||
			(tab.FilePath != "" && tab.FileIsDirty)
	}

	tabWidths := make([]int, numTabs)
	for i := range m.state.Tabs {
		unsavedLen := 0
		if isTabUnsaved(i) {
			unsavedLen = 2
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
		tabWidths[i] = 2 + (2 * padding) + prefixLen + unsavedLen + labelLen
	}

	start := m.state.ActiveIndex
	end := m.state.ActiveIndex
	currentWidth := tabWidths[m.state.ActiveIndex]
	indicatorWidth := 3

	for {
		expanded := false
		if start > 0 {
			nextW := tabWidths[start-1]
			leftSpace := 0
			if start-1 > 0 {
				leftSpace = indicatorWidth
			}
			rightSpace := 0
			if end < numTabs-1 {
				rightSpace = indicatorWidth
			}
			if currentWidth+nextW+leftSpace+rightSpace <= usableW {
				start--
				currentWidth += nextW
				expanded = true
			}
		}
		if end < numTabs-1 {
			nextW := tabWidths[end+1]
			leftSpace := 0
			if start > 0 {
				leftSpace = indicatorWidth
			}
			rightSpace := 0
			if end+1 < numTabs-1 {
				rightSpace = indicatorWidth
			}
			if currentWidth+nextW+leftSpace+rightSpace <= usableW {
				end++
				currentWidth += nextW
				expanded = true
			}
		}
		if !expanded {
			break
		}
	}

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
		limit := inactiveLabelLen
		if i == m.state.ActiveIndex {
			limit = activeLabelLen
		}
		label := truncate(tab.Title, limit)
		if isTabUnsaved(i) {
			label = "● " + label
		}

		icon := "📄"
		if tab.Pinned {
			icon = "📌"
		}

		if i == m.state.ActiveIndex {
			style := styleTabActive.Padding(0, padding)
			tabs = append(tabs, style.Render(fmt.Sprintf("%s %d: %s", icon, i+1, label)))
		} else {
			style := styleTabInactive.Padding(0, padding)
			tabs = append(tabs, style.Render(fmt.Sprintf("%d: %s", i+1, label)))
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

	// 1. Full Preview Mode
	if m.previewMode == previewFull {
		markdownText := RenderMarkdown(m.textareas[idx].Value())
		lines := strings.Split(markdownText, "\n")
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
		return styleContentBox.Width(m.width - 2).Render(previewBody)
	}

	// 2. Split Live Preview Mode (Editor on Left, Live Preview on Right)
	if m.previewMode == previewSplit && m.width >= 70 {
		halfW := (m.width - 4) / 2
		rightW := m.width - 2 - halfW - 2
		if halfW < 10 {
			halfW = 10
		}
		if rightW < 10 {
			rightW = 10
		}

		m.textareas[idx].SetWidth(halfW - 2)
		m.textareas[idx].SetHeight(contentH)

		leftBox := styleContentBox.Width(halfW).Render(m.textareas[idx].View())

		markdownText := RenderMarkdown(m.textareas[idx].Value())
		lines := strings.Split(markdownText, "\n")
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
		rightBox := styleContentBoxBlur.Width(rightW).Render(previewBody)

		return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	}

	// 3. Normal Editor Only
	contentW := m.width - 4
	m.textareas[idx].SetWidth(contentW)
	m.textareas[idx].SetHeight(contentH)

	var box lipgloss.Style
	if m.textareas[idx].Focused() {
		box = styleContentBox
	} else {
		box = styleContentBoxBlur
	}
	return box.Width(m.width - 2).Render(m.textareas[idx].View())
}

func (m model) renderHelpModal() string {
	contentH := m.getContentHeight()
	w := m.width - 10
	if w < 40 {
		w = 40
	}

	helpText := strings.Join([]string{
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colAccentLt)).Render("octoNote — Keyboard Cheat Sheet"),
		"─────────────────────────────────────────────────────────────",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colText)).Render("Navigation & Tabs:"),
		"  Tab / Shift+Tab   Cycle between scratch tabs",
		"  Ctrl+Right / Left Move to next / previous tab",
		"  Alt+1 ... Alt+9   Jump directly to tabs 1 through 9",
		"  Ctrl+N  /  F5     Create a new scratch tab",
		"  Ctrl+W  /  Ctrl+X Close current tab (prompts if unsaved)",
		"  Ctrl+E  /  F6     Rename active tab",
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colText)).Render("Editing & Search:"),
		"  Ctrl+P            Toggle rich Markdown Preview",
		"  Ctrl+F            Find / Search inside current note (Enter for next)",
		"  Ctrl+S  /  F2     Save note to disk file",
		"  Ctrl+O  /  F3     Open an external file into a tab",
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colText)).Render("P2P Magic Wormhole:"),
		"  Ctrl+T            Transfer / share active note to a peer",
		"  Ctrl+R            Receive shared note with wormhole code",
		"",
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colMuted)).Render("Press Esc, ?, or F1 to return to editing"),
	}, "\n")

	card := styleModal.Width(w).Render(helpText)
	return styleContentBox.Width(m.width - 2).Height(contentH + 2).Render(card)
}

func (m model) renderLegend() string {
	// 1. Errors
	if m.fileErr != "" {
		return styleLegend.Width(m.width).Render(styleFileErr.Render("✗ " + m.fileErr))
	}

	// 2. Tab Rename Prompt
	if m.renameMode {
		prompt := lipgloss.NewStyle().Foreground(lipgloss.Color(colAccentLt)).Bold(true).Render("✏️  Rename Tab: ") +
			styleFileInput.Render(m.renameInput+"▌") + "  " +
			styleKey.Render("↵") + " confirm  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}

	// 3. Find Prompt
	if m.findMode {
		countStr := "no matches"
		if len(m.findMatches) > 0 {
			countStr = fmt.Sprintf("match %d of %d", m.findMatchIdx+1, len(m.findMatches))
		}
		prompt := lipgloss.NewStyle().Foreground(lipgloss.Color(colAccentLt)).Bold(true).Render("🔍 Find: ") +
			styleFileInput.Render(m.findInput+"▌") + "  " +
			styleHeaderMeta.Render("("+countStr+")") + "  " +
			styleKey.Render("↵") + " next  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}

	// 4. File Confirmation Prompt
	if m.fileMode == filePromptConfirm {
		msg := styleFileErr.Render("Unsaved changes!") +
			styleFilePrompt.Render("  Save before closing?  ") +
			styleKey.Render("Y") + " save  " +
			styleKey.Render("N") + " discard  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(msg)
	}

	// 5. Open File Prompt
	if m.fileMode == filePromptOpen {
		if m.fileSubmitting {
			statusText := "Opening file picker…"
			if m.fileInput != "" {
				statusText = "Opening " + m.fileInput + " …"
			}
			return styleLegend.Width(m.width).Render(styleFilePrompt.Render(statusText))
		}
		input := styleFileInput.Render(m.fileInput + "▌")
		prompt := styleFilePrompt.Render("Open file: ") + input +
			"  " + styleKey.Render("↵") + " open  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}

	// 6. Save File Prompt
	if m.fileMode == filePromptSave {
		if m.fileSubmitting {
			statusText := "Saving file…"
			if m.fileInput != "" {
				statusText = "Saving " + m.fileInput + " …"
			}
			return styleLegend.Width(m.width).Render(styleFilePrompt.Render(statusText))
		}
		input := styleFileInput.Render(m.fileInput + "▌")
		prompt := styleFilePrompt.Render("Save to disk: ") + input +
			"  " + styleKey.Render("↵") + " save  " +
			styleKey.Render("Esc") + " cancel"
		return styleLegend.Width(m.width).Render(prompt)
	}

	// 7. Wormhole Overlays
	if m.shareMode == shareSending {
		var status string
		if m.shareCode == "connecting…" {
			status = styleShareInfo.Render("opening wormhole…")
		} else {
			status = "wormhole code: " + styleShareCode.Render(m.shareCode) +
				styleShareInfo.Render("  waiting for peer…  ") +
				styleKey.Render("^C") + " cancel"
		}
		return styleLegend.Width(m.width).Render(status)
	}
	if m.shareMode == shareReceive {
		input := styleShareCode.Render("_" + m.shareInput + "_")
		prompt := styleShareInfo.Render("enter wormhole code: ") + input +
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

	// 8. Default Legend
	modeText := "EDIT"
	if m.previewMode == previewFull {
		modeText = "PREVIEW"
	} else if m.previewMode == previewSplit {
		modeText = "SPLIT PREVIEW"
	}
	modePill := styleModePill.Render(modeText)

	var shortcuts []struct{ key, desc string }
	if m.previewMode == previewFull {
		shortcuts = []struct{ key, desc string }{
			{"^P", "edit"},
			{"↑/↓", "scroll"},
			{"^F", "find"},
			{"Tab", "tabs"},
			{"?", "help"},
			{"^C", "quit"},
		}
	} else if m.previewMode == previewSplit {
		shortcuts = []struct{ key, desc string }{
			{"^P", "full"},
			{"^N", "new"},
			{"^W", "close"},
			{"^F", "find"},
			{"^S", "save"},
			{"Tab", "tabs"},
			{"F1", "help"},
			{"^C", "quit"},
		}
	} else {
		shortcuts = []struct{ key, desc string }{
			{"^P", "split"},
			{"^N", "new"},
			{"^W", "close"},
			{"^E", "rename"},
			{"^F", "find"},
			{"^S", "save"},
			{"^O", "open"},
			{"^T", "share"},
			{"Tab", "cycle"},
			{"F1", "help"},
			{"^C", "quit"},
		}
	}

	var parts []string
	parts = append(parts, modePill)
	for _, s := range shortcuts {
		parts = append(parts, styleKey.Render(s.key)+" "+s.desc)
	}

	idx := m.state.ActiveIndex
	tab := m.state.Tabs[idx]
	var saveStatus string
	unsaved := (tab.FilePath == "" && strings.TrimSpace(m.textareas[idx].Value()) != "") ||
		(tab.FilePath != "" && tab.FileIsDirty)

	switch {
	case tab.FilePath != "" && tab.FileIsDirty:
		saveStatus = styleUnsaved.Render("● " + filepath.Base(tab.FilePath) + " (unsaved - ^S)")
	case tab.FilePath != "" && !tab.FileIsDirty:
		saveStatus = styleSaved.Render("✓ " + filepath.Base(tab.FilePath) + " (saved)")
	case tab.FilePath == "" && unsaved:
		saveStatus = styleUnsaved.Render("● " + tab.Title + " (unsaved to disk - ^S)")
	default:
		saveStatus = styleSaved.Render("✓ saved " + m.lastSaved.Format("15:04:05"))
	}

	usableWidth := m.width - 2
	if usableWidth < 10 {
		usableWidth = 10
	}

	left := strings.Join(parts, " ")
	gap := usableWidth - visibleLen(left) - visibleLen(saveStatus)
	if gap < 2 {
		return styleLegend.Width(m.width).Render(left + "\n" + saveStatus)
	}

	return styleLegend.Width(m.width).Render(left + strings.Repeat(" ", gap) + saveStatus)
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

func deleteLastWord(s string) string {
	runes := []rune(strings.TrimRight(s, " \t"))
	i := len(runes) - 1
	for i >= 0 && runes[i] != ' ' && runes[i] != '/' && runes[i] != '\\' {
		i--
	}
	if i < 0 {
		return ""
	}
	return string(runes[:i+1])
}

func launchGUI() error {
	if runtime.GOOS == "darwin" {
		candidates := []string{
			"gui/build/bin/octoNote.app",
			"./octonote-gui",
			"/Applications/octoNote.app",
			filepath.Join(os.Getenv("HOME"), "Applications/octoNote.app"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				cmd := exec.Command("open", c)
				return cmd.Start()
			}
		}
	} else if runtime.GOOS == "windows" {
		candidates := []string{
			"octonote-gui.exe",
			"./octonote-gui.exe",
			"gui/build/bin/octonote.exe",
			filepath.Join(os.Getenv("USERPROFILE"), "octonote-gui.exe"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				cmd := exec.Command("cmd.exe", "/C", "start", "", c)
				return cmd.Start()
			}
		}
		if path, err := exec.LookPath("octonote-gui.exe"); err == nil {
			cmd := exec.Command("cmd.exe", "/C", "start", "", path)
			return cmd.Start()
		}
	} else {
		candidates := []string{
			"./octonote-gui",
			"gui/build/bin/octonote",
			"/usr/local/bin/octonote-gui",
			"/usr/bin/octonote-gui",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				cmd := exec.Command(c)
				return cmd.Start()
			}
		}
		if path, err := exec.LookPath("octonote-gui"); err == nil {
			cmd := exec.Command(path)
			return cmd.Start()
		}
	}

	return fmt.Errorf("octoNote desktop GUI app not found. Build it with 'make gui'")
}

func openFileIntoState(st *core.State, path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	// Check if already open
	for i, t := range st.Tabs {
		if t.FilePath == absPath {
			st.ActiveIndex = i
			return
		}
	}

	content, err := os.ReadFile(absPath)
	body := ""
	if err == nil {
		body = string(content)
	}
	title := filepath.Base(absPath)
	tab := core.Tab{
		ID:          fmt.Sprintf("%x", time.Now().UnixNano()),
		Title:       title,
		Body:        body,
		FilePath:    absPath,
		FileIsDirty: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	st.Tabs = append(st.Tabs, tab)
	st.ActiveIndex = len(st.Tabs) - 1
}

// ── Main Entrypoint ───────────────────────────────────────────────────────────

func main() {
	var targetFile string

	if len(os.Args) > 1 {
		arg := os.Args[1]

		// 1. Version
		if arg == "-v" || arg == "--version" || arg == "-version" {
			fmt.Printf("octonote v%s\n", version)
			os.Exit(0)
		}

		// 2. Help
		if arg == "-h" || arg == "--help" || arg == "-help" {
			fmt.Printf("octonote v%s — Lightning-fast, crash-proof multi-tab scratchpad\n\n", version)
			fmt.Println("Usage:")
			fmt.Println("  octonote                    Open the terminal scratchpad")
			fmt.Println("  octonote <file>             Open or edit a file in a scratchpad tab")
			fmt.Println("  octonote gui, --gui         Launch the desktop GUI application")
			fmt.Println("  octonote -v, --version      Print version")
			fmt.Println("  octonote --update           Check for and install updates")
			fmt.Println("  octonote -h, --help         Show this help message")
			fmt.Println("\nKeybindings in TUI:")
			fmt.Println("  Tab / Shift+Tab             Cycle tabs")
			fmt.Println("  Alt+1 ... Alt+9             Jump to tab 1-9")
			fmt.Println("  Ctrl+N / F5                 New tab")
			fmt.Println("  Ctrl+W / Ctrl+X / F4        Close tab")
			fmt.Println("  Ctrl+E / F6                 Rename tab")
			fmt.Println("  Ctrl+F                      Find in note")
			fmt.Println("  Ctrl+P                      Toggle Markdown preview")
			fmt.Println("  Ctrl+S / F2                 Save note to disk")
			fmt.Println("  Ctrl+O / F3                 Open file")
			fmt.Println("  Ctrl+T / Ctrl+R             P2P Magic Wormhole share & receive")
			fmt.Println("  F1 / ?                      Quick cheat sheet")
			fmt.Println("  Ctrl+C                      Quit (everything auto-saved)")
			os.Exit(0)
		}

		// 3. Desktop GUI launch
		if arg == "gui" || arg == "--gui" || arg == "-g" || arg == "app" {
			if err := launchGUI(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ octoNote desktop app launched.")
			os.Exit(0)
		}

		// 4. Update
		if arg == "--update" || arg == "-update" {
			updateCommand()
		}

		// 5. File argument
		if !strings.HasPrefix(arg, "-") {
			targetFile = arg
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
		st = core.State{
			Version:     2,
			ActiveIndex: 0,
			Tabs:        []core.Tab{core.NewTab("scratch")},
		}
	}

	if targetFile != "" {
		openFileIntoState(&st, targetFile)
		s.Save(st)
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

	fmt.Println("Downloading update...")
	var arch string
	switch runtime.GOARCH {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	default:
		fmt.Fprintf(os.Stderr, "Unsupported architecture: %s\n", runtime.GOARCH)
		os.Exit(1)
	}

	var osName string
	var ext string
	switch runtime.GOOS {
	case "darwin":
		osName = "darwin"
	case "linux":
		osName = "linux"
	case "windows":
		osName = "windows"
		ext = ".exe"
	default:
		fmt.Fprintf(os.Stderr, "Unsupported OS: %s\n", runtime.GOOS)
		os.Exit(1)
	}

	binaryName := fmt.Sprintf("octonote-%s-%s%s", osName, arch, ext)
	downloadURL := fmt.Sprintf("https://github.com/divyo-argha/octonote/releases/download/%s/%s", rel.TagName, binaryName)

	resp, err = client.Get(downloadURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error downloading binary: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: download URL returned status %s\n", resp.Status)
		os.Exit(1)
	}

	tmpFile, err := os.CreateTemp("", "octonote-update-*"+ext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp file: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, resp.Body)
	tmpFile.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error saving binary: %v\n", err)
		os.Exit(1)
	}

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting permissions: %v\n", err)
		os.Exit(1)
	}

	oldPath := execPath + ".old"
	_ = os.Remove(oldPath)
	err = os.Rename(execPath, oldPath)
	if err != nil {
		err = copyFile(tmpFile.Name(), execPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error replacing binary: %v\n", err)
			os.Exit(1)
		}
	} else {
		err = os.Rename(tmpFile.Name(), execPath)
		if err != nil {
			_ = copyFile(tmpFile.Name(), execPath)
		}
	}

	_ = os.Remove(oldPath)
	fmt.Printf("\n✓ Successfully updated octonote to v%s!\n", latest)
	os.Exit(0)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func isNewerVersion(latest, current string) bool {
	var lMajor, lMinor, lPatch int
	var cMajor, cMinor, cPatch int

	fmt.Sscanf(latest, "%d.%d.%d", &lMajor, &lMinor, &lPatch)
	fmt.Sscanf(current, "%d.%d.%d", &cMajor, &cMinor, &cPatch)

	if lMajor > cMajor {
		return true
	}
	if lMajor == cMajor && lMinor > cMinor {
		return true
	}
	if lMajor == cMajor && lMinor == cMinor && lPatch > cPatch {
		return true
	}
	return false
}
