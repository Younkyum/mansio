# Third-Party Licenses

Mansio is licensed under [GPL-3.0-or-later](LICENSE). It bundles or
depends on the following third-party components, each licensed under
the terms shown below. All listed licenses are compatible with GPL-3.0.

---

## Backend (Go) — statically linked into the `mansio` binary

| Module | Version | License | Source |
|---|---|---|---|
| github.com/creack/pty | v1.1.24 | MIT | https://github.com/creack/pty |
| github.com/google/uuid | v1.6.0 | BSD-3-Clause | https://github.com/google/uuid |
| github.com/gorilla/websocket | v1.5.3 | BSD-2-Clause | https://github.com/gorilla/websocket |
| golang.org/x/crypto | v0.50.0 | BSD-3-Clause | https://cs.opensource.google/go/x/crypto |
| golang.org/x/sys | v0.43.0 | BSD-3-Clause | https://cs.opensource.google/go/x/sys |
| modernc.org/sqlite | v1.50.0 | BSD-3-Clause | https://gitlab.com/cznic/sqlite |
| modernc.org/libc | v1.72.0 | BSD-3-Clause | https://gitlab.com/cznic/libc |
| modernc.org/mathutil | v1.7.1 | BSD-3-Clause | https://gitlab.com/cznic/mathutil |
| modernc.org/memory | v1.11.0 | BSD-3-Clause | https://gitlab.com/cznic/memory |
| github.com/dustin/go-humanize | v1.0.1 | MIT | https://github.com/dustin/go-humanize |
| github.com/mattn/go-isatty | v0.0.20 | MIT | https://github.com/mattn/go-isatty |
| github.com/ncruces/go-strftime | v1.0.0 | MIT | https://github.com/ncruces/go-strftime |
| github.com/remyoudompheng/bigfft | (pinned) | BSD-3-Clause | https://github.com/remyoudompheng/bigfft |

Each of these licenses preserves the original copyright notice and
permits redistribution. Full license texts ship inside the Go module
cache and are reproduced in releases per their requirements.

---

## Frontend (npm) — bundled into the embedded JavaScript

| Package | Version | License | Source |
|---|---|---|---|
| react | 19.x | MIT | https://github.com/facebook/react |
| react-dom | 19.x | MIT | https://github.com/facebook/react |
| scheduler | 0.27.x | MIT | https://github.com/facebook/react |
| @xterm/xterm | 6.x | MIT | https://github.com/xtermjs/xterm.js |
| @xterm/addon-fit | 0.11.x | MIT | https://github.com/xtermjs/xterm.js |
| @xterm/addon-web-links | 0.12.x | MIT | https://github.com/xtermjs/xterm.js |
| @xterm/addon-webgl | 0.19.x | MIT | https://github.com/xtermjs/xterm.js |
| zustand | 5.x | MIT | https://github.com/pmndrs/zustand |
| csstype | 3.x | MIT | https://github.com/frenic/csstype |
| @types/react | 19.x | MIT | https://github.com/DefinitelyTyped/DefinitelyTyped |

---

## External runtime dependencies — invoked, not bundled

These programs are executed by Mansio at runtime but are **not**
linked, embedded, or modified by it. Their licenses do not affect the
license of Mansio itself; they are listed here for completeness.

| Tool | License | Required by |
|---|---|---|
| tmux | ISC (BSD-style) | All modes — terminal sessions |
| Node.js | MIT | Docker mode (preinstalled in container) |
| Python 3 | PSF License (MIT-compatible) | Docker mode (preinstalled in container) |

---

## Reproducing this list

Backend:

```bash
go mod download -json | python3 -c "
import json,sys,os
d=sys.stdin.read(); dec=json.JSONDecoder(); i=0
while i<len(d):
    o,e=dec.raw_decode(d,i); i=e
    while i<len(d) and d[i] in ' \n\r\t': i+=1
    p=o.get('Path'); dr=o.get('Dir')
    if not dr: continue
    for fn in ('LICENSE','LICENSE.md','LICENSE.txt','COPYING','LICENCE'):
        fp=os.path.join(dr,fn)
        if os.path.exists(fp):
            print(p, '->', open(fp).read(80).replace('\n',' ')); break"
```

Frontend:

```bash
cd frontend && npx --yes license-checker --production --summary
```
