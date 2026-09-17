# 🌊 SKYWAVE PRIME

<div align="center">

**The Ultimate Pro Terminal Chat Client & Self-Hostable Decentralized Node.**  
*Engineered in Go with Bubble Tea & Lip Gloss.*

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![TUI](https://img.shields.io/badge/TUI-Bubble%20Tea%20Prime-F25D94?style=flat)](https://github.com/charmbracelet/bubbletea)
[![Styling](https://img.shields.io/badge/Style-Lip%20Gloss-7D56F4?style=flat)](https://github.com/charmbracelet/lipgloss)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

</div>

---

## ⚡ What Makes Skywave Next-Gen?

Unlike basic "nested box" terminal scripts, **Skywave Prime** delivers a true power-user experience inspired by `lazygit`, `k9s`, and `Raycast`:

- **🎮 OP Multi-Pane Navigation**:
  - `Tab` / `Shift+Tab`: Effortlessly cycle focus between **Channels [1]**, **Chat Viewport [2]**, **Online Souls [3]**, and **Input [4]**.
  - **Illuminated Active Pane**: High-contrast neon borders with vibrant headers and active cursors (`▶ #room`, `▶ @user`).
  - `↑` / `↓` or `k` / `j`: Smooth cursor movement in any pane.
  - `Enter` / `Space` in Channels: Instant channel switch!
  - `Enter` in Users: Auto-populates `/dm @user ` prompt!
  - `y` in Chat: Yanks message text directly to your OS clipboard!
- **🔍 Spotlight Quick Jumper (`Ctrl+K` / `Ctrl+P`)**:
  - Floating Raycast/VSCode-style command palette with real-time fuzzy filtering.
  - Instantly jump to any `#channel`, `@user` DM, or execute commands (`/theme`, `/party`, `/hack`, `/server`, `/leave`) by typing 2 letters!
- **🌐 Multi-Server Switcher Drawer (`Ctrl+S`)**:
  - Switch live between the **Official Skywave Hub** (`wss://hub.skywave.chat/ws`), local node, or private VPS without restarting the app!
  - Save custom server bookmarks into `~/.skywave/servers.json`.
  - Add new servers with `A`.
- **🔒 AES-256-GCM End-to-End Encryption (E2EE)**:
  - Create secret rooms via `Ctrl+N` with a local passphrase.
  - Messages are encrypted client-side; server nodes relay only ciphertext.
  - Visual `[🔒 E2EE SHIELD ACTIVE]` banner in the chat header.
- **✨ Rich Chat Feed & Slack/Discord-Style Grouping**:
  - Consecutive messages from the same sender within 90s merge cleanly without repeating headers.
  - **Emoji Shortcodes**: `:fire:` → 🔥, `:rocket:` → 🚀, `:skull:` → 💀, `:heart:` → ❤️, `:100:` → 💯, `:party:` → 🎉, and 30+ more!
  - **Markdown Codeblocks**: Formatted with dedicated syntax containers (` ```go ... ``` `).
  - **URL Highlighting**: Clickable or highlighted link detection.
  - **@Mention Highlighting**: Visual badge tags when your name is called.
- **🎨 6 Pro Color Themes (`Ctrl+T`)**:
  - Instant theme cycling: `skywave` · `tokyo` · `cyberpunk` · `nord` · `dracula` · `mono`.
- **🔔 Audio Chime & Visual Toasts (`Ctrl+B`)**:
  - Terminal audio bell chime on DM or `@mention` (toggleable on/off).
  - Sleek top-right animated status toast pill.
- **👻 Persistent Anonymous Ghost Accounts**:
  - Zero emails, passwords, or personal data.
  - Generates a persistent anonymous keypair (`~/.skywave/identity.json`) so your callsign stays yours across reboots.

---

## 🚀 Quick Start

### 1. Build Binaries
```bash
go build -o bin/skywave.exe ./cmd/skywave
go build -o bin/skywave-server.exe ./cmd/skywave-server
```

### 2. Launch the Server
```bash
# Start your own node on 0.0.0.0:8080
./bin/skywave-server

# Or private password-protected node:
./bin/skywave-server -name "Dark Wave Node" -password "topsecret"
```

### 3. Launch the Client (TUI)
```bash
# Connect with saved anonymous callsign
./bin/skywave

# Connect with custom callsign
./bin/skywave -nick neo

# Connect to any remote server
./bin/skywave -server ws://your-vps.com:8080/ws
```

---

## ⌨️ Pro Keybindings & Shortcuts

| Key | Scope | Action |
| :--- | :--- | :--- |
| **`Tab` / `Shift+Tab`** | Global | Cycle active pane (`Input` → `Channels` → `Chat` → `Users`) |
| **`Ctrl+K` / `Ctrl+P`** | Global | **Spotlight Quick Jumper** (Fuzzy find rooms, users, actions) |
| **`Ctrl+S`** | Global | **Server Switcher Drawer** (Switch or add nodes live) |
| **`Ctrl+N`** | Global | **Create Channel / E2EE Encrypted Room** |
| **`Ctrl+T`** | Global | **Cycle Color Themes** (`skywave`, `tokyo`, `cyberpunk`, `nord`...) |
| **`Ctrl+B`** | Global | **Toggle Audio Chime Bell** (`🔔 ENABLED` / `🔕 MUTED`) |
| **`F1` / `Ctrl+H`** | Global | **Command Deck / Hotkey Manual** |
| **`Esc`** | Any Pane | Cancel modals / jump immediately back to typing |
| **`↑` / `↓` or `k` / `j`** | Channels [1] | Navigate room cursor (`▶ #room`) |
| **`Enter` / `Space`** | Channels [1] | Switch to highlighted channel |
| **`n`** | Channels [1] | Create new channel dialog |
| **`↑` / `↓` or `k` / `j`** | Chat [2] | Scroll chat line-by-line |
| **`g` / `G`** | Chat [2] | Jump to top / bottom of chat history |
| **`y`** | Chat [2] | **Yank/Copy last message to OS clipboard** |
| **`↑` / `↓` or `k` / `j`** | Users [3] | Navigate online members cursor |
| **`Enter`** | Users [3] | Open `/dm @user ` prompt |
| **`Enter`** | Input [4] | Send message or execute slash command |

---

## 💬 Slash Commands

- `/join <#room> [passcode]` — Switch room or tune into E2EE room
- `/leave` — Return to `#general`
- `/dm <@user> <message>` — Send encrypted 1-on-1 direct message
- `/nick <new_name>` — Rename callsign (persisted to anonymous account)
- `/server` — Open multi-server switcher drawer
- `/theme <name>` — Set theme directly (`skywave`, `tokyo`, `cyberpunk`, `nord`, `dracula`, `mono`)
- `/topic <text>` — Set channel description
- `/whoami` — View your anonymous account ID & statistics
- `/party` — Toggle confetti party mode
- `/hack`, `/matrix` — Hacker easter eggs
- `/clear` — Wipe current viewport
- `/quit` — Exit Skywave
