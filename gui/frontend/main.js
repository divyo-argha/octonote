/* ══════════════════════════════════════════════════════════════════════════════
   octoNote v2.0 — Modern Frontend Logic & State Controller
   ══════════════════════════════════════════════════════════════════════════════ */

// ── DOM Element References ───────────────────────────────────────────────────

const titlebar             = document.getElementById('titlebar');
const tabbar               = document.getElementById('tabbar');
const btnNewTab            = document.getElementById('btn-new-tab');
const btnSidebarToggle     = document.getElementById('btn-sidebar-toggle');
const sidebarPane          = document.getElementById('sidebar-pane');
const editor               = document.getElementById('editor');
const editorMain           = document.getElementById('editor-main');
const editorHighlight      = document.getElementById('editor-highlight');
const editorActiveLine     = document.getElementById('editor-active-line');
const lineGutter           = document.getElementById('line-gutter');
const previewPane          = document.getElementById('preview-pane');

// Action Buttons
const btnCommandPalette    = document.getElementById('btn-command-palette');
// Lock variables removed
const btnPreview           = document.getElementById('btn-preview');
const btnZen               = document.getElementById('btn-zen');
const btnMinimize          = document.getElementById('btn-minimize');
const btnMaximize          = document.getElementById('btn-maximize');
const btnClose             = document.getElementById('btn-close');

// Sidebar Elements
const sidebarSearchInput   = document.getElementById('sidebar-search-input');
const sidebarNotesList     = document.getElementById('sidebar-notes-list');
const btnSidebarNewNote    = document.getElementById('btn-sidebar-new-note');

// Dev Tools Elements
const btnToolJsonPrettify  = document.getElementById('btn-tool-json-prettify');
const btnToolJsonMinify    = document.getElementById('btn-tool-json-minify');
const metricWords          = document.getElementById('metric-words');
const metricChars          = document.getElementById('metric-chars');
const metricLines          = document.getElementById('metric-lines');
const metricReading        = document.getElementById('metric-reading');

// Status Bar Elements
const statusSave           = document.getElementById('status-save');
const statusSaveText       = document.getElementById('status-save-text');
const statusFile           = document.getElementById('status-file');
// Removed lock state status
const statusReading        = document.getElementById('status-reading');
const statusWords          = document.getElementById('status-words');
const statusChars          = document.getElementById('status-chars');
const statusPos            = document.getElementById('status-pos');
const statusTabs           = document.getElementById('status-tabs');

// Command Palette Elements
const commandPalette       = document.getElementById('command-palette');
const commandPaletteInput  = document.getElementById('command-palette-input');
const commandPaletteResults= document.getElementById('command-palette-results');

// Find and Replace Elements
const searchBar            = document.getElementById('search-bar');
const searchFindInput      = document.getElementById('search-find-input');
const searchReplaceInput   = document.getElementById('search-replace-input');
const searchCount          = document.getElementById('search-count');
const btnSearchPrev        = document.getElementById('btn-search-prev');
const btnSearchNext        = document.getElementById('btn-search-next');
const btnSearchClose       = document.getElementById('btn-search-close');
const btnReplaceOne        = document.getElementById('btn-replace-one');
const btnReplaceAll        = document.getElementById('btn-replace-all');

let searchMatches = [];
let currentSearchMatchIndex = -1;

// Toast Container
const toastContainer       = document.getElementById('toast-container');

// ── Application State ─────────────────────────────────────────────────────────

let state = { tabs: [], active_index: 0 };
let isSidebarOpen = true;
let isPreviewOpen = false;
let isZenMode = false;
let saveTimer = null;
let commandPaletteSelectedIndex = 0;
let filteredCommands = [];

// Settings (persisted to localStorage)
const settings = JSON.parse(localStorage.getItem('octonote_settings')) || {
  theme: 'dark',
  font: 'jetbrains',
  fontSize: 15,
  wordWrap: true,
  lineNumbers: true,
  activeLine: true,
  tabLayout: 'horizontal',
};

// ── Theme & Preferences Controller ───────────────────────────────────────────

