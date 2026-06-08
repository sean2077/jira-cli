# jira-cli

Agent-first command line access to private Jira Server instances.

`jira-cli` is a small Go binary named `jira`. It is built for agents and humans
who need predictable Jira automation from a shell: compact output by default,
stable `--json` for parsing, and `--raw` when developing against Jira's REST
responses.

## Why

Many Jira automation tools assume Jira Cloud or newer Jira Server APIs. The
companion `jira-mcp-server` project was created for the same compatibility
pressure: some organizations are still on Jira Server 8.x, including 8.1,
because upgrades are blocked by legacy dependencies, compliance policy, cost,
or plugin compatibility.

This project gives those environments a direct CLI path instead of requiring a
long-running MCP server or browser automation. It targets public Jira REST
surfaces first, especially `/rest/api/2` on Jira Server 8.1, and keeps risky
operations explicit with `--dry-run` and `--yes`.

For endpoints that do not need a first-class command yet, `jira api` provides a
guarded public JSON REST pass-through for `/rest/api/2` and `/rest/agile/1.0`.
It rejects absolute URLs, `latest`, auth-session paths, traversal, embedded
query strings, and fragments; write methods require `--dry-run` or `--yes`, and
live Agile pass-through writes also require `--force`.

## Install

Install the latest release on Linux or macOS:

```bash
curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh
jira version
```

Install to a user-local directory:

```bash
mkdir -p "$HOME/.local/bin"
curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh -s -- --dir "$HOME/.local/bin"
```

Pin a release only when needed:

```bash
curl -fsSL https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.sh | sh -s -- v1.0.0
```

Windows PowerShell:

```powershell
$installer = Join-Path $env:TEMP "jira-install.ps1"
Invoke-WebRequest "https://github.com/sean2077/jira-cli/raw/refs/heads/main/scripts/install.ps1" -OutFile $installer
powershell -ExecutionPolicy Bypass -File $installer -AddToPath
jira version
```

Release assets are direct binaries plus `checksums.txt`; installer scripts live
in the repository.

## Configure

The fastest path is environment variables:

```bash
export JIRA_BASE_URL="https://jira.example.com"
export JIRA_USER="agent"
export JIRA_API_TOKEN="..."

jira probe
```

`JIRA_USER_EMAIL` can replace `JIRA_USER`, and `JIRA_PASSWORD` can replace
`JIRA_API_TOKEN` for Jira Server password-style auth. Run `jira config doctor`
when configuration is unclear.

## Use

The examples use `T1` as a placeholder test project key. Replace it with a
project key from your Jira instance.

```bash
jira search 'project = T1 ORDER BY updated DESC' --limit 10
jira issue T1-123
jira comments T1-123
jira attachments T1-123
jira filters
jira components T1
jira mine --days 3
jira dashboards --filter my
jira attachment add T1-123 --file ./screenshot.png --dry-run
jira create --project T1 --issue-type Task --summary 'Fix login' --component UI --due 2026-06-30 --attach ./screenshot.png --dry-run
jira api get filter/favourite --query expand=sharedUsers --json
jira create --project T1 --issue-type Bug --summary 'Fix login' --dry-run
jira --json issue T1-123
```

Default output is compact and agent-readable. Use `--json` for stable parsing
and `--raw` for Jira response discovery.

Writes should start with `--dry-run`. Actual Jira writes require explicit
confirmation with `--yes`, such as `jira comment T1-123 --body '...' --yes` or
`jira delete T1-123 --yes`. Attachment upload uses a typed multipart command,
`jira attachment add <KEY> --file PATH --yes`, rather than the generic JSON
`jira api` pass-through. For create/update, common Web-form fields have direct
flags: `--component`, `--version`, `--due`, and `--priority`; use `--field`
for less common Jira fields. `jira create` also accepts `--attach PATH` so a
new issue and its first evidence file can be created in one command.

## Agent Skill

Install the bundled skill after the binary is on `PATH`:

```bash
jira skill install
jira skill install --global
```

The skill teaches agents to prefer `jira` over ad hoc `curl`, run `jira probe`
before assuming server capabilities, and avoid destructive actions unless the
user clearly requested them.

## Documentation

- [CLI contract](docs/cli-contract.md): command shape, flags, output modes, and
  safety rules.
- [Skill contract](docs/skill-contract.md): agent workflow and skill install
  behavior.
- [Jira Server 8.1 REST notes](docs/jira-8.1-rest-research.md): supported REST
  boundary and dashboard constraints.
- [Test coverage](docs/test-coverage.md): fake-server coverage, live-test
  gates, and release checks.

## Develop

```bash
go test ./...
go vet ./...
scripts/build-release.sh v0.0.0-test
```

Live Jira checks are opt-in with `JIRA_LIVE_TEST=1` and the environment
variables from `.env.example`. Release binaries are uploaded by the GitHub
Release workflow after a release is published.
