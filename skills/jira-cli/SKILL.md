---
name: jira-cli
description: "Use when Codex needs to interact with a private Jira instance through the `jira` CLI: searching issues, inspecting issue details, creating/updating/commenting/transitioning issues, discovering dashboards, summarizing boards or sprints, probing Jira Server 8.1 capabilities, or diagnosing Jira CLI configuration. Prefer this skill over handwritten curl or MCP usage for Jira work."
---

# jira-cli

Use the `jira` CLI for private Jira work. Prefer compact output first, because
the CLI is designed for low-token agent workflows.

## First Steps

Run a probe when auth, API support, dashboard support, Agile support, fields, or
permissions are unknown:

```bash
jira probe
```

Use `--json` only when exact parsing is needed across follow-up steps:

```bash
jira probe --json
```

Use `jira --help`, `jira help <command>`, or `jira <command> --help` when a
command shape or flag is unclear. The CLI uses Cobra help and suggestions where
the command scope can be resolved; other usage errors point to the nearest
command help surface.

## Install

If the `jira-cli` skill is missing from the current project, install the bundled
skill with:

```bash
jira skill install
```

Use `jira skill install --global` for the user-level `~/.agents/skills` target.

## Common Commands

Search:

```bash
jira search 'project = JCLI ORDER BY updated DESC' --limit 10
```

Inspect:

```bash
jira issue JCLI-123
jira comments JCLI-123
jira worklogs JCLI-123
jira attachments JCLI-123
jira remote-links JCLI-123
```

Comment:

```bash
jira comment JCLI-123 --body 'Validated locally; ready for review.' --dry-run
jira create --project JCLI --issue-type Task --summary 'Fix login' --component UI --due 2026-06-30 --attach ./screenshot.png --dry-run
```

Transition:

```bash
jira transitions JCLI-123
jira transition JCLI-123 --id 31 --dry-run
```

Dashboards:

```bash
jira dashboards --filter my
jira dashboard 10000 --json
jira dashboard item property 10000 20000 issue.support --json
```

Jira 8.1 metadata:

```bash
jira filters
jira components T1
jira versions T1
jira roles T1
jira assignable --project T1 --query jdoe
jira mypermissions --project T1
```

Long-tail public JSON REST:

```bash
jira api get filter/favourite --query expand=sharedUsers --json
jira api post filter --body '{"name":"triage","jql":"project = T1"}' --dry-run
```

Attachments:

```bash
jira attachment add JCLI-123 --file ./screenshot.png --dry-run
jira attachment download 10001 --file ./screenshot.png
jira attachment delete 10001 --dry-run
```

Destructive commands require explicit confirmation:

```bash
jira delete JCLI-123 --yes
```

## Rules

- Prefer `jira` over handwritten `curl`.
- Use compact output by default.
- Use `--json` for scripts or exact field parsing.
- Use `--raw` only for debugging or missing-field discovery.
- Dashboard `--json` includes stable fields plus `rawDashboard` or
  `rawDashboards` for public fields the CLI has not modeled yet.
- Use `jira api` only for documented public JSON REST paths under
  `/rest/api/2` or `/rest/agile/1.0`; it rejects `latest`, auth-session,
  absolute URLs, traversal, embedded query, fragment paths, and special
  multipart/binary endpoints. Use typed attachment commands for issue
  attachments.
- Run `jira probe` before assuming dashboard, Agile, custom-field, workflow, or
  permission behavior.
- Use dashboard item property set/delete only with `--dry-run` or `--yes`; the
  CLI does not support private dashboard layout or gadget-management endpoints.
- Use issue property set/delete and `jira api` writes only with `--dry-run` or
  `--yes`; live `jira api` writes under `/rest/agile/1.0` also require
  `--force`.
- Use normal Jira writes such as create, update, comment, assign, transition,
  watch, worklog add, link create/delete, and Agile move only with `--dry-run`
  or `--yes`.
- For create/update, prefer direct common field flags (`--component`,
  `--version`, `--due`, `--priority`) before falling back to raw `--field`.
- Use `jira create ... --attach PATH` when a new issue should include an
  evidence file immediately.
- Use attachment upload/delete only with `--dry-run` or `--yes`; download
  refuses to overwrite local files unless `--force` is present.
- Never echo secrets.
- Do not run destructive Jira commands unless the user clearly requested the
  action and the command includes the required confirmation flag.

## References

For command contracts and implementation guidance, read the repository docs:

- `docs/cli-contract.md`
- `docs/jira-8.1-rest-research.md`
- `docs/skill-contract.md`
