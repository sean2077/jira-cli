# Jira Server 8.1 REST Research

## Sources

Primary sources used for this plan:

- Jira Server platform REST API reference, Jira 8.1.0:
  <https://docs.atlassian.com/software/jira/docs/api/REST/8.1.0/>
- Jira Agile Server REST API reference, 8.x reference surface:
  <https://docs.atlassian.com/jira-software/REST/8.20.1/>
- Atlassian Jira REST API examples:
  <https://developer.atlassian.com/server/jira/platform/jira-rest-api-examples/>
- Atlassian attachment upload REST KB:
  <https://support.atlassian.com/jira/kb/how-to-add-an-attachment-to-a-jira-issue-using-rest-api/>
- Atlassian public issue JSWSERVER-20177 about missing Jira Software REST docs
  and WADL workaround:
  <https://jira.atlassian.com/browse/JSWSERVER-20177>

## Platform REST API

Jira Server 8.1.0 documents the platform REST API as useful for apps,
integrations, and scripted interactions. It uses JSON over standard HTTP
methods.

Important version decision:

- Use `/rest/api/2`.
- Do not default to `/rest/api/latest`.

Reason: the Jira 8.1.0 docs state that the current API version is `2` and that
`latest` is symbolic. For a private legacy target, stable old-instance behavior
is more valuable than automatic drift.

## Authentication

The Jira 8.1.0 platform docs list OAuth and HTTP Basic authentication. For this
tool:

- v1 supports Basic Auth because it is practical for private bot/script usage.
- OAuth is deferred.
- Password/token material can come from env, profile `token_env`, or plaintext
  profile `token`/`password`. Plaintext profile secrets are supported for local
  private automation but are unencrypted; prefer env or a future keychain where
  config files may be shared, backed up, or committed.

## Platform Endpoints in V1

High-confidence 8.1.0 platform endpoints:

| Capability | Endpoint | V1 command direction |
| --- | --- | --- |
| Search issues | `GET /rest/api/2/search` | `jira search` |
| Create issue | `POST /rest/api/2/issue` | `jira create` |
| Create metadata | `GET /rest/api/2/issue/createmeta` | used by `jira create --meta` |
| Get issue | `GET /rest/api/2/issue/{issueIdOrKey}` | `jira issue` |
| Edit issue | `PUT /rest/api/2/issue/{issueIdOrKey}` | `jira update` |
| Edit metadata | `GET /rest/api/2/issue/{issueIdOrKey}/editmeta` | used by update guidance |
| Delete issue | `DELETE /rest/api/2/issue/{issueIdOrKey}` | `jira delete --yes` |
| Assign issue | `PUT /rest/api/2/issue/{issueIdOrKey}/assignee` | `jira assign` |
| Add comment | `POST /rest/api/2/issue/{issueIdOrKey}/comment` | `jira comment` |
| Get transitions | `GET /rest/api/2/issue/{issueIdOrKey}/transitions` | `jira transitions` |
| Transition issue | `POST /rest/api/2/issue/{issueIdOrKey}/transitions` | `jira transition` |
| Watchers | `/rest/api/2/issue/{issueIdOrKey}/watchers` | `jira watchers/watch/unwatch` |
| Worklogs | `/rest/api/2/issue/{issueIdOrKey}/worklog` | `jira worklog` |
| Issue links | `/rest/api/2/issueLink` and issue link types | `jira link` |
| Projects | `/rest/api/2/project` | `jira projects/project` |
| Users | `/rest/api/2/user` and user search endpoints | `jira users search` |
| Fields | `/rest/api/2/field` | `jira fields` |
| Issue types | `/rest/api/2/issuetype` | `jira issuetypes` |
| Priorities | `/rest/api/2/priority` | `jira priorities` |
| Statuses | `/rest/api/2/status` | `jira statuses` |

## Search and Token Efficiency

Search supports a `fields` parameter. The 8.1.0 docs note that search defaults
to navigable fields, while get-issue defaults differ. Therefore:

- `jira search` must always request explicit fields.
- compact output should default to fields like key, summary, status, priority,
  assignee, project, created, updated.
- details/comments/worklogs should be opt-in, not default search payload.

## Create and Update Metadata

Create and edit behavior depends on Jira screens:

- Create fields should be derived from `createmeta`.
- Edit fields should be derived from `editmeta`.
- If a field is not on the relevant screen, Jira may reject it even if the field
  exists globally.

CLI consequence:

- `jira create` and `jira update` should provide concise error guidance that
  suggests metadata commands when Jira rejects fields.
- The CLI should not pretend all fields are always writable.

## Jira Server Identity Model

Jira Server 8.x uses user `name`/`key` shapes. The CLI should not default to
Cloud `accountId` semantics.

CLI consequence:

- `jira assign` v1 accepts Server username/key.
- Any future Cloud support must be a separate compatibility layer.

## 204 Empty Successes

Several Jira Server write operations return `204 No Content` on success.

CLI consequence:

- Treat 204 as success.
- Compact output must print meaningful confirmation, for example
  `OK update issue=JCLI-123`.
- Never print `null`, `undefined`, or an empty raw body as if it were useful
  data.

## Pagination

Jira docs state that each API may have different item limits, and consumers may
receive fewer results than requested.

CLI consequence:

- Pagination must advance by the actual number of items returned, not by the
  requested `maxResults`.
- Compact output should expose enough pagination state for agents:
  `next="--start-at 50"`.
- JSON output should include `startAt`, `maxResults`, `total` when present, and
  `nextStartAt` when known.

## Agile REST API

Jira Software/Agile REST uses `/rest/agile/1.0`.

Important caveat:

- Public exact-version Agile docs for Jira Software 8.1 may be incomplete or
  hard to access.
- Atlassian public issue JSWSERVER-20177 notes missing Jira Software REST API
  docs for several 7.x/8.x versions and gives a WADL workaround:
  `<JIRA_URL>/rest/agile/1.0/application.wadl`.

V1 policy:

- Implement Agile commands only after `jira probe` confirms availability.
- Prefer safe read endpoints first.
- Treat move/rank/sprint mutation endpoints as high-risk and dry-run-first.

Agile endpoints to research/implement after probe:

| Capability | Endpoint | V1 direction |
| --- | --- | --- |
| Boards | `GET /rest/agile/1.0/board` | `jira boards` |
| Board issues | `GET /rest/agile/1.0/board/{boardId}/issue` | `jira board issues` |
| Backlog issues | `GET /rest/agile/1.0/board/{boardId}/backlog` | `jira backlog` |
| Board sprints | `GET /rest/agile/1.0/board/{boardId}/sprint` | `jira sprints` |
| Sprint details | `GET /rest/agile/1.0/sprint/{sprintId}` | `jira sprint` |
| Sprint issues | `GET /rest/agile/1.0/sprint/{sprintId}/issue` | `jira sprint issues` |
| Epic details | `GET /rest/agile/1.0/epic/{epicIdOrKey}` | `jira epic` |
| Epic issues | `GET /rest/agile/1.0/epic/{epicIdOrKey}/issue` | `jira epic issues` |
| Move to backlog | `POST /rest/agile/1.0/backlog/issue` | dry-run-first |
| Move to sprint | `POST /rest/agile/1.0/sprint/{sprintId}/issue` | dry-run-first |

## Dashboard Endpoints

Jira Server 8.1 exposes stable platform REST endpoints for read-only dashboard
discovery and dashboard item properties:

| Capability | Endpoint | CLI direction |
| --- | --- | --- |
| Dashboards | `GET /rest/api/2/dashboard` | `jira dashboards` |
| Dashboard details | `GET /rest/api/2/dashboard/{id}` | `jira dashboard` |
| Dashboard item property keys | `GET /rest/api/2/dashboard/{dashboardId}/items/{itemId}/properties` | `jira dashboard item properties` |
| Dashboard item property | `GET /rest/api/2/dashboard/{dashboardId}/items/{itemId}/properties/{propertyKey}` | `jira dashboard item property` |
| Set dashboard item property | `PUT /rest/api/2/dashboard/{dashboardId}/items/{itemId}/properties/{propertyKey}` | dry-run/yes-gated |
| Delete dashboard item property | `DELETE /rest/api/2/dashboard/{dashboardId}/items/{itemId}/properties/{propertyKey}` | dry-run/yes-gated |

The public Jira Server 8.1 REST contract does not provide stable dashboard
creation, sharing, layout, or gadget add/move/configuration endpoints. Do not
automate those through private UI or gadget endpoints in this CLI.

## Additional Jira 8.1 Functions Worth Considering

Implemented or covered after the Jira 8.1 coverage pass:

- issue comments read/list
- issue attachments list/get/upload/download/delete
- issue remote links
- issue properties
- worklogs read/get
- filters and favorites
- components and versions
- project roles and project statuses
- assignable users and permissions discovery
- resolutions
- guarded public JSON REST pass-through through `jira api`

Still deferred unless a concrete agent workflow needs them:

- group administration
- audit records if available in the target edition
- avatar and remaining upload endpoints, because they need careful file IO and
  special headers such as `X-Atlassian-Token`

## Public JSON REST Pass-Through

The `jira api get|post|put|delete <PATH>` command covers the long tail of
documented Jira Server 8.1 public JSON REST endpoints without adding first-class
commands for every admin or low-frequency resource.

Safety boundary:

- accepts only `/rest/api/2/...`, `/rest/agile/1.0/...`, `api/2/...`,
  `agile/1.0/...`, or platform-resource shorthand such as `filter/favourite`
- rejects absolute URLs, scheme-relative URLs, `latest`, auth session paths,
  traversal, encoded slash/traversal, embedded query strings, and fragments
- write methods require `--dry-run` or `--yes`
- live Agile pass-through writes require `--force` in addition to `--yes` because
  typed Agile commands provide safer previews for common workflows
- JSON bodies are validated and sent as JSON values, preserving large numeric
  literals through the raw-message pattern used by entity properties
- multipart/binary/special-header flows remain out of generic pass-through;
  issue attachments are covered by typed `jira attachment` commands. Downloads
  use metadata `content` URLs after same-origin validation rather than assuming
  a synthetic REST content path.

## Do Not Prioritize in V1

- issue type scheme administration
- permission scheme administration
- workflow scheme administration
- global settings
- avatar upload
- upgrade endpoints
- plugin/admin APIs

Reason: these are low-frequency, high-risk, and not required for the initial
agent-efficiency goal.
