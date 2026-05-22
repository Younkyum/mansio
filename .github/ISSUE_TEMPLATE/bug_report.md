---
name: Bug report
about: Something isn't working as documented
title: "[bug] "
labels: bug
assignees: ''
---

## Summary

<!-- One or two sentences describing the bug. -->

## Steps to reproduce

1.
2.
3.

## Expected behavior

<!-- What did you expect to happen? -->

## Actual behavior

<!-- What actually happened? Include any error messages, console output, or screenshots. -->

## Environment

- **Mansio version / commit:** <!-- e.g. v0.1.0 or commit SHA -->
- **Install mode:** <!-- native (Linux/macOS) or Docker -->
- **OS:** <!-- e.g. Ubuntu 24.04, macOS 14.5 -->
- **Browser:** <!-- e.g. Chrome 130, Safari 18, Firefox 131 -->
- **Mobile / desktop:** <!-- if mobile, what device + viewport -->
- **tmux version:** <!-- output of `tmux -V` -->

## Logs

<!--
  Native (Linux):  journalctl -u mansio@<user> -e
  Native (macOS):  tail -n 200 ~/Library/Logs/mansio/stderr.log
  Docker:          docker compose logs --tail=200 mansio
-->

```
<paste relevant log lines here>
```

## Additional context

<!-- Anything else we should know — recent config changes, network setup (e.g. behind Cloudflare Tunnel), reverse proxy, etc. -->
