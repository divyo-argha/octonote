# octoNote

**A lightning-fast, crash-proof, multi-tab scratchpad for your terminal.**  
Stop hitting Ctrl+S. Stop losing quick ideas. Stop naming throwaway files.

[![Latest Release](https://img.shields.io/github/v/release/divyo-argha/octonote?style=flat-square&color=00FFAA&label=latest)](https://github.com/divyo-argha/octonote/releases)
[![npm](https://img.shields.io/npm/v/octonote?style=flat-square&color=CB3837&logo=npm&logoColor=white&label=npm)](https://www.npmjs.com/package/octonote)
[![total downloads](https://img.shields.io/npm/dt/octonote?style=flat-square&color=CB3837&logo=npm&logoColor=white&label=total%20downloads)](https://www.npmjs.com/package/octonote)
[![License-Apache_2.0](https://img.shields.io/badge/License-Apache_2.0-blue?style=flat-square)](LICENSE)

[The Problem](#-the-problem) · [Installation](#-installation) · [Quick Start](#-quick-start) · [Why octoNote](#-why-octonote) · [Features](#-features) · [Shortcuts](#-keyboard-shortcuts) · [Persistence](#-persistence)

---

![macOS](https://img.shields.io/badge/macOS-supported-000000?style=for-the-badge&logo=apple&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-supported-FCC624?style=for-the-badge&logo=linux&logoColor=black)
![Windows](https://img.shields.io/badge/Windows-supported-0078D4?style=for-the-badge&logo=windows&logoColor=white)
![Terminal TUI](https://img.shields.io/badge/Terminal_TUI-supported-4E9A06?style=for-the-badge&logo=gnubash&logoColor=white)

---


---

## 😤 The Problem

You're a developer coding away in your terminal. You need to quickly copy a JSON payload, sketch a rough database schema, draft an email, or keep a temporary todo checklist.

And this happens every day:

```
# You open nano or vim.
# You type your notes.
# Later, you want to close it:
"Save modified buffer?"  ← 💀 annoying prompt.

# Or you name it temp_123.txt and throw it in ~/Desktop.
# Now your Desktop is a graveyard of 80 unnamed text files.
# Or your terminal session closes, and your unsaved buffer is gone forever.
```

**octoNote is the permanent fix.** A dedicated scratchpad workspace that opens instantly inside your terminal, supports multiple tabs, and auto-saves every single character to disk. You never name a scratch file, and you never hit save.

---

## 📦 Installation

This npm package automatically downloads the optimized Go binary matching your operating system and architecture upon installation.

```bash
# Install TUI (CLI) globally
npm install -g octonote

# Run TUI instantly without install
npx octonote
```



---

## ⚡ Quick Start

Launch the terminal scratchpad with one command:

```bash
octonote
```

```
✦ octonote
 1: scratch   2: ideas   3: todo   [+]
╭──────────────────────────────────────────╮
│ Start typing…                            │
│                                          │
╰──────────────────────────────────────────╯
^N new  ^W close  ^→/← switch  Tab cycle  ^C quit          ✓ saved 21:04:55
```

Write notes, open multiple tabs, exit anytime, and reopen. Everything is exactly where you left it.

---

## 🛠️ Command Flags

| Flag | Action |
|---|---|
| `-v`, `--version` | Print the installed version of octoNote |
| `--update` | Check for updates (or prompt to update npm/standalone versions) |
| `-h`, `--help` | Show CLI usage help message |

---

## 🏆 Why octoNote?

| Feature | octoNote | `nano` / `vim` | standard Notes App |
|---------|:--------:|:--------------:|:------------------:|
| Instantly open | ✅ | ✅ | ❌ |
| Keyboard-driven TUI | ✅ | ✅ | ❌ |
| Multi-tab workspace | ✅ | ❌ | ❌ |
| Auto-save on every keystroke | ✅ | ❌ | ⚠️ sync delay |
| Crash-proof atomic writes | ✅ | ❌ | ❌ |
| Zero file naming overhead | ✅ | ❌ | ❌ |

---

## ✨ Features

- 📝 **Zero-Management Notes** — Create and destroy tabs on the fly with shortcuts. No "Save as..." prompts. No files or directory trees to organize.
- 🛡️ **Crash-Proof Engine** — Saves state on every single keystroke. Uses atomic temp-file swaps so you never lose or corrupt your notes.
- 💻 **Platform-Native Storage** — Saves to standardized locations depending on your OS.
- 🚀 **Built in Go** — Ultra-lightweight, extremely fast startup, minimal memory consumption.

---

## ⌨️ Keyboard Shortcuts

| Shortcut | Action |
|---|---|
| `Ctrl+N` | Create new tab |
| `Ctrl+W` | Delete / close current tab |
| `Ctrl+Tab` | Switch to next tab |
| `Ctrl+Shift+Tab` | Switch to previous tab |
| `Ctrl+1` ... `Ctrl+9` | Jump directly to tab by index |
| `Tab` | Cycle focus between tab bar and text area |
| `Ctrl+C` | Quit application |

---

## 📁 Persistence

Notes are saved as `state.json` inside your platform's standard configuration directory:

- **macOS:** `~/Library/Application Support/octonote/state.json`
- **Linux:** `~/.config/octonote/state.json`
- **Windows:** `%APPDATA%\octonote\state.json`

---

## 📄 License

Apache License, Version 2.0 — see [LICENSE](https://github.com/divyo-argha/octonote/blob/main/LICENSE) for details.
