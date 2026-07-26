/* ══════════════════════════════════════════════════════════════════════════════
   octoNote v2.0 — Frontend Logic
   ══════════════════════════════════════════════════════════════════════════════ */

// ── DOM References ────────────────────────────────────────────────────────────

const tabbar       = document.getElementById('tabbar');
const btnNewTab    = document.getElementById('btn-new-tab');
const editor       = document.getElementById('editor');
const editorMain   = document.getElementById('editor-main');
const editorHighlight = document.getElementById('editor-highlight');
const editorActiveLine = document.getElementById('editor-active-line');
const lineGutter   = document.getElementById('line-gutter');
const previewPane  = document.getElementById('preview-pane');
const statusSave   = document.getElementById('status-save');
const statusSaveTx = document.getElementById('status-save-text');
const statusPos    = document.getElementById('status-pos');
const statusWords  = document.getElementById('status-words');
const statusChars  = document.getElementById('status-chars');
const statusTabs   = document.getElementById('status-tabs');
const statusFile   = document.getElementById('status-file');
const btnMinimize  = document.getElementById('btn-minimize');
const btnMaximize  = document.getElementById('btn-maximize');
const btnClose     = document.getElementById('btn-close');
const btnOntop     = document.getElementById('btn-ontop');
const btnPreview   = document.getElementById('btn-preview');
const btnHistory   = document.getElementById('btn-history');
const historyPanel = document.getElementById('history-panel');
const historyList  = document.getElementById('history-list');
const btnHistoryClose   = document.getElementById('btn-history-close');
const btnHistoryRestore = document.getElementById('btn-history-restore');

// File modals
const openModal        = document.getElementById('open-modal');
const openPathInput    = document.getElementById('open-path-input');
const openModalErr     = document.getElementById('open-modal-err');
const openModalConfirm = document.getElementById('open-modal-confirm');
const openModalClose   = document.getElementById('open-modal-close');
const saveasModal        = document.getElementById('saveas-modal');
const saveasPathInput    = document.getElementById('saveas-path-input');
const saveasModalErr     = document.getElementById('saveas-modal-err');
const saveasModalConfirm = document.getElementById('saveas-modal-confirm');
const saveasModalClose   = document.getElementById('saveas-modal-close');

// Find/Replace bar
const searchBar          = document.getElementById('search-bar');
const searchFindInput    = document.getElementById('search-find-input');
const searchCount        = document.getElementById('search-count');
const btnSearchPrev      = document.getElementById('btn-search-prev');
const btnSearchNext      = document.getElementById('btn-search-next');
const btnSearchCase      = document.getElementById('btn-search-case');
const btnSearchRegex     = document.getElementById('btn-search-regex');
const btnSearchClose     = document.getElementById('btn-search-close');
const btnSearchReplaceToggle = document.getElementById('btn-search-replace-toggle');
const searchReplaceRow   = document.getElementById('search-replace-row');
const searchReplaceInput = document.getElementById('search-replace-input');
const btnReplaceOne      = document.getElementById('btn-replace-one');
const btnReplaceAll      = document.getElementById('btn-replace-all');

// Command palette
const commandPalette      = document.getElementById('command-palette');
const commandPaletteInput = document.getElementById('command-palette-input');
const commandPaletteResults = document.getElementById('command-palette-results');

// Context menu
const contextMenu    = document.getElementById('context-menu');
const ctxRename      = document.getElementById('ctx-rename');
const ctxDuplicate   = document.getElementById('ctx-duplicate');
const ctxPin         = document.getElementById('ctx-pin');
const ctxCloseOthers = document.getElementById('ctx-close-others');
const ctxClose       = document.getElementById('ctx-close');

// Settings
const settingsPanel  = document.getElementById('settings-panel');
const settingsClose  = document.getElementById('settings-close');
const settingFont    = document.getElementById('setting-font');
const settingFontsize = document.getElementById('setting-fontsize');
const settingWordwrap = document.getElementById('setting-wordwrap');
const settingAutotitle = document.getElementById('setting-autotitle');
const settingLinenums = document.getElementById('setting-linenums');
const settingActiveline = document.getElementById('setting-activeline');
const btnSettingsStorage = document.getElementById('btn-settings-storage');
const themePills     = document.querySelectorAll('.theme-pill');

// Shortcuts overlay
const shortcutsOverlay = document.getElementById('shortcuts-overlay');
const shortcutsClose   = document.getElementById('shortcuts-close');

// Export panel
const exportPanel = document.getElementById('export-panel');
const exportClose = document.getElementById('export-close');

// Onboarding
const welcomeOverlay = document.getElementById('welcome-overlay');

// Toast container
const toastContainer = document.getElementById('toast-container');

// ── State ─────────────────────────────────────────────────────────────────────

let state = { tabs: [], active_index: 0 };
let saveTimer = null;
let saving = false;
let isAlwaysOnTop = false;
let pendingCloseAfterSave = false;
let isPreviewOpen = false;
let isDistractionFree = false;
let contextMenuTargetIdx = -1;
let selectedHistoryIdx = -1;

// Version History: per-tab ring buffer (max 50 snapshots)
const tabHistory = new Map(); // tabID -> [{timestamp, body}]
const HISTORY_MAX = 50;

// Search state
const searchState = {
  query: '',
  caseSensitive: false,
  useRegex: false,
  matches: [],
  currentMatch: -1,
};

// Auto-title debounce
let autoTitleTimer = null;

// Per-tab font size overrides
const tabFontSizes = new Map(); // tabID -> px number

// Settings (persisted to localStorage)
const settings = {
  theme: 'dark',
  font: 'jetbrains',
  fontSize: 15,
  wordWrap: true,
  autoTitle: true,
  lineNumbers: true,
  activeLine: true,
};

const SAVE_DEBOUNCE_MS = 50;
const AUTO_TITLE_DEBOUNCE_MS = 1000;

// ── Markdown Highlight Engine ─────────────────────────────────────────────────

