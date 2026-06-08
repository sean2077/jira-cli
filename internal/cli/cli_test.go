package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCobraHelpSurfaces(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{
		"agent-first Jira Server CLI",
		"Usage:",
		"Command Groups:",
		"create",
		"dashboard",
		"move",
		"skill",
		"Flags:",
		"--json",
		`Use "jira help <command>" or "jira <command> --help" for command-specific help.`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q in:\n%s", want, out)
		}
	}

	tests := []struct {
		name   string
		args   []string
		want   []string
		forbid []string
	}{
		{
			name: "create help",
			args: []string{"create", "--help"},
			want: []string{"Usage:", "jira create", "--project", "--issue-type", "--summary", "--field", "--dry-run"},
		},
		{
			name: "api post help",
			args: []string{"help", "api", "post"},
			want: []string{"Usage:", "jira api post PATH", "--body", "--query", "--force"},
		},
		{
			name:   "api get help",
			args:   []string{"help", "api", "get"},
			want:   []string{"Usage:", "jira api get PATH", "--query", "--force"},
			forbid: []string{"--body"},
		},
		{
			name: "dashboard property help",
			args: []string{"dashboard", "item", "property", "set", "--help"},
			want: []string{"Usage:", "jira dashboard item property set DASHBOARD_ID ITEM_ID KEY", "--body", "--yes"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout.Reset()
			stderr.Reset()
			code := Main(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			out := stdout.String()
			for _, want := range tt.want {
				if !strings.Contains(out, want) {
					t.Fatalf("help missing %q in:\n%s", want, out)
				}
			}
			for _, forbid := range tt.forbid {
				if strings.Contains(out, forbid) {
					t.Fatalf("help contained forbidden %q in:\n%s", forbid, out)
				}
			}
		})
	}
}

func TestSkillInstallDryRunUsesProjectDefault(t *testing.T) {
	temp := t.TempDir()
	t.Chdir(temp)

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"skill", "install", "--dry-run"}, &stdout, &stderr, Runtime{Env: map[string]string{}})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	target := filepath.Join(temp, ".agents", "skills", "jira-cli")
	if !strings.Contains(stdout.String(), "OK skill install dry-run target="+target+" scope=project exists=false") {
		t.Fatalf("stdout = %q, want target %q", stdout.String(), target)
	}
	if _, err := os.Stat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry-run created target or returned unexpected stat error: %v", err)
	}
}

func TestSkillInstallWritesEmbeddedSkill(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills-root")

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"skill", "install", "--target", root}, &stdout, &stderr, Runtime{Env: map[string]string{}})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	target := filepath.Join(root, "jira-cli")
	skillBody, err := os.ReadFile(filepath.Join(target, "SKILL.md"))
	if err != nil {
		t.Fatalf("read installed SKILL.md: %v", err)
	}
	if !strings.Contains(string(skillBody), "name: jira-cli") {
		t.Fatalf("installed SKILL.md does not look like jira-cli skill:\n%s", skillBody)
	}
	if _, err := os.Stat(filepath.Join(target, "agents", "openai.yaml")); err != nil {
		t.Fatalf("openai agent metadata was not installed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "embed.go")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("internal Go embed file leaked into installed skill: %v", err)
	}
	if !strings.Contains(stdout.String(), "scope=custom") {
		t.Fatalf("stdout = %q, want custom scope", stdout.String())
	}
}

func TestSkillInstallRejectsExistingTargetUnlessForced(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills-root")
	target := filepath.Join(root, "jira-cli")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"skill", "install", "--target", root}, &stdout, &stderr, Runtime{Env: map[string]string{}})
	if code != 1 {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "use --force") {
		t.Fatalf("stderr = %q, want force hint", stderr.String())
	}
}

func TestSkillInstallForceReplacesExistingTarget(t *testing.T) {
	root := filepath.Join(t.TempDir(), "skills-root")
	target := filepath.Join(root, "jira-cli")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	stale := filepath.Join(target, "stale.txt")
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"skill", "install", "--target", root, "--force"}, &stdout, &stderr, Runtime{Env: map[string]string{}})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("force install did not remove stale file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "SKILL.md")); err != nil {
		t.Fatalf("force install did not write skill: %v", err)
	}
}

