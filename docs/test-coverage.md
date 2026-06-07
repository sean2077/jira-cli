# Test Coverage

This project uses two coverage layers:

- fake Jira server tests for deterministic command behavior, methods, payloads,
  output shape, pagination, errors, and safety guards
- optional live Jira smoke tests for real connectivity and read-only API
  compatibility

Live tests are opt-in through `.env`/`.envrc`. By default they do not create
persistent Jira artifacts. Future live write tests must require
`JIRA_LIVE_PROJECT`, which should point to a dedicated Jira CLI test project
such as `JCLI`, create only `JIRA_CLI_TEST_*` artifacts in that project, and
print manual cleanup instructions if automated cleanup fails.

## Verification Commands

```bash
go test ./... -count=1
go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/jira-cli-allpkg.cover
go tool cover -func=/tmp/jira-cli-allpkg.cover | tail -n 1

direnv exec . go test ./internal/cli -run TestLive -count=1 -v
```

## Command Matrix

| Command | Fake Jira coverage | Live coverage | Notes |
| --- | --- | --- | --- |
| `version` | yes | not needed | no Jira access |
| `config doctor` | yes | not needed | verifies secret redaction |
| `probe` | yes | yes | read-only |
| `whoami` | yes | yes | read-only |
| `search` | yes | yes | read-only |
| `bulk search` | yes | no | read-only; fake covers partial failure |
| `issue` | yes | yes | read-only |
| `comments` | yes | no | read-only |
| `comment get` | yes | no | read-only |
| `links` | yes | yes | read-only |
| `remote-links` | yes | no | read-only |
| `remote-link` | yes | no | read-only |
| `projects` | yes | yes | read-only |
| `project` | yes | no | read-only |
| `components` | yes | no | read-only |
| `versions` | yes | no | read-only |
| `roles` | yes | no | read-only |
| `role` | yes | no | read-only |
| `project-statuses` | yes | no | read-only |
| `fields` | yes | yes | read-only |
| `users search` | yes | yes | read-only |
| `assignable` | yes | no | read-only |
| `issuetypes` | yes | no | read-only |
| `priorities` | yes | no | read-only |
| `statuses` | yes | no | read-only |
| `resolutions` | yes | no | read-only |
| `resolution` | yes | no | read-only |
| `workflows` | yes | no | read-only |
| `filters` | yes | no | read-only favorites |
| `filter` | yes | no | read-only |
| `permissions` | yes | no | read-only |
| `mypermissions` | yes | no | read-only |
| `dashboards` | yes | no | read-only |
| `dashboard` | yes | no | read-only |
| `dashboard item properties` | yes | no | read-only |
| `dashboard item property` | yes | no | read-only |
| `dashboard item property set --dry-run` | yes | no | no Jira access |
| `dashboard item property set --yes` | yes | no | write; fake only |
| `dashboard item property delete --dry-run` | yes | no | no Jira access |
| `dashboard item property delete --yes` | yes | no | destructive guard plus fake delete |
| `api get` | yes | no | guarded public JSON REST pass-through |
| `api post/put/delete --dry-run` | yes | no | no Jira access |
| `api post/put/delete --yes` | yes | no | write guard plus fake server; Agile live writes require `--force` |
| `issue properties` | yes | no | read-only |
| `issue property` | yes | no | read-only |
| `issue property set --dry-run` | yes | no | no Jira access |
| `issue property set --yes` | yes | no | write; fake only |
| `issue property delete --dry-run` | yes | no | no Jira access |
| `issue property delete --yes` | yes | no | destructive guard plus fake delete |
| `boards` | yes | yes | Agile read-only |
| `board` | yes | no | Agile read-only |
| `board issues` | yes | no | Agile read-only |
| `backlog` | yes | no | Agile read-only |
| `sprints` | yes | no | Agile read-only |
| `sprint` | yes | no | Agile read-only |
| `sprint issues` | yes | no | Agile read-only |
| `sprint summary` | yes | no | Agile read-only |
| `epic` | yes | no | Agile read-only |
| `epic issues` | yes | no | Agile read-only |
| `mine` | yes | yes | composed JQL, read-only |
| `stale` | yes | yes | composed JQL, read-only |
| `blockers` | yes | yes | derived from issue links |
| `create --meta` | yes | no | read-only createmeta |
| `create --yes` | yes | no | write; fake only |
| `update --yes` | yes | no | write; fake plus dry-run |
| `comment --yes` | yes | no | write; fake only |
| `assign --yes` | yes | no | write; fake only |
| `transitions` | yes | no | read-only |
| `transition --yes` | yes | no | write; fake only |
| `delete` | yes | no | destructive guard plus fake delete |
| `watchers` | yes | no | read-only |
| `watch --yes` | yes | no | write; fake only |
| `unwatch --yes` | yes | no | write; fake only |
| `worklog add --yes` | yes | no | write; fake only |
| `worklogs` | yes | no | read-only |
| `worklog get` | yes | no | read-only |
| `attachments` | yes | no | read-only |
| `attachment` | yes | no | read-only metadata |
| `attachment add --dry-run` | yes | no | no Jira access |
| `attachment add --yes` | yes | no | multipart upload; fake only |
| `attachment download` | yes | no | metadata-driven binary content; fake only |
| `attachment delete --dry-run` | yes | no | no Jira access |
| `attachment delete --yes` | yes | no | destructive guard plus fake delete |
| `link create --yes` | yes | no | write; fake only |
| `link delete --dry-run` | yes | no | no Jira access |
| `link delete --yes` | yes | no | destructive guard plus fake delete |
| `move sprint --dry-run` | yes | no | no Jira access |
| `move sprint --yes` | yes | no | write; fake only |
| `move backlog --dry-run` | yes | yes | no Jira write |
| `move backlog --yes` | yes | no | write; fake only |