function applySettings() {
  document.documentElement.setAttribute('data-theme', settings.theme);
  
  // Tab Layout
  if (tabbar) {
    tabbar.style.display = (settings.tabLayout === 'vertical') ? 'none' : 'flex';
  }

  // Font Family
  if (settings.font === 'fira') {
    editor.style.fontFamily = "'Fira Code', monospace";
  } else if (settings.font === 'inter') {
    editor.style.fontFamily = "'Inter', sans-serif";
  } else if (settings.font === 'outfit') {
    editor.style.fontFamily = "'Outfit', sans-serif";
  } else if (settings.font === 'system') {
    editor.style.fontFamily = "monospace";
  } else {
    editor.style.fontFamily = "'JetBrains Mono', monospace";
  }

  // Font Size
  editor.style.fontSize = settings.fontSize + 'px';
  editorHighlight.style.fontSize = settings.fontSize + 'px';
  lineGutter.style.fontSize = settings.fontSize + 'px';

  const fontSizeVal = document.getElementById('setting-fontsize-val');
  if (fontSizeVal) fontSizeVal.textContent = settings.fontSize + 'px';

  // Word Wrap
  editor.style.whiteSpace = settings.wordWrap ? 'pre-wrap' : 'pre';
  editorHighlight.style.whiteSpace = settings.wordWrap ? 'pre-wrap' : 'pre';

  // Line Numbers
  lineGutter.style.display = settings.lineNumbers ? 'block' : 'none';

  // Active Line
  editorActiveLine.style.display = settings.activeLine ? 'block' : 'none';

  // Update theme pills active state
  document.querySelectorAll('.theme-pill').forEach(pill => {
    pill.classList.toggle('active', pill.getAttribute('data-theme') === settings.theme);
  });

  const tabLayoutSel = document.getElementById('setting-tab-layout');
  if (tabLayoutSel) tabLayoutSel.value = settings.tabLayout || 'horizontal';

  localStorage.setItem('octonote_settings', JSON.stringify(settings));
}

// ── Init & State Sync ─────────────────────────────────────────────────────────

window.addEventListener('DOMContentLoaded', () => {
  applySettings();
  setupEventListeners();
  loadStateFromBackend();

  // Listen to state changes from Wails backend
  if (window.runtime) {
    window.runtime.EventsOn('state:changed', (newState) => {
      if (!newState || !newState.tabs) return;
      const currentActive = state.active_index;
      state = newState;
      if (currentActive >= 0 && currentActive < state.tabs.length) {
        state.active_index = currentActive;
      }
      renderTabs();
      renderSidebarNotes();
      updateEditorContent();
    });
  }
});

async function loadStateFromBackend() {
  if (window.go && window.go.main && window.go.main.App) {
    try {
      state = await window.go.main.App.GetState();
      renderTabs();
      renderSidebarNotes();
      updateEditorContent();
    } catch (err) {
      console.error('Failed to load state:', err);
    }
  }
}

// ── Event Listeners Setup ─────────────────────────────────────────────────────