func TestSkillInstallJSONAndGlobalTarget(t *testing.T) {
	home := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--json", "skill", "install", "--global", "--dry-run"}, &stdout, &stderr, Runtime{Env: map[string]string{"HOME": home}})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var result struct {
		OK     bool   `json:"ok"`
		Kind   string `json:"kind"`
		Skill  string `json:"skill"`
		Scope  string `json:"scope"`
		Target string `json:"target"`
		DryRun bool   `json:"dryRun"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode JSON: %v; stdout=%q", err, stdout.String())
	}
	if !result.OK || result.Kind != "skill_install" || result.Skill != "jira-cli" || result.Scope != "global" || !result.DryRun {
		t.Fatalf("unexpected result: %#v", result)
	}
	wantTarget := filepath.Join(home, ".agents", "skills", "jira-cli")
	if result.Target != wantTarget {
		t.Fatalf("target = %q, want %q", result.Target, wantTarget)
	}
}

func TestVersionCompactAndJSON(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "compact default", args: []string{"version"}, want: "jira " + Version + "\n"},
		{name: "json", args: []string{"--json", "version"}, want: `{"ok":true,"kind":"version","version":"` + Version + `"}` + "\n"},
		{name: "json then compact", args: []string{"--json", "--compact", "version"}, want: "jira " + Version + "\n"},
		{name: "compact then json", args: []string{"--compact", "--json", "version"}, want: `{"ok":true,"kind":"version","version":"` + Version + `"}` + "\n"},
		{name: "raw then compact", args: []string{"--raw", "--compact", "version"}, want: "jira " + Version + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Main(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCobraActionMapMatchesRunnableLeaves(t *testing.T) {
	opts := DefaultOptions()
	var stdout, stderr bytes.Buffer
	exitCode := 0
	root := newRootCommand(&opts, &stdout, &stderr, Runtime{}, &exitCode)

	actions := commandActions()
	runnable := map[string]bool{}
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd == nil || cmd.Hidden || cmd.Name() == "help" {
			return
		}
		if cmd.Runnable() && cmd.Annotations["jira-cli-parent"] != "true" {
			path := cmd.CommandPath()
			if path != "jira" && path != "jira version" {
				key := strings.TrimPrefix(path, "jira ")
				runnable[key] = true
				if _, ok := actions[key]; !ok {
					t.Errorf("missing action for Cobra command %q", key)
				}
			}
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(root)

	for key := range actions {
		if !runnable[key] {
			t.Errorf("action %q has no runnable Cobra command", key)
		}
	}
}

func TestCobraGlobalFlagsReachCommands(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Query().Get("jql"), "project = JCLI"; got != want {
			t.Fatalf("jql = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("maxResults"), "10"; got != want {
			t.Fatalf("maxResults = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("startAt"), "5"; got != want {
			t.Fatalf("startAt = %q, want %q", got, want)
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "agent" || pass != "secret-token" {
			t.Fatalf("basic auth = %q/%q ok=%t", user, pass, ok)
		}
		writeJSONBody(`{"startAt":5,"maxResults":10,"total":0,"issues":[]}`)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{
		"--profile", "private",
		"--base-url=" + server.URL,
		"--type", "server",
		"--user", "agent",
		"--token-env", "JIRA_PRIVATE_TOKEN",
		"--json",
		"--limit", "25",
		"--page-size=10",
		"--start-at", "5",
		"--timeout", "5s",
		"search",
		"project = JCLI",
	}, &stdout, &stderr, Runtime{Env: map[string]string{"JIRA_PRIVATE_TOKEN": "secret-token"}})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"kind":"issue_search"`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestCobraUsageErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing flag value", args: []string{"--base-url"}, want: "flag needs an argument"},
		{name: "invalid limit", args: []string{"search", "--limit", "0", "project = JCLI"}, want: "--limit must be a positive integer"},
		{name: "invalid start", args: []string{"search", "--start-at", "-1", "project = JCLI"}, want: "--start-at must be a non-negative integer"},
		{name: "invalid timeout", args: []string{"search", "--timeout", "nope", "project = JCLI"}, want: "invalid argument"},
		{name: "unknown flag", args: []string{"search", "--unknown", "project = JCLI"}, want: "unknown flag"},
		{name: "command-specific flag", args: []string{"issue", "JCLI-1", "--summary", "nope"}, want: "unknown flag"},
		{name: "api get body flag", args: []string{"api", "get", "filter/favourite", "--body", "{}"}, want: "unknown flag"},
		{name: "missing arg", args: []string{"issue"}, want: "accepts 1 arg"},
		{name: "unknown command suggestion", args: []string{"craete"}, want: "create"},
		{name: "nested typo nearest help", args: []string{"issue", "proeprty", "JCLI-1"}, want: `HINT use "jira issue --help"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, Runtime{})
			if code != 1 || !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("code = %d stdout=%q stderr=%q, want %q", code, stdout.String(), stderr.String(), tt.want)
			}
		})
	}
}

func TestProbeCompactAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/serverInfo", writeJSONBody(`{"version":"8.1.0"}`))
	mux.HandleFunc("/rest/api/2/myself", writeJSONBody(`{"name":"jdoe","key":"JDOE","displayName":"Jane Doe"}`))
	mux.HandleFunc("/rest/api/2/search", writeJSONBody(`{"startAt":0,"maxResults":0,"total":0,"issues":[]}`))
	mux.HandleFunc("/rest/api/2/project", writeJSONBody(`[{"key":"JCLI","name":"Jira CLI Test"}]`))
	mux.HandleFunc("/rest/api/2/field", writeJSONBody(`[{"id":"summary","name":"Summary","custom":false}]`))
	mux.HandleFunc("/rest/api/2/dashboard", writeJSONBody(`{"startAt":0,"maxResults":1,"total":0,"dashboards":[]}`))
	mux.HandleFunc("/rest/agile/1.0/board", writeJSONBody(`{"maxResults":1,"values":[]}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"probe"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	want := "OK probe server=8.1.0 api=/rest/api/2 dashboard=available agile=available user=jdoe\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestProbeMetadataChecksWhenInputsProvided(t *testing.T) {
	seen := map[string]bool{}
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/serverInfo", writeJSONBody(`{"version":"8.1.0"}`))
	mux.HandleFunc("/rest/api/2/myself", writeJSONBody(`{"name":"jdoe"}`))
	mux.HandleFunc("/rest/api/2/search", writeJSONBody(`{"startAt":0,"maxResults":0,"total":0,"issues":[]}`))
	mux.HandleFunc("/rest/api/2/project", writeJSONBody(`[]`))
	mux.HandleFunc("/rest/api/2/field", writeJSONBody(`[]`))
	mux.HandleFunc("/rest/api/2/dashboard", writeJSONBody(`{"startAt":0,"maxResults":1,"total":0,"dashboards":[]}`))
	mux.HandleFunc("/rest/api/2/issue/createmeta", func(w http.ResponseWriter, r *http.Request) {
		seen["createmeta"] = true
		if r.URL.Query().Get("projectKeys") != "JCLI" || r.URL.Query().Get("issuetypeNames") != "Bug" {
			t.Fatalf("createmeta query = %s", r.URL.RawQuery)
		}
		writeJSONBody(`{"projects":[]}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/editmeta", func(w http.ResponseWriter, r *http.Request) {
		seen["editmeta"] = true
		writeJSONBody(`{"fields":{}}`)(w, r)
	})
	mux.HandleFunc("/rest/agile/1.0/board", writeJSONBody(`{"maxResults":1,"values":[]}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--json", "probe", "--project", "JCLI", "--issue-type", "Bug", "--issue", "JCLI-1"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !seen["createmeta"] || !seen["editmeta"] {
		t.Fatalf("metadata endpoints not called: %#v", seen)
	}
	var result struct {
		CreateMeta bool `json:"createMeta"`
		EditMeta   bool `json:"editMeta"`
		Dashboards bool `json:"dashboards"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode probe JSON: %v", err)
	}
	if !result.CreateMeta || !result.EditMeta || !result.Dashboards {
		t.Fatalf("metadata result = %#v", result)
	}
}

func TestSearchCompactGoldenPaginatesByActualItems(t *testing.T) {
	var starts []string
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("fields"); got != "summary,status,priority,assignee,project,created,updated" {
			t.Fatalf("fields = %q", got)
		}
		start := r.URL.Query().Get("startAt")
		starts = append(starts, start)
		switch start {
		case "0":
			writeJSONBody(`{"startAt":0,"maxResults":1,"total":3,"issues":[{"key":"JCLI-1","fields":{"summary":"First issue","status":{"name":"Open"},"priority":{"name":"P2"},"assignee":{"name":"jdoe"}}}]}`)(w, r)
		case "1":
			writeJSONBody(`{"startAt":1,"maxResults":1,"total":3,"issues":[{"key":"JCLI-2","fields":{"summary":"Second issue","status":{"name":"In Progress"},"priority":{"name":"P1"}}}]}`)(w, r)
		default:
			t.Fatalf("unexpected startAt %q", start)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"search", "project = JCLI", "--limit", "2", "--page-size", "2"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if got, want := strings.Join(starts, ","), "0,1"; got != want {
		t.Fatalf("startAt sequence = %q, want %q", got, want)
	}
	want := "JCLI-1 Open P2 jdoe First issue\nJCLI-2 In Progress P1 - Second issue\n2 issues total=3 next=\"--start-at 2\"\n"
	if got := stdout.String(); got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestSearchJSONSchemaIsStable(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", writeJSONBody(`{"startAt":0,"maxResults":50,"total":1,"issues":[{"key":"JCLI-1","fields":{"summary":"First issue","status":{"name":"Open"},"priority":{"name":"P2"},"assignee":{"name":"jdoe"}}}]}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--json", "search", "project = JCLI", "--limit", "1"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var body struct {
		OK    bool `json:"ok"`
		Kind  string
		Items []struct {
			Key     string `json:"key"`
			Summary string `json:"summary"`
			Status  string `json:"status"`
		} `json:"items"`
		Page struct {
			StartAt    int `json:"startAt"`
			MaxResults int `json:"maxResults"`
			Total      int `json:"total"`
		} `json:"page"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
		t.Fatalf("decode json: %v; stdout=%q", err, stdout.String())
	}
	if !body.OK || body.Kind != "issue_search" || len(body.Items) != 1 || body.Items[0].Key != "JCLI-1" || body.Items[0].Summary != "First issue" || body.Items[0].Status != "Open" || body.Page.Total != 1 {
		t.Fatalf("unexpected schema body: %#v", body)
	}
}

func TestReadOnlyCommandsAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/myself", writeJSONBody(`{"name":"jdoe","key":"JDOE","emailAddress":"jdoe@example.com","displayName":"Jane Doe"}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1", writeJSONBody(`{"key":"JCLI-1","fields":{"summary":"First issue","status":{"name":"Open"},"priority":{"name":"P2"},"assignee":{"name":"jdoe"},"issuelinks":[{"id":"300","type":{"name":"Blocks","outward":"blocks","inward":"is blocked by"},"outwardIssue":{"key":"JCLI-2","fields":{"summary":"Blocking issue","status":{"name":"Open"}}}}]}}`))
	mux.HandleFunc("/rest/api/2/project", writeJSONBody(`[{"key":"JCLI","name":"Jira CLI Test Project"}]`))
	mux.HandleFunc("/rest/api/2/project/JCLI", writeJSONBody(`{"key":"JCLI","name":"Jira CLI Test Project","lead":{"name":"lead"}}`))
	mux.HandleFunc("/rest/api/2/field", writeJSONBody(`[{"id":"customfield_10016","name":"Story Points","custom":true}]`))
	mux.HandleFunc("/rest/api/2/user/search", writeJSONBody(`[{"name":"jdoe","key":"JDOE","emailAddress":"jdoe@example.com","displayName":"Jane Doe"}]`))
	mux.HandleFunc("/rest/api/2/issuetype", writeJSONBody(`[{"id":"1","name":"Bug"}]`))
	mux.HandleFunc("/rest/api/2/priority", writeJSONBody(`[{"id":"2","name":"P2"}]`))
	mux.HandleFunc("/rest/api/2/status", writeJSONBody(`[{"id":"3","name":"Open"}]`))
	mux.HandleFunc("/rest/api/2/workflow", writeJSONBody(`[{"id":"classic","name":"Classic Workflow"}]`))
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "whoami", args: []string{"whoami"}, want: "jdoe key=JDOE email=jdoe@example.com display=\"Jane Doe\"\n"},
		{name: "issue", args: []string{"issue", "JCLI-1"}, want: "JCLI-1 Open P2 jdoe First issue\n"},
		{name: "links", args: []string{"links", "JCLI-1"}, want: "300 Blocks blocks JCLI-2 Open - - Blocking issue\n"},
		{name: "projects", args: []string{"projects"}, want: "JCLI Jira CLI Test Project\n"},
		{name: "project", args: []string{"project", "JCLI"}, want: "JCLI Jira CLI Test Project lead=lead\n"},
		{name: "users search", args: []string{"users", "search", "--query", "jdoe"}, want: "jdoe key=JDOE email=jdoe@example.com display=\"Jane Doe\"\n1 users\n"},
		{name: "fields", args: []string{"fields"}, want: "customfield_10016 Story Points custom=true\n"},
		{name: "issuetypes", args: []string{"issuetypes"}, want: "1 Bug\n"},
		{name: "priorities", args: []string{"priorities"}, want: "2 P2\n"},
		{name: "statuses", args: []string{"statuses"}, want: "3 Open\n"},
		{name: "workflows", args: []string{"workflows"}, want: "classic Classic Workflow\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJira81AdditionalCommandsAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/comment", writeJSONBody(`{"startAt":0,"maxResults":1,"total":1,"comments":[{"id":"101","body":"first comment","author":{"name":"jdoe"}}]}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/comment/101", writeJSONBody(`{"id":"101","body":"first comment","author":{"name":"jdoe"}}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/worklog", writeJSONBody(`{"startAt":0,"maxResults":1,"total":1,"worklogs":[{"id":"201","comment":"done","timeSpent":"1h","author":{"name":"jdoe"}}]}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/worklog/201", writeJSONBody(`{"id":"201","comment":"done","timeSpent":"1h","author":{"name":"jdoe"}}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/remotelink", writeJSONBody(`[{"id":301,"relationship":"relates","object":{"title":"docs","url":"https://example.com/doc"}}]`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/remotelink/301", writeJSONBody(`{"id":301,"relationship":"relates","object":{"title":"docs","url":"https://example.com/doc"}}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/properties", writeJSONBody(`{"keys":[{"self":"http://jira/rest/api/2/issue/JCLI-1/properties/agent.state","key":"agent.state"}]}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/properties/agent.state", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSONBody(`{"key":"agent.state","value":{"status":"seen"}}`)(w, r)
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read issue property body: %v", err)
			}
			if got, want := string(body), `{"status":"seen","sequence":9007199254740993123}`; got != want {
				t.Fatalf("issue property body = %q, want %q", got, want)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("issue property method = %s", r.Method)
		}
	})
	mux.HandleFunc("/rest/api/2/filter/favourite", writeJSONBody(`[{"id":"10000","name":"My Filter","jql":"project = JCLI","favourite":true,"owner":{"name":"jdoe"}}]`))
	mux.HandleFunc("/rest/api/2/filter/10000", writeJSONBody(`{"id":"10000","name":"My Filter","jql":"project = JCLI","favourite":true,"owner":{"name":"jdoe"}}`))
	mux.HandleFunc("/rest/api/2/project/JCLI/components", writeJSONBody(`[{"id":"400","name":"UI"}]`))
	mux.HandleFunc("/rest/api/2/project/JCLI/versions", writeJSONBody(`[{"id":"500","name":"1.0","released":false,"archived":false}]`))
	mux.HandleFunc("/rest/api/2/project/JCLI/role", writeJSONBody(`{"Developers":"http://jira/rest/api/2/project/JCLI/role/10002"}`))
	mux.HandleFunc("/rest/api/2/project/JCLI/role/10002", writeJSONBody(`{"id":10002,"name":"Developers"}`))
	mux.HandleFunc("/rest/api/2/project/JCLI/statuses", writeJSONBody(`[{"name":"Bug","statuses":[{"name":"Open"}]}]`))
	mux.HandleFunc("/rest/api/2/user/assignable/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("project") != "JCLI" || query.Get("username") != "jdoe" || query.Get("startAt") != "0" || query.Get("maxResults") != "50" {
			t.Fatalf("assignable query = %s", r.URL.RawQuery)
		}
		writeJSONBody(`[{"name":"jdoe","key":"JDOE","displayName":"Jane Doe"}]`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/permissions", writeJSONBody(`{"permissions":{"BROWSE_PROJECTS":{"key":"BROWSE_PROJECTS","name":"Browse Projects"}}}`))
	mux.HandleFunc("/rest/api/2/mypermissions", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("projectKey") != "JCLI" {
			t.Fatalf("mypermissions query = %s", r.URL.RawQuery)
		}
		writeJSONBody(`{"permissions":{"BROWSE_PROJECTS":{"havePermission":true}}}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/resolution", writeJSONBody(`[{"id":"1","name":"Fixed"}]`))
	mux.HandleFunc("/rest/api/2/resolution/1", writeJSONBody(`{"id":"1","name":"Fixed"}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "comments", args: []string{"comments", "JCLI-1"}, want: "101 jdoe first comment\n1 comments total=1\n"},
		{name: "comment get", args: []string{"comment", "get", "JCLI-1", "101"}, want: "101 jdoe first comment\n"},
		{name: "worklogs", args: []string{"worklogs", "JCLI-1"}, want: "201 jdoe 1h done\n1 worklogs total=1\n"},
		{name: "worklog get", args: []string{"worklog", "get", "JCLI-1", "201"}, want: "201 jdoe 1h done\n"},
		{name: "remote links", args: []string{"remote-links", "JCLI-1"}, want: "301 relates docs https://example.com/doc\n1 remote-links\n"},
		{name: "remote link", args: []string{"remote-link", "JCLI-1", "301"}, want: "301 relates docs https://example.com/doc\n"},
		{name: "issue properties", args: []string{"issue", "properties", "JCLI-1"}, want: "agent.state self=http://jira/rest/api/2/issue/JCLI-1/properties/agent.state\n1 properties\n"},
		{name: "issue property", args: []string{"issue", "property", "JCLI-1", "agent.state"}, want: "agent.state value={\"status\":\"seen\"}\n"},
		{name: "filters", args: []string{"filters"}, want: "10000 My Filter owner=jdoe favourite=true jql=\"project = JCLI\"\n1 filters\n"},
		{name: "filter", args: []string{"filter", "10000"}, want: "10000 My Filter owner=jdoe favourite=true jql=\"project = JCLI\"\n"},
		{name: "components", args: []string{"components", "JCLI"}, want: "400 UI\n1 components\n"},
		{name: "versions", args: []string{"versions", "JCLI"}, want: "500 1.0 released=false archived=false\n1 versions\n"},
		{name: "roles", args: []string{"roles", "JCLI"}, want: "Developers http://jira/rest/api/2/project/JCLI/role/10002\n1 roles\n"},
		{name: "role", args: []string{"role", "JCLI", "10002"}, want: "{\"id\":10002,\"name\":\"Developers\"}\n"},
		{name: "project statuses", args: []string{"project-statuses", "JCLI"}, want: "1 project-statuses\n"},
		{name: "assignable", args: []string{"assignable", "--project", "JCLI", "--query", "jdoe"}, want: "jdoe key=JDOE email=- display=\"Jane Doe\"\n1 assignable\n"},
		{name: "permissions", args: []string{"permissions"}, want: "{\"permissions\":{\"BROWSE_PROJECTS\":{\"key\":\"BROWSE_PROJECTS\",\"name\":\"Browse Projects\"}}}\n"},
		{name: "mypermissions", args: []string{"mypermissions", "--project", "JCLI"}, want: "{\"permissions\":{\"BROWSE_PROJECTS\":{\"havePermission\":true}}}\n"},
		{name: "resolutions", args: []string{"resolutions"}, want: "1 Fixed\n"},
		{name: "resolution", args: []string{"resolution", "1"}, want: "1 Fixed\n"},
		{name: "issue property set", args: []string{"issue", "property", "set", "JCLI-1", "agent.state", "--body", `{"status":"seen","sequence":9007199254740993123}`, "--yes"}, want: "OK issue_property_set issue=JCLI-1 key=agent.state\n"},
		{name: "issue property delete", args: []string{"issue", "property", "delete", "JCLI-1", "agent.state", "--yes"}, want: "OK issue_property_delete issue=JCLI-1 key=agent.state\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIPassThroughAgainstFakeServer(t *testing.T) {
	var postCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/jira/rest/api/2/filter/favourite", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("expand") != "sharedUsers" {
			t.Fatalf("api get request = %s %s", r.Method, r.URL.String())
		}
		writeJSONBody(`[{"id":"10000"}]`)(w, r)
	})
	mux.HandleFunc("/jira/rest/agile/1.0/board", writeJSONBody(`{"startAt":0,"maxResults":1,"total":0,"values":[]}`))
	mux.HandleFunc("/jira/rest/api/2/filter", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("api post method = %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read api body: %v", err)
		}
		if got, want := string(body), `{"name":"Huge","sequence":9007199254740993123}`; got != want {
			t.Fatalf("api body = %q, want %q", got, want)
		}
		postCalled = true
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"10001"}`))
	})
	mux.HandleFunc("/jira/rest/api/2/project/JCLI/avatar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("avatar JSON method = %s", r.Method)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read avatar JSON body: %v", err)
		}
		if got, want := string(body), `{"id":"10000"}`; got != want {
			t.Fatalf("avatar JSON body = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	rt := Runtime{
		Env: map[string]string{
			"JIRA_BASE_URL":  server.URL + "/jira",
			"JIRA_USER":      "agent",
			"JIRA_API_TOKEN": "secret",
		},
		HTTPClient: server.Client(),
	}

	t.Run("platform get with query and context path", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "get", "/rest/api/2/filter/favourite", "--query", "expand=sharedUsers"}, &stdout, &stderr, rt)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "OK api_get status=200 path=/rest/api/2/filter/favourite body=[{\"id\":\"10000\"}]\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})

	t.Run("agile shorthand", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "get", "agile/1.0/board"}, &stdout, &stderr, rt)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "path=/rest/agile/1.0/board") {
			t.Fatalf("stdout = %q", stdout.String())
		}
	})

	t.Run("write requires explicit intent", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "post", "filter", "--body", `{"name":"Huge"}`}, &stdout, &stderr, rt)
		if code != 1 || !strings.Contains(stderr.String(), "requires --dry-run or --yes") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("dry run skips config", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "delete", "filter/10001", "--query", "expand=owner", "--dry-run"}, &stdout, &stderr, Runtime{})
		if code != 0 || !strings.Contains(stdout.String(), "DRY-RUN api_delete") || !strings.Contains(stdout.String(), "query=expand=owner") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("post yes preserves json body", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "post", "filter", "--body", `{"name":"Huge","sequence":9007199254740993123}`, "--yes"}, &stdout, &stderr, rt)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "OK api_post path=/rest/api/2/filter status=201\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})

	t.Run("avatar json write allowed", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "put", "project/JCLI/avatar", "--body", `{"id":"10000"}`, "--yes"}, &stdout, &stderr, rt)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "OK api_put path=/rest/api/2/project/JCLI/avatar status=204\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
	if !postCalled {
		t.Fatal("api post was not called")
	}
}

func TestAPIPassThroughAgileWriteRequiresForce(t *testing.T) {
	var called bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/agile/1.0/sprint/7/issue", func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Fatalf("agile write method = %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	rt := runtimeForServer(server)

	t.Run("dry run does not require force", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "post", "agile/1.0/sprint/7/issue", "--body", `{"issues":["JCLI-1"]}`, "--dry-run"}, &stdout, &stderr, Runtime{})
		if code != 0 || !strings.Contains(stdout.String(), "DRY-RUN api_post") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("yes without force is rejected before http", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "post", "agile/1.0/sprint/7/issue", "--body", `{"issues":["JCLI-1"]}`, "--yes"}, &stdout, &stderr, rt)
		if code != 1 || !strings.Contains(stderr.String(), "Agile writes require --force") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if called {
			t.Fatal("agile write guard allowed an HTTP request")
		}
	})

	t.Run("yes force is explicit escape hatch", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"api", "post", "agile/1.0/sprint/7/issue", "--body", `{"issues":["JCLI-1"]}`, "--yes", "--force"}, &stdout, &stderr, rt)
		if code != 0 {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if got, want := stdout.String(), "OK api_post path=/rest/agile/1.0/sprint/7/issue status=204\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestAPIPassThroughRejectsUnsafePathsBeforeHTTP(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	rt := runtimeForServer(server)

	tests := []string{
		"https://jira.example.com/rest/api/2/filter",
		"//jira.example.com/rest/api/2/filter",
		"/rest/api/latest/filter",
		"/rest/auth/1/session",
		"../filter",
		"filter/%2e%2e",
		"filter%2fsecret",
		"filter?expand=owner",
		"filter#frag",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime([]string{"api", "get", path}, &stdout, &stderr, rt)
			if code != 1 {
				t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
	if called {
		t.Fatal("unsafe path validation allowed an HTTP request")
	}
}

func TestAPIPassThroughRejectsExtraArgsAndEmptyBodyBeforeHTTP(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	rt := runtimeForServer(server)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "get extra arg", args: []string{"api", "get", "filter", "extra"}, want: "accepts 1 arg"},
		{name: "write extra arg", args: []string{"api", "delete", "filter/10000", "extra", "--yes"}, want: "accepts 1 arg"},
		{name: "empty body", args: []string{"api", "post", "filter", "--body=", "--yes"}, want: "--body must be valid JSON"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, rt)
			if code != 1 || !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
	if called {
		t.Fatal("api validation allowed an HTTP request")
	}
}

func TestAPIPassThroughRejectsSpecialHeaderEndpointsBeforeDryRun(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "attachment upload",
			args: []string{"api", "post", "issue/JCLI-1/attachments", "--body", `{}`, "--dry-run"},
			want: "attachment upload",
		},
		{
			name: "avatar temporary upload",
			args: []string{"api", "post", "avatar/temporary", "--body", `{}`, "--dry-run"},
			want: "avatar",
		},
		{
			name: "attachment content download",
			args: []string{"api", "get", "attachment/content/10001"},
			want: "attachment content",
		},
		{
			name: "attachment metadata",
			args: []string{"api", "get", "attachment/10001"},
			want: "attachment endpoints",
		},
		{
			name: "attachment write",
			args: []string{"api", "delete", "attachment/10001", "--dry-run"},
			want: "attachment write",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, Runtime{})
			if code != 1 || !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestAPIPassThroughAllowsReadOnlyAvatarJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/avatar/project/system", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("avatar method = %s", r.Method)
		}
		writeJSONBody(`{"system":[{"id":"10000"}]}`)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"api", "get", "avatar/project/system"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "OK api_get status=200 path=/rest/api/2/avatar/project/system body={\"system\":[{\"id\":\"10000\"}]}\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestAttachmentCommandsAgainstFakeServer(t *testing.T) {
	tempDir := t.TempDir()
	uploadPath := filepath.Join(tempDir, "proof.txt")
	if err := os.WriteFile(uploadPath, []byte("proof"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	downloadPath := filepath.Join(tempDir, "download.txt")
	var uploadCalled bool
	var deleteCalled bool
	var serverURL string

	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/JCLI-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("fields") != "attachment" {
			t.Fatalf("attachments request = %s %s", r.Method, r.URL.String())
		}
		writeJSONBody(`{"key":"JCLI-1","fields":{"attachment":[{"id":"700","filename":"proof.txt","size":5,"mimeType":"text/plain","author":{"name":"jdoe"}}]}}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/attachments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("attachment upload method = %s", r.Method)
		}
		if got := r.Header.Get("X-Atlassian-Token"); got != "no-check" {
			t.Fatalf("X-Atlassian-Token = %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Fatalf("Content-Type = %q", r.Header.Get("Content-Type"))
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("missing multipart file: %v", err)
		}
		defer file.Close()
		if header.Filename != "proof.txt" {
			t.Fatalf("filename = %q", header.Filename)
		}
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read multipart file: %v", err)
		}
		if string(body) != "proof" {
			t.Fatalf("multipart body = %q", string(body))
		}
		uploadCalled = true
		writeJSONBody(`[{"id":"700","filename":"proof.txt","size":5,"mimeType":"text/plain"}]`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/attachment/700", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSONBody(`{"id":"700","filename":"proof.txt","size":5,"mimeType":"text/plain","content":"`+serverURL+`/secure/attachment/700/proof.txt","author":{"name":"jdoe"}}`)(w, r)
		case http.MethodDelete:
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("attachment method = %s", r.Method)
		}
	})
	mux.HandleFunc("/secure/attachment/700/proof.txt", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("attachment content method = %s", r.Method)
		}
		if r.Header.Get("Authorization") == "" {
			t.Fatal("attachment content download missing auth header")
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("proof"))
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	serverURL = server.URL
	rt := runtimeForServer(server)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "list", args: []string{"attachments", "JCLI-1"}, want: "700 proof.txt size=5 mime=text/plain\n1 attachments\n"},
		{name: "get", args: []string{"attachment", "700"}, want: "700 proof.txt size=5 mime=text/plain\n"},
		{name: "add dry-run no config", args: []string{"attachment", "add", "JCLI-1", "--file", uploadPath, "--dry-run"}, want: "DRY-RUN attachment_add file=" + uploadPath + " issue=JCLI-1\n"},
		{name: "add yes", args: []string{"attachment", "add", "JCLI-1", "--file", uploadPath, "--yes"}, want: "700 proof.txt size=5 mime=text/plain\n1 attachments uploaded\n"},
		{name: "delete dry-run no config", args: []string{"attachment", "delete", "700", "--dry-run"}, want: "DRY-RUN attachment_delete id=700\n"},
		{name: "delete yes", args: []string{"attachment", "delete", "700", "--yes"}, want: "OK attachment_delete id=700\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			testRT := rt
			if strings.Contains(tt.name, "no config") {
				testRT = Runtime{}
			}
			code := MainWithRuntime(tt.args, &stdout, &stderr, testRT)
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("download", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"attachment", "download", "700", "--file", downloadPath}, &stdout, &stderr, rt)
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "OK attachment_download bytes=5 file="+downloadPath+" id=700\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
		body, err := os.ReadFile(downloadPath)
		if err != nil {
			t.Fatalf("read download: %v", err)
		}
		if string(body) != "proof" {
			t.Fatalf("download body = %q", string(body))
		}
	})

	t.Run("download refuses overwrite", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"attachment", "download", "700", "--file", downloadPath}, &stdout, &stderr, rt)
		if code != 1 || !strings.Contains(stderr.String(), "output exists") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("download force refuses directory target", func(t *testing.T) {
		dirTarget := filepath.Join(tempDir, "download-dir")
		if err := os.Mkdir(dirTarget, 0o700); err != nil {
			t.Fatalf("mkdir target: %v", err)
		}
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"attachment", "download", "700", "--file", dirTarget, "--force"}, &stdout, &stderr, rt)
		if code != 1 || !strings.Contains(stderr.String(), "must not be a directory") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
		if info, err := os.Stat(dirTarget); err != nil || !info.IsDir() {
			t.Fatalf("directory target should remain a directory; info=%v err=%v", info, err)
		}
	})

	if !uploadCalled {
		t.Fatal("attachment upload was not called")
	}
	if !deleteCalled {
		t.Fatal("attachment delete was not called")
	}
}

