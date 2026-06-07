# Codex Skill Contract

## Skill Identity

- Skill folder: `skills/jira-cli`
- Skill name in frontmatter: `jira-cli`
- Primary command taught by the skill: `jira`
- Release binary installer: `jira skill install`
- Default install target: `.agents/skills/jira-cli`
- Global install target: `$HOME/.agents/skills/jira-cli`
- Canonical Codex user skill target, when manually copying:
  `${CODEX_HOME:-$HOME/.codex}/skills/jira-cli`

## Trigger Intent

The skill should trigger when Codex needs to interact with private Jira:

- search Jira issues
- inspect an issue
- summarize project/sprint/board work
- add Jira comments
- transition issues
- create/update/assign Jira issues
- inspect fields, statuses, projects, users, dashboards, boards, or sprints
- inspect Jira dashboards and dashboard item properties
- diagnose Jira CLI configuration

## Skill Body Policy

Keep `SKILL.md` lean. It should include:

- when to use the skill
- first command to run
- output mode guidance
- safety rules
- compact examples
- references to this repository's docs for deeper command details

Do not include:

- full REST endpoint tables
- long Jira tutorials
- raw private examples
- secrets
- extensive troubleshooting copied from docs

## Required Agent Workflow

1. Prefer `jira` CLI over handwritten `curl`.
2. Run `jira probe` when capability, auth, dashboard/Agile availability, or
   custom field behavior is unknown.
3. Use compact output by default.
4. Use `--json` when exact fields need to be parsed across steps.
5. Use `--raw` only for debugging, schema discovery, or command development.
6. Avoid live writes unless the user requested or authorized them.
7. Require explicit `--yes` for destructive operations.
8. Never echo secrets.

## Installation

Preferred install after the `jira` binary is on `PATH`:

```bash
jira skill install
jira skill install --global
```

`jira skill install` defaults to the current project's `.agents/skills` root.
Use `--global` for `$HOME/.agents/skills`, `--target <DIR>` for a custom skill
root, and `--force` to replace an existing `jira-cli` skill.

The repository skill is also discoverable through the open `skills` CLI:

```bash
npx --yes skills add sean2077/jira-cli --skill jira-cli --copy --full-depth
npx --yes skills add sean2077/jira-cli --skill jira-cli --copy --full-depth --global
```

## Skill Examples

Search:

```bash
jira search 'project = JCLI AND status != Done ORDER BY updated DESC' --limit 10
```

Inspect:

```bash
jira issue JCLI-123
```

Structured parse:

```bash
jira search 'assignee = currentUser() ORDER BY updated DESC' --limit 20 --json
```

Comment:

```bash
jira comment JCLI-123 --body 'Validated locally; ready for review.' --dry-run
```

Transition:

```bash
jira transitions JCLI-123
jira transition JCLI-123 --id 31 --dry-run
```

Dashboard discovery:

```bash
jira dashboards --filter my
jira dashboard 10000 --json
jira dashboard item properties 10000 20000 --json
jira dashboard item property 10000 20000 issue.support --json
```

Dashboard `--json` responses expose stable dashboard fields plus
`rawDashboard`/`rawDashboards` so agents can inspect public fields the CLI has
not modeled yet.

Destructive:

```bash
jira delete JCLI-123 --yes
```

## Safety Text for SKILL.md

The skill should include this rule in plain language:

> Do not run destructive Jira commands unless the user has clearly requested the
> action and the command includes the required explicit confirmation flag.

When editing the actual skill, keep the final wording concise and avoid long
quoted blocks.

## Validation

Use the skill creator validation tool when available:

```bash
python "${CODEX_HOME:-$HOME/.codex}/skills/.system/skill-creator/scripts/quick_validate.py" skills/jira-cli
npx --yes skills add . --list --full-depth
```

If that path is unavailable, run the equivalent local validator from the current
Codex skill-creator installation.

Forward-test prompt:

```text
Use the jira-cli skill from this repository's skills/jira-cli directory to find
my open Jira issues and summarize the top blockers. Do not use curl unless the
skill tells you to.
```

Expected behavior:

- agent reads the skill
- agent calls `jira probe` or `jira search`
- agent prefers compact output first
- agent uses `--json` only if parsing is necessary