function setupEventListeners() {
  // Sidebar Toggle
  btnSidebarToggle.addEventListener('click', toggleSidebar);

  // New Tab
  btnNewTab.addEventListener('click', createNewTab);
  btnSidebarNewNote.addEventListener('click', createNewTab);

  // Editor Input & Cursor Events
  editor.addEventListener('input', handleEditorInput);
  editor.addEventListener('keyup', updateCursorPosAndMetrics);
  editor.addEventListener('click', updateCursorPosAndMetrics);
  editor.addEventListener('scroll', syncEditorScroll);

  // Sidebar Tab Switcher
  document.querySelectorAll('.sidebar-tab').forEach(tabBtn => {
    tabBtn.addEventListener('click', (e) => {
      document.querySelectorAll('.sidebar-tab').forEach(t => t.classList.remove('sidebar-tab--active'));
      document.querySelectorAll('.sidebar-view').forEach(v => v.classList.remove('sidebar-view--active'));
      
      const targetView = tabBtn.getAttribute('data-view');
      tabBtn.classList.add('sidebar-tab--active');
      document.getElementById('view-' + targetView).classList.add('sidebar-view--active');
    });
  });

  // Sidebar Notes Search
  sidebarSearchInput.addEventListener('input', () => {
    renderSidebarNotes();
  });

  // Command Palette
  btnCommandPalette.addEventListener('click', showCommandPalette);
  
  // Find and Replace
  searchFindInput.addEventListener('input', updateSearchMatches);
  searchFindInput.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      e.shiftKey ? navigateSearch(-1) : navigateSearch(1);
    } else if (e.key === 'Escape') {
      toggleSearchBar();
    }
  });
  btnSearchNext.addEventListener('click', () => navigateSearch(1));
  btnSearchPrev.addEventListener('click', () => navigateSearch(-1));
  btnSearchClose.addEventListener('click', toggleSearchBar);
  btnReplaceOne.addEventListener('click', replaceCurrentMatch);
  btnReplaceAll.addEventListener('click', replaceAllMatches);

  // Removed btnLockToggle
  btnPreview.addEventListener('click', togglePreview);
  btnZen.addEventListener('click', toggleZenMode);

  // Window Controls
  if (btnMinimize) btnMinimize.addEventListener('click', () => window.runtime?.WindowMinimise());
  if (btnMaximize) btnMaximize.addEventListener('click', () => window.runtime?.WindowToggleMaximise());
  if (btnClose) btnClose.addEventListener('click', () => window.runtime?.Quit());

  // Dev Tools Buttons
  if (btnToolJsonPrettify) btnToolJsonPrettify.addEventListener('click', () => runJsonFormat(false));
  if (btnToolJsonMinify) btnToolJsonMinify.addEventListener('click', () => runJsonFormat(true));

  // Case Conversion Buttons
  document.querySelectorAll('.tool-btn-case').forEach(btn => {
    btn.addEventListener('click', () => {
      const mode = btn.getAttribute('data-case');
      runCaseTransform(mode);
    });
  });

  // Quick Template Buttons
  document.querySelectorAll('.tool-btn-template').forEach(btn => {
    btn.addEventListener('click', () => {
      const tmplName = btn.getAttribute('data-template');
      insertTemplate(tmplName);
    });
  });

  // Theme Pills in Settings
  document.querySelectorAll('.theme-pill').forEach(pill => {
    pill.addEventListener('click', () => {
      settings.theme = pill.getAttribute('data-theme');
      applySettings();
      showToast(`Theme changed to ${settings.theme}`);
    });
  });

  // Settings Controls
  document.getElementById('setting-tab-layout')?.addEventListener('change', (e) => {
    settings.tabLayout = e.target.value;
    applySettings();
  });

  document.getElementById('setting-font')?.addEventListener('change', (e) => {
    settings.font = e.target.value;
    applySettings();
  });

  document.getElementById('setting-fontsize')?.addEventListener('input', (e) => {
    settings.fontSize = parseInt(e.target.value, 10);
    applySettings();
  });

  document.getElementById('setting-wordwrap')?.addEventListener('change', (e) => {
    settings.wordWrap = e.target.checked;
    applySettings();
  });

  document.getElementById('setting-linenums')?.addEventListener('change', (e) => {
    settings.lineNumbers = e.target.checked;
    applySettings();
  });

  document.getElementById('setting-activeline')?.addEventListener('change', (e) => {
    settings.activeLine = e.target.checked;
    applySettings();
  });

  // Export ZIP
  document.getElementById('btn-export-zip')?.addEventListener('click', async () => {
    if (window.go?.main?.App) {
      showToast('Exporting scratchpads…');
    }
  });

  // Command Palette Keyboard Navigation
  commandPaletteInput.addEventListener('input', renderCommandPaletteResults);
  commandPaletteInput.addEventListener('keydown', handleCommandPaletteKeydown);
  commandPalette.querySelector('.command-palette__backdrop').addEventListener('click', hideCommandPalette);

  // Keyboard Shortcuts (Global)
  window.addEventListener('keydown', handleGlobalKeydown);
}

// ── Tab Management ────────────────────────────────────────────────────────────