func TestAttachmentDownloadRejectsUnsafeContentURLBeforeWrite(t *testing.T) {
	tempDir := t.TempDir()
	downloadPath := filepath.Join(tempDir, "download.txt")
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/attachment/700", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("attachment method = %s", r.Method)
		}
		writeJSONBody(`{"id":"700","filename":"proof.txt","content":"https://evil.example/secure/attachment/700/proof.txt"}`)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"attachment", "download", "700", "--file", downloadPath}, &stdout, &stderr, runtimeForServer(server))
	if code != 1 || !strings.Contains(stderr.String(), "must stay on the Jira host") {
		t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(downloadPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("download file should not exist; stat err=%v", err)
	}
}

func TestCreateWithAttachmentAgainstFakeServer(t *testing.T) {
	tempDir := t.TempDir()
	uploadPath := filepath.Join(tempDir, "proof.txt")
	if err := os.WriteFile(uploadPath, []byte("proof"), 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	var uploadCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("create method = %s", r.Method)
		}
		writeJSONBody(`{"id":"10000","key":"JCLI-10"}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/attachments", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("attachment upload method = %s", r.Method)
		}
		if got := r.Header.Get("X-Atlassian-Token"); got != "no-check" {
			t.Fatalf("X-Atlassian-Token = %q", got)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("missing multipart file: %v", err)
		}
		defer file.Close()
		body, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read multipart file: %v", err)
		}
		if header.Filename != "proof.txt" || string(body) != "proof" {
			t.Fatalf("uploaded filename=%q body=%q", header.Filename, string(body))
		}
		uploadCalled = true
		writeJSONBody(`[{"id":"700","filename":"proof.txt","size":5,"mimeType":"text/plain"}]`)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--attach", uploadPath, "--yes"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "OK create attachments=1 issue=JCLI-10\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if !uploadCalled {
		t.Fatal("attachment upload was not called")
	}
}

func TestRawIssuePrintsUnmodifiedJiraBody(t *testing.T) {
	raw := `not-json-from-jira`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/JCLI-1", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(raw))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--raw", "issue", "JCLI-1"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != raw {
		t.Fatalf("stdout = %q, want raw body %q", got, raw)
	}
}

func TestRawSearchPrintsUnmodifiedJiraBody(t *testing.T) {
	raw := `schema-debug-body`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("jql"); got != "project = JCLI" {
			t.Fatalf("jql = %q", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(raw))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--raw", "search", "project = JCLI"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != raw {
		t.Fatalf("stdout = %q, want raw body %q", got, raw)
	}
}

func TestProbeRawIsRejectedBeforeConfigAndJiraCall(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--raw", "probe"}, &stdout, &stderr, Runtime{})
	if code != 1 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "--raw is not supported for probe") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestCreateMetaIsReadOnly(t *testing.T) {
	var createmetaCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/createmeta", func(w http.ResponseWriter, r *http.Request) {
		createmetaCalled = true
		if r.Method != http.MethodGet {
			t.Fatalf("createmeta method = %s", r.Method)
		}
		query := r.URL.Query()
		if query.Get("projectKeys") != "JCLI" || query.Get("issuetypeNames") != "Bug" || query.Get("expand") != "projects.issuetypes.fields" {
			t.Fatalf("createmeta query = %s", r.URL.RawQuery)
		}
		writeJSONBody(`{"projects":[{"key":"JCLI","issuetypes":[{"name":"Bug","fields":{"summary":{"name":"Summary","required":true},"components":{"name":"Component/s","required":true,"schema":{"type":"array","items":"component","system":"components"},"allowedValues":[{"id":"400","name":"UI"}]},"description":{"name":"Description","required":true,"schema":{"type":"string","system":"description"}},"duedate":{"name":"Due Date","required":true,"schema":{"type":"date","system":"duedate"}},"versions":{"name":"Affects Version/s","required":true,"schema":{"type":"array","items":"version","system":"versions"},"allowedValues":[{"id":"499","name":"0.9","released":true},{"id":"500","name":"1.0","released":false}]}}}]}]}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue", func(_ http.ResponseWriter, r *http.Request) {
		t.Fatalf("create --meta must not call %s %s", r.Method, r.URL.Path)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"create", "--project", "JCLI", "--issue-type", "Bug", "--meta"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !createmetaCalled {
		t.Fatal("createmeta endpoint was not called")
	}
	out := stdout.String()
	for _, want := range []string{
		"OK createmeta project=JCLI issue-type=Bug",
		"4 additional required fields",
		"components Component/s allowed=400:UI example=--component 400",
		"description Description example=--body '...'",
		"duedate Due Date example=--due YYYY-MM-DD",
		"versions Affects Version/s allowed=500:1.0,499:0.9 example=--version 500",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q in %q", want, out)
		}
	}
}

