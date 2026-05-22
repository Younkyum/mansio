<p align="center">
  <img src="frontend/public/icon-512.png?v=2" alt="Mansio" width="128" />
</p>

<h1 align="center">Mansio</h1>

<p align="center">
  <a href="README.ko.md">한국어</a> · <a href="README.zh-CN.md">中文</a> · <strong>English</strong>
</p>

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg?logo=react)](https://react.dev/)
[![tmux](https://img.shields.io/badge/tmux-persistent-1BB91F.svg)](https://github.com/tmux/tmux)
[![License: GPL v3+](https://img.shields.io/badge/License-GPLv3+-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

> **Persistent workspace for AI coding agents.**
> Run Claude Code (or any long-lived shell process) on a server, close the browser, hop devices — and come back hours later to find your agent, your build, your `vim` exactly where you left them. tmux on the inside, a real UI on the outside.

[Quick Start](#quick-start) • [Usage Guide](#usage-guide) • [Deployment](#deployment) • [Architecture](#architecture) • [Troubleshooting](#troubleshooting)

---

## Screenshots

<p align="center">
  <img src="docs/screenshots/desktop.png?v=2" alt="Mansio desktop UI — sidebar, tab bar, and persistent terminal" width="820" />
  <br />
  <em>Desktop — workspaces, tab bar, and a tmux-backed terminal panel.</em>
</p>

<p align="center">
  <img src="docs/screenshots/mobile.png?v=2" alt="Mansio mobile UI — collapsible sidebar with dedicated input bar" width="320" />
  <br />
  <em>Mobile — collapsible sidebar, IME-safe input bar, on-screen modifier keys.</em>
</p>

---

## Why I built this

I run **Claude Code** (and other AI coding agents) on a remote dev box. The work is long: multi-step refactors, slow builds, agent loops that chew on a problem for an hour. But every time I closed the browser tab, walked away from my desk, or jumped to my phone, the workflow broke — the SSH session died, the scrollback evaporated, and whatever the agent was in the middle of got yanked out from under it.

So I wrapped `tmux` in a real browser UX. **Mansio gives the agent a persistent home** — a workspace that doesn't care whether I'm at my desk, on my phone, or rebooting the laptop. Close the tab. Walk to a meeting. Reopen on a different device. The agent is still running, the scrollback is intact, the prompt is exactly where I left it.

That's the whole pitch: **a place your agent can live, not just a session it borrows.**

---

## Why Mansio?

- **Self-hosted SSH replacement in the browser** — no client install, just a URL.
- **Survives everything** — close the browser, restart the server, switch networks; your shell, your `vim`, your long-running `npm run build` all stay alive thanks to `tmux`.
- **Single binary** — ~10 MB Go binary with the React frontend embedded. The only runtime dependency is `tmux`.
- **Works on your phone** — tap-friendly UI, dedicated mobile input bar, IME-safe (Korean / Chinese / Japanese).
- **Two deployment modes** — native install for full host access (SSH-equivalent), or Docker for an isolated playground.

---

## Quick Start

### TL;DR — get a terminal in your browser in 60 seconds

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Linux
sudo bash deploy/install.sh

# macOS
bash deploy/install.sh
```

Open **http://localhost:8080**, set a password on the first screen, and you're done.

> **Want to try it without touching your host?** Use Docker instead:
> ```bash
> docker compose up -d --build
> ```

---

## Usage Guide

This section walks through everything you'll do day-to-day. Skip to the part you need.

### 1. First Launch — Set Your Password

When you open `http://localhost:8080` for the first time, Mansio shows a setup screen asking you to create a password. This is stored as a bcrypt hash on the server and unlocks future logins via an HttpOnly session cookie (7-day expiry).

> **Native install:** the password protects SSH-equivalent access to your host. Use a strong one.
> **Docker mode:** the container is isolated, but the password still protects whatever you mount into it.

After login you see the main UI:

```
┌──────────────┬──────────────────────────────────────────┐
│  Workspaces  │  ┌──┬──┬──┬──┐                         │
│  ─────────   │  │T1│T2│T3│ +│   tab bar                │
│  ▸ default   │  └──┴──┴──┴──┘                         │
│    work      │                                          │
│    server    │   $ ls                                   │
│              │   README.md  go.mod  internal/           │
│              │   $ █                                    │
│  [☀ ☾ ⚙]    │                                          │
└──────────────┴──────────────────────────────────────────┘
   Sidebar               Terminal panel
```

### 2. Workspaces & Tabs

Workspaces are top-level groups (left sidebar). Each workspace holds one or more tabs (top of the terminal panel). Every tab is its own persistent `tmux` session.

| Action | How |
|---|---|
| **Create workspace** | Click **+** at the top of the sidebar |
| **Rename / delete workspace** | Right-click (long-press on mobile) the workspace name → context menu |
| **Switch workspace** | Click another workspace in the sidebar — instant, no re-fit, scrollback preserved |
| **Create tab** | Click **+** at the right end of the tab bar |
| **Rename / delete tab** | Right-click (long-press on mobile) the tab → context menu |
| **Switch tab** | Click another tab in the tab bar |

The sidebar shows each workspace's **last-active terminal's CWD** as a subtitle, polled every 5 s while the page is visible. Useful for picking the right project at a glance.

> **Instant switching:** every open terminal stays mounted in the background (VS Code-style detach/attach). Switching workspaces is immediate, never re-fits a hidden terminal to 0×0, and never drops scrollback.

### 3. Persistent Sessions — How It Actually Behaves

Every tab is backed by a tmux session named `lt_<id>`. That means:

- **Close the browser tab** → processes keep running in tmux.
- **Reopen the browser** → reattach with full scrollback restored.
- **Server restart** *(native install)* → tmux survives the Go process; reattach as if nothing happened.
- **Container restart** *(Docker)* → tmux dies with the container; tabs reload empty (the metadata persists, but the running processes don't).
- **Delete a tab** → `tmux kill-session` runs and the processes are terminated.

A long-running build, a `vim` session, an `htop`, an interactive REPL — leave them, walk away, come back.

### 4. Keyboard & Mouse

| Input | Effect |
|---|---|
| **Shift + Enter** | Send a literal newline without submitting (helpful for multi-line input in REPLs and AI CLIs that submit on plain Enter) |
| **Mouse wheel** | Scroll terminal scrollback (tmux mouse mode is on by default) |
| **Click + drag** | Select text natively in the terminal |
| **Right-click on tab/workspace** | Context menu (rename / delete) |
| **Drag handle between sidebar and terminal** | Resize the sidebar (140–400 px) |

> **Copy / paste:** most browsers honor `Cmd+C` / `Ctrl+C` for selected text; `Cmd+V` / `Ctrl+V` pastes at the prompt. If the shell is doing something with `Ctrl+C` (sending SIGINT), select first, then use the system menu to copy.

### 5. Drag-and-Drop File Upload

Drop a file (or several) onto the terminal pane:

1. The file is POSTed to `/api/v1/sessions/:id/upload` as `multipart/form-data`.
2. It's saved under `~/uploads/` (or the equivalent inside the Docker container) with collision-safe naming.
3. The resulting **absolute path** is automatically pasted at your prompt, ready for the next command.

```
$ █
   [drag-drop image.png]
$ /home/mansio/uploads/image.png█
   [now type whatever:]
$ python process.py /home/mansio/uploads/image.png
```

Default cap: **100 MiB per upload**. Path traversal and NUL bytes are rejected server-side.

### 6. Theme — Light, Dark, System

Bottom of the sidebar has three icons:

- **☀ Light** — pin to light mode
- **☾ Dark** — pin to dark mode
- **⚙ System** — follow OS preference (default)

Both UI and xterm.js palettes are tuned for ≥4.5:1 WCAG contrast and verified by `theme.test.ts`.

### 7. Mobile

On narrow screens (<640 px wide):

- The sidebar collapses behind a hamburger button at the top of the tab bar.
- Tap the hamburger → sidebar slides in from the left with a backdrop overlay.
- A **dedicated mobile input bar** appears below the terminal. This bypasses xterm.js's hidden textarea so IME (Korean / Chinese / Japanese composition) and keyboard suggestions work correctly.
- All tap targets are at least 44 × 44 px.
- iOS focus-zoom is suppressed (the input uses 16 px font + scale tricks) so the page doesn't zoom in on every focus.

### 8. Logout

Click the user / power icon in the sidebar (or hit `/api/v1/auth/logout` directly). The server-side session is invalidated immediately; the cookie is cleared.

---

## Deployment

Mansio has two deployment modes. Pick one:

| | **Native Install** | **Docker** |
|---|---|---|
| Access level | Full host (SSH-equivalent) | Isolated container |
| tmux survives server restart | ✅ Yes | ❌ No (dies with container) |
| Best for | Personal dev box, home server | Sandbox, demos, untrusted use |
| Disk footprint | ~10 MB binary + tmux | ~1 GB image (Ubuntu + Node + Python) |

### Option 1: Native Install

The web terminal will have the same access as logging into the machine directly — same files, same tools, same environment.

**Prerequisites:** Go 1.26+, Node.js 20+, npm, tmux, git

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Linux
sudo bash deploy/install.sh

# macOS (no sudo on the script itself; it sudo's internally for /usr/local/bin)
bash deploy/install.sh
```

The installer detects the OS, builds from source, installs the binary to `/usr/local/bin/mansio`, and registers a service.

**Installer flags:**

| Flag | Description | Default |
|---|---|---|
| `--host HOST` | Server host | `127.0.0.1` |
| `--port PORT` | Server port | `8080` |
| `--data-dir DIR` | SQLite database directory | Linux: `/var/lib/mansio`, macOS: `~/.local/share/mansio` |
| `--user USER` | System user to run as | current user |
| `--help` | Show help | — |

#### Linux (systemd)

```bash
# Status / restart / logs
systemctl status mansio@$(whoami)
systemctl restart mansio@$(whoami)
journalctl -u mansio@$(whoami) -f

# Custom host/port/data dir
sudo bash deploy/install.sh --host 127.0.0.1 --port 3000 --data-dir /var/lib/mansio

# Uninstall (keeps data dir)
sudo bash deploy/uninstall.sh

# Wipe data too
sudo rm -rf /var/lib/mansio
```

Data dir: `/var/lib/mansio` · Service unit: `/etc/systemd/system/mansio@.service`

#### macOS (launchd)

```bash
launchctl list | grep mansio                       # status
launchctl stop  com.mansio.mansio           # stop
launchctl start com.mansio.mansio           # start
tail -f ~/Library/Logs/mansio/stdout.log           # logs

# Uninstall (keeps data dir + logs)
bash deploy/uninstall.sh
```

Data dir: `~/.local/share/mansio` · Logs: `~/Library/Logs/mansio/` · plist: `~/Library/LaunchAgents/com.mansio.mansio.plist`

> **macOS Full Disk Access:** macOS sandboxes access to `~/Documents`, `~/Desktop`, etc. On first launch Mansio hits `/api/v1/health`, and if those directories are unreadable, the web UI shows a full-screen modal with step-by-step instructions: System Settings → Privacy & Security → Full Disk Access → add `/usr/local/bin/mansio`. The installer also opens System Settings to the right pane automatically.

#### Cloudflare Tunnel

Works out of the box. Keep Mansio bound to loopback and point your tunnel at `http://localhost:8080` — Cloudflare handles HTTPS and WebSocket proxying automatically.

```bash
cloudflared tunnel --url http://localhost:8080
```

For a permanent tunnel, follow Cloudflare's named-tunnel docs and route a hostname to `http://localhost:8080`.

### Option 2: Docker

Runs in an isolated **Ubuntu 24.04** container preloaded with **Node.js 20**, **Python 3**, **build-essential**, **zsh**, **git**, **tmux**, and CJK fonts. The home directory persists via a Docker volume.

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio
docker compose up -d --build
# Open http://localhost:8080
```

The compose file publishes `127.0.0.1:8080` by default. Set `MANSIO_PORT=3000` to change the host port.

**Persists across container restarts:**
- `/home/mansio` → installed tools, project files, shell configs (volume `mansio-home`)
- `/data` → workspace/session metadata (volume `mansio-data`)

**Does NOT persist:**
- tmux sessions (running processes) — killed when the container restarts
- System packages installed via `apt` — bake them into the `Dockerfile` to make them permanent

**Common operations:**

```bash
docker compose logs -f               # follow logs
docker compose restart               # restart (loses tmux)
docker compose down                  # stop + remove (keeps volumes)
docker compose down -v               # stop + remove + WIPE all data
docker compose exec mansio bash    # shell into the container
```

### CLI Options (the binary itself)

| Flag | Description | Default |
|---|---|---|
| `--host` | Server host | `127.0.0.1` |
| `--port` | Server port | `8080` |
| `--data-dir` | SQLite database directory | `./data` |

Run directly:

```bash
./mansio --host 127.0.0.1 --port 9000 --data-dir /tmp/mansio-data
```

> **Upgrading from earlier builds:** the server now binds to `127.0.0.1` by default instead of all interfaces. Pass `--host 0.0.0.0` if you need external access (and front it with HTTPS). Env vars previously prefixed `LOCITERM_*` are now `MANSIO_*` (e.g. `MANSIO_PORT`, `MANSIO_HOST`, `MANSIO_DATA_DIR`).

---

## Troubleshooting

| Symptom | Cause / Fix |
|---|---|
| **Web UI shows "Permission Required" modal (macOS)** | Add `/usr/local/bin/mansio` to System Settings → Privacy & Security → Full Disk Access. Click "I've fixed it — Check again". |
| **`systemctl status mansio@<user>` shows failure** | `journalctl -u mansio@<user> -e` for the actual error. Common: port 8080 already taken — reinstall with `--port`. |
| **Tabs are empty after a Docker restart** | Expected — tmux dies with the container. Use native install if you need tmux to survive restarts. |
| **Can't paste / clipboard blocked** | Browser permission. Some browsers require HTTPS for the Clipboard API; front the server with Cloudflare Tunnel. |
| **iOS keyboard zooms in on focus** | Already mitigated (16 px font + scale). If you still see it, hard-refresh the page; old build cached. |
| **CJK characters render as boxes** | Native install: install a CJK font on your OS. Docker: already includes `fonts-noto-cjk`. |
| **"WebSocket connection failed"** | Check that your reverse proxy forwards `Upgrade` / `Connection` headers. Cloudflare Tunnel does this by default. |
| **Forgot password** | Native: stop the service, delete the password row from `<data-dir>/mansio.db` (or just delete the DB and lose all metadata), restart. Docker: `docker compose down -v && docker compose up -d --build`. |

---

## Architecture

```
Browser                            Go server (single binary)
┌─────────────────────┐           ┌──────────────────────────────────┐
│ React + xterm.js    │           │ net/http ServeMux                │
│                     │           │                                  │
│ Sidebar ──REST──────────────────> /api/v1/workspaces               │
│ TabBar  ──REST──────────────────> /api/v1/sessions                 │
│ Drop    ──multipart─────────────> /api/v1/sessions/:id/upload      │
│ xterm.js ═══WS══════════════════> /api/v1/ws/terminal/:id          │
│  binary frames (I/O) │          │   ├── tmux.Manager               │
│  JSON (control)      │          │   │   └── tmux sessions (persist)│
│                      │          │   └── store (SQLite)             │
└──────────────────────┘          └──────────────────────────────────┘
```

### Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React 19, TypeScript, xterm.js, Zustand, Vite |
| Backend | Go (stdlib `net/http`), gorilla/websocket, creack/pty |
| Persistence | tmux (sessions), SQLite via `modernc.org/sqlite` (metadata) |
| Auth | bcrypt + HttpOnly session cookie (7-day expiry) |
| Deploy | systemd (Linux) · launchd (macOS) · Docker multi-stage build (Ubuntu 24.04) |

### How tmux Persistence Works

```
1. Tab created    → tmux new-session -d -s lt_{id} -c $HOME
2. Browser opens  → creack/pty spawns "tmux attach -t lt_{id}"
                    PTY fd is bridged to WebSocket (binary frames)
3. Browser closes → PTY (attach process) terminates
                    tmux session keeps running in the background
4. Reconnect      → new "tmux attach" → scrollback + processes restored
5. Tab deleted    → tmux kill-session -t lt_{id}
```

The tmux server runs independently from the Go process. Even if the Go server crashes or restarts, tmux sessions survive (native install only — Docker containers lose tmux sessions on restart).

### WebSocket Protocol

Two frame types over a single connection:

| Direction | Type | Content |
|---|---|---|
| Client → Server | Binary | Terminal stdin (keystrokes) |
| Server → Client | Binary | Terminal stdout (output) |
| Client → Server | Text (JSON) | `{ type: "resize", cols, rows }` |
| Server → Client | Text (JSON) | `{ type: "attached" }`, `{ type: "pong" }` |

Binary frames carry raw terminal I/O with zero encoding overhead.

### REST API

```
GET    /api/v1/health                # Liveness + macOS permission status

POST   /api/v1/auth/setup            # First-run password setup
POST   /api/v1/auth/login            # Login
POST   /api/v1/auth/logout           # Logout
GET    /api/v1/auth/check            # Check auth state

GET    /api/v1/workspaces            # List workspaces
POST   /api/v1/workspaces            # Create workspace
PATCH  /api/v1/workspaces/:id        # Rename workspace
DELETE /api/v1/workspaces/:id        # Delete workspace (cascades sessions + tmux)

GET    /api/v1/workspaces/:wid/sessions   # List sessions
POST   /api/v1/workspaces/:wid/sessions   # Create session
PATCH  /api/v1/sessions/:id               # Rename session
DELETE /api/v1/sessions/:id               # Delete session (kills tmux)

POST   /api/v1/sessions/:id/upload        # multipart/form-data file upload
GET    /api/v1/ws/terminal/:sessionId     # WebSocket terminal
```

---

## Project Structure

```
mansio/
├── cmd/mansio/main.go              # Entrypoint, embed.FS, graceful shutdown
├── internal/
│   ├── server/                       # HTTP routing, auth middleware, /health
│   ├── api/                          # REST handlers (workspace, session, auth, upload)
│   ├── ws/                           # WebSocket upgrade + PTY bridge
│   ├── tmux/                         # tmux session lifecycle management
│   ├── store/                        # SQLite persistence + migrations
│   └── model/                        # Data structs
├── frontend/src/
│   ├── components/
│   │   ├── Auth/LoginForm.tsx        # Login / setup form
│   │   ├── Sidebar/Sidebar.tsx       # Workspace list + theme toggle + context menu
│   │   └── Terminal/                 # TabBar, TerminalPanel, TerminalView, MobileInputBar
│   ├── hooks/
│   │   ├── useTerminal.ts            # xterm.js + WebSocket lifecycle
│   │   ├── useEffectiveTheme.ts      # system/light/dark resolver
│   │   ├── useMediaQuery.ts          # Mobile breakpoint detector
│   │   └── shiftEnter.ts             # Shift+Enter → literal newline
│   ├── stores/
│   │   ├── appStore.ts               # Zustand: workspaces/sessions/active
│   │   └── themeStore.ts             # Persisted theme mode
│   ├── api/upload.ts                 # Multipart upload client
│   └── lib/
│       ├── theme.ts                  # Light + dark UI palettes & xterm themes
│       └── contrast.ts               # WCAG contrast helper (used by tests)
├── deploy/
│   ├── install.sh                    # Cross-platform installer (Linux+macOS)
│   ├── uninstall.sh                  # Cross-platform uninstaller
│   └── mansio.service              # systemd unit template (Linux)
├── Dockerfile                        # Multi-stage build (Ubuntu 24.04 runtime)
├── docker-compose.yml                # Docker deployment with persistent volumes
└── Makefile
```

---

## Development

```bash
# Tests
make test              # Run all tests (Go + frontend)
make test-go           # Go tests only
make test-frontend     # Frontend tests only

# Dev mode (two terminals)
make dev-backend       # Terminal 1: Go server on :8080
make dev-frontend      # Terminal 2: Vite dev server with proxy

# Build a single self-contained binary
make build             # → ./mansio

# Clean build artifacts
make clean
```

The Vite dev server proxies API + WebSocket calls to `localhost:8080`, so you get hot-reload on the frontend while the Go backend keeps running.

---

## Design Decisions

| Decision | Rationale |
|---|---|
| **Go stdlib `net/http`** | ~14 endpoints. Go 1.22+ ServeMux handles method+path routing natively — no router dependency. |
| **modernc.org/sqlite** | Pure Go, no CGo. Static binary, easy cross-compilation. |
| **tmux for persistence** | Sessions survive browser close AND server restart. Independent process. |
| **Binary WebSocket frames** | Zero encoding overhead. Critical for high-throughput terminal output. |
| **HttpOnly session cookie (not JWT)** | Simpler and revocable for single-user self-hosting. |
| **Per-effective-theme xterm palette** | Light/dark themes verified against ≥4.5:1 contrast in `theme.test.ts`. |
| **Ubuntu 24.04 (Docker)** | glibc-based for tool compatibility (Node.js, AI CLIs, etc.). |
| **Dedicated mobile input bar** | xterm.js's hidden textarea breaks IME composition on mobile keyboards; a real `<textarea>` is the cleanest fix. |

---

## Security Notes

- **Native install grants the same access level as SSH** — use a strong password.
- **Always front the server with HTTPS in production** (Cloudflare Tunnel recommended for the easiest path).
- **Restrict port access** via firewall or VPN whenever possible.
- **Docker mode provides isolation** — host files outside the mounted volumes are not accessible.
- **Uploads are sanitized** (no path traversal, no NUL bytes) and capped at 100 MiB per upload.
- **Sessions expire after 7 days**; logout invalidates immediately.
- **Password is bcrypt-hashed** at cost 10; the plaintext is never stored or logged.

---

## Roadmap

- [ ] Code Review panel (git diff viewer)
- [ ] Multi-user support (per-user workspace isolation)
- [ ] Tab drag-to-reorder
- [ ] Terminal scrollback search (Ctrl+Shift+F)
- [ ] Terminal split panes (horizontal / vertical within a tab)
- [ ] Custom theme presets
- [ ] Built-in HTTPS/TLS support (Let's Encrypt or self-signed)
- [ ] OAuth login (GitHub, Google)
- [ ] 2FA (TOTP)

See [TODO.md](TODO.md) for the full backlog.

---

## License

**GPL-3.0-or-later** — see [LICENSE](LICENSE) for the full text.

Mansio is free software: you can redistribute it and/or modify it
under the terms of the GNU General Public License as published by the
Free Software Foundation, either version 3 of the License, or (at your
option) any later version.

This is a copyleft license: any fork, redistribution, or modified
version you publish must also be licensed under GPL-3.0-or-later and
ship its source code. For third-party components bundled with
Mansio, see [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md).