function renderTabs() {
  // Clear non-new tabs
  const existingTabs = tabbar.querySelectorAll('.tab:not(.tab--new)');
  existingTabs.forEach(t => t.remove());

  if (!state.tabs || state.tabs.length === 0) return;

  state.tabs.forEach((tab, index) => {
    const tabEl = document.createElement('button');
    const dirtyClass = tab.file_is_dirty ? 'tab--dirty' : '';
    tabEl.className = `tab ${index === state.active_index ? 'tab--active' : ''} ${dirtyClass}`;
    tabEl.setAttribute('role', 'tab');
    tabEl.setAttribute('aria-selected', index === state.active_index ? 'true' : 'false');
    tabEl.style.setProperty('--wails-draggable', 'no-drag');

    let titleText = tab.title || `tab ${index + 1}`;
    let pinHtml = tab.pinned ? '<svg class="tab__pin-icon" viewBox="0 0 24 24" fill="currentColor"><path d="M16 12V4h1V2H7v2h1v8l-2 2v2h5v6l1 1 1-1v-6h5v-2l-2-2z"/></svg>' : '';
    let dirtyHtml = tab.file_is_dirty ? '<span class="tab__dirty-dot" title="Unsaved changes"></span>' : '';

    tabEl.innerHTML = `
      ${pinHtml}
      <span class="tab__title">${escapeHtml(titleText)}</span>
      ${dirtyHtml}
      ${state.tabs.length > 1 && !tab.pinned ? `<span class="tab__close" title="Close tab">&times;</span>` : ''}
    `;

    // Tab click
    tabEl.addEventListener('click', (e) => {
      if (e.target.classList.contains('tab__close')) {
        e.stopPropagation();
        closeTab(index);
      } else {
        switchActiveTab(index);
      }
    });

    // Double click to rename
    tabEl.addEventListener('dblclick', () => {
      const newTitle = prompt('Rename scratchpad:', tab.title);
      if (newTitle !== null) {
        renameTab(index, newTitle);
      }
    });

    tabbar.insertBefore(tabEl, btnNewTab);
  });

  statusTabs.textContent = `${state.tabs.length} tab${state.tabs.length > 1 ? 's' : ''}`;
}

async function createNewTab() {
  if (window.go?.main?.App) {
    state = await window.go.main.App.NewTab();
    renderTabs();
    renderSidebarNotes();
    updateEditorContent();
  }
}

async function switchActiveTab(index) {
  if (index < 0 || index >= state.tabs.length) return;
  if (index === state.active_index) return;

  // Flush pending save for current tab before switching
  if (saveTimer) {
    clearTimeout(saveTimer);
    saveTimer = null;
    const oldIdx = state.active_index;
    const body = editor.value;
    if (window.go?.main?.App && state.tabs[oldIdx]) {
      state.tabs[oldIdx].body = body;
      window.go.main.App.SaveTab(oldIdx, body, 0);
    }
  }

  state.active_index = index;
  renderTabs();
  renderSidebarNotes();
  updateEditorContent();

  if (window.go?.main?.App) {
    window.go.main.App.SetActiveTab(index);
  }
}

async function closeTab(index) {
  if (window.go?.main?.App) {
    state = await window.go.main.App.CloseTab(index);
    renderTabs();
    renderSidebarNotes();
    updateEditorContent();
  }
}

async function renameTab(index, title) {
  if (window.go?.main?.App) {
    state = await window.go.main.App.RenameTab(index, title);
    renderTabs();
    renderSidebarNotes();
  }
}

// ── Sidebar Explorer ──────────────────────────────────────────────────────────

function renderSidebarNotes() {
  sidebarNotesList.innerHTML = '';
  if (!state.tabs) return;

  const query = sidebarSearchInput.value.toLowerCase().trim();

  state.tabs.forEach((tab, index) => {
    if (query && !tab.title.toLowerCase().includes(query) && !tab.body.toLowerCase().includes(query)) {
      return;
    }

    const card = document.createElement('div');
    const dirtyCardClass = tab.file_is_dirty ? 'note-item-card--dirty' : '';
    card.className = `note-item-card ${index === state.active_index ? 'note-item-card--active' : ''} ${dirtyCardClass}`;
    
    const wordCount = tab.body ? tab.body.trim().split(/\s+/).filter(Boolean).length : 0;
    const dirtyTagHtml = tab.file_is_dirty ? '<span class="note-dirty-tag">Unsaved</span>' : '';

    card.innerHTML = `
      <div class="note-item-header">
        <span class="note-item-title">${escapeHtml(tab.title || 'Untitled')}</span>
        ${tab.pinned ? '📌' : ''}
        ${dirtyTagHtml}
      </div>
      <div class="note-item-sub">${wordCount} words • ${tab.body.length} chars</div>
    `;

    card.addEventListener('click', () => switchActiveTab(index));
    sidebarNotesList.appendChild(card);
  });
}

function toggleSidebar() {
  isSidebarOpen = !isSidebarOpen;
  sidebarPane.classList.toggle('sidebar-pane--collapsed', !isSidebarOpen);
  btnSidebarToggle.setAttribute('aria-expanded', isSidebarOpen ? 'true' : 'false');
}

// ── Editor & Auto-Save ────────────────────────────────────────────────────────