func TestCreateMetaListsAvailableIssueTypesWhenNameDoesNotMatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/createmeta", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("projectKeys") != "JCLI" {
			t.Fatalf("createmeta query = %s", r.URL.RawQuery)
		}
		if query.Get("issuetypeNames") == "Task" {
			writeJSONBody(`{"projects":[{"key":"JCLI","issuetypes":[]}]}`)(w, r)
			return
		}
		if query.Get("issuetypeNames") != "" {
			t.Fatalf("unexpected issue type query = %s", r.URL.RawQuery)
		}
		writeJSONBody(`{"projects":[{"key":"JCLI","issuetypes":[{"name":"任务"},{"name":"Bug"}]}]}`)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"create", "--project", "JCLI", "--issue-type", "Task", "--meta"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"0 issue-types matched", "available issue-types: Bug, 任务"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q in %q", want, out)
		}
	}
}

func TestCreateMetaRawPrintsUnmodifiedJiraBody(t *testing.T) {
	raw := `non-json-createmeta`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/createmeta", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("createmeta method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(raw))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"--raw", "create", "--project", "JCLI", "--issue-type", "Bug", "--meta"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if got := stdout.String(); got != raw {
		t.Fatalf("stdout = %q, want raw body %q", got, raw)
	}
}

func TestDashboardCommandsAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("dashboards method = %s", r.Method)
		}
		query := r.URL.Query()
		if query.Get("filter") != "my" || query.Get("startAt") != "0" || query.Get("maxResults") != "1" {
			t.Fatalf("dashboards query = %s", r.URL.RawQuery)
		}
		writeJSONBody(`{"startAt":0,"maxResults":1,"total":2,"dashboards":[{"id":"10000","name":"System Dashboard","self":"http://jira/rest/api/2/dashboard/10000","view":"http://jira/secure/Dashboard.jspa?selectPageId=10000"}]}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/dashboard/10000", writeJSONBody(`{"id":"10000","name":"System Dashboard","self":"http://jira/rest/api/2/dashboard/10000","view":"http://jira/secure/Dashboard.jspa?selectPageId=10000"}`))
	mux.HandleFunc("/rest/api/2/dashboard/10000/items/20000/properties", writeJSONBody(`{"keys":[{"self":"http://jira/rest/api/2/dashboard/10000/items/20000/properties/issue.support","key":"issue.support"}]}`))
	mux.HandleFunc("/rest/api/2/dashboard/10000/items/20000/properties/issue.support", writeJSONBody(`{"key":"issue.support","value":{"hipchat.room.id":"support-123","support.time":"1m"}}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "dashboards", args: []string{"dashboards", "--filter", "my", "--limit", "1"}, want: "10000 System Dashboard view=http://jira/secure/Dashboard.jspa?selectPageId=10000\n1 dashboards total=2 next=\"--start-at 1\"\n"},
		{name: "dashboard", args: []string{"dashboard", "10000"}, want: "10000 System Dashboard view=http://jira/secure/Dashboard.jspa?selectPageId=10000\n"},
		{name: "property keys", args: []string{"dashboard", "item", "properties", "10000", "20000"}, want: "issue.support self=http://jira/rest/api/2/dashboard/10000/items/20000/properties/issue.support\n1 properties\n"},
		{name: "property get", args: []string{"dashboard", "item", "property", "10000", "20000", "issue.support"}, want: "issue.support value={\"hipchat.room.id\":\"support-123\",\"support.time\":\"1m\"}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDashboardJSONAndRawOutputsAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"startAt":0,"maxResults":1,"total":1,"dashboards":[{"id":"10000","name":"System Dashboard","self":"http://jira/rest/api/2/dashboard/10000","view":"http://jira/secure/Dashboard.jspa?selectPageId=10000","extraPanelHint":{"itemIds":["20000"]}}]}`))
	})
	rawMux := http.NewServeMux()
	rawMux.HandleFunc("/rest/api/2/dashboard", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(`raw-dashboard-list`))
	})
	mux.HandleFunc("/rest/api/2/dashboard/10000", writeJSONBody(`{"id":"10000","name":"System Dashboard","self":"http://jira/rest/api/2/dashboard/10000","view":"http://jira/secure/Dashboard.jspa?selectPageId=10000","extraPanelHint":{"itemIds":["20000"]}}`))
	mux.HandleFunc("/rest/api/2/dashboard/10000/items/20000/properties/issue.support", writeJSONBody(`{"key":"issue.support","value":{"columns":["status","assignee"],"limit":25}}`))
	server := httptest.NewServer(mux)
	defer server.Close()
	rawServer := httptest.NewServer(rawMux)
	defer rawServer.Close()

	t.Run("raw dashboards", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--raw", "dashboards"}, &stdout, &stderr, runtimeForServer(rawServer))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "raw-dashboard-list"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})

	t.Run("json dashboards preserve raw public objects", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "dashboards", "--limit", "1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK    bool `json:"ok"`
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
			RawDashboards []struct {
				ExtraPanelHint struct {
					ItemIDs []string `json:"itemIds"`
				} `json:"extraPanelHint"`
			} `json:"rawDashboards"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode dashboards JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || len(body.Items) != 1 || body.Items[0].ID != "10000" || len(body.RawDashboards) != 1 || len(body.RawDashboards[0].ExtraPanelHint.ItemIDs) != 1 || body.RawDashboards[0].ExtraPanelHint.ItemIDs[0] != "20000" {
			t.Fatalf("unexpected dashboards body: %#v", body)
		}
	})

	t.Run("json dashboard preserves raw public object", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "dashboard", "10000"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK        bool   `json:"ok"`
			Kind      string `json:"kind"`
			Dashboard struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"dashboard"`
			RawDashboard struct {
				ExtraPanelHint struct {
					ItemIDs []string `json:"itemIds"`
				} `json:"extraPanelHint"`
			} `json:"rawDashboard"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode dashboard JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || body.Kind != "dashboard" || body.Dashboard.ID != "10000" || body.Dashboard.Name != "System Dashboard" || len(body.RawDashboard.ExtraPanelHint.ItemIDs) != 1 || body.RawDashboard.ExtraPanelHint.ItemIDs[0] != "20000" {
			t.Fatalf("unexpected dashboard body: %#v", body)
		}
	})

	t.Run("json property", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "dashboard", "item", "property", "10000", "20000", "issue.support"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK       bool   `json:"ok"`
			Kind     string `json:"kind"`
			Property struct {
				Key   string `json:"key"`
				Value struct {
					Columns []string `json:"columns"`
					Limit   int      `json:"limit"`
				} `json:"value"`
			} `json:"property"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode dashboard property JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || body.Kind != "dashboard_item_property" || body.Property.Key != "issue.support" || body.Property.Value.Limit != 25 || len(body.Property.Value.Columns) != 2 {
			t.Fatalf("unexpected dashboard property body: %#v", body)
		}
	})
}

