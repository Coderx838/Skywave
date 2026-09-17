# Skywave

A high-performance terminal chat client and lightweight WebSocket relay server implemented in Go. Built using Charm's Bubble Tea TUI framework and Lip Gloss styling engine.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![TUI Architecture](https://img.shields.io/badge/Architecture-Elm%20%2F%20Bubble%20Tea-F25D94?style=flat)](https://github.com/charmbracelet/bubbletea)
[![Protocol](https://img.shields.io/badge/Protocol-WebSocket%20%2F%20JSON-7D56F4?style=flat)](https://w3.org/TR/websockets/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Overview

Skywave is designed for developers, systems engineers, and power users who operate primarily within terminal environments. It provides a keyboard-driven interface with low resource consumption, zero tracking, client-side encryption capabilities, and simple self-hosting.

### Core Capabilities

- **Multi-Pane Interface**: Four-quadrant layout providing focused views for channel navigation, scrollable message streams, presence tracking, and command input.
- **Fuzzy Quick Jumper**: Global command palette for rapid switching between channels, direct messages, and system actions.
- **End-to-End Encryption (E2EE)**: Optional AES-256-GCM client-side encryption for secure rooms. Server instances route ciphertext and cannot inspect plaintext payloads.
- **Decentralized Node Switching**: Connect to local nodes, private self-hosted servers, or community instances without client restarts.
- **Cryptographic Identity**: Persistent local keypairs for identity verification without requiring email addresses or third-party OAuth providers.
- **Configurable Persistence**: Ephemeral in-memory operation by default, with adjustable buffer depth for room history.

---

## Architecture

```
+-------------------------------------------------------------+
|                      Skywave Client                         |
|  +----------------+  +-------------------+  +------------+  |
|  | Channels View  |  | Message Viewport  |  | User List  |  |
|  +----------------+  +-------------------+  +------------+  |
|  | Command / Chat Input Buffer                           |  |
|  +-------------------------------------------------------+  |
|  | Bubble Tea Runtime | Lip Gloss Engine | Local Keystore|  |
+------------------------------+------------------------------+
                               |
                   WebSocket (JSON Frames / TLS)
                               |
+------------------------------v------------------------------+
|                      Skywave Server                         |
|  +-------------------------------------------------------+  |
|  | HTTP /ws Upgrade Handler & Rate Limiter               |  |
|  +-------------------------------------------------------+  |
|  | Hub Event Loop (Broadcast, Join, Leave, Direct Msg)   |  |
|  +-------------------------------------------------------+  |
|  | Ephemeral Ring Buffers (Configurable History Depth)   |  |
+-------------------------------------------------------------+
```

---

## Installation

### Prerequisites

- Go 1.24 or higher
- Standard POSIX or Windows terminal emulator with ANSI color support

### Build from Source

Clone the repository and compile the binaries:

```bash
git clone https://github.com/Coderx838/Skywave.git
cd skywave

# Compile the client
go build -o bin/skywave ./cmd/skywave

# Compile the server
go build -o bin/skywave-server ./cmd/skywave-server
```

On Windows:
```powershell
go build -o bin\skywave.exe .\cmd\skywave
go build -o bin\skywave-server.exe .\cmd\skywave-server
```

---

## Server Deployment

The server binary is lightweight and has zero external dependencies outside of Go standard libraries and `gorilla/websocket`.

### Running Locally

```bash
./bin/skywave-server -port 8080 -host 0.0.0.0
```

### Server Configuration Options

| Flag | Env Variable | Default | Description |
| :--- | :--- | :--- | :--- |
| `-port` | `PORT` | `8080` | Port for the HTTP/WebSocket listener |
| `-host` | — | `0.0.0.0` | Network interface address to bind |
| `-name` | — | `Skywave Prime Node` | Public identifier for the node |
| `-password` | — | `""` (none) | Require authentication token to access node |
| `-motd` | — | `Welcome to Skywave...` | Message displayed to users on connect |
| `-history` | — | `50` | In-memory message backlog size per channel (`0` for ephemeral) |

### Docker Deployment

A multi-stage `Dockerfile` is provided in the repository root:

```bash
# Build container image
docker build -t skywave-server .

# Run container
docker run -d -p 8080:8080 --name skywave skywave-server
```

### Cloud Deployment (Render, Fly.io, Railway)

The server automatically detects the `PORT` environment variable injected by PaaS providers.

1. Push your repository to GitHub.
2. Link the repository to a new Web Service.
3. Set the environment to **Docker**.
4. Configure the optional health check path to `/health`.
5. Connect your client via `wss://<your-service-domain>/ws`.

---

## Client Usage

Start the interactive terminal interface:

```bash
# Default connection to local server (ws://localhost:8080/ws)
./bin/skywave

# Connect to a remote server
./bin/skywave -server wss://your-node-domain.com/ws

# Specify callsign at startup
./bin/skywave -nick Ghost
```

---

## Keyboard Navigation

### Global Shortcuts

| Key Combination | Function |
| :--- | :--- |
| `Tab` / `Shift+Tab` | Cycle focus between panes (`Input` -> `Channels` -> `Chat` -> `Users`) |
| `Ctrl+K` / `Ctrl+P` | Open Spotlight Quick Jumper palette |
| `Ctrl+S` | Open Server Switcher drawer |
| `Ctrl+N` | Create new channel or encrypted E2EE room |
| `Ctrl+T` | Cycle visual themes |
| `Ctrl+B` | Toggle terminal bell notifications |
| `F1` / `Ctrl+H` | Display keyboard reference dialog |
| `Esc` | Close active dialog or return focus to input |

### Pane-Specific Controls

#### Channels Pane
- `Up` / `Down` or `k` / `j`: Move selection cursor
- `Enter` / `Space`: Switch to selected channel
- `n`: Prompt new channel dialog

#### Chat Viewport
- `Up` / `Down` or `k` / `j`: Scroll message history line-by-line
- `PageUp` / `PageDown`: Scroll message history by page
- `g` / `G`: Jump to beginning / end of message stream
- `y`: Copy latest visible message to system clipboard

#### Users Pane
- `Up` / `Down` or `k` / `j`: Move selection cursor
- `Enter`: Pre-populate direct message command buffer

---

## Command Reference

Commands begin with a forward slash (`/`) and are entered in the input field:

| Command | Arguments | Description |
| :--- | :--- | :--- |
| `/join` | `<#channel> [key]` | Switch to channel, optionally unlocking E2EE room with key |
| `/leave` | — | Leave current channel and return to `#general` |
| `/dm` | `<@user> <message>` | Send a private message directly to a user |
| `/nick` | `<name>` | Update active callsign |
| `/topic` | `<text>` | Set topic for the current channel |
| `/server` | — | Open the server connection manager |
| `/theme` | `[name]` | Set color palette (`skywave`, `tokyo`, `cyberpunk`, `nord`, `dracula`, `mono`) |
| `/whoami` | — | Display cryptographic identifier and connection status |
| `/clear` | — | Clear messages in the active viewport buffer |
| `/quit` | — | Terminate the application |

---

## Security Model

- **Transport Security**: Deploying behind TLS reverse proxies (such as Nginx, Caddy, or Cloudflare) ensures all WebSocket communications are protected using standard WSS encryption.
- **End-to-End Encryption**: When creating private rooms via `Ctrl+N`, messages are encrypted with AES-256-GCM on the sender's client before transmission. Decryption occurs exclusively on clients possessing the shared room secret.
- **Zero Logging Mode**: By specifying `-history 0` on the server, messages are strictly relayed to active connections and never retained in memory.

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