function updateEditorContent() {
  if (!state.tabs || state.tabs.length === 0) return;
  const activeTab = state.tabs[state.active_index];
  if (!activeTab) return;

  if (editor.value !== activeTab.body) {
    editor.value = activeTab.body || '';
  }

  statusFile.textContent = activeTab.file_path || '';
  statusFile.hidden = !activeTab.file_path;

  updateLineNumbers();
  updateCursorPosAndMetrics();
  updateEditorDirtyUI(!!activeTab.file_is_dirty);
  if (isPreviewOpen) renderMarkdownPreview();
}

function handleEditorInput() {
  if (state.tabs && state.tabs[state.active_index]) {
    state.tabs[state.active_index].body = editor.value;
    state.tabs[state.active_index].file_is_dirty = true;
    updateTabDirtyState(state.active_index, true);
    updateEditorDirtyUI(true, 'unsaved');
  }
  updateLineNumbers();
  updateCursorPosAndMetrics();
  triggerAutoSave();
  if (isPreviewOpen) renderMarkdownPreview();
}

function triggerAutoSave() {
  statusSaveText.textContent = 'Saving…';
  statusSave.classList.remove('status-indicator--saved');
  updateEditorDirtyUI(true, 'saving');

  const activeIdx = state.active_index;
  const currentBody = editor.value;

  clearTimeout(saveTimer);
  saveTimer = setTimeout(async () => {
    if (window.go?.main?.App && state.tabs[activeIdx]) {
      const lines = currentBody.substring(0, editor.selectionStart || 0).split('\n');
      const cursorLine = lines.length - 1;

      await window.go.main.App.SaveTab(activeIdx, currentBody, cursorLine);
      if (state.tabs[activeIdx]) {
        state.tabs[activeIdx].file_is_dirty = false;
      }
      if (activeIdx === state.active_index) {
        statusSaveText.textContent = 'Saved';
        statusSave.classList.add('status-indicator--saved');
        updateTabDirtyState(activeIdx, false);
        updateEditorDirtyUI(false);
      }
    }
  }, 300);
}

function updateTabDirtyState(index, isDirty) {
  const tabs = tabbar.querySelectorAll('.tab:not(.tab--new)');
  const tabEl = tabs[index];
  if (!tabEl) return;

  tabEl.classList.toggle('tab--dirty', isDirty);
  let dirtyDot = tabEl.querySelector('.tab__dirty-dot');

  if (isDirty) {
    if (!dirtyDot) {
      dirtyDot = document.createElement('span');
      dirtyDot.className = 'tab__dirty-dot';
      dirtyDot.title = 'Unsaved changes';
      const closeBtn = tabEl.querySelector('.tab__close');
      if (closeBtn) {
        tabEl.insertBefore(dirtyDot, closeBtn);
      } else {
        tabEl.appendChild(dirtyDot);
      }
    }
  } else {
    if (dirtyDot) {
      dirtyDot.remove();
    }
  }
}

let savedBadgeTimer = null;

function updateEditorDirtyUI(isDirty, status = 'unsaved') {
  let dirtyBadge = document.getElementById('editor-dirty-badge');
  if (!dirtyBadge) {
    dirtyBadge = document.createElement('div');
    dirtyBadge.id = 'editor-dirty-badge';
    dirtyBadge.className = 'editor-dirty-badge';
    editorMain.prepend(dirtyBadge);
  }

  clearTimeout(savedBadgeTimer);

  if (isDirty) {
    editorMain.classList.add('editor-main--dirty');
    dirtyBadge.classList.remove('editor-dirty-badge--saved');
    dirtyBadge.hidden = false;
    if (status === 'saving') {
      dirtyBadge.innerHTML = `<span class="dirty-badge-dot dirty-badge-dot--saving"></span> Saving…`;
    } else {
      dirtyBadge.innerHTML = `<span class="dirty-badge-dot dirty-badge-dot--unsaved"></span> Unsaved`;
    }
  } else {
    editorMain.classList.remove('editor-main--dirty');
    dirtyBadge.classList.add('editor-dirty-badge--saved');
    dirtyBadge.innerHTML = `<span class="dirty-badge-dot dirty-badge-dot--saved"></span> Saved`;
    dirtyBadge.hidden = false;

    savedBadgeTimer = setTimeout(() => {
      dirtyBadge.hidden = true;
    }, 1500);
  }
}

function updateLineNumbers() {
  if (!settings.lineNumbers) return;
  const lines = editor.value.split('\n').length;
  let numbersHtml = '';
  for (let i = 1; i <= lines; i++) {
    numbersHtml += `<div>${i}</div>`;
  }
  lineGutter.innerHTML = numbersHtml;
}