func TestDashboardItemPropertyWritesAgainstFakeServer(t *testing.T) {
	var putCalled, deleteCalled bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/dashboard/10000/items/20000/properties/issue.support", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read dashboard property body: %v", err)
			}
			if got, want := string(body), `{"panelId":9007199254740993123,"jql":"project = JCLI"}`; got != want {
				t.Fatalf("property body = %q, want %q", got, want)
			}
			putCalled = true
			w.WriteHeader(http.StatusCreated)
		case http.MethodDelete:
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("dashboard property method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("set dry-run", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"dashboard", "item", "property", "set", "10000", "20000", "issue.support", "--body", `{"limit":25}`, "--dry-run"}, &stdout, &stderr, Runtime{})
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got := stdout.String(); !strings.Contains(got, "DRY-RUN dashboard_item_property_set") || !strings.Contains(got, "body={\"limit\":25}") {
			t.Fatalf("stdout = %q", got)
		}
	})

	t.Run("set requires explicit intent", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"dashboard", "item", "property", "set", "10000", "20000", "issue.support", "--body", `{"limit":25}`}, &stdout, &stderr, Runtime{})
		if code != 1 || !strings.Contains(stderr.String(), "requires --dry-run or --yes") {
			t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
		}
	})

	t.Run("set yes", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"dashboard", "item", "property", "set", "10000", "20000", "issue.support", "--body", `{"panelId":9007199254740993123,"jql":"project = JCLI"}`, "--yes"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "OK dashboard_item_property_set dashboard=10000 item=20000 key=issue.support\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})

	t.Run("delete dry-run", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"dashboard", "item", "property", "delete", "10000", "20000", "issue.support", "--dry-run"}, &stdout, &stderr, Runtime{})
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got := stdout.String(); !strings.Contains(got, "DRY-RUN dashboard_item_property_delete") {
			t.Fatalf("stdout = %q", got)
		}
	})

	t.Run("delete yes", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"dashboard", "item", "property", "delete", "10000", "20000", "issue.support", "--yes"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "OK dashboard_item_property_delete dashboard=10000 item=20000 key=issue.support\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})

	if !putCalled || !deleteCalled {
		t.Fatalf("dashboard property writes missing: put=%t delete=%t", putCalled, deleteCalled)
	}
}

