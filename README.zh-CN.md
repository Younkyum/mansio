<p align="center">
  <img src="frontend/public/icon-512.png?v=2" alt="Mansio" width="128" />
</p>

<h1 align="center">Mansio</h1>

<p align="center">
  <a href="README.md">English</a> · <strong>中文</strong> · <a href="README.ko.md">한국어</a>
</p>

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg?logo=react)](https://react.dev/)
[![tmux](https://img.shields.io/badge/tmux-persistent-1BB91F.svg)](https://github.com/tmux/tmux)
[![License: GPL v3+](https://img.shields.io/badge/License-GPLv3+-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

> **为 AI 编码 Agent 打造的持久工作区。**
> 把 Claude Code（或任何长时间运行的 shell 进程）跑在服务器上，关掉浏览器、换设备 — 几小时后回来，agent、构建、`vim` 全都停在你离开的那一刻。里子是 tmux，面子是真正的 UI。

[快速开始](#快速开始) • [使用指南](#使用指南) • [部署](#部署) • [架构](#架构) • [故障排查](#故障排查)

---

## 截图

<p align="center">
  <img src="docs/screenshots/desktop.png?v=2" alt="Mansio 桌面 UI — 侧边栏、标签栏与持久化终端" width="820" />
  <br />
  <em>桌面端 — 工作区、标签栏与基于 tmux 的持久化终端面板。</em>
</p>

<p align="center">
  <img src="docs/screenshots/mobile.png?v=2" alt="Mansio 移动 UI — 可折叠侧边栏与专用输入栏" width="320" />
  <br />
  <em>移动端 — 可折叠侧边栏、IME 安全输入栏、屏幕内辅助键。</em>
</p>

---

## 为什么做这个

我把 **Claude Code**（以及其他 AI 编码 agent）跑在远端开发机上。任务通常都很长 —— 多步重构、慢编译、跑一小时的 agent 循环。但只要我关掉浏览器标签、离开座位，或切到手机，工作流就断了：SSH 会话挂掉、滚动缓冲丢光、agent 正在做的上下文被整个抽走。

于是我用真正的浏览器 UX 把 `tmux` 包了起来。**Mansio 给 agent 一个持久的家** —— 一个不在乎你坐在桌前、握着手机、还是在重启笔电的工作区。关掉标签。去开个会。换台设备打开，agent 还在跑，滚动缓冲还在，提示符正好停在你离开的位置。

一句话：**让 agent 能"住下来"的空间，而不是临时借用的会话。**

---

## 为什么选择 Mansio？

- **浏览器中的自托管 SSH 替代品** — 无需安装客户端，只需一个 URL。
- **生存能力极强** — 关闭浏览器、重启服务器、切换网络都没问题，得益于 `tmux`，你的 shell、`vim`、运行中的 `npm run build` 全都不会丢。
- **单文件二进制** — 约 10 MB 的 Go 二进制文件，内嵌 React 前端。运行时唯一依赖是 `tmux`。
- **手机也能用** — 触摸友好的 UI、专用移动输入栏、IME 安全（中/韩/日）。
- **两种部署模式** — 原生安装提供完整主机访问（与 SSH 等价），或使用 Docker 提供隔离沙盒。

---

## 快速开始

### 60 秒内在浏览器中打开终端

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Linux
sudo bash deploy/install.sh

# macOS
bash deploy/install.sh
```

打开 **http://localhost:8080**，在首屏设置密码，完成。

> **想在不动主机的情况下试用？** 用 Docker：
> ```bash
> docker compose up -d --build
> ```

---

## 使用指南

本节涵盖你日常会用到的全部操作。需要哪一部分就跳到哪一部分。

### 1. 首次启动 — 设置密码

首次访问 `http://localhost:8080` 时，会出现设置密码的初始化界面。密码以 bcrypt 哈希存储于服务器，后续登录通过 HttpOnly 会话 Cookie（7 天过期）保持。

> **原生安装：** 密码保护的是与 SSH 等价的主机访问权限，请使用强密码。
> **Docker 模式：** 容器是隔离的，但密码同样保护你挂载进去的目录。

登录后的主界面：

```
┌──────────────┬──────────────────────────────────────────┐
│  Workspaces  │  ┌──┬──┬──┬──┐                         │
│  ─────────   │  │T1│T2│T3│ +│   标签栏                  │
│  ▸ default   │  └──┴──┴──┴──┘                         │
│    work      │                                          │
│    server    │   $ ls                                   │
│              │   README.md  go.mod  internal/           │
│              │   $ █                                    │
│  [☀ ☾ ⚙]    │                                          │
└──────────────┴──────────────────────────────────────────┘
   侧边栏              终端面板
```

### 2. 工作区与标签页

工作区是顶级分组（左侧侧边栏），每个工作区包含一个或多个标签页（终端面板顶部）。每个标签页都对应一个独立、持久的 `tmux` 会话。

| 操作 | 方法 |
|---|---|
| **创建工作区** | 点击侧边栏顶部的 **+** |
| **重命名 / 删除工作区** | 右键单击（移动端长按）工作区名 → 上下文菜单 |
| **切换工作区** | 点击侧边栏中的另一个工作区 — 即时切换、不会重新 fit、滚动历史保留 |
| **创建标签页** | 点击标签栏右端的 **+** |
| **重命名 / 删除标签页** | 右键单击（移动端长按）标签页 → 上下文菜单 |
| **切换标签页** | 点击标签栏中的另一个标签 |

侧边栏显示每个工作区**最后活动终端的 CWD** 作为副标题，页面可见时每 5 秒轮询一次。便于一眼挑出正确的项目。

> **即时切换：** 所有打开的终端都在后台保持挂载（VS Code 风格的 detach/attach）。切换工作区即时完成、不会把隐藏终端 fit 到 0×0、滚动历史保留完整。

### 3. 持久会话 — 实际行为

每个标签页背后是一个名为 `lt_<id>` 的 tmux 会话。这意味着：

- **关闭浏览器标签页** → tmux 内的进程继续运行。
- **重新打开浏览器** → 重新挂载，滚动历史完整恢复。
- **重启服务器**（原生安装） → tmux 独立于 Go 进程，仿佛什么都没发生过。
- **重启容器**（Docker） → tmux 随容器死亡；标签页重载为空（元数据保留，但运行中的进程消失）。
- **删除标签页** → 执行 `tmux kill-session`，进程终止。

长时间构建、`vim` 会话、`htop`、交互式 REPL —— 放着不管，离开，回来就行。

### 4. 键盘与鼠标

| 输入 | 效果 |
|---|---|
| **Shift + Enter** | 输入字面换行而不提交命令（在 REPL / AI CLI 多行输入中很有用） |
| **鼠标滚轮** | 滚动终端历史（默认启用 tmux mouse mode） |
| **点击 + 拖动** | 在终端中原生选中文本 |
| **右键标签页 / 工作区** | 上下文菜单（重命名 / 删除） |
| **拖动侧边栏与终端之间的把手** | 调整侧边栏宽度（140–400 px） |

> **复制 / 粘贴：** 大多数浏览器对选中的文本支持 `Cmd+C` / `Ctrl+C`；用 `Cmd+V` / `Ctrl+V` 粘贴到提示符。如果 shell 把 `Ctrl+C` 解释成 SIGINT，请先选中再用系统菜单复制。

### 5. 拖放上传文件

把文件（或多个文件）拖到终端面板上：

1. 文件以 `multipart/form-data` POST 至 `/api/v1/sessions/:id/upload`。
2. 保存到 `~/uploads/`（或 Docker 容器内对应路径），自动避免文件名冲突。
3. 生成的**绝对路径**会自动粘贴到提示符，可直接用于下一条命令。

```
$ █
   [拖放 image.png]
$ /home/mansio/uploads/image.png█
   [继续输入：]
$ python process.py /home/mansio/uploads/image.png
```

默认上限：**每次上传 100 MiB**。服务器端拒绝路径穿越和 NUL 字节。

### 6. 主题 — 浅色、深色、系统

侧边栏底部有三个图标：

- **☀ Light** — 固定为浅色模式
- **☾ Dark** — 固定为深色模式
- **⚙ System** — 跟随系统偏好（默认）

UI 与 xterm.js 调色板均按 ≥4.5:1 WCAG 对比度调优，并由 `theme.test.ts` 验证。

### 7. 移动端

在窄屏（<640 px）上：

- 侧边栏折叠到标签栏顶部的汉堡按钮后面。
- 点击汉堡 → 侧边栏从左侧滑入，伴随背景遮罩。
- 终端下方出现**专用移动输入栏**。它绕过了 xterm.js 的隐藏 textarea，让 IME（中/韩/日组合输入）和键盘建议正常工作。
- 所有点击区域至少 44 × 44 px。
- 抑制了 iOS 聚焦缩放（输入使用 16 px 字号 + scale 技巧），不会每次聚焦都放大页面。

### 8. 登出

点击侧边栏的用户 / 电源图标，或直接调用 `/api/v1/auth/logout`。服务器端会话立即失效，Cookie 被清除。

---

## 部署

Mansio 有两种部署模式，二选一：

| | **原生安装** | **Docker** |
|---|---|---|
| 访问级别 | 完整主机（SSH 等价） | 隔离容器 |
| 服务器重启后 tmux 是否存活 | ✅ 是 | ❌ 否（随容器消亡） |
| 适合场景 | 个人开发机、家庭服务器 | 沙盒、演示、不可信使用 |
| 磁盘占用 | ~10 MB 二进制 + tmux | ~1 GB 镜像（Ubuntu + Node + Python） |

### 方式一：原生安装

Web 终端将拥有与直接登录主机相同的访问权限 — 相同的文件、工具与环境。

**前置条件：** Go 1.26+, Node.js 20+, npm, tmux, git

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Linux
sudo bash deploy/install.sh

# macOS（脚本本身无需 sudo，仅在写入 /usr/local/bin 时内部调用 sudo）
bash deploy/install.sh
```

安装脚本会自动检测操作系统、从源码构建、将二进制安装到 `/usr/local/bin/mansio`，并注册系统服务。

**安装脚本参数：**

| 参数 | 说明 | 默认值 |
|---|---|---|
| `--host HOST` | 服务器主机 | `127.0.0.1` |
| `--port PORT` | 服务器端口 | `8080` |
| `--data-dir DIR` | SQLite 数据库目录 | Linux: `/var/lib/mansio`, macOS: `~/.local/share/mansio` |
| `--user USER` | 运行用户 | 当前用户 |
| `--help` | 显示帮助 | — |

#### Linux (systemd)

```bash
# 状态 / 重启 / 日志
systemctl status mansio@$(whoami)
systemctl restart mansio@$(whoami)
journalctl -u mansio@$(whoami) -f

# 自定义主机/端口/数据目录
sudo bash deploy/install.sh --host 127.0.0.1 --port 3000 --data-dir /var/lib/mansio

# 卸载（保留数据目录）
sudo bash deploy/uninstall.sh

# 同时清除数据
sudo rm -rf /var/lib/mansio
```

数据目录：`/var/lib/mansio` · 服务单元：`/etc/systemd/system/mansio@.service`

#### macOS (launchd)

```bash
launchctl list | grep mansio                       # 状态
launchctl stop  com.mansio.mansio           # 停止
launchctl start com.mansio.mansio           # 启动
tail -f ~/Library/Logs/mansio/stdout.log           # 日志

# 卸载（保留数据 + 日志）
bash deploy/uninstall.sh
```

数据目录：`~/.local/share/mansio` · 日志：`~/Library/Logs/mansio/` · plist：`~/Library/LaunchAgents/com.mansio.mansio.plist`

> **macOS 完全磁盘访问权限：** macOS 会沙盒化对 `~/Documents`、`~/Desktop` 等目录的访问。Mansio 在首次启动时会调用 `/api/v1/health` 检查权限，若被阻止则在 Web UI 中以全屏模态显示分步指引（系统设置 → 隐私与安全性 → 完全磁盘访问 → 添加 `/usr/local/bin/mansio`）。安装脚本也会自动打开对应的系统设置面板。

#### Cloudflare Tunnel

开箱即用。保持 Mansio 绑定到 loopback，并将隧道指向 `http://localhost:8080`，Cloudflare 会自动处理 HTTPS 与 WebSocket 代理。

```bash
cloudflared tunnel --url http://localhost:8080
```

如需永久隧道，请按照 Cloudflare 的 Named Tunnel 文档将一个主机名路由到 `http://localhost:8080`。

### 方式二：Docker

在预装 **Node.js 20**、**Python 3**、**build-essential**、**zsh**、**git**、**tmux** 与 CJK 字体的 **Ubuntu 24.04** 隔离容器中运行。主目录通过 Docker 卷持久化。

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio
docker compose up -d --build
# 访问 http://localhost:8080
```

compose 文件默认只发布到 `127.0.0.1:8080`。如需修改主机端口，请设置 `MANSIO_PORT=3000`。

**容器重启后保留：**
- `/home/mansio` → 已安装的工具、项目文件、Shell 配置（卷 `mansio-home`）
- `/data` → 工作区/会话元数据（卷 `mansio-data`）

**不保留：**
- tmux 会话（运行中的进程） — 容器重启时终止
- 通过 `apt` 安装的系统包 — 需写入 `Dockerfile` 才能永久保留

**常用操作：**

```bash
docker compose logs -f               # 跟随日志
docker compose restart               # 重启（tmux 丢失）
docker compose down                  # 停止 + 移除（保留卷）
docker compose down -v               # 停止 + 移除 + 清除所有数据
docker compose exec mansio bash    # 进入容器 shell
```

### CLI 选项（二进制本身）

| 参数 | 说明 | 默认值 |
|---|---|---|
| `--host` | 服务器主机 | `127.0.0.1` |
| `--port` | 服务器端口 | `8080` |
| `--data-dir` | SQLite 数据库目录 | `./data` |

直接运行：

```bash
./mansio --host 127.0.0.1 --port 9000 --data-dir /tmp/mansio-data
```

> **从旧版本升级：** 服务器默认绑定从所有接口更改为 `127.0.0.1`。如需外部访问，请显式传入 `--host 0.0.0.0`（建议通过 HTTPS 反向代理）。`LOCITERM_*` 环境变量已重命名为 `MANSIO_*`（例如 `MANSIO_PORT`、`MANSIO_HOST`、`MANSIO_DATA_DIR`）。

---

## 故障排查

| 现象 | 原因 / 解决方法 |
|---|---|
| **Web UI 显示 "Permission Required" 模态（macOS）** | 在系统设置 → 隐私与安全性 → 完全磁盘访问中添加 `/usr/local/bin/mansio`，点击 "I've fixed it — Check again"。 |
| **`systemctl status mansio@<user>` 显示失败** | 用 `journalctl -u mansio@<user> -e` 查看具体错误。常见原因：8080 端口被占用 — 用 `--port` 重装。 |
| **Docker 重启后标签页为空** | 符合预期 — tmux 随容器消亡。如需 tmux 在重启后存活，请使用原生安装。 |
| **无法粘贴 / 剪贴板被拦截** | 浏览器权限问题。某些浏览器要求 HTTPS 才能使用 Clipboard API；用 Cloudflare Tunnel 反代。 |
| **iOS 聚焦时缩放** | 已缓解（16 px 字号 + scale）。如仍出现，请强制刷新页面，可能是旧版本被缓存。 |
| **CJK 字符显示为方框** | 原生安装：在系统中安装 CJK 字体。Docker 已包含 `fonts-noto-cjk`。 |
| **"WebSocket connection failed"** | 检查反向代理是否转发 `Upgrade` / `Connection` 头。Cloudflare Tunnel 默认会处理。 |
| **忘记密码** | 原生：停止服务，删除 `<data-dir>/mansio.db` 中的密码行（或直接删除 DB —— 同时丢失元数据），重启。Docker：`docker compose down -v && docker compose up -d --build`。 |

---

## 架构

```
浏览器                              Go 服务器（单文件二进制）
┌─────────────────────┐           ┌──────────────────────────────────┐
│ React + xterm.js    │           │ net/http ServeMux                │
│                     │           │                                  │
│ 侧边栏 ──REST──────────────────> /api/v1/workspaces               │
│ 标签栏 ──REST──────────────────> /api/v1/sessions                  │
│ Drop  ──multipart─────────────> /api/v1/sessions/:id/upload      │
│ xterm.js ═══WS═════════════════> /api/v1/ws/terminal/:id           │
│  二进制帧（I/O）      │          │   ├── tmux.Manager               │
│  JSON（控制）         │          │   │   └── tmux 会话（持久）       │
│                      │          │   └── store（SQLite）             │
└──────────────────────┘          └──────────────────────────────────┘
```

### 技术栈

| 层级 | 技术 |
|---|---|
| 前端 | React 19, TypeScript, xterm.js, Zustand, Vite |
| 后端 | Go（stdlib `net/http`）, gorilla/websocket, creack/pty |
| 持久化 | tmux（会话）, SQLite via `modernc.org/sqlite`（元数据） |
| 认证 | bcrypt + HttpOnly 会话 Cookie（7 天过期） |
| 部署 | systemd（Linux）· launchd（macOS）· Docker 多阶段构建（Ubuntu 24.04） |

### tmux 持久化原理

```
1. 创建标签页  → tmux new-session -d -s lt_{id} -c $HOME
2. 浏览器连接  → creack/pty 启动 "tmux attach -t lt_{id}"
                PTY fd 桥接到 WebSocket（二进制帧）
3. 浏览器关闭  → PTY（attach 进程）终止
                tmux 会话在后台继续运行
4. 重新连接    → 新的 "tmux attach" → 滚动历史 + 进程恢复
5. 删除标签页  → tmux kill-session -t lt_{id}
```

tmux 服务器独立于 Go 进程运行。即使 Go 服务器崩溃或重启，tmux 会话也不会丢失（仅限原生安装 — Docker 容器重启时 tmux 会话会丢失）。

### WebSocket 协议

同一连接上使用两种帧类型：

| 方向 | 类型 | 内容 |
|---|---|---|
| 客户端 → 服务器 | Binary | 终端 stdin（键盘输入） |
| 服务器 → 客户端 | Binary | 终端 stdout（输出） |
| 客户端 → 服务器 | Text（JSON） | `{ type: "resize", cols, rows }` |
| 服务器 → 客户端 | Text（JSON） | `{ type: "attached" }`, `{ type: "pong" }` |

二进制帧传输原始终端 I/O，零编码开销。

### REST API

```
GET    /api/v1/health                # 健康检查 + macOS 权限状态

POST   /api/v1/auth/setup            # 首次密码设置
POST   /api/v1/auth/login            # 登录
POST   /api/v1/auth/logout           # 登出
GET    /api/v1/auth/check            # 检查认证状态

GET    /api/v1/workspaces            # 列出工作区
POST   /api/v1/workspaces            # 创建工作区
PATCH  /api/v1/workspaces/:id        # 重命名工作区
DELETE /api/v1/workspaces/:id        # 删除工作区（级联删除会话 + tmux）

GET    /api/v1/workspaces/:wid/sessions   # 列出会话
POST   /api/v1/workspaces/:wid/sessions   # 创建会话
PATCH  /api/v1/sessions/:id               # 重命名会话
DELETE /api/v1/sessions/:id               # 删除会话（终止 tmux）

POST   /api/v1/sessions/:id/upload        # multipart/form-data 文件上传
GET    /api/v1/ws/terminal/:sessionId     # WebSocket 终端
```

---

## 项目结构

```
mansio/
├── cmd/mansio/main.go              # 入口点、embed.FS、优雅关闭
├── internal/
│   ├── server/                       # HTTP 路由、认证中间件、/health
│   ├── api/                          # REST 处理器（workspace, session, auth, upload）
│   ├── ws/                           # WebSocket 升级 + PTY 桥接
│   ├── tmux/                         # tmux 会话生命周期管理
│   ├── store/                        # SQLite 持久化 + 迁移
│   └── model/                        # 数据结构体
├── frontend/src/
│   ├── components/
│   │   ├── Auth/LoginForm.tsx        # 登录/设置表单
│   │   ├── Sidebar/Sidebar.tsx       # 工作区列表 + 主题切换 + 上下文菜单
│   │   └── Terminal/                 # TabBar, TerminalPanel, TerminalView, MobileInputBar
│   ├── hooks/
│   │   ├── useTerminal.ts            # xterm.js + WebSocket 生命周期
│   │   ├── useEffectiveTheme.ts      # system/light/dark 解析器
│   │   ├── useMediaQuery.ts          # 移动端断点检测
│   │   └── shiftEnter.ts             # Shift+Enter → 字面换行
│   ├── stores/
│   │   ├── appStore.ts               # Zustand：工作区/会话/活动状态
│   │   └── themeStore.ts             # 持久化的主题模式
│   ├── api/upload.ts                 # 多部分上传客户端
│   └── lib/
│       ├── theme.ts                  # 浅色/深色 UI 调色板与 xterm 主题
│       └── contrast.ts               # WCAG 对比度工具（测试中使用）
├── deploy/
│   ├── install.sh                    # 跨平台安装脚本（Linux+macOS）
│   ├── uninstall.sh                  # 跨平台卸载脚本
│   └── mansio.service              # systemd 单元模板（Linux）
├── Dockerfile                        # 多阶段构建（Ubuntu 24.04 运行时）
├── docker-compose.yml                # Docker 部署（含持久卷）
└── Makefile
```

---

## 开发

```bash
# 测试
make test              # 运行所有测试（Go + 前端）
make test-go           # 仅 Go 测试
make test-frontend     # 仅前端测试

# 开发模式（两个终端）
make dev-backend       # 终端 1：Go 服务器（:8080）
make dev-frontend      # 终端 2：Vite 开发服务器（代理）

# 构建单文件自包含二进制
make build             # → ./mansio

# 清理构建产物
make clean
```

Vite 开发服务器会把 API 与 WebSocket 请求代理到 `localhost:8080`，让你在 Go 后端持续运行的同时享受前端热重载。

---

## 设计决策

| 决策 | 理由 |
|---|---|
| **Go stdlib `net/http`** | 约 14 个端点。Go 1.22+ ServeMux 原生支持方法+路径路由 — 无需额外路由库。 |
| **modernc.org/sqlite** | 纯 Go 实现，无需 CGo。支持静态二进制和交叉编译。 |
| **tmux 持久化** | 会话在浏览器关闭和服务器重启后均存活。独立进程。 |
| **二进制 WebSocket 帧** | 零编码开销。高吞吐量终端输出必需。 |
| **HttpOnly 会话 Cookie（非 JWT）** | 单用户自托管场景更简单且可撤销。 |
| **按生效主题分别配色** | 浅色/深色主题在 `theme.test.ts` 中均经 ≥4.5:1 对比度验证。 |
| **Ubuntu 24.04（Docker）** | 基于 glibc，工具兼容性更好（Node.js、AI CLI 等）。 |
| **专用移动输入栏** | xterm.js 的隐藏 textarea 会破坏移动键盘的 IME 组合 — 真正的 `<textarea>` 是最干净的解决方案。 |

---

## 安全注意事项

- **原生安装与 SSH 具有相同的访问级别** — 请使用强密码。
- **生产环境务必使用 HTTPS**（推荐 Cloudflare Tunnel，最简单的路径）。
- **限制端口访问** — 尽可能放在防火墙或 VPN 之后。
- **Docker 模式提供隔离** — 无法访问挂载卷之外的主机文件。
- **上传经过 sanitize**（防路径穿越、NUL 字节）并限制为每次 100 MiB。
- **会话 7 天后过期**；登出立即失效。
- **密码以 bcrypt 哈希存储**（cost 10）。明文不会被存储或记录。

---

## 路线图

- [ ] 代码审查面板（git diff 查看器）
- [ ] 多用户支持（按用户隔离工作区）
- [ ] 标签页拖拽排序
- [ ] 终端滚动历史搜索（Ctrl+Shift+F）
- [ ] 终端分屏（标签页内 horizontal/vertical split）
- [ ] 自定义主题预设
- [ ] 内置 HTTPS/TLS 支持（Let's Encrypt 或自签名）
- [ ] OAuth 登录（GitHub、Google）
- [ ] 2FA（TOTP）

完整待办见 [TODO.md](TODO.md)。

---

## 许可证

**GPL-3.0-or-later** — 完整许可证文本见 [LICENSE](LICENSE)。

Mansio 是自由软件。你可以根据自由软件基金会发布的 GNU 通用公共
许可证（第 3 版，或由你选择的任何更新版本）的条款重新分发和/或
修改它。

GPL 是 copyleft 许可证：任何对 Mansio 的派生、再分发、修改后再
发布的版本，都必须以 GPL-3.0-or-later 重新许可，并附带源代码。
Mansio 捆绑的第三方组件的许可证详见
[THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md)。