function updateHighlight() {
  let text = editor.value;
  // Escape HTML
  text = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');

  // Fenced code blocks (``` ... ```)
  text = text.replace(/^(```(\w*))\n([\s\S]*?)(^```)/gm,
    (_, open, lang, body, close) =>
      `<span class="md-fence">${open}</span><span class="md-code-lang">${lang}</span>\n<span class="md-fence">${escHtml(body)}</span><span class="md-fence">${close}</span>`
  );

  // Headers (h1-h6) — each level styled differently
  text = text.replace(/^(#{6}\s+)(.*)$/gm, '<span class="md-h4">$1$2</span>');
  text = text.replace(/^(#{5}\s+)(.*)$/gm, '<span class="md-h4">$1$2</span>');
  text = text.replace(/^(#{4}\s+)(.*)$/gm, '<span class="md-h4">$1$2</span>');
  text = text.replace(/^(#{3}\s+)(.*)$/gm, '<span class="md-h3">$1$2</span>');
  text = text.replace(/^(#{2}\s+)(.*)$/gm, '<span class="md-h2">$1$2</span>');
  text = text.replace(/^(#{1}\s+)(.*)$/gm, '<span class="md-h1">$1$2</span>');

  // Horizontal rules
  text = text.replace(/^(---+|\*\*\*+|___+)$/gm, '<span class="md-hr">$1</span>');

  // Task lists
  text = text.replace(/^(\s*[-*+]\s+\[x\]\s+)(.*)$/gim, '<span class="md-task-done">$1$2</span>');
  text = text.replace(/^(\s*[-*+]\s+\[ \]\s+)(.*)$/gm,  '<span class="md-task-open">$1$2</span>');

  // Lists
  text = text.replace(/^(\s*[-*+]\s+)(.*)$/gm, '<span class="md-list">$1</span>$2');

  // Blockquotes
  text = text.replace(/^(\s*&gt;\s+)(.*)$/gm, '<span class="md-quote">$1$2</span>');

  // Bold+italic (*** or ___)
  text = text.replace(/(\*\*\*|___)([^\*_]+)(\*\*\*|___)/g, '<span class="md-bold md-italic">$1$2$3</span>');
  // Bold (** or __)
  text = text.replace(/(\*\*|__)([^\*_]+)(\*\*|__)/g, '<span class="md-bold">$1$2$3</span>');
  // Italic (* or _)
  text = text.replace(/(\*|_)([^\*_\n]+)(\*|_)/g, '<span class="md-italic">$1$2$3</span>');
  // Strikethrough (~~)
  text = text.replace(/(~~)([^~\n]+)(~~)/g, '<span class="md-strike">$1$2$3</span>');

  // Inline code (`code`)
  text = text.replace(/(`[^`\n]+`)/g, '<span class="md-code">$1</span>');

  // Images (before links)
  text = text.replace(/(!\[.*?\]\(.*?\))/g, '<span class="md-image">$1</span>');

  // Links [text](url)
  text = text.replace(/(\[.*?\]\(.*?\))/g, '<span class="md-link">$1</span>');

  // Search match highlighting (layered on top)
  if (searchState.query && searchState.matches.length > 0) {
    // handled separately via overlay approach
  }

  if (text.endsWith('\n')) text += ' ';
  editorHighlight.innerHTML = text;
}

function escHtml(str) {
  return str.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

// ── Markdown Preview Renderer ─────────────────────────────────────────────────

function renderMarkdownPreview(text) {
  if (!isPreviewOpen) return;

  let html = escHtml(text);

  // Fenced code blocks
  html = html.replace(/^```(\w*)\n([\s\S]*?)^```/gm, (_, lang, code) =>
    `<pre><code${lang ? ` data-lang="${lang}"` : ''}>${code.trimEnd()}</code></pre>`
  );

  // HR
  html = html.replace(/^(---+|\*\*\*+)$/gm, '<hr/>');

  // Headers
  html = html.replace(/^######\s+(.*)$/gm, '<h6>$1</h6>');
  html = html.replace(/^#####\s+(.*)$/gm,  '<h5>$1</h5>');
  html = html.replace(/^####\s+(.*)$/gm,   '<h4>$1</h4>');
  html = html.replace(/^###\s+(.*)$/gm,    '<h3>$1</h3>');
  html = html.replace(/^##\s+(.*)$/gm,     '<h2>$1</h2>');
  html = html.replace(/^#\s+(.*)$/gm,      '<h1>$1</h1>');

  // Task lists (before normal lists)
  html = html.replace(/^\s*[-*+]\s+\[x\]\s+(.*)/gim, '<li style="list-style:none"><input type="checkbox" checked disabled/> $1</li>');
  html = html.replace(/^\s*[-*+]\s+\[ \]\s+(.*)/gm,  '<li style="list-style:none"><input type="checkbox" disabled/> $1</li>');

  // Blockquotes
  html = html.replace(/^&gt;\s+(.*)/gm, '<blockquote>$1</blockquote>');

  // Bold+italic
  html = html.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>');
  html = html.replace(/___(.+?)___/g, '<strong><em>$1</em></strong>');
  // Bold
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
  html = html.replace(/__(.+?)__/g, '<strong>$1</strong>');
  // Italic
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
  html = html.replace(/_(.+?)_/g, '<em>$1</em>');
  // Strikethrough
  html = html.replace(/~~(.+?)~~/g, '<del>$1</del>');

  // Inline code
  html = html.replace(/`([^`\n]+)`/g, '<code>$1</code>');

  // Images (before links)
  html = html.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img src="$2" alt="$1" style="max-width:100%"/>');

  // Links
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');

  // Unordered lists
  html = html.replace(/^[\s]*[-*+]\s+(.*)/gm, '<li>$1</li>');

  // Ordered lists
  html = html.replace(/^[\s]*\d+\.\s+(.*)/gm, '<li>$1</li>');

  // Tables
  html = html.replace(/^\|(.+)\|$/gm, (_, row) => {
    if (/^[-| ]+$/.test(row)) return '<tr class="__sep__"></tr>';
    const cells = row.split('|').map(c => `<td>${c.trim()}</td>`).join('');
    return `<tr>${cells}</tr>`;
  });
  html = html.replace(/(<tr>.*?<\/tr>)/gs, m => {
    return `<table>${m.replace(/<tr class="__sep__"><\/tr>/g,'')}</table>`;
  });

  // Paragraphs — wrap standalone lines
  html = html.replace(/^(?!<[hH\d]|<pre|<hr|<blockquote|<ul|<ol|<li|<table|<tr)([^\n<].+)$/gm, '<p>$1</p>');

  previewPane.innerHTML = html;
}

// ── Line Numbers Gutter ───────────────────────────────────────────────────────

function updateLineGutter() {
  if (!settings.lineNumbers) { lineGutter.style.display = 'none'; return; }
  lineGutter.style.display = '';

  const lines = editor.value.split('\n');
  const lineCount = lines.length;
  const cursorLine = getCursorLine();

  // Reuse existing elements if count unchanged
  const existing = lineGutter.children;
  if (existing.length !== lineCount) {
    lineGutter.innerHTML = '';
    for (let i = 0; i < lineCount; i++) {
      const span = document.createElement('span');
      span.className = 'line-gutter__line' + (i === cursorLine ? ' line-gutter__line--active' : '');
      span.textContent = i + 1;
      lineGutter.appendChild(span);
    }
  } else {
    for (let i = 0; i < lineCount; i++) {
      existing[i].className = 'line-gutter__line' + (i === cursorLine ? ' line-gutter__line--active' : '');
    }
  }

  // Sync gutter scroll with editor scroll
  lineGutter.scrollTop = editor.scrollTop;
}

// ── Active Line Highlight ─────────────────────────────────────────────────────

function updateActiveLine() {
  if (!settings.activeLine) { editorActiveLine.style.display = 'none'; return; }
  editorActiveLine.style.display = '';

  const lineHeight = parseFloat(getComputedStyle(editor).lineHeight);
  const paddingTop = parseFloat(getComputedStyle(editor).paddingTop);
  const cursorLine = getCursorLine();

  const top = paddingTop + cursorLine * lineHeight - editor.scrollTop;
  editorActiveLine.style.top = top + 'px';
  editorActiveLine.style.height = lineHeight + 'px';
  editorActiveLine.style.left = (settings.lineNumbers ? 'var(--gutter-w)' : '0');
  editorActiveLine.style.right = '0';
  editorActiveLine.style.position = 'absolute';
}

// ── Tabs Rendering ────────────────────────────────────────────────────────────

function renderTabs() {
  tabbar.querySelectorAll('.tab:not(.tab--new)').forEach(el => el.remove());

  state.tabs.forEach((tab, i) => {
    const btn = document.createElement('button');
    btn.className = 'tab'
      + (i === state.active_index ? ' tab--active' : '')
      + (tab.pinned ? ' tab--pinned' : '');
    btn.setAttribute('role', 'tab');
    btn.setAttribute('aria-selected', i === state.active_index ? 'true' : 'false');
    btn.setAttribute('aria-controls', 'editor');
    btn.setAttribute('id', `tab-${tab.id}`);
    btn.setAttribute('title', `${tab.title}\nDouble-click to rename · Right-click for options`);
    btn.dataset.index = i;
    btn.draggable = !tab.pinned;

    const idxSpan = document.createElement('span');
    idxSpan.className = 'tab__index';
    idxSpan.textContent = i + 1;
    idxSpan.setAttribute('aria-hidden', 'true');

    const titleSpan = document.createElement('span');
    titleSpan.className = 'tab__title';
    const isUnsaved = tab.file_is_dirty || (!tab.file_path && (tab.body || '').trim().length > 0);
    titleSpan.textContent = (isUnsaved ? '● ' : '') + tab.title;
    if (isUnsaved) titleSpan.style.color = 'var(--col-warn)';

    btn.append(idxSpan);

    if (tab.pinned) {
      const pinIcon = document.createElement('span');
      pinIcon.className = 'tab__pin-icon';
      pinIcon.textContent = '📌';
      pinIcon.setAttribute('aria-hidden', 'true');
      btn.append(pinIcon);
    }

    btn.append(titleSpan);

    if (!tab.pinned) {
      const closeBtn = document.createElement('button');
      closeBtn.className = 'tab__close';
      closeBtn.setAttribute('aria-label', `Close tab: ${tab.title}`);
      closeBtn.setAttribute('title', 'Close tab (Ctrl+W)');
      closeBtn.textContent = '×';
      closeBtn.addEventListener('click', e => { e.stopPropagation(); closeTab(i); });
      btn.append(closeBtn);
    }

    btn.addEventListener('click', () => switchTab(i));
    btn.addEventListener('dblclick', () => startRename(btn, titleSpan, i));
    btn.addEventListener('contextmenu', e => openContextMenu(e, i));

    // Drag-and-drop reordering
    btn.addEventListener('dragstart', e => {
      e.dataTransfer.setData('text/plain', String(i));
      btn.classList.add('tab--dragging');
    });
    btn.addEventListener('dragend', () => btn.classList.remove('tab--dragging'));
    btn.addEventListener('dragover', e => {
      e.preventDefault();
      btn.classList.add('tab--drag-over');
    });
    btn.addEventListener('dragleave', () => btn.classList.remove('tab--drag-over'));
    btn.addEventListener('drop', async e => {
      e.preventDefault();
      btn.classList.remove('tab--drag-over');
      const fromIdx = parseInt(e.dataTransfer.getData('text/plain'), 10);
      const toIdx = i;
      if (fromIdx !== toIdx) await reorderTab(fromIdx, toIdx);
    });

    tabbar.insertBefore(btn, btnNewTab);

    if (tab._new) {
      btn.classList.add('tab--entering');
      delete tab._new;
    }
  });

  const count = state.tabs.length;
  statusTabs.textContent = `${count} tab${count !== 1 ? 's' : ''}`;

  const activeTabEl = tabbar.querySelector('.tab--active');
  if (activeTabEl) {
    setTimeout(() => activeTabEl.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' }), 50);
  }
}

// ── Editor Load ───────────────────────────────────────────────────────────────

function loadActiveTabIntoEditor() {
  const tab = state.tabs[state.active_index];
  if (!tab) return;

  const tabId = tab.id;
  const savedSize = tabFontSizes.get(tabId);
  if (savedSize) {
    editor.style.fontSize = savedSize + 'px';
    editorHighlight.style.fontSize = savedSize + 'px';
  } else {
    editor.style.fontSize = '';
    editorHighlight.style.fontSize = '';
  }

  editor.value = tab.body || '';
  updateHighlight();
  updateLineGutter();
  updateActiveLine();
  renderMarkdownPreview(editor.value);

  try {
    const lines = editor.value.split('\n');
    let pos = 0;
    for (let l = 0; l < Math.min(tab.cursor_line || 0, lines.length - 1); l++) {
      pos += lines[l].length + 1;
    }
    editor.setSelectionRange(pos, pos);
  } catch (_) {}
  editor.focus();
  updateFileStatus();
  updateCursorStatus();
}

// ── Save Logic ────────────────────────────────────────────────────────────────

function scheduleSave() {
  clearTimeout(saveTimer);
  setSaveStatus('saving');
  saveTimer = setTimeout(flushSave, SAVE_DEBOUNCE_MS);
}

async function flushSave() {
  const idx = state.active_index;
  const tab = state.tabs[idx];
  if (!tab) return;

  const body = editor.value;
  const cursorLine = getCursorLine();

  state.tabs[idx].body = body;
  state.tabs[idx].cursor_line = cursorLine;

  try {
    saving = true;
    await window.go.main.App.SaveTab(idx, body, cursorLine);
    setSaveStatus('saved');
    scheduleHistorySnapshot(tab.id, body);
  } catch (err) {
    console.error('octoNote: save failed', err);
    setSaveStatus('error');
    showToast('Save failed — ' + String(err), 'error');
  } finally {
    saving = false;
  }
}

// ── History Management ────────────────────────────────────────────────────────

function scheduleHistorySnapshot(tabId, body) {
  const history = tabHistory.get(tabId) || [];
  const lastEntry = history[history.length - 1];
  // Only snapshot if content actually changed
  if (lastEntry && lastEntry.body === body) return;
  history.push({ timestamp: new Date(), body });
  if (history.length > HISTORY_MAX) history.shift();
  tabHistory.set(tabId, history);
  // Refresh history panel if open
  if (!historyPanel.hidden) renderHistoryPanel();
}

function renderHistoryPanel() {
  const tab = state.tabs[state.active_index];
  if (!tab) return;
  const history = tabHistory.get(tab.id) || [];
  historyList.innerHTML = '';
  if (history.length === 0) {
    historyList.innerHTML = '<p style="padding:12px;color:var(--col-muted);font-size:0.75rem">No snapshots yet. History is captured as you type.</p>';
    return;
  }
  [...history].reverse().forEach((entry, reversedIdx) => {
    const idx = history.length - 1 - reversedIdx;
    const btn = document.createElement('button');
    btn.className = 'history-entry' + (idx === selectedHistoryIdx ? ' history-entry--active' : '');
    btn.setAttribute('role', 'listitem');
    const timeStr = entry.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    const preview = entry.body.replace(/\n/g, ' ').slice(0, 60);
    btn.innerHTML = `<div class="history-entry__time">${timeStr}</div><div class="history-entry__preview">${escHtml(preview) || '<em>empty</em>'}</div>`;
    btn.addEventListener('click', () => {
      selectedHistoryIdx = idx;
      btnHistoryRestore.disabled = false;
      renderHistoryPanel();
      // Preview the snapshot in the editor without saving
      editor.value = entry.body;
      updateHighlight();
    });
    historyList.appendChild(btn);
  });
}

btnHistoryRestore.addEventListener('click', async () => {
  const tab = state.tabs[state.active_index];
  if (!tab) return;
  const history = tabHistory.get(tab.id) || [];
  const entry = history[selectedHistoryIdx];
  if (!entry) return;
  editor.value = entry.body;
  updateHighlight();
  updateLineGutter();
  await flushSave();
  showToast('Version restored ✓', 'success');
  btnHistoryRestore.disabled = true;
  selectedHistoryIdx = -1;
  renderHistoryPanel();
});

btnHistory.addEventListener('click', () => {
  const wasHidden = historyPanel.hidden;
  historyPanel.hidden = !wasHidden;
  btnHistory.setAttribute('aria-expanded', String(wasHidden));
  btnHistory.classList.toggle('share-btn--active', wasHidden);
  if (wasHidden) renderHistoryPanel();
});
btnHistoryClose.addEventListener('click', () => {
  historyPanel.hidden = true;
  btnHistory.setAttribute('aria-expanded', 'false');
  btnHistory.classList.remove('share-btn--active');
});

// ── Tab Operations ────────────────────────────────────────────────────────────

async function switchTab(idx) {
  if (idx === state.active_index) return;
  clearTimeout(saveTimer);
  if (editor.value !== (state.tabs[state.active_index]?.body ?? '')) await flushSave();
  try {
    state = await window.go.main.App.SetActiveTab(idx);
    renderTabs();
    loadActiveTabIntoEditor();
    clearSearch();
  } catch (err) {
    console.error('octoNote: switch tab failed', err);
    showToast('Failed to switch tab', 'error');
  }
}

async function newTab() {
  try {
    state = await window.go.main.App.NewTab();
    if (state.tabs.length > 0) state.tabs[state.tabs.length - 1]._new = true;
    renderTabs();
    loadActiveTabIntoEditor();
  } catch (err) {
    console.error('octoNote: new tab failed', err);
    showToast('Failed to create tab', 'error');
  }
}

async function closeTab(idx) {
  const tab = state.tabs[idx];
  if (tab?.pinned) { showToast('Unpin this tab before closing', 'warn'); return; }
  try {
    state = await window.go.main.App.CloseTab(idx);
    renderTabs();
    loadActiveTabIntoEditor();
  } catch (err) {
    console.error('octoNote: close tab failed', err);
    showToast('Failed to close tab', 'error');
  }
}

async function duplicateTab(idx) {
  try {
    if (window.go?.main?.App?.DuplicateTab) {
      state = await window.go.main.App.DuplicateTab(idx);
    } else {
      // Fallback: manual duplicate
      const tab = state.tabs[idx];
      state = await window.go.main.App.NewTab();
      const newIdx = state.active_index;
      const newTitle = (tab.title || 'tab') + ' (copy)';
      state = await window.go.main.App.RenameTab(newIdx, newTitle);
      await window.go.main.App.SaveTab(newIdx, tab.body || '', 0);
      state.tabs[newIdx].body = tab.body || '';
    }
    renderTabs();
    loadActiveTabIntoEditor();
    showToast('Tab duplicated', 'success');
  } catch (err) {
    showToast('Failed to duplicate tab', 'error');
  }
}

async function reorderTab(fromIdx, toIdx) {
  try {
    if (window.go?.main?.App?.ReorderTabs) {
      state = await window.go.main.App.ReorderTabs(fromIdx, toIdx);
    } else {
      // Fallback: reorder locally
      const tabs = [...state.tabs];
      const [moved] = tabs.splice(fromIdx, 1);
      tabs.splice(toIdx, 0, moved);
      state.tabs = tabs;
      let newActive = state.active_index;
      if (state.active_index === fromIdx) newActive = toIdx;
      else if (fromIdx < state.active_index && toIdx >= state.active_index) newActive--;
      else if (fromIdx > state.active_index && toIdx <= state.active_index) newActive++;
      state.active_index = newActive;
    }
    renderTabs();
    loadActiveTabIntoEditor();
  } catch (err) {
    showToast('Failed to reorder tabs', 'error');
  }
}

async function pinTab(idx, pinned) {
  try {
    if (window.go?.main?.App?.PinTab) {
      state = await window.go.main.App.PinTab(idx, pinned);
    } else {
      state.tabs[idx].pinned = pinned;
      await window.go.main.App.SaveTab(idx, state.tabs[idx].body, state.tabs[idx].cursor_line);
    }
    renderTabs();
    showToast(pinned ? 'Tab pinned 📌' : 'Tab unpinned', 'success');
  } catch (err) {
    showToast('Failed to pin tab', 'error');
  }
}

async function closeOtherTabs(keepIdx) {
  const toClose = state.tabs
    .map((_, i) => i)
    .filter(i => i !== keepIdx && !state.tabs[i].pinned)
    .reverse();
  for (const i of toClose) {
    state = await window.go.main.App.CloseTab(i);
  }
  renderTabs();
  loadActiveTabIntoEditor();
}

function startRename(tabBtn, titleSpan, idx) {
  if (tabBtn.querySelector('.tab-rename-input')) return;
  const originalTitle = state.tabs[idx].title;
  const input = document.createElement('input');
  input.type = 'text';
  input.className = 'tab-rename-input';
  input.value = originalTitle;
  input.maxLength = 64;
  input.setAttribute('aria-label', 'Rename tab');

  titleSpan.replaceWith(input);
  input.focus(); input.select();

  async function commitRename() {
    const newTitle = input.value.trim() || originalTitle;
    try {
      state = await window.go.main.App.RenameTab(idx, newTitle);
      renderTabs();
    } catch (_) { renderTabs(); }
  }

  input.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); commitRename(); }
    if (e.key === 'Escape') { e.preventDefault(); renderTabs(); }
    e.stopPropagation();
  });
  input.addEventListener('blur', commitRename);
}

// ── Auto-title from Content ───────────────────────────────────────────────────

function scheduleAutoTitle() {
  if (!settings.autoTitle) return;
  clearTimeout(autoTitleTimer);
  autoTitleTimer = setTimeout(() => {
    const idx = state.active_index;
    const tab = state.tabs[idx];
    if (!tab) return;
    // Only auto-title if tab still has the default "tab N" name
    if (!tab.title.match(/^tab \d+$/i)) return;

    const firstLine = (editor.value || '').split('\n').find(l => l.trim().length > 0) || '';
    const title = firstLine.replace(/^#+\s*/, '').trim().slice(0, 40);
    if (title && title !== tab.title) {
      window.go.main.App.RenameTab(idx, title).then(newState => {
        state = newState;
        renderTabs();
      }).catch(() => {});
    }
  }, AUTO_TITLE_DEBOUNCE_MS);
}

// ── Context Menu ──────────────────────────────────────────────────────────────

function openContextMenu(e, idx) {
  e.preventDefault();
  contextMenuTargetIdx = idx;
  const tab = state.tabs[idx];
  ctxPin.textContent = tab.pinned ? '📌 Unpin Tab' : '📌 Pin Tab';
  contextMenu.style.left = Math.min(e.clientX, window.innerWidth - 200) + 'px';
  contextMenu.style.top  = Math.min(e.clientY, window.innerHeight - 200) + 'px';
  contextMenu.hidden = false;
  ctxRename.focus();
}

function closeContextMenu() {
  contextMenu.hidden = true;
  contextMenuTargetIdx = -1;
}

ctxRename.addEventListener('click', () => {
  const btn = document.getElementById(`tab-${state.tabs[contextMenuTargetIdx]?.id}`);
  const titleSpan = btn?.querySelector('.tab__title');
  if (btn && titleSpan) startRename(btn, titleSpan, contextMenuTargetIdx);
  closeContextMenu();
});
ctxDuplicate.addEventListener('click', () => { duplicateTab(contextMenuTargetIdx); closeContextMenu(); });
ctxPin.addEventListener('click', () => {
  const t = state.tabs[contextMenuTargetIdx];
  if (t) pinTab(contextMenuTargetIdx, !t.pinned);
  closeContextMenu();
});
ctxCloseOthers.addEventListener('click', () => { closeOtherTabs(contextMenuTargetIdx); closeContextMenu(); });
ctxClose.addEventListener('click', () => { closeTab(contextMenuTargetIdx); closeContextMenu(); });

document.addEventListener('click', e => {
  if (!contextMenu.contains(e.target)) closeContextMenu();
});
document.addEventListener('keydown', e => {
  if (e.key === 'Escape') closeContextMenu();
});

// ── Command Palette ───────────────────────────────────────────────────────────

const COMMANDS = [
  { id: 'new-tab',       label: 'New Tab',              icon: '📄', shortcut: 'Ctrl+N',        action: newTab },
  { id: 'close-tab',     label: 'Close Tab',            icon: '✕',  shortcut: 'Ctrl+W',        action: () => closeTab(state.active_index) },
  { id: 'duplicate-tab', label: 'Duplicate Tab',        icon: '⎘',  shortcut: 'Ctrl+D',        action: () => duplicateTab(state.active_index) },
  { id: 'pin-tab',       label: 'Pin / Unpin Tab',      icon: '📌', shortcut: '',              action: () => { const t = state.tabs[state.active_index]; pinTab(state.active_index, !t?.pinned); } },
  { id: 'open-file',     label: 'Open File…',           icon: '📂', shortcut: 'Ctrl+O',        action: openOpenModal },
  { id: 'save-file',     label: 'Save to File',         icon: '💾', shortcut: 'Ctrl+S',        action: handleSave },
  { id: 'save-as',       label: 'Save As…',             icon: '💾', shortcut: 'Ctrl+Shift+S',  action: () => openSaveAsModal() },
  { id: 'find',          label: 'Find in Tab',          icon: '🔍', shortcut: 'Ctrl+F',        action: openSearch },
  { id: 'replace',       label: 'Find & Replace',       icon: '🔄', shortcut: 'Ctrl+H',        action: openSearchReplace },
  { id: 'preview',       label: 'Toggle Markdown Preview', icon: '👁', shortcut: 'Ctrl+M',     action: togglePreview },
  { id: 'history',       label: 'Version History',      icon: '🕐', shortcut: 'Ctrl+Shift+H',  action: () => { historyPanel.hidden = !historyPanel.hidden; renderHistoryPanel(); } },
  { id: 'distraction',   label: 'Distraction-free Mode', icon: '🧘', shortcut: 'Ctrl+Shift+F', action: toggleDistractionFree },
  { id: 'export',        label: 'Export…',              icon: '📤', shortcut: '',              action: () => { exportPanel.hidden = false; } },
  { id: 'settings',      label: 'Settings',             icon: '⚙',  shortcut: 'Ctrl+,',        action: () => { settingsPanel.hidden = false; } },
  { id: 'shortcuts',     label: 'Keyboard Shortcuts',   icon: '⌨',  shortcut: '?',             action: () => { shortcutsOverlay.hidden = false; } },
  { id: 'theme-dark',    label: 'Theme: Dark',          icon: '🌙', shortcut: '',              action: () => applyTheme('dark') },
  { id: 'theme-light',   label: 'Theme: Light',         icon: '☀', shortcut: '',              action: () => applyTheme('light') },
  { id: 'theme-system',  label: 'Theme: System',        icon: '🖥', shortcut: '',              action: () => applyTheme('system') },
];

let paletteActiveIdx = -1;
let paletteItems = [];

function openCommandPalette() {
  commandPalette.hidden = false;
  commandPaletteInput.value = '';
  renderPaletteResults('');
  commandPaletteInput.focus();
}

function closeCommandPalette() {
  commandPalette.hidden = true;
  editor.focus();
}

function renderPaletteResults(query) {
  commandPaletteResults.innerHTML = '';
  paletteItems = [];
  paletteActiveIdx = -1;
  const q = query.toLowerCase().trim();

  // Tab results
  const matchingTabs = state.tabs.filter((t, i) =>
    !q || t.title.toLowerCase().includes(q) || (t.body || '').toLowerCase().includes(q)
  );
  if (matchingTabs.length > 0) {
    const label = document.createElement('div');
    label.className = 'command-palette__group-label';
    label.textContent = 'Tabs';
    commandPaletteResults.appendChild(label);
    matchingTabs.slice(0, 5).forEach(tab => {
      const idx = state.tabs.findIndex(t => t.id === tab.id);
      const preview = (tab.body || '').replace(/\n/g, ' ').slice(0, 60);
      const btn = makePaletteItem('📄', tab.title, preview, '', () => { switchTab(idx); closeCommandPalette(); });
      commandPaletteResults.appendChild(btn);
      paletteItems.push(btn);
    });
  }

  // Command results
  const matchingCmds = COMMANDS.filter(c => !q || c.label.toLowerCase().includes(q));
  if (matchingCmds.length > 0) {
    const label = document.createElement('div');
    label.className = 'command-palette__group-label';
    label.textContent = 'Commands';
    commandPaletteResults.appendChild(label);
    matchingCmds.slice(0, 10).forEach(cmd => {
      const btn = makePaletteItem(cmd.icon, cmd.label, '', cmd.shortcut, () => {
        closeCommandPalette();
        cmd.action();
      });
      commandPaletteResults.appendChild(btn);
      paletteItems.push(btn);
    });
  }

  if (paletteItems.length > 0) {
    paletteActiveIdx = 0;
    paletteItems[0].classList.add('command-palette__item--active');
  }
}

function makePaletteItem(icon, label, preview, shortcut, action) {
  const btn = document.createElement('button');
  btn.className = 'command-palette__item';
  btn.setAttribute('role', 'option');
  btn.innerHTML = `
    <span class="command-palette__item-icon">${icon}</span>
    <span class="command-palette__item-label">${escHtml(label)}</span>
    ${preview ? `<span class="command-palette__item-preview">${escHtml(preview)}</span>` : ''}
    ${shortcut ? `<span class="command-palette__item-kbd">${escHtml(shortcut)}</span>` : ''}
  `;
  btn.addEventListener('click', action);
  return btn;
}

commandPaletteInput.addEventListener('input', () => renderPaletteResults(commandPaletteInput.value));
commandPaletteInput.addEventListener('keydown', e => {
  if (e.key === 'ArrowDown') {
    e.preventDefault();
    if (paletteActiveIdx >= 0) paletteItems[paletteActiveIdx].classList.remove('command-palette__item--active');
    paletteActiveIdx = Math.min(paletteActiveIdx + 1, paletteItems.length - 1);
    paletteItems[paletteActiveIdx]?.classList.add('command-palette__item--active');
    paletteItems[paletteActiveIdx]?.scrollIntoView({ block: 'nearest' });
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    if (paletteActiveIdx >= 0) paletteItems[paletteActiveIdx].classList.remove('command-palette__item--active');
    paletteActiveIdx = Math.max(paletteActiveIdx - 1, 0);
    paletteItems[paletteActiveIdx]?.classList.add('command-palette__item--active');
    paletteItems[paletteActiveIdx]?.scrollIntoView({ block: 'nearest' });
  } else if (e.key === 'Enter') {
    e.preventDefault();
    paletteItems[paletteActiveIdx]?.click();
  } else if (e.key === 'Escape') {
    closeCommandPalette();
  }
  e.stopPropagation();
});
commandPalette.querySelector('.command-palette__backdrop').addEventListener('click', closeCommandPalette);

// ── Find & Replace ────────────────────────────────────────────────────────────

function openSearch(withReplace = false) {
  searchBar.hidden = false;
  searchFindInput.focus();
  searchFindInput.select();
  if (withReplace) {
    searchReplaceRow.hidden = false;
    btnSearchReplaceToggle.setAttribute('aria-expanded', 'true');
  }
  runSearch();
}

function openSearchReplace() { openSearch(true); }

function closeSearch() {
  searchBar.hidden = true;
  clearSearch();
  editor.focus();
}

function clearSearch() {
  searchState.query = '';
  searchState.matches = [];
  searchState.currentMatch = -1;
  updateSearchCount();
  updateHighlight();
}

function runSearch() {
  const query = searchFindInput.value;
  searchState.query = query;

  if (!query) {
    searchState.matches = [];
    searchState.currentMatch = -1;
    updateSearchCount();
    updateHighlight();
    return;
  }

  const text = editor.value;
  const matches = [];
  try {
    const flags = searchState.caseSensitive ? 'g' : 'gi';
    let pattern;
    if (searchState.useRegex) {
      pattern = new RegExp(query, flags);
    } else {
      pattern = new RegExp(query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), flags);
    }
    let m;
    while ((m = pattern.exec(text)) !== null) {
      matches.push({ start: m.index, end: m.index + m[0].length });
      if (matches.length > 500) break;
    }
  } catch (_) {}

  searchState.matches = matches;
  if (searchState.currentMatch >= matches.length) searchState.currentMatch = -1;
  if (matches.length > 0 && searchState.currentMatch < 0) searchState.currentMatch = 0;
  updateSearchCount();
  jumpToCurrentMatch();
}

function nextMatch() {
  if (!searchState.matches.length) return;
  searchState.currentMatch = (searchState.currentMatch + 1) % searchState.matches.length;
  updateSearchCount();
  jumpToCurrentMatch();
}

function prevMatch() {
  if (!searchState.matches.length) return;
  searchState.currentMatch = (searchState.currentMatch - 1 + searchState.matches.length) % searchState.matches.length;
  updateSearchCount();
  jumpToCurrentMatch();
}

function jumpToCurrentMatch() {
  const match = searchState.matches[searchState.currentMatch];
  if (!match) return;
  editor.setSelectionRange(match.start, match.end);
  editor.focus();
}

function updateSearchCount() {
  const total = searchState.matches.length;
  const cur = searchState.currentMatch + 1;
  searchCount.textContent = total > 0 ? `${cur} of ${total}` : (searchState.query ? 'No results' : '0 of 0');
}

function replaceOne() {
  const match = searchState.matches[searchState.currentMatch];
  if (!match) return;
  const repl = searchReplaceInput.value;
  editor.focus();
  editor.setSelectionRange(match.start, match.end);
  document.execCommand('insertText', false, repl);
  runSearch();
  scheduleSave();
}

function replaceAll() {
  const text = editor.value;
  const repl = searchReplaceInput.value;
  let result;
  try {
    const flags = searchState.caseSensitive ? 'g' : 'gi';
    let pattern;
    if (searchState.useRegex) {
      pattern = new RegExp(searchState.query, flags);
    } else {
      pattern = new RegExp(searchState.query.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), flags);
    }
    result = text.replace(pattern, repl);
  } catch (_) { return; }
  editor.value = result;
  updateHighlight();
  updateLineGutter();
  scheduleSave();
  runSearch();
  showToast(`Replaced ${searchState.matches.length} occurrence(s)`, 'success');
}

searchFindInput.addEventListener('input', runSearch);
searchFindInput.addEventListener('keydown', e => {
  if (e.key === 'Enter') { e.shiftKey ? prevMatch() : nextMatch(); e.preventDefault(); }
  if (e.key === 'Escape') closeSearch();
  e.stopPropagation();
});
btnSearchNext.addEventListener('click', nextMatch);
btnSearchPrev.addEventListener('click', prevMatch);
btnSearchClose.addEventListener('click', closeSearch);
btnSearchCase.addEventListener('click', () => {
  searchState.caseSensitive = !searchState.caseSensitive;
  btnSearchCase.classList.toggle('search-bar__btn--active', searchState.caseSensitive);
  btnSearchCase.setAttribute('aria-pressed', String(searchState.caseSensitive));
  runSearch();
});
btnSearchRegex.addEventListener('click', () => {
  searchState.useRegex = !searchState.useRegex;
  btnSearchRegex.classList.toggle('search-bar__btn--active', searchState.useRegex);
  btnSearchRegex.setAttribute('aria-pressed', String(searchState.useRegex));
  runSearch();
});
btnSearchReplaceToggle.addEventListener('click', () => {
  const expanded = searchReplaceRow.hidden;
  searchReplaceRow.hidden = !expanded;
  btnSearchReplaceToggle.setAttribute('aria-expanded', String(expanded));
});
btnReplaceOne.addEventListener('click', replaceOne);
btnReplaceAll.addEventListener('click', replaceAll);

// ── Preview Panel ─────────────────────────────────────────────────────────────

function togglePreview() {
  isPreviewOpen = !isPreviewOpen;
  previewPane.hidden = !isPreviewOpen;
  btnPreview.classList.toggle('share-btn--active', isPreviewOpen);
  btnPreview.setAttribute('aria-pressed', String(isPreviewOpen));
  if (isPreviewOpen) renderMarkdownPreview(editor.value);
}

btnPreview.addEventListener('click', togglePreview);

// ── Distraction-Free Mode ─────────────────────────────────────────────────────

function toggleDistractionFree() {
  isDistractionFree = !isDistractionFree;
  document.body.classList.toggle('distraction-free', isDistractionFree);
  if (!isDistractionFree) editor.focus();
}

// ── Theme Management ──────────────────────────────────────────────────────────

function applyTheme(theme) {
  settings.theme = theme;
  let resolved = theme;
  if (theme === 'system') {
    resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  document.documentElement.setAttribute('data-theme', resolved);
  localStorage.setItem('octonote-theme', theme);

  // Update settings panel pills
  themePills.forEach(p => {
    const isActive = p.dataset.themeVal === theme;
    p.classList.toggle('theme-pill--active', isActive);
    p.setAttribute('aria-pressed', String(isActive));
  });
}

function loadSettings() {
  const savedTheme = localStorage.getItem('octonote-theme') || 'dark';
  applyTheme(savedTheme);

  const savedFont = localStorage.getItem('octonote-font') || 'jetbrains';
  settings.font = savedFont;
  if (settingFont) settingFont.value = savedFont;
  applyFont(savedFont);

  const savedSize = parseInt(localStorage.getItem('octonote-fontsize') || '15', 10);
  settings.fontSize = savedSize;
  if (settingFontsize) settingFontsize.value = String(savedSize);
  applyFontSize(savedSize);

  const savedWrap = localStorage.getItem('octonote-wordwrap') !== 'false';
  settings.wordWrap = savedWrap;
  if (settingWordwrap) settingWordwrap.checked = savedWrap;
  applyWordWrap(savedWrap);

  const savedAutoTitle = localStorage.getItem('octonote-autotitle') !== 'false';
  settings.autoTitle = savedAutoTitle;
  if (settingAutotitle) settingAutotitle.checked = savedAutoTitle;

  const savedLinenums = localStorage.getItem('octonote-linenums') !== 'false';
  settings.lineNumbers = savedLinenums;
  if (settingLinenums) settingLinenums.checked = savedLinenums;

  const savedActiveline = localStorage.getItem('octonote-activeline') !== 'false';
  settings.activeLine = savedActiveline;
  if (settingActiveline) settingActiveline.checked = savedActiveline;
}

function applyFont(font) {
  const fontMap = {
    jetbrains: "'JetBrains Mono', monospace",
    fira:      "'Fira Code', monospace",
    cascadia:  "'Cascadia Code', monospace",
    system:    "monospace",
  };
  const val = fontMap[font] || fontMap.jetbrains;
  document.documentElement.style.setProperty('--font-editor', val);
}

function applyFontSize(px) {
  document.documentElement.style.setProperty('--font-size-editor', px + 'px');
}

function applyWordWrap(wrap) {
  editor.style.whiteSpace = wrap ? 'pre-wrap' : 'pre';
  editor.style.overflowX  = wrap ? 'hidden' : 'auto';
  editorHighlight.style.whiteSpace = wrap ? 'pre-wrap' : 'pre';
}

// Settings panel events
themePills.forEach(pill => {
  pill.addEventListener('click', () => applyTheme(pill.dataset.themeVal));
});

settingFont?.addEventListener('change', () => {
  settings.font = settingFont.value;
  localStorage.setItem('octonote-font', settingFont.value);
  applyFont(settingFont.value);
});

settingFontsize?.addEventListener('change', () => {
  const sz = parseInt(settingFontsize.value, 10);
  settings.fontSize = sz;
  localStorage.setItem('octonote-fontsize', String(sz));
  applyFontSize(sz);
});

settingWordwrap?.addEventListener('change', () => {
  settings.wordWrap = settingWordwrap.checked;
  localStorage.setItem('octonote-wordwrap', String(settingWordwrap.checked));
  applyWordWrap(settingWordwrap.checked);
});
settingAutotitle?.addEventListener('change', () => {
  settings.autoTitle = settingAutotitle.checked;
  localStorage.setItem('octonote-autotitle', String(settingAutotitle.checked));
});
settingLinenums?.addEventListener('change', () => {
  settings.lineNumbers = settingLinenums.checked;
  localStorage.setItem('octonote-linenums', String(settingLinenums.checked));
  updateLineGutter();
});
settingActiveline?.addEventListener('change', () => {
  settings.activeLine = settingActiveline.checked;
  localStorage.setItem('octonote-activeline', String(settingActiveline.checked));
  updateActiveLine();
});

btnSettingsStorage?.addEventListener('click', async () => {
  try {
    const dir = await window.go.main.App.GetStorageDir();
    showToast('Storage: ' + dir, 'info', 5000);
  } catch (_) {}
});

settingsPanel.querySelector('.settings-panel__backdrop')?.addEventListener('click', () => { settingsPanel.hidden = true; });
settingsClose?.addEventListener('click', () => { settingsPanel.hidden = true; });

// ── Shortcuts Overlay ─────────────────────────────────────────────────────────

shortcutsClose?.addEventListener('click', () => { shortcutsOverlay.hidden = true; });
shortcutsOverlay.querySelector('.shortcuts-overlay__backdrop')?.addEventListener('click', () => { shortcutsOverlay.hidden = true; });

// ── Export Panel ──────────────────────────────────────────────────────────────

exportClose?.addEventListener('click', () => { exportPanel.hidden = true; });
exportPanel.querySelector('.export-panel__backdrop')?.addEventListener('click', () => { exportPanel.hidden = true; });

document.getElementById('export-md')?.addEventListener('click', () => {
  exportContent('md');
  exportPanel.hidden = true;
});
document.getElementById('export-txt')?.addEventListener('click', () => {
  exportContent('txt');
  exportPanel.hidden = true;
});
document.getElementById('export-html')?.addEventListener('click', () => {
  exportContent('html');
  exportPanel.hidden = true;
});
document.getElementById('export-clip')?.addEventListener('click', async () => {
  try {
    await navigator.clipboard.writeText(editor.value);
    showToast('Copied to clipboard ✓', 'success');
  } catch (_) { showToast('Clipboard access denied', 'error'); }
  exportPanel.hidden = true;
});
document.getElementById('export-all')?.addEventListener('click', async () => {
  try {
    if (window.go?.main?.App?.ExportAllTabs) {
      const dir = await window.go.main.App.GetStorageDir();
      const err = await window.go.main.App.ExportAllTabs(dir);
      if (err) showToast(err, 'error');
      else showToast('All tabs exported to: ' + dir, 'success', 5000);
    } else {
      showToast('Export all tabs requires a newer build', 'warn');
    }
  } catch (e) { showToast(String(e), 'error'); }
  exportPanel.hidden = true;
});

function exportContent(format) {
  const tab = state.tabs[state.active_index];
  if (!tab) return;
  const filename = (tab.title || 'note') + '.' + (format === 'html' ? 'html' : format);
  let content = editor.value;

  if (format === 'html') {
    content = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"/><title>${escHtml(tab.title)}</title>
<style>body{max-width:720px;margin:40px auto;font-family:system-ui,sans-serif;line-height:1.7;color:#222;}</style>
</head><body><article>
${renderToHTMLString(content)}
</article></body></html>`;
  }

  const blob = new Blob([content], { type: 'text/plain' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
  showToast(`Exported as ${filename}`, 'success');
}

function renderToHTMLString(md) {
  // Quick export renderer
  const div = document.createElement('div');
  div.className = 'preview-pane';
  renderMarkdownPreview(md);
  return previewPane.innerHTML;
}

// ── Toast Notifications ───────────────────────────────────────────────────────

function showToast(msg, type = 'info', duration = 3000) {
  const toast = document.createElement('div');
  toast.className = `toast toast--${type}`;
  toast.setAttribute('role', 'status');
  const icons = { success: '✅', warn: '⚠️', error: '❌', info: 'ℹ️' };
  toast.innerHTML = `<span class="toast__icon">${icons[type] || 'ℹ️'}</span><span class="toast__msg">${escHtml(msg)}</span>`;
  toastContainer.appendChild(toast);

  setTimeout(() => {
    toast.classList.add('toast--exit');
    toast.addEventListener('animationend', () => toast.remove(), { once: true });
    setTimeout(() => toast.remove(), 300);
  }, duration);
}

// ── Status Updates ────────────────────────────────────────────────────────────

function getCursorLine() {
  const text = editor.value.substring(0, editor.selectionStart);
  return text.split('\n').length - 1;
}

function updateCursorStatus() {
  const fullText = editor.value;
  const text = fullText.substring(0, editor.selectionStart);
  const lines = text.split('\n');
  const line = lines.length;
  const col  = lines[lines.length - 1].length + 1;
  statusPos.textContent = `Ln ${line}, Col ${col}`;

  const words = fullText.trim() ? fullText.trim().split(/\s+/).length : 0;
  const chars = fullText.length;
  statusWords.textContent = `${words}w`;
  statusChars.textContent = `${chars}c`;

  updateLineGutter();
  updateActiveLine();
}

function getSaveTimestamp() {
  return new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function setSaveStatus(status) {
  const tab = state.tabs[state.active_index];
  if (!tab) return;
  statusSave.className = 'status-indicator';
  let iconHTML = '', text = '';

  if (status === 'saving') {
    statusSave.classList.add('status-indicator--saving');
    iconHTML = svgDot('var(--col-warn)', true);
    text = 'saving…';
  } else if (status === 'error') {
    statusSave.classList.add('status-indicator--error');
    iconHTML = svgDot('var(--col-danger)');
    text = 'save error';
  } else {
    const isUnsaved = tab.file_is_dirty || (!tab.file_path && (tab.body || '').trim().length > 0);
    if (tab.file_path) {
      const name = tab.file_path.split('/').pop();
      if (tab.file_is_dirty) {
        statusSave.classList.add('status-indicator--dirty');
        iconHTML = svgDot('var(--col-warn)');
        text = `unsaved — ${name}`;
      } else {
        statusSave.classList.add('status-indicator--saved');
        iconHTML = svgDot('var(--col-success)');
        text = `saved ${name}`;
      }
    } else {
      if (isUnsaved) {
        statusSave.classList.add('status-indicator--dirty');
        iconHTML = svgDot('var(--col-warn)');
        text = 'scratchpad';
      } else {
        statusSave.classList.add('status-indicator--empty');
        iconHTML = svgDot('var(--col-muted)');
        text = 'empty';
      }
    }
  }

  statusSave.innerHTML = `${iconHTML}<span id="status-save-text"></span>`;
  statusSave.querySelector('#status-save-text').textContent = text;
}

function svgDot(fill, animated = false) {
  const anim = animated ? ' style="animation:blink-saving 0.9s ease-in-out infinite"' : '';
  return `<svg class="status-icon" width="8" height="8" viewBox="0 0 8 8" aria-hidden="true"${anim}><circle cx="4" cy="4" r="4" fill="${fill}"/></svg>`;
}

function updateFileStatus() { setSaveStatus('saved'); }

// ── Keyboard Shortcuts ────────────────────────────────────────────────────────

function onKeyDown(e) {
  // Allow rename input to handle its own keys
  if (document.activeElement?.classList.contains('tab-rename-input')) return;

  // Allow modals to handle their own keys (except Escape)
  const anyModalOpen = !commandPalette.hidden || !openModal.hidden || !saveasModal.hidden || !settingsPanel.hidden || !shortcutsOverlay.hidden || !exportPanel.hidden;

  const ctrl = e.ctrlKey || e.metaKey;

  // Escape — close any open panel
  if (e.key === 'Escape') {
    if (!commandPalette.hidden)   { closeCommandPalette(); return; }
    if (!searchBar.hidden)         { closeSearch(); return; }
    if (!historyPanel.hidden)      { historyPanel.hidden = true; return; }
    if (!settingsPanel.hidden)     { settingsPanel.hidden = true; return; }
    if (!shortcutsOverlay.hidden)  { shortcutsOverlay.hidden = true; return; }
    if (!exportPanel.hidden)       { exportPanel.hidden = true; return; }
    if (!openModal.hidden)         { closeOpenModal(); return; }
    if (!saveasModal.hidden)       { closeSaveAsModal(); return; }
    if (!shareModal.hidden)        { closeShareModal(); return; }
    if (isDistractionFree)         { toggleDistractionFree(); return; }
    return;
  }

  if (anyModalOpen) return;

  // ? → shortcuts overlay (when editor not focused)
  if (e.key === '?' && document.activeElement !== editor) {
    shortcutsOverlay.hidden = false;
    return;
  }

  if (ctrl && e.key === 'n') { e.preventDefault(); newTab(); return; }
  if (ctrl && e.key === 'w') { e.preventDefault(); closeTab(state.active_index); return; }
  if (ctrl && e.key === 'd') { e.preventDefault(); duplicateTab(state.active_index); return; }
  if (ctrl && e.key === 'o') { e.preventDefault(); openOpenModal(); return; }
  if (ctrl && e.key === 's' && !e.shiftKey) { e.preventDefault(); handleSave(); return; }
  if (ctrl && e.key === 's' &&  e.shiftKey) { e.preventDefault(); openSaveAsModal(); return; }
  if (ctrl && e.key === 'p') { e.preventDefault(); openCommandPalette(); return; }
  if (ctrl && e.key === 'k') { e.preventDefault(); openCommandPalette(); return; }
  if (ctrl && e.key === 'f') { e.preventDefault(); openSearch(); return; }
  if (ctrl && e.key === 'h') { e.preventDefault(); openSearchReplace(); return; }
  if (ctrl && e.key === 'm') { e.preventDefault(); togglePreview(); return; }
  if (ctrl && e.key === ',') { e.preventDefault(); settingsPanel.hidden = false; return; }
  if (ctrl && e.shiftKey && e.key === 'F') { e.preventDefault(); toggleDistractionFree(); return; }
  if (ctrl && e.shiftKey && e.key === 'H') { e.preventDefault(); historyPanel.hidden = !historyPanel.hidden; renderHistoryPanel(); return; }

  // Font size
  if (ctrl && (e.key === '+' || e.key === '=')) {
    e.preventDefault();
    adjustTabFontSize(+1);
    return;
  }
  if (ctrl && e.key === '-') {
    e.preventDefault();
    adjustTabFontSize(-1);
    return;
  }
  if (ctrl && e.key === '0') {
    e.preventDefault();
    resetTabFontSize();
    return;
  }

  // Tab navigation
  if (ctrl && e.key === 'Tab' && !e.shiftKey) {
    e.preventDefault();
    switchTab((state.active_index + 1) % state.tabs.length);
    return;
  }
  if (ctrl && e.key === 'Tab' && e.shiftKey) {
    e.preventDefault();
    switchTab((state.active_index - 1 + state.tabs.length) % state.tabs.length);
    return;
  }
  if (ctrl && e.key >= '1' && e.key <= '9') {
    const idx = parseInt(e.key, 10) - 1;
    if (idx < state.tabs.length) { e.preventDefault(); switchTab(idx); }
    return;
  }
}

document.addEventListener('keydown', onKeyDown);

// ── Per-tab Font Size ─────────────────────────────────────────────────────────

function adjustTabFontSize(delta) {
  const tab = state.tabs[state.active_index];
  if (!tab) return;
  const current = tabFontSizes.get(tab.id) || settings.fontSize;
  const next = Math.max(10, Math.min(32, current + delta));
  tabFontSizes.set(tab.id, next);
  editor.style.fontSize = next + 'px';
  editorHighlight.style.fontSize = next + 'px';
  updateLineGutter();
}

function resetTabFontSize() {
  const tab = state.tabs[state.active_index];
  if (!tab) return;
  tabFontSizes.delete(tab.id);
  editor.style.fontSize = '';
  editorHighlight.style.fontSize = '';
  updateLineGutter();
}

// ── Editor Events ─────────────────────────────────────────────────────────────

editor.addEventListener('input', () => {
  scheduleSave();
  updateCursorStatus();
  updateHighlight();
  scheduleAutoTitle();
  if (searchState.query) runSearch();
  if (isPreviewOpen) renderMarkdownPreview(editor.value);
});

editor.addEventListener('scroll', () => {
  editorHighlight.scrollTop = editor.scrollTop;
  editorHighlight.scrollLeft = editor.scrollLeft;
  lineGutter.scrollTop = editor.scrollTop;
  updateActiveLine();
});

editor.addEventListener('keyup',        updateCursorStatus);
editor.addEventListener('click',        updateCursorStatus);
editor.addEventListener('mouseup',      updateCursorStatus);
document.addEventListener('selectionchange', () => {
  if (document.activeElement === editor) updateCursorStatus();
});

// ── Window Controls ───────────────────────────────────────────────────────────

btnMinimize?.addEventListener('click', () => window.runtime?.WindowMinimise?.());
btnMaximize?.addEventListener('click', () => window.runtime?.WindowToggleMaximise?.());
btnClose?.addEventListener('click',   () => window.runtime?.Quit?.());

btnOntop?.addEventListener('click', () => {
  isAlwaysOnTop = !isAlwaysOnTop;
  window.runtime?.WindowSetAlwaysOnTop?.(isAlwaysOnTop);
  btnOntop.classList.toggle('share-btn--active', isAlwaysOnTop);
  btnOntop.setAttribute('aria-pressed', String(isAlwaysOnTop));
});

btnNewTab.addEventListener('click', newTab);

// ── Onboarding / First Run ────────────────────────────────────────────────────

async function maybeShowOnboarding(isFirstRun) {
  if (!isFirstRun) return;

  welcomeOverlay.hidden = false;
  await new Promise(r => setTimeout(r, 2500));
  welcomeOverlay.classList.add('fade-out');
  await new Promise(r => setTimeout(r, 400));
  welcomeOverlay.hidden = true;

  // Pre-populate first tab with getting started guide
  const guide = `# Welcome to octoNote 🐙

> Your lightning-fast, crash-proof, multi-tab scratchpad.

## Getting Started

- **Multiple tabs** — Press \`Ctrl+N\` to create new tabs.
- **Switch tabs** — Use \`Ctrl+Tab\` or \`Ctrl+1\`…\`9\` to jump.
- **Rename tabs** — Double-click any tab to rename it.
- **Auto-save** — Every keystroke is saved. You never need to press Ctrl+S.

## Key Features

- 🔍 \`Ctrl+F\` — Find in current tab
- 👁 \`Ctrl+M\` — Toggle Markdown Preview (try it now!)
- ⌨ \`Ctrl+P\` — Command Palette (search everything)
- 📤 **Share** button — Send this tab to anyone, P2P encrypted

## Markdown Support

This is **bold**, this is *italic*, and \`this is code\`.

\`\`\`
function hello() {
  return "octoNote is awesome!";
}
\`\`\`

## File Support

- \`Ctrl+O\` — Open a file from disk
- \`Ctrl+S\` — Save to disk
- \`Ctrl+Shift+S\` — Save As…

---

*Delete this tab anytime. Everything auto-saves. Happy hacking!*
`;

  const idx = state.active_index;
  editor.value = guide;
  updateHighlight();
  await window.go.main.App.SaveTab(idx, guide, 0);
  await window.go.main.App.RenameTab(idx, 'Getting Started');
  state = await window.go.main.App.GetState();
  renderTabs();
}

// ── Init ──────────────────────────────────────────────────────────────────────

async function init() {
  loadSettings();
  try {
    state = await window.go.main.App.GetState();
    const isFirstRun = state.tabs.length === 1 && !state.tabs[0].body;
    renderTabs();
    loadActiveTabIntoEditor();
    await maybeShowOnboarding(isFirstRun);
  } catch (err) {
    console.error('octoNote: failed to load state', err);
    renderFallback();
  }

  document.addEventListener('keydown', onKeyDown);
  registerWailsEvents();
}

function renderFallback() {
  state = {
    tabs: [{ id: 'fallback', title: 'scratch', body: '', cursor_line: 0 }],
    active_index: 0,
  };
  renderTabs();
  editor.placeholder = 'Go bridge unavailable — running in offline mode.';
}

window.addEventListener('DOMContentLoaded', () => {
  let attempts = 0;
  const poll = setInterval(() => {
    if (window.go?.main?.App || attempts > 50) {
      clearInterval(poll);
      init();
    }
    attempts++;
  }, 50);
});

// ── Wails Events ──────────────────────────────────────────────────────────────

function registerWailsEvents() {
  const waitRuntime = setInterval(() => {
    if (!window.runtime?.EventsOn) return;
    clearInterval(waitRuntime);

    window.runtime.EventsOn('share:code', (code) => {
      shareCodeValue.textContent = code;
      shareHostStatus.textContent = '⏳ Waiting for peer to connect…';
    });

    window.runtime.EventsOn('share:done', () => {
      shareHostActive.setAttribute('hidden', '');
      shareHostDone.removeAttribute('hidden');
      showToast('Tab shared successfully! ✅', 'success');
      setTimeout(closeShareModal, 2000);
    });

    window.runtime.EventsOn('share:received', (payload) => {
      state = payload.state;
      renderTabs();
      loadActiveTabIntoEditor();
      shareGuestWaiting.setAttribute('hidden', '');
      shareGuestDone.removeAttribute('hidden');
      shareGuestDoneMsg.textContent = `✅ Tab "${payload.title}" added to your notebook!`;
      showToast(`Received tab: "${payload.title}"`, 'success');
      setTimeout(closeShareModal, 2000);
    });

    window.runtime.EventsOn('state:changed', (newSt) => {
      state = newSt;
      renderTabs();
      const tab = state.tabs[state.active_index];
      if (tab && editor.value !== (tab.body || '')) {
        const start = editor.selectionStart;
        const end   = editor.selectionEnd;
        editor.value = tab.body || '';
        updateHighlight();
        try { editor.setSelectionRange(start, end); } catch (_) {}
      }
      updateFileStatus();
      updateCursorStatus();
    });

    window.runtime.EventsOn('share:error', (msg) => {
      resetShareModal();
      showShareError(msg);
      showToast('Share error: ' + msg, 'error');
    });
  }, 100);
}

// ── Share Modal ───────────────────────────────────────────────────────────────

const shareModal      = document.getElementById('share-modal');
const btnShare        = document.getElementById('btn-share');
const shareModalClose = document.getElementById('share-modal-close');
const shareTabHost    = document.getElementById('share-tab-host');
const shareTabGuest   = document.getElementById('share-tab-guest');
const sharePanelHost  = document.getElementById('share-panel-host');
const sharePanelGuest = document.getElementById('share-panel-guest');
const shareHostIdle      = document.getElementById('share-host-idle');
const shareHostActive    = document.getElementById('share-host-active');
const shareHostDone      = document.getElementById('share-host-done');
const shareCodeValue     = document.getElementById('share-code-value');
const btnCopyCode        = document.getElementById('btn-copy-code');
const btnGenerateCode    = document.getElementById('btn-generate-code');
const btnCancelShare     = document.getElementById('btn-cancel-share');
const shareHostStatus    = document.getElementById('share-host-status');
const shareSenderName    = document.getElementById('share-sender-name');
const sharePreviewTitle  = document.getElementById('share-preview-title');
const shareSenderChip    = document.getElementById('share-sender-chip');
const shareSenderChipName = document.getElementById('share-sender-chip-name');
const shareGuestIdle    = document.getElementById('share-guest-idle');
const shareGuestWaiting = document.getElementById('share-guest-waiting');
const shareGuestDone    = document.getElementById('share-guest-done');
const shareGuestDoneMsg = document.getElementById('share-guest-done-msg');
const shareCodeInput    = document.getElementById('share-code-input');
const btnReceiveCode    = document.getElementById('btn-receive-code');
const btnCancelReceive  = document.getElementById('btn-cancel-receive');
const shareError        = document.getElementById('share-error');

function openShareModal() {
  resetShareModal();
  const activeTab = state.tabs?.[state.active_index];
  if (activeTab) sharePreviewTitle.textContent = activeTab.title || 'scratch';
  shareModal.removeAttribute('hidden');
  btnShare.classList.add('share-btn--active');
  shareSenderName.focus();
}

function closeShareModal() {
  shareModal.setAttribute('hidden', '');
  btnShare.classList.remove('share-btn--active');
  window.go?.main?.App?.ShareCancel?.();
  editor.focus();
}

function resetShareModal() {
  shareError.setAttribute('hidden', '');
  shareError.textContent = '';
  shareHostIdle.removeAttribute('hidden');
  shareHostActive.setAttribute('hidden', '');
  shareHostDone.setAttribute('hidden', '');
  shareCodeValue.textContent = '—';
  shareSenderName.value = '';
  shareSenderChip.setAttribute('hidden', '');
  shareSenderChipName.textContent = '';
  shareGuestIdle.removeAttribute('hidden');
  shareGuestWaiting.setAttribute('hidden', '');
  shareGuestDone.setAttribute('hidden', '');
  shareCodeInput.value = '';
}

function switchShareTab(tab) {
  const isHost = tab === 'host';
  shareTabHost.classList.toggle('share-tab--active', isHost);
  shareTabGuest.classList.toggle('share-tab--active', !isHost);
  shareTabHost.setAttribute('aria-selected', isHost ? 'true' : 'false');
  shareTabGuest.setAttribute('aria-selected', !isHost ? 'true' : 'false');
  sharePanelHost.toggleAttribute('hidden', !isHost);
  sharePanelGuest.toggleAttribute('hidden', isHost);
}

function showShareError(msg) {
  shareError.textContent = '⚠ ' + msg;
  shareError.removeAttribute('hidden');
}

btnShare.addEventListener('click', openShareModal);
shareModalClose.addEventListener('click', closeShareModal);
shareModal.querySelector('.share-modal__backdrop').addEventListener('click', closeShareModal);
shareTabHost.addEventListener('click', () => switchShareTab('host'));
shareTabGuest.addEventListener('click', () => switchShareTab('guest'));

btnGenerateCode.addEventListener('click', async () => {
  const label = shareSenderName.value.trim();
  shareError.setAttribute('hidden', '');
  shareHostIdle.setAttribute('hidden', '');
  shareHostActive.removeAttribute('hidden');
  shareCodeValue.textContent = 'opening wormhole…';
  shareHostStatus.textContent = '⏳ Connecting to relay…';
  if (label) { shareSenderChipName.textContent = label; shareSenderChip.removeAttribute('hidden'); }
  else        { shareSenderChip.setAttribute('hidden', ''); }
  await flushSave();
  window.go.main.App.ShareSend(label);
});

btnCopyCode.addEventListener('click', async () => {
  const code = shareCodeValue.textContent;
  if (!code || code === '—' || code === 'opening wormhole…') return;
  try {
    await navigator.clipboard.writeText(code);
    btnCopyCode.classList.add('copied');
    showToast('Code copied to clipboard ✓', 'success');
    setTimeout(() => btnCopyCode.classList.remove('copied'), 1500);
  } catch (_) {}
});

btnCancelShare.addEventListener('click', () => { window.go.main.App.ShareCancel(); resetShareModal(); });

btnReceiveCode.addEventListener('click', startReceive);
shareCodeInput.addEventListener('keydown', e => { if (e.key === 'Enter') startReceive(); });

function startReceive() {
  const code = shareCodeInput.value.trim();
  if (!code) { showShareError('Please enter a share code.'); return; }
  shareError.setAttribute('hidden', '');
  shareGuestIdle.setAttribute('hidden', '');
  shareGuestWaiting.removeAttribute('hidden');
  window.go.main.App.ShareReceive(code);
}

btnCancelReceive.addEventListener('click', () => {
  window.go.main.App.ShareCancel();
  resetShareModal();
  switchShareTab('guest');
});

// ── File Open / Save ──────────────────────────────────────────────────────────

async function handleSave() {
  await flushSave();
  const idx = state.active_index;
  const tab = state.tabs[idx];
  if (!tab) return;
  if (tab.file_path) {
    const errMsg = await window.go.main.App.SaveCurrentFile(idx, editor.value);
    if (errMsg) { showToast(errMsg, 'error'); }
    else {
      tab.file_is_dirty = false;
      setSaveStatus('saved');
      renderTabs();
      showToast('Saved ✓', 'success');
    }
  } else {
    openSaveAsModal();
  }
}

function openOpenModal() {
  openModalErr.setAttribute('hidden', '');
  openModalErr.textContent = '';
  openPathInput.value = '';
  openModal.removeAttribute('hidden');
  openPathInput.focus();
}

function closeOpenModal() { openModal.setAttribute('hidden', ''); editor.focus(); }

async function confirmOpen() {
  const path = openPathInput.value.trim();
  if (!path) return;
  openModalErr.setAttribute('hidden', '');
  const result = await window.go.main.App.OpenFile(path);
  if (result.error) {
    openModalErr.textContent = result.error;
    openModalErr.removeAttribute('hidden');
    return;
  }
  closeOpenModal();
  const currentTab = state.tabs[state.active_index];
  const currentEmpty = !currentTab?.body?.trim() && !currentTab?.file_path;
  if (currentEmpty) {
    const idx = state.active_index;
    const filename = path.split('/').pop();
    state.tabs[idx].title = filename;
    state.tabs[idx].body = result.content;
    state.tabs[idx].file_path = path;
    state.tabs[idx].file_is_dirty = false;
    editor.value = result.content;
    await window.go.main.App.SaveFileAs(idx, path, result.content);
    renderTabs(); updateFileStatus(); setSaveStatus('saved');
    showToast('Opened: ' + filename, 'success');
  } else {
    state = await window.go.main.App.NewTab();
    const newIdx = state.active_index;
    const filename = path.split('/').pop();
    await window.go.main.App.SaveFileAs(newIdx, path, result.content);
    state = await window.go.main.App.GetState();
    renderTabs(); loadActiveTabIntoEditor(); updateFileStatus(); setSaveStatus('saved');
    showToast('Opened: ' + filename, 'success');
  }
}

openModalConfirm.addEventListener('click', confirmOpen);
openModalClose.addEventListener('click', closeOpenModal);
openModal.querySelector('.open-modal__backdrop').addEventListener('click', closeOpenModal);
openPathInput.addEventListener('keydown', e => {
  if (e.key === 'Enter') { e.preventDefault(); confirmOpen(); }
  if (e.key === 'Escape') { e.preventDefault(); closeOpenModal(); }
  e.stopPropagation();
});

function openSaveAsModal(afterSaveClose = false) {
  pendingCloseAfterSave = afterSaveClose;
  saveasModalErr.setAttribute('hidden', '');
  saveasModalErr.textContent = '';
  const tab = state.tabs[state.active_index];
  saveasPathInput.value = tab?.file_path || '';
  saveasModal.removeAttribute('hidden');
  saveasPathInput.focus();
  saveasPathInput.select();
}

function closeSaveAsModal() {
  saveasModal.setAttribute('hidden', '');
  pendingCloseAfterSave = false;
  editor.focus();
}

async function confirmSaveAs() {
  const path = saveasPathInput.value.trim();
  if (!path) return;
  saveasModalErr.setAttribute('hidden', '');
  await flushSave();
  const idx = state.active_index;
  const content = editor.value;
  const errMsg = await window.go.main.App.SaveFileAs(idx, path, content);
  if (errMsg) {
    saveasModalErr.textContent = errMsg;
    saveasModalErr.removeAttribute('hidden');
    return;
  }
  state.tabs[idx].file_path = path;
  state.tabs[idx].file_is_dirty = false;
  state.tabs[idx].title = path.split('/').pop();
  closeSaveAsModal();
  renderTabs(); updateFileStatus(); setSaveStatus('saved');
  showToast('Saved to: ' + path.split('/').pop(), 'success');
  if (pendingCloseAfterSave) {
    pendingCloseAfterSave = false;
    state = await window.go.main.App.CloseTab(idx);
    renderTabs(); loadActiveTabIntoEditor(); updateFileStatus();
  }
}

saveasModalConfirm.addEventListener('click', confirmSaveAs);
saveasModalClose.addEventListener('click', closeSaveAsModal);
saveasModal.querySelector('.open-modal__backdrop').addEventListener('click', closeSaveAsModal);
saveasPathInput.addEventListener('keydown', e => {
  if (e.key === 'Enter') { e.preventDefault(); confirmSaveAs(); }
  if (e.key === 'Escape') { e.preventDefault(); closeSaveAsModal(); }
  e.stopPropagation();
});

// ── System theme change listener ──────────────────────────────────────────────

window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
  if (settings.theme === 'system') applyTheme('system');
});