func TestAgileReadOnlyCommandsAgainstFakeServer(t *testing.T) {
	issuePage := `{"startAt":0,"maxResults":1,"total":1,"issues":[{"key":"JCLI-7","fields":{"summary":"Agile issue","status":{"name":"Open"},"priority":{"name":"P2"},"assignee":{"name":"jdoe"}}}]}`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/agile/1.0/board", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("startAt") == "" {
			writeJSONBody(`{"maxResults":1,"values":[{"id":42,"name":"Team Board","type":"scrum"}]}`)(w, r)
			return
		}
		writeJSONBody(`{"startAt":0,"maxResults":50,"total":1,"isLast":true,"values":[{"id":42,"name":"Team Board","type":"scrum"}]}`)(w, r)
	})
	mux.HandleFunc("/rest/agile/1.0/board/42", writeJSONBody(`{"id":42,"name":"Team Board","type":"scrum"}`))
	mux.HandleFunc("/rest/agile/1.0/board/42/issue", writeJSONBody(issuePage))
	mux.HandleFunc("/rest/agile/1.0/board/42/backlog", writeJSONBody(issuePage))
	mux.HandleFunc("/rest/agile/1.0/board/42/sprint", writeJSONBody(`{"startAt":0,"maxResults":50,"total":1,"isLast":true,"values":[{"id":7,"name":"Sprint 7","state":"active"}]}`))
	mux.HandleFunc("/rest/agile/1.0/sprint/7", writeJSONBody(`{"id":7,"name":"Sprint 7","state":"active"}`))
	mux.HandleFunc("/rest/agile/1.0/sprint/7/issue", writeJSONBody(issuePage))
	mux.HandleFunc("/rest/agile/1.0/epic/JCLI-9", writeJSONBody(`{"id":9,"key":"JCLI-9","name":"Epic Name","summary":"Epic Summary"}`))
	mux.HandleFunc("/rest/agile/1.0/epic/JCLI-9/issue", writeJSONBody(issuePage))
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "boards", args: []string{"boards"}, want: "42 scrum Team Board\n1 boards total=1\n"},
		{name: "board", args: []string{"board", "42"}, want: "42 scrum Team Board\n"},
		{name: "board issues", args: []string{"board", "issues", "42"}, want: "JCLI-7 Open P2 jdoe Agile issue\n1 issues total=1\n"},
		{name: "backlog", args: []string{"backlog", "42"}, want: "JCLI-7 Open P2 jdoe Agile issue\n1 issues total=1\n"},
		{name: "sprints", args: []string{"sprints", "42"}, want: "7 active Sprint 7\n1 sprints total=1\n"},
		{name: "sprint", args: []string{"sprint", "7"}, want: "7 active Sprint 7\n"},
		{name: "sprint issues", args: []string{"sprint", "issues", "7"}, want: "JCLI-7 Open P2 jdoe Agile issue\n1 issues total=1\n"},
		{name: "sprint summary", args: []string{"sprint", "summary", "--sprint", "7"}, want: "sprint=7 issues=1 total=1 statuses=Open:1\n"},
		{name: "epic", args: []string{"epic", "JCLI-9"}, want: "JCLI-9 Epic Name\n"},
		{name: "epic issues", args: []string{"epic", "issues", "JCLI-9"}, want: "JCLI-7 Open P2 jdoe Agile issue\n1 issues total=1\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAgileCommandsReturnMissingCapabilityWhenProbeFails(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/agile/1.0/board", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"errorMessages":["Agile is unavailable"]}`, http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"boards"}, &stdout, &stderr, runtimeForServer(server))
	if code != 5 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "ERR jira") || !strings.Contains(stderr.String(), "404") {
		t.Fatalf("stderr = %q, want Jira missing capability error", stderr.String())
	}
}

func TestAgentEfficiencyCommandsAgainstFakeServer(t *testing.T) {
	var lastJQL string
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
		lastJQL = r.URL.Query().Get("jql")
		writeJSONBody(`{"startAt":0,"maxResults":1,"total":1,"issues":[{"key":"JCLI-8","fields":{"summary":"Composed issue","status":{"name":"Open"},"priority":{"name":"P2"},"assignee":{"name":"jdoe"}}}]}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-1", writeJSONBody(`{"key":"JCLI-1","fields":{"issuelinks":[{"id":"300","type":{"name":"Blocks","outward":"blocks","inward":"is blocked by"},"outwardIssue":{"key":"JCLI-2","fields":{"summary":"Blocking issue","status":{"name":"Open"}}}}]}}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("mine", func(t *testing.T) {
		lastJQL = ""
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"mine", "--days", "3", "--limit", "1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "JCLI-8 Open P2 jdoe Composed issue") {
			t.Fatalf("stdout = %q", stdout.String())
		}
		if !strings.Contains(lastJQL, "assignee = currentUser()") || !strings.Contains(lastJQL, "updated >= -3d") {
			t.Fatalf("jql = %q", lastJQL)
		}
	})

	t.Run("stale", func(t *testing.T) {
		lastJQL = ""
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"stale", "--project", "JCLI", "--days", "14", "--status", "Open", "--limit", "1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "JCLI-8 Open P2 jdoe Composed issue") {
			t.Fatalf("stdout = %q", stdout.String())
		}
		for _, want := range []string{"project = JCLI", "resolution = EMPTY", "updated <= -14d", "status = Open"} {
			if !strings.Contains(lastJQL, want) {
				t.Fatalf("jql missing %q in %q", want, lastJQL)
			}
		}
	})

	t.Run("blockers", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"blockers", "--issue", "JCLI-1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if got, want := stdout.String(), "300 Blocks blocks JCLI-2 Open - - Blocking issue\n"; got != want {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	})
}

func TestSafeWriteCommandsAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("create method = %s", r.Method)
		}
		var body map[string]map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		fields := body["fields"]
		if fields["summary"] != "New issue" {
			t.Fatalf("create summary = %#v", fields["summary"])
		}
		components, ok := fields["components"].([]any)
		if !ok || len(components) != 1 {
			t.Fatalf("create components = %#v", fields["components"])
		}
		component, ok := components[0].(map[string]any)
		if !ok || component["id"] != "10000" {
			t.Fatalf("create component = %#v", components[0])
		}
		versions, ok := fields["versions"].([]any)
		if !ok || len(versions) != 1 {
			t.Fatalf("create versions = %#v", fields["versions"])
		}
		version, ok := versions[0].(map[string]any)
		if !ok || version["id"] != "10113" {
			t.Fatalf("create version = %#v", versions[0])
		}
		if fields["duedate"] != "2026-06-30" {
			t.Fatalf("create duedate = %#v", fields["duedate"])
		}
		priority, ok := fields["priority"].(map[string]any)
		if !ok || priority["id"] != "3" {
			t.Fatalf("create priority = %#v", fields["priority"])
		}
		writeJSONBody(`{"id":"10000","key":"JCLI-10"}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			var body map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			if body["fields"]["summary"] != "Updated" {
				t.Fatalf("update body = %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected issue method %s", r.Method)
		}
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/comment", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("comment method = %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode comment body: %v", err)
		}
		if body["body"] != "Looks good" {
			t.Fatalf("comment body = %#v", body)
		}
		writeJSONBody(`{"id":"101"}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/assignee", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("assign method = %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/transitions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSONBody(`{"transitions":[{"id":"31","name":"Done","to":{"name":"Done"}}]}`)(w, r)
		case http.MethodPost:
			var body map[string]map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode transition body: %v", err)
			}
			if body["transition"]["id"] != "31" {
				t.Fatalf("transition body = %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("transition method = %s", r.Method)
		}
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/watchers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSONBody(`{"isWatching":true,"watchCount":2,"watchers":[{"name":"agent"},{"name":"jdoe"}]}`)(w, r)
		case http.MethodPost:
			var user string
			if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
				t.Fatalf("decode watcher body: %v", err)
			}
			if user != "agent" {
				t.Fatalf("watch user = %q", user)
			}
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			if got := r.URL.Query().Get("username"); got != "agent" {
				t.Fatalf("unwatch username = %q", got)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("watchers method = %s", r.Method)
		}
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/worklog", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("worklog method = %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode worklog body: %v", err)
		}
		if body["timeSpent"] != "1h" {
			t.Fatalf("worklog body = %#v", body)
		}
		writeJSONBody(`{"id":"201"}`)(w, r)
	})
	mux.HandleFunc("/rest/api/2/issueLink", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("link create method = %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/rest/api/2/issueLink/300", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("link delete method = %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "create", args: []string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--component", "10000", "--version", "10113", "--due", "2026-06-30", "--priority", "3", "--yes"}, want: "OK create issue=JCLI-10\n"},
		{name: "update 204", args: []string{"update", "JCLI-10", "--field", "summary=Updated", "--yes"}, want: "OK update issue=JCLI-10\n"},
		{name: "comment", args: []string{"comment", "JCLI-10", "--body", "Looks good", "--yes"}, want: "OK comment id=101 issue=JCLI-10\n"},
		{name: "assign 204", args: []string{"assign", "JCLI-10", "--assignee", "jdoe", "--yes"}, want: "OK assign assignee=jdoe issue=JCLI-10\n"},
		{name: "transitions", args: []string{"transitions", "JCLI-10"}, want: "31 Done to=Done\n"},
		{name: "transition 204", args: []string{"transition", "JCLI-10", "--id", "31", "--yes"}, want: "OK transition id=31 issue=JCLI-10\n"},
		{name: "delete 204", args: []string{"delete", "JCLI-10", "--yes"}, want: "OK delete issue=JCLI-10\n"},
		{name: "watchers", args: []string{"watchers", "JCLI-10"}, want: "watching=true count=2\n"},
		{name: "watch 204", args: []string{"watch", "JCLI-10", "--yes"}, want: "OK watch issue=JCLI-10 user=agent\n"},
		{name: "unwatch 204", args: []string{"unwatch", "JCLI-10", "--yes"}, want: "OK unwatch issue=JCLI-10 user=agent\n"},
		{name: "worklog", args: []string{"worklog", "add", "JCLI-10", "--time", "1h", "--comment", "done", "--yes"}, want: "OK worklog_add id=201 issue=JCLI-10\n"},
		{name: "link create", args: []string{"link", "create", "JCLI-10", "--target", "JCLI-11", "--link-type", "Blocks", "--yes"}, want: "OK link_create source=JCLI-10 target=JCLI-11 type=Blocks\n"},
		{name: "link delete 204", args: []string{"link", "delete", "300", "--yes"}, want: "OK link_delete id=300\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if strings.Contains(stdout.String(), "null") || strings.Contains(stdout.String(), "undefined") {
				t.Fatalf("compact output leaked empty-body sentinel: %q", stdout.String())
			}
		})
	}
}

func TestWriteCommandsRequireIntentBeforeConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "create", args: []string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue"}},
		{name: "update", args: []string{"update", "JCLI-10", "--field", "summary=Updated"}},
		{name: "comment", args: []string{"comment", "JCLI-10", "--body", "Looks good"}},
		{name: "assign", args: []string{"assign", "JCLI-10", "--assignee", "jdoe"}},
		{name: "transition", args: []string{"transition", "JCLI-10", "--id", "31"}},
		{name: "watch", args: []string{"watch", "JCLI-10"}},
		{name: "unwatch", args: []string{"unwatch", "JCLI-10"}},
		{name: "worklog", args: []string{"worklog", "add", "JCLI-10", "--time", "1h"}},
		{name: "link create", args: []string{"link", "create", "JCLI-10", "--target", "JCLI-11"}},
		{name: "link delete", args: []string{"link", "delete", "300"}},
		{name: "move sprint", args: []string{"move", "sprint", "--sprint", "7", "--issue", "JCLI-10"}},
		{name: "move backlog", args: []string{"move", "backlog", "--issue", "JCLI-10"}},
		{name: "attachment add", args: []string{"attachment", "add", "JCLI-10", "--file", "/does/not/matter"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, Runtime{})
			if code != 1 || !strings.Contains(stderr.String(), "requires --dry-run or --yes") {
				t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestStructuredFieldRejectsInvalidJSONBeforeHTTP(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--field", "components=[{", "--yes"}, &stdout, &stderr, runtimeForServer(server))
	if code != 1 || !strings.Contains(stderr.String(), "object/array values must be valid JSON") {
		t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if called {
		t.Fatal("invalid structured field allowed an HTTP request")
	}
}

func TestCommonIssueFieldFlagsRejectInvalidDueBeforeHTTP(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--due", "2026/06/30", "--yes"}, &stdout, &stderr, runtimeForServer(server))
	if code != 1 || !strings.Contains(stderr.String(), "--due must be YYYY-MM-DD") {
		t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if called {
		t.Fatal("invalid due date allowed an HTTP request")
	}
}

func TestJSONReadOnlyViewsAgainstFakeServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/JCLI-1", writeJSONBody(`{"key":"JCLI-1","fields":{"summary":"First issue","issuelinks":[{"id":"301","type":{"name":"Blocks","outward":"blocks","inward":"is blocked by"},"inwardIssue":{"key":"JCLI-0","fields":{"summary":"Parent blocker","status":{"name":"Open"}}}}]}}`))
	mux.HandleFunc("/rest/api/2/user/search", writeJSONBody(`[{"name":"jdoe","key":"JDOE","emailAddress":"jdoe@example.com","displayName":"Jane Doe"}]`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/transitions", writeJSONBody(`{"transitions":[{"id":"31","name":"Done","to":{"name":"Done"}}]}`))
	mux.HandleFunc("/rest/api/2/issue/JCLI-1/watchers", writeJSONBody(`{"isWatching":true,"watchCount":1,"watchers":[{"name":"jdoe","key":"JDOE"}]}`))
	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("links", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "links", "JCLI-1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK    bool   `json:"ok"`
			Kind  string `json:"kind"`
			Issue string `json:"issue"`
			Links []struct {
				ID        string `json:"id"`
				Direction string `json:"direction"`
				Issue     struct {
					Key     string `json:"key"`
					Summary string `json:"summary"`
				} `json:"issue"`
			} `json:"links"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode links JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || body.Kind != "links" || body.Issue != "JCLI-1" || len(body.Links) != 1 || body.Links[0].Direction != "is blocked by" || body.Links[0].Issue.Key != "JCLI-0" {
			t.Fatalf("unexpected links body: %#v", body)
		}
	})

	t.Run("users search", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "users", "search", "--query", "jdoe"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK    bool   `json:"ok"`
			Kind  string `json:"kind"`
			Users []struct {
				Name         string `json:"name"`
				Key          string `json:"key"`
				EmailAddress string `json:"emailAddress"`
			} `json:"users"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode users JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || body.Kind != "users_search" || len(body.Users) != 1 || body.Users[0].Name != "jdoe" || body.Users[0].Key != "JDOE" {
			t.Fatalf("unexpected users body: %#v", body)
		}
	})

	t.Run("transitions", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "transitions", "JCLI-1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK          bool   `json:"ok"`
			Kind        string `json:"kind"`
			Transitions []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				To   string `json:"to"`
			} `json:"transitions"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode transitions JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || body.Kind != "transitions" || len(body.Transitions) != 1 || body.Transitions[0].ID != "31" || body.Transitions[0].To != "Done" {
			t.Fatalf("unexpected transitions body: %#v", body)
		}
	})

	t.Run("watchers", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--json", "watchers", "JCLI-1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var body struct {
			OK       bool   `json:"ok"`
			Kind     string `json:"kind"`
			Watchers struct {
				IsWatching bool `json:"isWatching"`
				WatchCount int  `json:"watchCount"`
				Watchers   []struct {
					Name string `json:"name"`
					Key  string `json:"key"`
				} `json:"watchers"`
			} `json:"watchers"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("decode watchers JSON: %v; stdout=%q", err, stdout.String())
		}
		if !body.OK || body.Kind != "watchers" || !body.Watchers.IsWatching || body.Watchers.WatchCount != 1 || len(body.Watchers.Watchers) != 1 || body.Watchers.Watchers[0].Key != "JDOE" {
			t.Fatalf("unexpected watchers body: %#v", body)
		}
	})
}

func TestTransitionByNameAgainstFakeServer(t *testing.T) {
	var posted bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/transitions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSONBody(`{"transitions":[{"id":"31","name":"Done","to":{"name":"Done"}}]}`)(w, r)
		case http.MethodPost:
			posted = true
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode transition body: %v", err)
			}
			transition, ok := body["transition"].(map[string]any)
			if !ok || transition["id"] != "31" {
				t.Fatalf("transition payload = %#v", body)
			}
			update, ok := body["update"].(map[string]any)
			if !ok {
				t.Fatalf("missing comment update in payload %#v", body)
			}
			comments, ok := update["comment"].([]any)
			if !ok || len(comments) != 1 {
				t.Fatalf("comment update = %#v", update["comment"])
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("transition method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"transition", "JCLI-10", "--name", "Done", "--comment", "closing", "--yes"}, &stdout, &stderr, runtimeForServer(server))
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if !posted {
		t.Fatal("transition POST was not called after resolving name")
	}
	if got, want := stdout.String(), "OK transition id=31 issue=JCLI-10\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestTransitionAmbiguousNameDoesNotPost(t *testing.T) {
	var posted bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue/JCLI-10/transitions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posted = true
			t.Fatal("ambiguous transition name must not post")
		}
		writeJSONBody(`{"transitions":[{"id":"31","name":"Done"},{"id":"41","name":"Done"}]}`)(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"transition", "JCLI-10", "--name", "Done", "--yes"}, &stdout, &stderr, runtimeForServer(server))
	if code != 1 {
		t.Fatalf("code = %d, stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if posted || !strings.Contains(stderr.String(), "ambiguous") {
		t.Fatalf("posted=%t stderr=%q", posted, stderr.String())
	}
}

func TestAgileMoveYesAgainstFakeServer(t *testing.T) {
	var sprintMoved, backlogMoved bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/agile/1.0/board", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Get("maxResults") != "1" {
			t.Fatalf("agile probe request = %s %s", r.Method, r.URL.String())
		}
		writeJSONBody(`{"maxResults":1,"values":[]}`)(w, r)
	})
	mux.HandleFunc("/rest/agile/1.0/sprint/7/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("move sprint method = %s", r.Method)
		}
		var body struct {
			Issues []string `json:"issues"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode move sprint body: %v", err)
		}
		if got, want := strings.Join(body.Issues, ","), "JCLI-10,JCLI-11"; got != want {
			t.Fatalf("move sprint issues = %q, want %q", got, want)
		}
		sprintMoved = true
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/rest/agile/1.0/backlog/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("move backlog method = %s", r.Method)
		}
		var body struct {
			Issues []string `json:"issues"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode move backlog body: %v", err)
		}
		if got, want := strings.Join(body.Issues, ","), "JCLI-12"; got != want {
			t.Fatalf("move backlog issues = %q, want %q", got, want)
		}
		backlogMoved = true
		w.WriteHeader(http.StatusNoContent)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "move sprint", args: []string{"move", "sprint", "--sprint", "7", "--issue", "JCLI-10", "JCLI-11", "--yes"}, want: "OK move_sprint issues=[JCLI-10 JCLI-11] sprint=7\n"},
		{name: "move backlog", args: []string{"move", "backlog", "--issue", "JCLI-12", "--yes"}, want: "OK move_backlog issues=[JCLI-12]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
		})
	}
	if !sprintMoved || !backlogMoved {
		t.Fatalf("move calls missing: sprint=%t backlog=%t", sprintMoved, backlogMoved)
	}
}

