# CLI Contract

## Name

- Repository: `jira-cli`
- Binary: `jira`
- Skill source: `skills/jira-cli`

## Command Shape

Use short noun/verb commands optimized for agent calls:

```text
jira probe
jira whoami
jira search '<JQL>'
jira issue JCLI-123
jira comments JCLI-123
jira worklogs JCLI-123
jira attachments JCLI-123
jira filters
jira components T1
jira dashboards --filter my
jira dashboard 10000
jira api get filter/favourite --query expand=sharedUsers
jira comment JCLI-123 --body '...' --yes
jira transitions JCLI-123
jira transition JCLI-123 --id 31 --dry-run
jira delete JCLI-123 --yes
```

The command layer is Cobra-backed. The public help surfaces are:

```text
jira --help
jira help <command>
jira <command> --help
```

Cobra owns command selection, flag parsing, argument validation, and near-match
unknown-command suggestions where the command scope can be resolved. Other
usage/validation failures keep the compact `ERR usage ...` prefix and point to
the nearest command help surface, but exact wording may follow Cobra/pflag
conventions instead of the earlier static help text.

Avoid deeply nested commands unless the Jira domain requires it:

```text
jira board issues 42
jira sprint issues 123
jira issue property JCLI-123 agent.state
jira worklog add JCLI-123 --time 1h --comment '...' --yes
jira attachment add JCLI-123 --file ./screenshot.png --yes
jira attachment download 10001 --file ./screenshot.png
jira dashboard item property 10000 20000 issue.support
```

## Global Flags

| Flag | Meaning |
| --- | --- |
| `--profile <name>` | Select named profile |
| `--base-url <url>` | Override Jira base URL |
| `--type server|cloud` | v1 accepts only `server` |
| `--user <name>` | Override username |
| `--token-env <name>` | Read token/password from env var |
| `--compact` | Compact output, default |
| `--json` | Stable JSON output |
| `--raw` | Unmodified Jira response |
| `--limit <n>` | Max items across pages |
| `--page-size <n>` | Requested page size |
| `--start-at <n>` | Start offset for paged endpoints |
| `--timeout <duration>` | HTTP timeout |
| `--dry-run` | Show intended write |
| `--yes` | Confirm write/destructive execution |
| `--force` | Replace an existing local install target where supported |

## Configuration Precedence

1. CLI flags
2. Environment variables
3. Selected profile
4. Default profile

Required config for v1:

- base URL
- Jira type: `server`
- username
- token/password via env

## Output Modes

### Compact

Default. Must be optimized for agent token efficiency.

Rules:

- One item per line for lists.
- Stable field order.
- No decorative formatting.
- Include next-page hint when applicable.
- Include `OK` prefix for successful writes.
- Include `ERR` prefix for compact errors.

### JSON

Use CLI-owned stable schemas:

```json
{
  "ok": true,
  "kind": "issue",
  "issue": {
    "key": "JCLI-123",
    "summary": "Fix login redirect",
    "status": "Open"
  }
}
```

Do not expose raw Jira JSON as the default `--json` shape. Raw Jira JSON belongs
behind `--raw`. Discovery commands whose Jira Server 8.1 schemas vary by
permissions or plugins may use a stable `{ok, kind, raw}` wrapper in `--json`;
the wrapper is stable, while the nested `raw` payload is explicitly raw-backed.

### Raw

Raw mode prints the Jira response body. It is for debugging, schema discovery,
and new command development. Commands that compose multiple Jira calls rather
than returning one Jira response, such as `jira probe` and `jira bulk search`,
must reject `--raw` and tell the caller to use `--json`.

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | success |
| 1 | usage/validation error |
| 2 | authentication/authorization error |
| 3 | Jira API application error |
| 4 | network/TLS/proxy/timeout error |
| 5 | missing capability |
| 6 | partial batch success |

## Required Command Contracts

### `jira probe`

Purpose: read-only capability discovery.

`--raw` is not supported because probe composes several endpoint checks into one
capability result.

Compact example:

```text
OK probe server=8.1.0 api=/rest/api/2 dashboard=available agile=available user=jdoe
```

JSON must include:

- server version when available
- base URL
- authenticated user
- platform API availability
- Agile API availability
- dashboard API availability
- project/field/search capability status
- warnings

### `jira search <JQL>`

Must:

- request explicit fields
- support `--limit`, `--page-size`, `--start-at`
- compact output one issue per line
- include total and next-page hint when known

Compact example:

```text
JCLI-123 Open P2 jdoe Fix login redirect
JCLI-124 In Progress P1 - Investigate token refresh failure
2 issues total=18 next="--start-at 2"
```

### `jira issue <KEY>`

Must:

- default to useful issue summary, not all fields
- include comments/worklogs/links only when flags request them
- expose `--json` for exact parsing

### `jira comments <KEY>` and `jira comment get <KEY> <ID>`

Must:

- use `GET /rest/api/2/issue/{issueIdOrKey}/comment`
- support pagination flags for comment lists
- expose compact, `--json`, and unmodified `--raw` output
- preserve existing `jira comment <KEY> --body TEXT` add-comment behavior

### `jira worklogs <KEY>` and `jira worklog get <KEY> <ID>`

Must:

- use `GET /rest/api/2/issue/{issueIdOrKey}/worklog`
- support pagination flags for worklog lists
- expose compact, `--json`, and unmodified `--raw` output
- preserve existing `jira worklog add <KEY>` behavior
- require `--dry-run` or `--yes` for `jira worklog add <KEY>`

### Jira Attachments

Must:

- use `GET /rest/api/2/issue/{issueIdOrKey}?fields=attachment` for `jira attachments <KEY>`
- use `GET /rest/api/2/attachment/{id}` for `jira attachment <ID>` and `jira attachment get <ID>`
- use `POST /rest/api/2/issue/{issueIdOrKey}/attachments` for `jira attachment add <KEY> --file PATH`
- send multipart form field `file` and `X-Atlassian-Token: no-check` for upload
- require `--dry-run` or `--yes` for upload
- use `GET /rest/api/2/attachment/{id}` metadata first, then download the
  same-origin `content` URL for `jira attachment download <ID> --file PATH`
- refuse to overwrite local download files unless `--force` is present
- use `DELETE /rest/api/2/attachment/{id}` for `jira attachment delete <ID>`
- require `--dry-run` or `--yes` for delete
- expose compact, `--json`, and unmodified `--raw` metadata/upload response output

### `jira issue properties/property`

Must:

- use `GET|PUT|DELETE /rest/api/2/issue/{issueIdOrKey}/properties`
- preserve arbitrary JSON values under stable `--json`
- require `--body <JSON>` for `set`
- require `--dry-run` or `--yes` for `set` and `delete`

### `jira filters` and `jira filter <ID>`

Must:

- use `GET /rest/api/2/filter/favourite` for favorite filter discovery
- use `GET /rest/api/2/filter/{id}` for filter detail
- expose compact, `--json`, and unmodified `--raw` output

### Jira 8.1 Metadata Discovery

Must expose read-only compact, `--json`, and `--raw` output for:

- `jira remote-links <KEY>` and `jira remote-link <KEY> <ID>`
- `jira components <PROJECT>`
- `jira versions <PROJECT>`
- `jira roles <PROJECT>` and `jira role <PROJECT> <ROLE_ID>`
- `jira project-statuses <PROJECT>`
- `jira assignable --project KEY [--query TEXT]`
- `jira permissions`
- `jira mypermissions [--project KEY|--issue KEY]`
- `jira resolutions` and `jira resolution <ID>`

`role`, `project-statuses`, `permissions`, and `mypermissions` may use the
stable `{ok, kind, raw}` JSON wrapper because Jira Server can vary these payloads
by permission model, plugins, and project configuration.

### `jira api get|post|put|delete <PATH>`

Purpose: guarded public JSON REST pass-through for documented Jira Server 8.1
endpoints that do not yet need first-class command modeling.

Must:

- accept only public Jira REST paths under `/rest/api/2` and
  `/rest/agile/1.0`, plus safe shorthand forms
- reject absolute URLs, scheme-relative URLs, `/rest/api/latest`,
  `/rest/auth/1/session`, traversal, encoded slash/traversal, embedded query,
  and fragment paths
- support repeated `--query k=v`
- support `--body JSON` for non-GET methods
- reject `api get ... --body`
- require `--dry-run` or `--yes` for non-GET methods
- require `--force` in addition to `--yes` for live `/rest/agile/1.0` writes;
  `--dry-run` remains available without configuration
- preserve unmodified Jira response bodies with `--raw`
- keep multipart upload, attachment content, avatar upload, and special-header
  flows out of generic pass-through; use typed attachment commands for issue
  attachments

### `jira create`

Must:

- accept project, type, summary, optional body/fields
- support first-class common create/update field flags:
  `--component ID|NAME`, `--version ID|NAME`, `--due YYYY-MM-DD`, and
  `--priority ID|NAME`
