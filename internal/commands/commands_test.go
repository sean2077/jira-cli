package commands

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sean2077/jira-cli/internal/jira"
)

func TestSafeAttachmentContentURL(t *testing.T) {
	tests := []struct {
		name    string
		base    string
		content string
		want    string
		wantErr string
	}{
		{name: "relative on host", base: "https://jira.example.com", content: "/secure/attachment/700/p.txt", want: "https://jira.example.com/secure/attachment/700/p.txt"},
		{name: "absolute on host", base: "https://jira.example.com", content: "https://jira.example.com/secure/x", want: "https://jira.example.com/secure/x"},
		{name: "empty", base: "https://jira.example.com", content: "", wantErr: "content URL"},
		{name: "cross host", base: "https://jira.example.com", content: "https://evil.example.com/x", wantErr: "stay on the Jira host"},
		{name: "scheme downgrade", base: "https://jira.example.com", content: "http://jira.example.com/x", wantErr: "stay on the Jira host"},
		{name: "credentials", base: "https://jira.example.com", content: "https://u:p@jira.example.com/x", wantErr: "must not include credentials"},
		{name: "fragment", base: "https://jira.example.com", content: "https://jira.example.com/x#f", wantErr: "must not include a fragment"},
		{name: "base path escape", base: "https://jira.example.com/jira", content: "https://jira.example.com/other", wantErr: "stay under the Jira base path"},
		{name: "base path ok", base: "https://jira.example.com/jira", content: "https://jira.example.com/jira/secure/x", want: "https://jira.example.com/jira/secure/x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := safeAttachmentContentURL(tt.base, tt.content)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePublicRESTPathAccepts(t *testing.T) {
	tests := []struct {
		in       string
		wantAPI  jira.API
		wantPath string
	}{
		{"/rest/api/2/search", jira.PlatformAPI, "/rest/api/2/search"},
		{"search", jira.PlatformAPI, "/rest/api/2/search"},
		{"api/2/issue/JCLI-1", jira.PlatformAPI, "/rest/api/2/issue/JCLI-1"},
		{"rest/agile/1.0/board", jira.AgileAPI, "/rest/agile/1.0/board"},
		{"agile/1.0/board/5/sprint", jira.AgileAPI, "/rest/agile/1.0/board/5/sprint"},
	}
	for _, tt := range tests {
		api, _, path, err := parsePublicRESTPath(tt.in)
		if err != nil {
			t.Fatalf("%q: unexpected err %v", tt.in, err)
		}
		if api != tt.wantAPI || path != tt.wantPath {
			t.Fatalf("%q -> api=%v path=%q, want api=%v path=%q", tt.in, api, path, tt.wantAPI, tt.wantPath)
		}
	}
}

func TestParsePublicRESTPathRejects(t *testing.T) {
	tests := []struct{ in, wantErr string }{
		{"rest/api/latest/search", "/rest/api/2"},
		{"api/latest/x", "api/2"},
		{"rest/auth/1/session", "auth-session"},
		{"issue/%2f/x", "encoded slash"},
		{"rest/api/2/../../etc", "unsafe api path segment"},
		{"http://evil.example.com/x", "absolute URL"},
		{"//evil.example.com/x", "scheme-relative"},
		{"issue?expand=x", "query or fragment"},
		{"issue#frag", "query or fragment"},
		{"rest/foo", "under /rest/api/2"},
		{"", "api path is required"},
	}
	for _, tt := range tests {
		if _, _, _, err := parsePublicRESTPath(tt.in); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Fatalf("%q: err = %v, want containing %q", tt.in, err, tt.wantErr)
		}
	}
}

func TestValidateAPIPassThroughEndpoint(t *testing.T) {
	if err := validateAPIPassThroughEndpoint(jira.AgileAPI, http.MethodPost, []string{"sprint", "7", "issue"}, false); err == nil || !strings.Contains(err.Error(), "require --force") {
		t.Fatalf("agile write without force = %v, want force requirement", err)
	}
	if err := validateAPIPassThroughEndpoint(jira.AgileAPI, http.MethodPost, []string{"sprint", "7", "issue"}, true); err != nil {
		t.Fatalf("agile write with force = %v, want allowed", err)
	}
	if err := validateAPIPassThroughEndpoint(jira.PlatformAPI, http.MethodGet, []string{"attachment", "700"}, false); err == nil || !strings.Contains(err.Error(), "attachment") {
		t.Fatalf("attachment get = %v, want rejection", err)
	}
	if err := validateAPIPassThroughEndpoint(jira.PlatformAPI, http.MethodPost, []string{"issue", "JCLI-1", "attachments"}, false); err == nil || !strings.Contains(err.Error(), "attachment upload") {
		t.Fatalf("attachment upload = %v, want rejection", err)
	}
	if err := validateAPIPassThroughEndpoint(jira.PlatformAPI, http.MethodPost, []string{"user", "avatar", "temporary"}, false); err == nil || !strings.Contains(err.Error(), "avatar") {
		t.Fatalf("avatar upload = %v, want rejection", err)
	}
	if err := validateAPIPassThroughEndpoint(jira.PlatformAPI, http.MethodGet, []string{"search"}, false); err != nil {
		t.Fatalf("platform get = %v, want allowed", err)
	}
}

func TestValidIssueKey(t *testing.T) {
	for _, k := range []string{"JCLI-1", "ABC-123", "A1-9"} {
		if !validIssueKey(k) {
			t.Fatalf("validIssueKey(%q) = false, want true", k)
		}
	}
	for _, k := range []string{"", "JCLI", "jcli-1", "JCLI-", "-1", "JCLI-1a", "JCLI 1", "*", "JCLI-1 "} {
		if validIssueKey(k) {
			t.Fatalf("validIssueKey(%q) = true, want false", k)
		}
	}
}

func TestJQLValue(t *testing.T) {
	cases := map[string]string{
		"OPEN":        "OPEN",
		"In Progress": `"In Progress"`,
		`a"b`:         `"a\"b"`,
		"":            `""`,
	}
	for in, want := range cases {
		if got := jqlValue(in); got != want {
			t.Fatalf("jqlValue(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCollectPagedAccumulatesToTotal(t *testing.T) {
	pages := [][]int{{1, 2}, {3, 4}, {5}}
	calls := 0
	items, page, err := collectPaged(0, 2, 10, func(start, size int) ([]int, int, bool, error) {
		idx := start / 2
		if idx >= len(pages) {
			return nil, 5, false, nil
		}
		calls++
		return pages[idx], 5, false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
	if len(items) != 5 {
		t.Fatalf("items = %v, want 5 elements", items)
	}
	if page.Total != 5 || page.NextStartAt != 0 {
		t.Fatalf("page = %+v, want Total=5 NextStartAt=0", page)
	}
}

func TestCollectPagedBareArrayStopsOnShortPage(t *testing.T) {
	pages := [][]int{{1, 2}, {3}}
	items, page, err := collectPaged(0, 2, 100, func(start, size int) ([]int, int, bool, error) {
		idx := start / 2
		if idx >= len(pages) {
			return nil, -1, false, nil
		}
		return pages[idx], -1, false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %v, want 3 elements", items)
	}
	if page.NextStartAt != 0 {
		t.Fatalf("NextStartAt = %d, want 0", page.NextStartAt)
	}
}

func TestCollectPagedHonorsLimitAndEmitsNext(t *testing.T) {
	items, page, err := collectPaged(0, 10, 25, func(start, size int) ([]int, int, bool, error) {
		return make([]int, size), 100, false, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 25 {
		t.Fatalf("items = %d, want 25", len(items))
	}
	if page.NextStartAt != 25 {
		t.Fatalf("NextStartAt = %d, want 25", page.NextStartAt)
	}
}
