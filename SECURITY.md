# Security Policy

## Supported versions

Mansio is pre-1.0. Only the latest commit on `main` receives security fixes.

## Reporting a vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Instead, report privately via one of the following:

- **GitHub Security Advisories** — https://github.com/Younkyum/Mansio/security/advisories/new (preferred — encrypted, threaded)
- **Email** — `jinyounkyum@gmail.com` with the subject prefix `[mansio-security]`

Please include:

- A description of the vulnerability and its impact.
- Steps to reproduce, or a proof-of-concept.
- The affected version (commit SHA or release tag).
- Your name / handle for credit (optional).

## What to expect

- **Acknowledgement** within 72 hours.
- **Initial assessment** within 7 days.
- **Coordinated disclosure** — we'll work with you on a fix and a public advisory. Please keep the issue confidential until a fix is released.

## Scope

In scope:

- The Go server (`internal/`, `cmd/mansio/`).
- The React frontend (`frontend/`).
- The deployment scripts (`deploy/`) and `Dockerfile`.
- The bundled WebSocket protocol and REST API.

Out of scope:

- Vulnerabilities that require local access to the host already running Mansio (Mansio grants SSH-equivalent access by design when installed natively — this is documented).
- Issues in upstream dependencies — please report those upstream first; we'll bump versions when patches land.
- Social engineering, physical attacks, DoS via resource exhaustion against an unprotected server.

## Hardening recommendations for operators

- Always front the server with HTTPS in production (Cloudflare Tunnel is the easiest path).
- Restrict port 8080 to a private network or VPN whenever possible.
- Use a strong, unique password — it controls SSH-equivalent access.
- Keep `tmux`, your OS, and Mansio itself up to date.