- support repeated `--attach PATH` on `jira create`; after issue creation
  succeeds, upload each file through the typed multipart attachment endpoint
- map numeric component/version/priority values to Jira `{id: ...}` objects and
  non-numeric values to `{name: ...}` objects
- validate `--due` locally as `YYYY-MM-DD`
- parse `--field name={...}` and `--field name=[...]` as JSON object/array
  values so required Jira fields such as components and versions can be set;
  keep other `--field name=value` values as strings
- require `--dry-run` or `--yes` for issue creation
- support read-only `--meta` lookup through `GET /rest/api/2/issue/createmeta`
  without requiring a summary or creating an issue
- make compact `--meta` output actionable: list additional required fields,
  allowed values where available, and copy-ready direct flag, `--field`, or
  `--body` examples
- when a requested issue type does not match the project metadata, list available
  project issue type names in compact output
- suggest `createmeta` when Jira rejects field payloads
- support dry-run display of the actual `fields` object that would be sent to
  Jira, including structured object/array `--field` values and planned
  attachments

### `jira update <KEY>`

Must:

- accept field updates
- support the same common field flags as `jira create`
- parse `--field name={...}` and `--field name=[...]` as JSON object/array
  values; keep other `--field name=value` values as strings
- require `--dry-run` or `--yes`
- suggest `editmeta` when Jira rejects field payloads
- support dry-run display

### `jira transition <KEY>`

Must:

- prefer `--id`
- allow `--name` only when it resolves to exactly one transition
- support optional comment if Jira accepts it
- require `--dry-run` or `--yes`

### Other first-class Jira writes

Must require `--dry-run` or `--yes` before reading Jira configuration or sending
the request for:

- `jira comment <KEY> --body TEXT`
- `jira assign <KEY> --assignee NAME`
- `jira watch <KEY>` and `jira unwatch <KEY>`
- `jira link create <SOURCE_KEY> --target KEY`
- `jira link delete <ID>`
- `jira move sprint --sprint ID --issue KEY`
- `jira move backlog --issue KEY`

### `jira delete <KEY>`

Must:

- require `--yes`
- refuse inferred or wildcard issue keys
- render compact success on 204

### `jira dashboards`

Must:

- use `GET /rest/api/2/dashboard`
- support `--filter favourite|my`
- support `--limit`, `--page-size`, and `--start-at`
- expose compact, `--json`, and unmodified `--raw` output
- include dashboard id, name, and view URL when available
- include `rawDashboards` in `--json` for unmodeled public response fields

### `jira dashboard <ID>`

Must:

- use `GET /rest/api/2/dashboard/{id}`
- expose compact, `--json`, and unmodified `--raw` output
- include `rawDashboard` in `--json` for unmodeled public response fields
- treat 404 as a normal Jira application/permission error for a specific id

### `jira dashboard item properties`

Must:

- use `GET /rest/api/2/dashboard/{dashboardId}/items/{itemId}/properties`
- expose property keys in compact output and stable `--json`
- support `--raw` for schema discovery

### `jira dashboard item property`

Must:

- use the Jira Server 8.1 dashboard item property endpoints only:
  `GET|PUT|DELETE /rest/api/2/dashboard/{dashboardId}/items/{itemId}/properties/{propertyKey}`
- preserve arbitrary JSON values under stable `--json`
- require `--body <JSON>` for `set`
- require `--dry-run` or `--yes` for `set` and `delete`
- avoid private dashboard creation, layout, gadget add/move/configuration, or UI-only endpoints

### `jira skill install`

Purpose: install the bundled `jira-cli` skill from the release binary.

Must:

- default to `.agents/skills/jira-cli` under the current working directory
- support `--global` for `~/.agents/skills/jira-cli`
- support `--target <DIR>` for custom skill roots, installing into
  `<DIR>/jira-cli`
- support `--dry-run`, `--force`, compact output, and `--json`
- reject `--raw`
- refuse to overwrite an existing target unless `--force` is present

## Help Text Requirements

Help is generated by Cobra from the command tree:

```text
agent-first Jira Server CLI

Usage:
  jira <command> [flags]
  jira help <command>

Command Groups:
  Core: probe, whoami, search, issue, comments, worklogs, attachments, links
  Writes: create, update, comment, assign, transition, watch, unwatch, delete

Flags:
  --profile string
  --json
  --raw
```

Root help should stay compact and scannable. Command-specific help must expose
the command usage and its local flags. Long examples and REST behavior details
belong under `docs/`.
