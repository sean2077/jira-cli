# v1.2.0

Agent-first CLI for Jira Server 8.1. This release fixes correctness bugs found
in a full code review, adds two requested features, and hardens the release
pipeline. No breaking changes.

## Features

- **Bearer auth** for Jira Server / Data Center 8.14+ personal access tokens:
  `--auth bearer` (also `JIRA_AUTH_SCHEME=bearer`, or `auth = "bearer"` in a
  profile). HTTP Basic remains the default and is correct for Jira 8.1.
- **Inline issue detail**: `jira issue <KEY> --links --comments --worklogs`
  includes those sections in compact and JSON output.

## Fixes

- `jira create --attach` no longer silently drops attachments in `--raw` mode;
  and when an attachment upload fails after the issue is created, the created
  issue key is now reported (exit code 6) instead of empty output — so
  automation does not retry into a duplicate issue.
- `jira assignable --issue <KEY>` now works (the flag was advertised in the
  error message but never registered).
- `--limit` now means "max items across pages" for the list commands
  (`comments`, `worklogs`, `dashboards`, `boards`, `sprints`, `backlog`,
  board/sprint/epic issues, `users search`, `assignable`), matching the
  documented contract; `users search` and `assignable` emit a next-page hint
  instead of silently truncating.
- The HTTP client refuses redirects off the Jira host (SSRF hardening) and
  surfaces the real network error (exit code 4) on a failed attachment upload.
- `--type cloud` fails fast with a clear message; `jira probe` reports
  `ok: false` when a probed capability is unavailable.

## Install

```bash
curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh -s -- v1.2.0
jira version
```

Prebuilt binaries for linux, macOS, and windows (amd64 and arm64) are attached;
verify them against `checksums.txt` (sha256).