function updateCursorPosAndMetrics() {
  const val = editor.value;
  const selStart = editor.selectionStart;

  // Cursor position
  const lines = val.substr(0, selStart).split('\n');
  const lineNum = lines.length;
  const colNum = lines[lines.length - 1].length + 1;
  statusPos.textContent = `Ln ${lineNum}, Col ${colNum}`;

  // Active line highlight offset
  const lineHeight = 22.4; // 14px * 1.6
  editorActiveLine.style.top = (12 + (lineNum - 1) * lineHeight) + 'px';

  // Metrics
  const chars = val.length;
  const words = val.trim() ? val.trim().split(/\s+/).filter(Boolean).length : 0;
  const lineCount = val ? val.split('\n').length : 0;

  statusWords.textContent = `${words}w`;
  statusChars.textContent = `${chars}c`;

  if (metricWords) metricWords.textContent = words;
  if (metricChars) metricChars.textContent = chars;
  if (metricLines) metricLines.textContent = lineCount;

  // Update real-time counts in the sidebar Notes Explorer
  const activeSidebarNoteSub = document.querySelector('.note-item-card--active .note-item-sub');
  if (activeSidebarNoteSub) {
    activeSidebarNoteSub.textContent = `${words} words • ${chars} chars`;
  }

  // Reading time
  const readingSec = Math.ceil((words / 200) * 60);
  const readingText = readingSec >= 60 ? `${Math.floor(readingSec / 60)} min read` : `${readingSec} sec read`;
  statusReading.textContent = `⏱ ${readingText}`;
  if (metricReading) metricReading.textContent = readingText;
}

function syncEditorScroll() {
  lineGutter.scrollTop = editor.scrollTop;
  if (isPreviewOpen) {
    const percentage = editor.scrollTop / (editor.scrollHeight - editor.clientHeight || 1);
    previewPane.scrollTop = percentage * (previewPane.scrollHeight - previewPane.clientHeight);
  }
}

// ── Live Split Markdown Preview ───────────────────────────────────────────────

function togglePreview() {
  isPreviewOpen = !isPreviewOpen;
  previewPane.hidden = !isPreviewOpen;
  btnPreview.setAttribute('aria-pressed', isPreviewOpen ? 'true' : 'false');
  if (isPreviewOpen) renderMarkdownPreview();
}