func TestWriteErrorsSuggestMetadataProbe(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/2/issue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("create method = %s", r.Method)
		}
		http.Error(w, `{"errorMessages":["Field customfield_10016 cannot be set"]}`, http.StatusBadRequest)
	})
	mux.HandleFunc("/rest/api/2/issue/JCLI-10", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("update method = %s", r.Method)
		}
		http.Error(w, `{"errors":{"customfield_10016":"Field cannot be set"}}`, http.StatusBadRequest)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "create", args: []string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--field", "customfield_10016=3", "--yes"}, want: "createmeta"},
		{name: "update", args: []string{"update", "JCLI-10", "--field", "customfield_10016=3", "--yes"}, want: "editmeta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, runtimeForServer(server))
			if code != 3 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "HINT") || !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr = %q, want hint for %s", stderr.String(), tt.want)
			}
		})
	}
}

func TestDestructiveCommandGuards(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()

	tests := [][]string{
		{"delete", "JCLI-10"},
		{"delete", "*", "--yes"},
		{"link", "delete", "300"},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime(args, &stdout, &stderr, runtimeForServer(server))
		if code == 0 {
			t.Fatalf("MainWithRuntime(%#v) succeeded unexpectedly", args)
		}
	}
	if called {
		t.Fatal("destructive guard allowed an HTTP request")
	}
}

func TestExtraPositionalsRejectedBeforeExecution(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))
	defer server.Close()
	rt := runtimeForServer(server)

	tests := []struct {
		name string
		args []string
	}{
		{name: "read issue", args: []string{"issue", "JCLI-1", "EXTRA"}},
		{name: "update", args: []string{"update", "JCLI-1", "EXTRA", "--field", "summary=Updated", "--yes"}},
		{name: "comment", args: []string{"comment", "JCLI-1", "EXTRA", "--body", "ignored", "--yes"}},
		{name: "delete", args: []string{"delete", "JCLI-1", "EXTRA", "--yes"}},
		{name: "link delete", args: []string{"link", "delete", "300", "EXTRA", "--yes"}},
		{name: "attachment delete", args: []string{"attachment", "delete", "700", "EXTRA", "--yes"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, rt)
			if code != 1 || !strings.Contains(stderr.String(), "ERR usage") {
				t.Fatalf("code = %d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
	if called {
		t.Fatal("extra positional validation allowed an HTTP request")
	}
}

func TestCommandExitCodes(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"version"}, &stdout, &stderr, Runtime{})
		if code != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("usage", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"--limit", "0", "search", "x"}, &stdout, &stderr, Runtime{})
		if code != 1 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("auth", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"errorMessages":["denied"]}`, http.StatusForbidden)
		}))
		defer server.Close()
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"whoami"}, &stdout, &stderr, runtimeForServer(server))
		if code != 2 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("application", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"errorMessages":["bad jql"]}`, http.StatusBadRequest)
		}))
		defer server.Close()
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"search", "bad"}, &stdout, &stderr, runtimeForServer(server))
		if code != 3 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("missing issue is application error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"errorMessages":["issue does not exist"]}`, http.StatusNotFound)
		}))
		defer server.Close()
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"issue", "JCLI-404"}, &stdout, &stderr, runtimeForServer(server))
		if code != 3 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("network", func(t *testing.T) {
		rt := Runtime{
			Env: map[string]string{
				"JIRA_BASE_URL":  "https://jira.example.com",
				"JIRA_USER":      "agent",
				"JIRA_API_TOKEN": "secret",
			},
			HTTPClient: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("network down")
			})},
		}
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"whoami"}, &stdout, &stderr, rt)
		if code != 4 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("missing capability", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"errorMessages":["missing"]}`, http.StatusNotFound)
		}))
		defer server.Close()
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"boards"}, &stdout, &stderr, runtimeForServer(server))
		if code != 5 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
	})

	t.Run("partial batch", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("/rest/api/2/search", func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Query().Get("jql") {
			case "good":
				writeJSONBody(`{"startAt":0,"maxResults":1,"total":1,"issues":[{"key":"JCLI-1","fields":{"summary":"Good","status":{"name":"Open"},"priority":{"name":"P2"},"assignee":{"name":"jdoe"}}}]}`)(w, r)
			case "bad":
				http.Error(w, `{"errorMessages":["bad jql"]}`, http.StatusBadRequest)
			default:
				t.Fatalf("unexpected jql %q", r.URL.Query().Get("jql"))
			}
		})
		server := httptest.NewServer(mux)
		defer server.Close()
		var stdout, stderr bytes.Buffer
		code := MainWithRuntime([]string{"bulk", "search", "good", "bad", "--limit", "1"}, &stdout, &stderr, runtimeForServer(server))
		if code != 6 {
			t.Fatalf("code = %d, stderr = %q stdout = %q", code, stderr.String(), stdout.String())
		}
		out := stdout.String()
		for _, want := range []string{"query=1 ok", "JCLI-1 Open P2 jdoe Good", "query=2 err"} {
			if !strings.Contains(out, want) {
				t.Fatalf("stdout missing %q in %q", want, out)
			}
		}
	})
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestWriteDryRunDoesNotRequireJiraConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "create", args: []string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--field", `components=[{"id":"10000"}]`, "--dry-run"}, want: `"components":[{"id":"10000"}]`},
		{name: "create attach", args: []string{"create", "--project", "JCLI", "--issue-type", "Bug", "--summary", "New issue", "--attach", "/tmp/proof.txt", "--dry-run"}, want: "attachments=[/tmp/proof.txt]"},
		{name: "update", args: []string{"update", "JCLI-10", "--field", "summary=Updated", "--dry-run"}, want: "DRY-RUN update"},
		{name: "update due", args: []string{"update", "JCLI-10", "--due", "2026-06-30", "--dry-run"}, want: `"duedate":"2026-06-30"`},
		{name: "link delete", args: []string{"link", "delete", "300", "--dry-run"}, want: "DRY-RUN link_delete id=300"},
		{name: "move sprint", args: []string{"move", "sprint", "--sprint", "7", "--issue", "JCLI-10", "--dry-run"}, want: "DRY-RUN move_sprint issues=[JCLI-10] sprint=7"},
		{name: "move backlog", args: []string{"move", "backlog", "--issue", "JCLI-10", "--issue", "JCLI-11", "--dry-run"}, want: "DRY-RUN move_backlog issues=[JCLI-10 JCLI-11]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := MainWithRuntime(tt.args, &stdout, &stderr, Runtime{})
			if code != 0 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if got := stdout.String(); !strings.Contains(got, tt.want) || strings.Contains(got, "secret") {
				t.Fatalf("stdout = %q", got)
			}
		})
	}
}

func TestConfigDoctorDoesNotPrintSecret(t *testing.T) {
	rt := Runtime{Env: map[string]string{
		"JIRA_BASE_URL":  "https://jira.example.com",
		"JIRA_USER":      "agent",
		"JIRA_API_TOKEN": "super-secret",
	}}

	var stdout, stderr bytes.Buffer
	code := MainWithRuntime([]string{"config", "doctor"}, &stdout, &stderr, rt)
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "OK config") {
		t.Fatalf("stdout = %q, want OK config", out)
	}
	if strings.Contains(out, "super-secret") || strings.Contains(stderr.String(), "super-secret") {
		t.Fatalf("config doctor leaked secret: stdout=%q stderr=%q", out, stderr.String())
	}
}

func runtimeForServer(server *httptest.Server) Runtime {
	return Runtime{
		Env: map[string]string{
			"JIRA_BASE_URL":  server.URL,
			"JIRA_USER":      "agent",
			"JIRA_API_TOKEN": "secret",
		},
		HTTPClient: server.Client(),
	}
}

func writeJSONBody(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}
}
