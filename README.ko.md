<p align="center">
  <img src="frontend/public/icon-512.png?v=2" alt="Mansio" width="128" />
</p>

<h1 align="center">Mansio</h1>

<p align="center">
  <a href="README.md">English</a> · <a href="README.zh-CN.md">中文</a> · <strong>한국어</strong>
</p>

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg?logo=react)](https://react.dev/)
[![tmux](https://img.shields.io/badge/tmux-persistent-1BB91F.svg)](https://github.com/tmux/tmux)
[![License: GPL v3+](https://img.shields.io/badge/License-GPLv3+-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)

> **AI 코딩 에이전트를 위한 영구 워크스페이스.**
> Claude Code(또는 어떤 장시간 셸 프로세스든)를 서버에서 돌리다가, 브라우저를 닫고, 기기를 옮겨도 — 몇 시간 뒤 돌아오면 에이전트도, 빌드도, `vim`도 떠난 자리 그대로. 안쪽은 tmux, 바깥쪽은 제대로 된 UI.

[빠른 시작](#빠른-시작) • [사용법](#사용법) • [배포](#배포) • [아키텍처](#아키텍처) • [트러블슈팅](#트러블슈팅)

---

## 스크린샷

<p align="center">
  <img src="docs/screenshots/desktop.png?v=2" alt="Mansio 데스크톱 UI — 사이드바, 탭 바, 영구 터미널" width="820" />
  <br />
  <em>데스크톱 — 워크스페이스, 탭 바, tmux 기반 영구 터미널 패널.</em>
</p>

<p align="center">
  <img src="docs/screenshots/mobile.png?v=2" alt="Mansio 모바일 UI — 접히는 사이드바와 전용 입력 바" width="320" />
  <br />
  <em>모바일 — 접히는 사이드바, IME 안전 입력 바, 화면 내 보조 키.</em>
</p>

---

## 만든 이유

원격 서버에서 **Claude Code**(와 다른 AI 코딩 에이전트들)를 돌리고 있었습니다. 작업이 다 길어요 — 멀티스텝 리팩터, 긴 빌드, 한 시간씩 도는 에이전트 루프. 그런데 브라우저 탭을 닫거나, 자리에서 일어나거나, 폰으로 옮기는 순간 흐름이 끊겼습니다. SSH 세션이 죽고, 스크롤백이 사라지고, 에이전트가 작업 중이던 컨텍스트가 통째로 뽑혀나갔어요.

그래서 `tmux`를 제대로 된 브라우저 UX로 감쌌습니다. **Mansio은 에이전트에게 영구적인 집을 줍니다** — 내가 책상에 앉아 있든, 폰을 들고 있든, 노트북을 재부팅하든 신경 쓰지 않는 워크스페이스. 탭을 닫고. 미팅 다녀오고. 다른 기기로 다시 들어와도 에이전트는 그대로 돌아가고 있고, 스크롤백도 남아 있고, 프롬프트는 정확히 떠난 자리에 있습니다.

요약하면: **에이전트가 잠깐 빌리는 세션이 아니라, 살 수 있는 공간.**

---

## 왜 Mansio인가?

- **브라우저에서 동작하는 셀프호스트 SSH 대체** — 클라이언트 설치 없이 URL만 있으면 됩니다.
- **모든 것에서 살아남음** — 브라우저를 닫아도, 서버를 재시작해도, 네트워크를 바꿔도 셸/`vim`/실행 중인 `npm run build`가 `tmux` 덕분에 그대로 유지됩니다.
- **단일 바이너리** — React 프론트엔드를 임베드한 ~10 MB Go 바이너리. 런타임 의존성은 `tmux`뿐입니다.
- **모바일에서도 동작** — 터치 친화적인 UI, 전용 모바일 입력 바, IME 안전 (한/중/일).
- **두 가지 배포 모드** — 풀 호스트 접근(SSH 동등)을 원하면 네이티브 설치, 격리된 샌드박스를 원하면 Docker.

---

## 빠른 시작

### 60초 안에 브라우저에서 터미널 띄우기

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Linux
sudo bash deploy/install.sh

# macOS
bash deploy/install.sh
```

**http://localhost:8080** 접속, 첫 화면에서 비밀번호 설정. 끝.

> **호스트를 건드리지 않고 시도해보고 싶다면** Docker로:
> ```bash
> docker compose up -d --build
> ```

---

## 사용법

여기서는 매일 사용하는 동작을 모두 다룹니다. 필요한 부분만 점프해서 보세요.

### 1. 첫 실행 — 비밀번호 설정

`http://localhost:8080`에 처음 접속하면 비밀번호를 만드는 셋업 화면이 뜹니다. 비밀번호는 서버에 bcrypt 해시로 저장되며, 이후 로그인은 HttpOnly 세션 쿠키(7일 만료)로 처리됩니다.

> **네이티브 설치:** 비밀번호가 호스트의 SSH 동급 접근을 보호합니다. 강한 비밀번호를 사용하세요.
> **Docker 모드:** 컨테이너는 격리되지만, 마운트한 볼륨에 대한 접근은 비밀번호로 보호해야 합니다.

로그인 후 메인 UI:

```
┌──────────────┬──────────────────────────────────────────┐
│  Workspaces  │  ┌──┬──┬──┬──┐                         │
│  ─────────   │  │T1│T2│T3│ +│   탭바                   │
│  ▸ default   │  └──┴──┴──┴──┘                         │
│    work      │                                          │
│    server    │   $ ls                                   │
│              │   README.md  go.mod  internal/           │
│              │   $ █                                    │
│  [☀ ☾ ⚙]    │                                          │
└──────────────┴──────────────────────────────────────────┘
   사이드바             터미널 패널
```

### 2. 워크스페이스와 탭

워크스페이스는 최상위 그룹(왼쪽 사이드바)이며, 각 워크스페이스는 1개 이상의 탭(터미널 패널 위쪽)을 가집니다. 모든 탭은 자체 영구 `tmux` 세션입니다.

| 동작 | 방법 |
|---|---|
| **워크스페이스 만들기** | 사이드바 상단의 **+** 클릭 |
| **워크스페이스 이름 변경 / 삭제** | 워크스페이스 이름 우클릭(모바일은 길게 누르기) → 컨텍스트 메뉴 |
| **워크스페이스 전환** | 사이드바의 다른 워크스페이스 클릭 — 즉시 전환, 리핏 없음, 스크롤백 유지 |
| **탭 만들기** | 탭바 우측 끝의 **+** 클릭 |
| **탭 이름 변경 / 삭제** | 탭 우클릭(모바일은 길게 누르기) → 컨텍스트 메뉴 |
| **탭 전환** | 탭바에서 다른 탭 클릭 |

사이드바에는 각 워크스페이스의 **마지막 활성 터미널 CWD**가 부제로 표시됩니다. 페이지가 보이는 동안 5초마다 폴링됩니다. 한눈에 적절한 프로젝트를 고르는 데 유용합니다.

> **즉시 전환:** 모든 열린 터미널은 백그라운드에 마운트된 상태로 유지됩니다(VS Code 방식 detach/attach). 워크스페이스 전환이 즉각적이며, 숨겨진 터미널을 0×0으로 fit 하지 않고, 스크롤백을 잃지 않습니다.

### 3. 영구 세션 — 실제 동작 방식

모든 탭은 `lt_<id>` 이름의 tmux 세션으로 백킹됩니다. 즉:

- **브라우저 탭 닫기** → tmux 안의 프로세스는 계속 실행.
- **브라우저 다시 열기** → 풀 스크롤백과 함께 재접속.
- **서버 재시작** *(네이티브 설치)* → tmux는 Go 프로세스와 별개로 살아 있음. 아무 일도 없었다는 듯 재접속.
- **컨테이너 재시작** *(Docker)* → tmux가 컨테이너와 함께 죽음. 메타데이터는 유지되지만 실행 중이던 프로세스는 사라집니다.
- **탭 삭제** → `tmux kill-session` 실행, 프로세스 종료.

장시간 빌드, `vim` 세션, `htop`, 인터랙티브 REPL — 그대로 두고 자리를 떠났다가 돌아와도 됩니다.

### 4. 키보드와 마우스

| 입력 | 효과 |
|---|---|
| **Shift + Enter** | 명령을 실행하지 않고 리터럴 개행만 입력 (REPL/AI CLI에서 멀티라인 입력에 유용) |
| **마우스 휠** | 터미널 스크롤백 스크롤 (tmux mouse mode 기본 활성화) |
| **클릭 + 드래그** | 터미널에서 텍스트 선택 |
| **탭/워크스페이스 우클릭** | 컨텍스트 메뉴 (이름 변경 / 삭제) |
| **사이드바와 터미널 사이 핸들 드래그** | 사이드바 폭 조절 (140–400 px) |

> **복사 / 붙여넣기:** 대부분의 브라우저는 선택된 텍스트에 대해 `Cmd+C` / `Ctrl+C`를 인식합니다. `Cmd+V` / `Ctrl+V`로 프롬프트에 붙여 넣으세요. 셸이 `Ctrl+C`를 SIGINT로 처리하고 있다면, 먼저 선택한 뒤 시스템 메뉴로 복사하세요.

### 5. 드래그 앤 드롭 파일 업로드

터미널 패널에 파일(또는 여러 개)을 드롭하세요:

1. 파일이 `multipart/form-data`로 `/api/v1/sessions/:id/upload`에 POST 됩니다.
2. `~/uploads/` 아래에 충돌 회피 네이밍으로 저장됩니다 (Docker 컨테이너 내부 동등 경로 포함).
3. 결과 **절대 경로**가 프롬프트에 자동으로 붙여 넣어져 다음 명령에 바로 사용할 수 있습니다.

```
$ █
   [image.png 드래그 앤 드롭]
$ /home/mansio/uploads/image.png█
   [이어서 입력:]
$ python process.py /home/mansio/uploads/image.png
```

기본 한도: **업로드당 100 MiB**. 경로 탈출과 NUL 바이트는 서버에서 거부합니다.

### 6. 테마 — 라이트, 다크, 시스템

사이드바 하단에 세 개의 아이콘:

- **☀ Light** — 라이트 모드 고정
- **☾ Dark** — 다크 모드 고정
- **⚙ System** — OS 환경설정 따라가기 (기본값)

UI와 xterm.js 팔레트 모두 ≥4.5:1 WCAG 대비를 만족하도록 튜닝되어 있고, `theme.test.ts`에서 검증됩니다.

### 7. 모바일

좁은 화면(<640 px)에서는:

- 사이드바가 탭바 상단의 햄버거 버튼 뒤로 접힙니다.
- 햄버거를 탭하면 백드롭 오버레이와 함께 사이드바가 왼쪽에서 슬라이드 인.
- 터미널 아래에 **전용 모바일 입력 바**가 표시됩니다. xterm.js의 hidden textarea를 우회해 IME 조합(한/중/일) 및 키보드 자동완성이 정상 동작합니다.
- 모든 탭 영역은 최소 44 × 44 px.
- iOS 포커스 줌인이 억제되어 있어(폰트 16 px + scale 트릭) 포커스마다 페이지가 확대되지 않습니다.

### 8. 로그아웃

사이드바의 사용자/전원 아이콘을 클릭하거나 `/api/v1/auth/logout`을 직접 호출하세요. 서버 측 세션이 즉시 무효화되고 쿠키가 클리어됩니다.

---

## 배포

Mansio은 두 가지 배포 모드가 있습니다. 하나를 고르세요:

| | **네이티브 설치** | **Docker** |
|---|---|---|
| 접근 수준 | 풀 호스트 (SSH 동등) | 격리된 컨테이너 |
| 서버 재시작 시 tmux 생존 | ✅ 예 | ❌ 아니오 (컨테이너와 함께 종료) |
| 적합한 용도 | 개인 개발 머신, 홈 서버 | 샌드박스, 데모, 신뢰 불가 사용 |
| 디스크 풋프린트 | ~10 MB 바이너리 + tmux | ~1 GB 이미지 (Ubuntu + Node + Python) |

### 방법 1: 네이티브 설치

웹 터미널이 직접 로그인한 것과 동일한 환경을 가집니다 — 같은 파일, 같은 도구, 같은 환경.

**사전 요구사항:** Go 1.26+, Node.js 20+, npm, tmux, git

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio

# Linux
sudo bash deploy/install.sh

# macOS (스크립트 자체에는 sudo 불필요. /usr/local/bin 설치 시 내부적으로 sudo 호출)
bash deploy/install.sh
```

설치 스크립트는 OS를 감지하고, 소스에서 빌드하고, 바이너리를 `/usr/local/bin/mansio`에 설치한 뒤 서비스를 등록합니다.

**설치 스크립트 플래그:**

| 플래그 | 설명 | 기본값 |
|---|---|---|
| `--host HOST` | 서버 호스트 | `127.0.0.1` |
| `--port PORT` | 서버 포트 | `8080` |
| `--data-dir DIR` | SQLite 데이터베이스 디렉토리 | Linux: `/var/lib/mansio`, macOS: `~/.local/share/mansio` |
| `--user USER` | 실행할 시스템 사용자 | 현재 사용자 |
| `--help` | 도움말 표시 | — |

#### Linux (systemd)

```bash
# 상태 / 재시작 / 로그
systemctl status mansio@$(whoami)
systemctl restart mansio@$(whoami)
journalctl -u mansio@$(whoami) -f

# 호스트/포트/데이터 디렉토리 변경
sudo bash deploy/install.sh --host 127.0.0.1 --port 3000 --data-dir /var/lib/mansio

# 제거 (데이터 디렉토리 유지)
sudo bash deploy/uninstall.sh

# 데이터까지 삭제
sudo rm -rf /var/lib/mansio
```

데이터 디렉토리: `/var/lib/mansio` · 서비스 유닛: `/etc/systemd/system/mansio@.service`

#### macOS (launchd)

```bash
launchctl list | grep mansio                       # 상태
launchctl stop  com.mansio.mansio           # 중지
launchctl start com.mansio.mansio           # 시작
tail -f ~/Library/Logs/mansio/stdout.log           # 로그

# 제거 (데이터 + 로그 유지)
bash deploy/uninstall.sh
```

데이터 디렉토리: `~/.local/share/mansio` · 로그: `~/Library/Logs/mansio/` · plist: `~/Library/LaunchAgents/com.mansio.mansio.plist`

> **macOS Full Disk Access:** macOS는 `~/Documents`, `~/Desktop` 등에 대한 접근을 샌드박싱합니다. Mansio은 첫 실행 시 `/api/v1/health`를 호출해 권한을 확인하고, 접근이 막혀 있으면 웹 UI에 전체 화면 모달로 단계별 안내를 표시합니다 (System Settings → Privacy & Security → Full Disk Access → `/usr/local/bin/mansio` 추가). 설치 스크립트도 해당 시스템 설정 화면을 자동으로 열어줍니다.

#### Cloudflare Tunnel

별도 설정 없이 동작합니다. Mansio은 loopback에 바인딩한 상태로 두고, 터널을 `http://localhost:8080`에 연결하면 Cloudflare가 HTTPS와 WebSocket 프록시를 자동 처리합니다.

```bash
cloudflared tunnel --url http://localhost:8080
```

영구 터널이 필요하면 Cloudflare의 Named Tunnel 문서를 따라 호스트네임을 `http://localhost:8080`으로 라우팅하세요.

### 방법 2: Docker

**Node.js 20**, **Python 3**, **build-essential**, **zsh**, **git**, **tmux**, CJK 폰트가 미리 설치된 **Ubuntu 24.04** 격리 컨테이너에서 실행됩니다. 홈 디렉토리는 Docker 볼륨으로 영속화됩니다.

```bash
git clone https://github.com/Younkyum/Mansio.git
cd Mansio
docker compose up -d --build
# http://localhost:8080 접속
```

compose 파일은 기본적으로 `127.0.0.1:8080`에만 게시합니다. 호스트 포트를 바꾸려면 `MANSIO_PORT=3000`을 지정하세요.

**컨테이너 재시작 시 유지되는 것:**
- `/home/mansio` → 설치한 도구, 프로젝트 파일, 셸 설정 (볼륨 `mansio-home`)
- `/data` → 워크스페이스/세션 메타데이터 (볼륨 `mansio-data`)

**유지되지 않는 것:**
- tmux 세션 (실행 중 프로세스) — 컨테이너 재시작 시 종료
- `apt`로 설치한 시스템 패키지 — 영구 반영하려면 `Dockerfile`에 추가

**자주 쓰는 명령:**

```bash
docker compose logs -f               # 로그 팔로우
docker compose restart               # 재시작 (tmux 손실)
docker compose down                  # 중지 + 제거 (볼륨 유지)
docker compose down -v               # 중지 + 제거 + 모든 데이터 삭제
docker compose exec mansio bash    # 컨테이너 내부 셸 접속
```

### CLI 옵션 (바이너리 자체)

| 플래그 | 설명 | 기본값 |
|---|---|---|
| `--host` | 서버 호스트 | `127.0.0.1` |
| `--port` | 서버 포트 | `8080` |
| `--data-dir` | SQLite 데이터베이스 디렉토리 | `./data` |

직접 실행:

```bash
./mansio --host 127.0.0.1 --port 9000 --data-dir /tmp/mansio-data
```

> **이전 버전에서 업그레이드:** 서버 기본 바인딩이 모든 인터페이스에서 `127.0.0.1`로 변경되었습니다. 외부 접근이 필요하면 `--host 0.0.0.0`을 명시하세요 (HTTPS로 프록시 권장). `LOCITERM_*` 환경변수는 `MANSIO_*`로 변경되었습니다 (예: `MANSIO_PORT`, `MANSIO_HOST`, `MANSIO_DATA_DIR`).

---

## 트러블슈팅

| 증상 | 원인 / 해결 |
|---|---|
| **웹 UI에 "Permission Required" 모달 (macOS)** | System Settings → Privacy & Security → Full Disk Access에 `/usr/local/bin/mansio` 추가. "I've fixed it — Check again" 클릭. |
| **`systemctl status mansio@<user>` 가 실패 표시** | `journalctl -u mansio@<user> -e`로 실제 에러 확인. 흔한 원인: 8080 포트 점유 — `--port`로 다시 설치. |
| **Docker 재시작 후 탭이 비어있음** | 정상 동작 — tmux는 컨테이너와 함께 죽습니다. tmux를 재시작에서도 유지하려면 네이티브 설치를 사용하세요. |
| **붙여넣기 / 클립보드가 막힘** | 브라우저 권한 문제. 일부 브라우저는 Clipboard API에 HTTPS를 요구합니다. Cloudflare Tunnel로 프론트하세요. |
| **iOS에서 포커스 시 줌인** | 이미 완화됨 (16 px 폰트 + scale). 그래도 보인다면 페이지 강제 새로고침. 이전 빌드가 캐시된 상태일 수 있음. |
| **CJK 문자가 □로 표시됨** | 네이티브 설치: OS에 CJK 폰트 설치. Docker는 이미 `fonts-noto-cjk` 포함. |
| **"WebSocket connection failed"** | 리버스 프록시가 `Upgrade` / `Connection` 헤더를 포워딩하는지 확인. Cloudflare Tunnel은 기본으로 처리합니다. |
| **비밀번호 분실** | 네이티브: 서비스 중지 후 `<data-dir>/mansio.db`의 비밀번호 행 삭제(또는 DB 통째로 삭제 — 메타데이터까지 손실), 재시작. Docker: `docker compose down -v && docker compose up -d --build`. |

---

## 아키텍처

```
브라우저                            Go 서버 (단일 바이너리)
┌─────────────────────┐           ┌──────────────────────────────────┐
│ React + xterm.js    │           │ net/http ServeMux                │
│                     │           │                                  │
│ 사이드바 ──REST──────────────────> /api/v1/workspaces               │
│ 탭바    ──REST──────────────────> /api/v1/sessions                  │
│ Drop    ──multipart─────────────> /api/v1/sessions/:id/upload      │
│ xterm.js ═══WS═════════════════> /api/v1/ws/terminal/:id           │
│  binary 프레임 (I/O)  │         │   ├── tmux.Manager               │
│  JSON (제어)          │         │   │   └── tmux 세션 (영구)        │
│                      │          │   └── store (SQLite)             │
└──────────────────────┘          └──────────────────────────────────┘
```

### 기술 스택

| 계층 | 기술 |
|---|---|
| 프론트엔드 | React 19, TypeScript, xterm.js, Zustand, Vite |
| 백엔드 | Go (stdlib `net/http`), gorilla/websocket, creack/pty |
| 영속성 | tmux (세션), SQLite via `modernc.org/sqlite` (메타데이터) |
| 인증 | bcrypt + HttpOnly 세션 쿠키 (7일 만료) |
| 배포 | systemd (Linux) · launchd (macOS) · Docker 멀티스테이지 빌드 (Ubuntu 24.04) |

### tmux 영속성 동작 방식

```
1. 탭 생성     → tmux new-session -d -s lt_{id} -c $HOME
2. 브라우저 접속 → creack/pty가 "tmux attach -t lt_{id}" 실행
                  PTY fd를 WebSocket에 브릿지 (binary 프레임)
3. 브라우저 종료 → PTY (attach 프로세스)만 종료
                  tmux 세션은 백그라운드에서 계속 실행
4. 재접속      → 새로운 "tmux attach" → 스크롤백 + 프로세스 복원
5. 탭 삭제     → tmux kill-session -t lt_{id}
```

tmux 서버는 Go 프로세스와 독립적으로 동작합니다. Go 서버가 크래시하거나 재시작해도 tmux 세션은 유지됩니다 (네이티브 설치에만 해당 — Docker는 컨테이너 재시작 시 tmux 세션 소멸).

### WebSocket 프로토콜

하나의 연결에서 두 종류의 프레임을 사용합니다:

| 방향 | 타입 | 내용 |
|---|---|---|
| 클라이언트 → 서버 | Binary | 터미널 stdin (키 입력) |
| 서버 → 클라이언트 | Binary | 터미널 stdout (출력) |
| 클라이언트 → 서버 | Text (JSON) | `{ type: "resize", cols, rows }` |
| 서버 → 클라이언트 | Text (JSON) | `{ type: "attached" }`, `{ type: "pong" }` |

Binary 프레임은 인코딩 오버헤드 없이 터미널 I/O를 직접 전달합니다.

### REST API

```
GET    /api/v1/health                # 헬스체크 + macOS 권한 상태

POST   /api/v1/auth/setup            # 초기 비밀번호 설정
POST   /api/v1/auth/login            # 로그인
POST   /api/v1/auth/logout           # 로그아웃
GET    /api/v1/auth/check            # 인증 상태 확인

GET    /api/v1/workspaces            # 워크스페이스 목록
POST   /api/v1/workspaces            # 워크스페이스 생성
PATCH  /api/v1/workspaces/:id        # 워크스페이스 이름 변경
DELETE /api/v1/workspaces/:id        # 워크스페이스 삭제 (세션 + tmux 함께 삭제)

GET    /api/v1/workspaces/:wid/sessions   # 세션 목록
POST   /api/v1/workspaces/:wid/sessions   # 세션 생성
PATCH  /api/v1/sessions/:id               # 세션 이름 변경
DELETE /api/v1/sessions/:id               # 세션 삭제 (tmux 종료)

POST   /api/v1/sessions/:id/upload        # multipart/form-data 파일 업로드
GET    /api/v1/ws/terminal/:sessionId     # WebSocket 터미널
```

---

## 프로젝트 구조

```
mansio/
├── cmd/mansio/main.go              # 진입점, embed.FS, graceful shutdown
├── internal/
│   ├── server/                       # HTTP 라우팅, 인증 미들웨어, /health
│   ├── api/                          # REST 핸들러 (workspace, session, auth, upload)
│   ├── ws/                           # WebSocket 업그레이드 + PTY 브릿지
│   ├── tmux/                         # tmux 세션 라이프사이클 관리
│   ├── store/                        # SQLite 영속성 + 마이그레이션
│   └── model/                        # 데이터 구조체
├── frontend/src/
│   ├── components/
│   │   ├── Auth/LoginForm.tsx        # 로그인/설정 폼
│   │   ├── Sidebar/Sidebar.tsx       # 워크스페이스 목록 + 테마 토글 + 컨텍스트 메뉴
│   │   └── Terminal/                 # TabBar, TerminalPanel, TerminalView, MobileInputBar
│   ├── hooks/
│   │   ├── useTerminal.ts            # xterm.js + WebSocket 라이프사이클
│   │   ├── useEffectiveTheme.ts      # system/light/dark 해석기
│   │   ├── useMediaQuery.ts          # 모바일 브레이크포인트 감지
│   │   └── shiftEnter.ts             # Shift+Enter → 리터럴 개행
│   ├── stores/
│   │   ├── appStore.ts               # Zustand: 워크스페이스/세션/활성 상태
│   │   └── themeStore.ts             # 영속화된 테마 모드
│   ├── api/upload.ts                 # 멀티파트 업로드 클라이언트
│   └── lib/
│       ├── theme.ts                  # 라이트/다크 UI 팔레트와 xterm 테마
│       └── contrast.ts               # WCAG 대비 헬퍼 (테스트에서 사용)
├── deploy/
│   ├── install.sh                    # 크로스 플랫폼 설치 스크립트 (Linux+macOS)
│   ├── uninstall.sh                  # 크로스 플랫폼 제거 스크립트
│   └── mansio.service              # systemd 유닛 템플릿 (Linux)
├── Dockerfile                        # 멀티 스테이지 빌드 (Ubuntu 24.04 런타임)
├── docker-compose.yml                # Docker 배포 (영구 볼륨 포함)
└── Makefile
```

---

## 개발

```bash
# 테스트
make test              # 전체 테스트 (Go + 프론트엔드)
make test-go           # Go 테스트만
make test-frontend     # 프론트엔드 테스트만

# 개발 모드 (터미널 두 개)
make dev-backend       # 터미널 1: Go 서버 (:8080)
make dev-frontend      # 터미널 2: Vite 개발 서버 (프록시)

# 단일 자체 포함 바이너리 빌드
make build             # → ./mansio

# 빌드 산출물 정리
make clean
```

Vite 개발 서버는 API와 WebSocket 호출을 `localhost:8080`으로 프록시합니다. 따라서 Go 백엔드를 켜둔 채로 프론트엔드에서 핫 리로드를 즐길 수 있습니다.

---

## 설계 결정

| 결정 | 이유 |
|---|---|
| **Go stdlib `net/http`** | 약 14개 엔드포인트. Go 1.22+ ServeMux가 메서드+경로 라우팅을 기본 지원 — 라우터 의존성 없음. |
| **modernc.org/sqlite** | 순수 Go 구현, CGo 불필요. 정적 바이너리, 손쉬운 크로스 컴파일. |
| **tmux 기반 영속성** | 브라우저 종료 + 서버 재시작에도 세션 생존. 독립 프로세스. |
| **Binary WebSocket 프레임** | 인코딩 오버헤드 제로. 고출력 터미널에 필수. |
| **HttpOnly 세션 쿠키 (JWT 아님)** | 싱글유저 셀프호스팅에 더 간단하고 취소 가능. |
| **이펙티브 테마별 xterm 팔레트** | 라이트/다크 테마 모두 `theme.test.ts`에서 ≥4.5:1 대비 검증. |
| **Ubuntu 24.04 (Docker)** | glibc 기반으로 도구 호환성 확보 (Node.js, AI CLI 등). |
| **전용 모바일 입력 바** | xterm.js의 hidden textarea가 모바일 키보드 IME 조합을 깨뜨림 — 진짜 `<textarea>`가 가장 깔끔한 해법. |

---

## 보안 참고사항

- **네이티브 설치는 SSH와 동일한 접근 수준** — 강한 비밀번호 사용.
- **프로덕션에서는 반드시 HTTPS 사용** (Cloudflare Tunnel이 가장 쉬운 경로).
- **포트 접근 제한** — 가능하면 방화벽 또는 VPN 뒤에 둘 것.
- **Docker 모드는 격리 제공** — 마운트된 볼륨 외부 호스트 파일 접근 불가.
- **업로드는 sanitize 처리됨** (경로 탈출, NUL 바이트 차단)되며 업로드당 100 MiB 한도.
- **세션은 7일 후 만료**. 로그아웃 시 즉시 무효화.
- **비밀번호는 bcrypt 해시** (cost 10). 평문은 저장/로깅되지 않음.

---

## 로드맵

- [ ] 코드 리뷰 패널 (git diff 뷰어)
- [ ] 멀티유저 지원 (사용자별 워크스페이스 격리)
- [ ] 탭 드래그 정렬
- [ ] 터미널 스크롤백 검색 (Ctrl+Shift+F)
- [ ] 터미널 분할 패널 (탭 내 horizontal/vertical split)
- [ ] 커스텀 테마 프리셋
- [ ] HTTPS/TLS 내장 지원 (Let's Encrypt 또는 self-signed)
- [ ] OAuth 로그인 (GitHub, Google)
- [ ] 2FA (TOTP)

전체 백로그는 [TODO.md](TODO.md) 참조.

---

## 라이선스

**GPL-3.0-or-later** — 전체 라이선스 텍스트는 [LICENSE](LICENSE) 참조.

Mansio은 자유 소프트웨어입니다. Free Software Foundation이
공표한 GNU General Public License의 조건에 따라(버전 3 또는 — 사용자
선택에 따라 — 그 이후의 어떤 버전이든) 자유롭게 재배포 및 수정할 수
있습니다.

GPL은 copyleft 라이선스입니다. 즉, Mansio을 **포크/배포/수정 후
공개**하려면 그 결과물도 GPL-3.0-or-later로 라이선싱해야 하며 소스
코드를 함께 제공해야 합니다. Mansio에 번들된 서드파티 구성요소의
라이선스는 [THIRD_PARTY_LICENSES.md](THIRD_PARTY_LICENSES.md)
참조.