function renderMarkdownPreview() {
  const text = editor.value;
  let html = text
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    // Headers
    .replace(/^### (.*$)/gim, '<h3>$1</h3>')
    .replace(/^## (.*$)/gim, '<h2>$1</h2>')
    .replace(/^# (.*$)/gim, '<h1>$1</h1>')
    // Bold & Italic
    .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
    .replace(/\*(.*?)\*/g, '<em>$1</em>')
    // Code blocks
    .replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
    // Inline code
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    // Task lists
    .replace(/^- \[ \] (.*$)/gim, '<p>⏹ $1</p>')
    .replace(/^- \[x\] (.*$)/gim, '<p>✅ $1</p>')
    // Blockquotes
    .replace(/^> (.*$)/gim, '<blockquote>$1</blockquote>')
    // Paragraphs
    .replace(/\n\n/g, '</p><p>');

  previewPane.innerHTML = `<p>${html}</p>`;
}

// ── Zen / Distraction-Free Mode ───────────────────────────────────────────────

function toggleZenMode() {
  isZenMode = !isZenMode;
  document.body.classList.toggle('zen-mode', isZenMode);
  btnZen.setAttribute('aria-pressed', isZenMode ? 'true' : 'false');
  if (isZenMode && isSidebarOpen) toggleSidebar();
}

// ── Developer Power Tools ─────────────────────────────────────────────────────

async function runJsonFormat(minify) {
  const input = editor.value;
  if (!input.trim()) return;

  if (window.go?.main?.App) {
    const res = await window.go.main.App.FormatJSON(input, minify);
    if (res.error) {
      showToast('⚠️ ' + res.error);
    } else {
      editor.value = res.result;
      handleEditorInput();
      showToast(minify ? 'Minified JSON' : 'Formatted JSON');
    }
  }
}

async function runCaseTransform(mode) {
  const input = editor.value;
  if (!input) return;

  if (window.go?.main?.App) {
    const transformed = await window.go.main.App.TransformCase(input, mode);
    editor.value = transformed;
    handleEditorInput();
    showToast(`Transformed to ${mode}`);
  }
}

async function insertTemplate(name) {
  if (window.go?.main?.App) {
    const tmpl = await window.go.main.App.GetTemplate(name);
    editor.value = (editor.value ? editor.value + '\n\n' : '') + tmpl;
    handleEditorInput();
    showToast(`Inserted template`);
  }
}

// ── Command Palette ───────────────────────────────────────────────────────────

function getBuiltinCommands() {
  const cmds = [
    { id: 'new-tab', title: 'New Scratchpad Tab', shortcut: 'Ctrl+N', action: createNewTab },
    { id: 'close-tab', title: 'Close Active Tab', shortcut: 'Ctrl+W', action: () => closeTab(state.active_index) },
    { id: 'toggle-sidebar', title: 'Toggle Notes Sidebar', shortcut: 'Ctrl+B', action: toggleSidebar },
    { id: 'toggle-preview', title: 'Toggle Split Markdown Preview', shortcut: 'Ctrl+M', action: togglePreview },
    { id: 'toggle-zen', title: 'Toggle Zen Mode', shortcut: 'F11', action: toggleZenMode },
    { id: 'format-json', title: 'Format & Indent JSON', shortcut: 'Ctrl+Shift+J', action: () => runJsonFormat(false) },
    { id: 'minify-json', title: 'Minify JSON', shortcut: '', action: () => runJsonFormat(true) },
  ];

  // Add tabs to search
  if (state.tabs) {
    state.tabs.forEach((tab, idx) => {
      cmds.push({
        id: `switch-tab-${idx}`,
        title: `Switch to Tab ${idx + 1}: ${tab.title || 'Untitled'}`,
        shortcut: `⌘${idx + 1}`,
        action: () => switchActiveTab(idx)
      });
    });
  }

  return cmds;
}

function showCommandPalette() {
  commandPalette.hidden = false;
  commandPaletteInput.value = '';
  commandPaletteSelectedIndex = 0;
  renderCommandPaletteResults();
  commandPaletteInput.focus();
}

function hideCommandPalette() {
  commandPalette.hidden = true;
  editor.focus();
}

function renderCommandPaletteResults() {
  const query = commandPaletteInput.value.toLowerCase().trim();
  const allCmds = getBuiltinCommands();
  filteredCommands = allCmds.filter(c => c.title.toLowerCase().includes(query));

  commandPaletteResults.innerHTML = '';
  if (filteredCommands.length === 0) {
    commandPaletteResults.innerHTML = '<div class="command-item" style="color:var(--col-text-muted)">No matching commands or tabs</div>';
    return;
  }

  filteredCommands.forEach((cmd, idx) => {
    const item = document.createElement('div');
    item.className = `command-item ${idx === commandPaletteSelectedIndex ? 'command-item--selected' : ''}`;
    item.innerHTML = `
      <span>${escapeHtml(cmd.title)}</span>
      ${cmd.shortcut ? `<kbd class="kbd-sm">${cmd.shortcut}</kbd>` : ''}
    `;
    item.addEventListener('click', () => {
      cmd.action();
      hideCommandPalette();
    });
    commandPaletteResults.appendChild(item);
  });
}

function handleCommandPaletteKeydown(e) {
  if (e.key === 'ArrowDown') {
    e.preventDefault();
    commandPaletteSelectedIndex = (commandPaletteSelectedIndex + 1) % (filteredCommands.length || 1);
    renderCommandPaletteResults();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    commandPaletteSelectedIndex = (commandPaletteSelectedIndex - 1 + filteredCommands.length) % (filteredCommands.length || 1);
    renderCommandPaletteResults();
  } else if (e.key === 'Enter') {
    e.preventDefault();
    if (filteredCommands[commandPaletteSelectedIndex]) {
      filteredCommands[commandPaletteSelectedIndex].action();
      hideCommandPalette();
    }
  } else if (e.key === 'Escape') {
    hideCommandPalette();
  }
}

// ── Global Keyboard Shortcuts ─────────────────────────────────────────────────

function handleGlobalKeydown(e) {
  const isCmdOrCtrl = e.metaKey || e.ctrlKey;

  if (isCmdOrCtrl && e.key.toLowerCase() === 'p') {
    e.preventDefault();
    showCommandPalette();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 'b') {
    e.preventDefault();
    toggleSidebar();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 'm') {
    e.preventDefault();
    togglePreview();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 'n') {
    e.preventDefault();
    createNewTab();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 'f') {
    e.preventDefault();
    toggleSearchBar();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 'w') {
    e.preventDefault();
    if (state.tabs && state.tabs.length > 1) {
      closeTab(state.active_index);
    }
  } else if (e.ctrlKey && e.key === 'Tab') {
    e.preventDefault();
    if (!state.tabs || state.tabs.length === 0) return;
    const nextIdx = e.shiftKey
      ? (state.active_index - 1 + state.tabs.length) % state.tabs.length
      : (state.active_index + 1) % state.tabs.length;
    switchActiveTab(nextIdx);
  } else if (isCmdOrCtrl && e.key >= '1' && e.key <= '9') {
    e.preventDefault();
    const targetIdx = parseInt(e.key, 10) - 1;
    if (state.tabs && targetIdx < state.tabs.length) {
      switchActiveTab(targetIdx);
    }
// Removed lock shortcut
  }
}
// ── Find and Replace ────────────────────────────────────────────────────────────

function toggleSearchBar() {
  if (searchBar.hidden) {
    searchBar.hidden = false;
    searchFindInput.focus();
    const selText = editor.value.substring(editor.selectionStart, editor.selectionEnd);
    if (selText) {
      searchFindInput.value = selText;
    }
    updateSearchMatches();
  } else {
    searchBar.hidden = true;
    editor.focus();
  }
}

function updateSearchMatches() {
  const query = searchFindInput.value;
  searchMatches = [];
  currentSearchMatchIndex = -1;
  
  if (!query) {
    searchCount.textContent = '0 of 0';
    return;
  }

  const text = editor.value;
  let lowerText = text.toLowerCase();
  let lowerQuery = query.toLowerCase();
  let pos = 0;
  
  while ((pos = lowerText.indexOf(lowerQuery, pos)) !== -1) {
    searchMatches.push({ start: pos, end: pos + query.length });
    pos += query.length;
  }

  if (searchMatches.length > 0) {
    // Try to find the closest match after the cursor
    const cursorPos = editor.selectionStart;
    let found = searchMatches.findIndex(m => m.start >= cursorPos);
    if (found === -1) found = 0;
    
    currentSearchMatchIndex = found;
    highlightCurrentSearchMatch();
  } else {
    searchCount.textContent = '0 of 0';
  }
}

function navigateSearch(dir) {
  if (searchMatches.length === 0) return;
  currentSearchMatchIndex += dir;
  if (currentSearchMatchIndex >= searchMatches.length) {
    currentSearchMatchIndex = 0;
  } else if (currentSearchMatchIndex < 0) {
    currentSearchMatchIndex = searchMatches.length - 1;
  }
  highlightCurrentSearchMatch();
}

function highlightCurrentSearchMatch() {
  if (currentSearchMatchIndex >= 0 && currentSearchMatchIndex < searchMatches.length) {
    const match = searchMatches[currentSearchMatchIndex];
    editor.focus();
    editor.setSelectionRange(match.start, match.end);
    searchCount.textContent = `${currentSearchMatchIndex + 1} of ${searchMatches.length}`;
  }
}

function replaceCurrentMatch() {
  if (currentSearchMatchIndex >= 0 && currentSearchMatchIndex < searchMatches.length) {
    const match = searchMatches[currentSearchMatchIndex];
    const repText = searchReplaceInput.value;
    editor.setRangeText(repText, match.start, match.end, 'end');
    
    // Notify editor input event
    editor.dispatchEvent(new Event('input'));
    
    // Matches indices have changed, so we must recompute
    updateSearchMatches();
  }
}

function replaceAllMatches() {
  const query = searchFindInput.value;
  if (!query) return;
  const repText = searchReplaceInput.value;
  
  // Custom case-insensitive replaceAll using regex
  const escapedQuery = query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const regex = new RegExp(escapedQuery, 'gi');
  
  const text = editor.value;
  editor.value = text.replace(regex, repText);
  
  editor.dispatchEvent(new Event('input'));
  updateSearchMatches();
}


// ── Toast Notifications ───────────────────────────────────────────────────────

function showToast(msg) {
  const toast = document.createElement('div');
  toast.className = 'toast';
  toast.textContent = msg;
  toastContainer.appendChild(toast);
  setTimeout(() => toast.remove(), 2500);
}

// Helper: Escape HTML
function escapeHtml(str) {
  return (str || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
