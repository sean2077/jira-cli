package jira

import (
	"net/url"
	"testing"
)

func TestBuildURLPlatform(t *testing.T) {
	got, err := BuildURL("https://jira.example.com/jira/", PlatformAPI, "issue", "JCLI-123")
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}
	want := "https://jira.example.com/jira/rest/api/2/issue/JCLI-123"
	if got != want {
		t.Fatalf("BuildURL = %q, want %q", got, want)
	}
}

func TestBuildURLAgile(t *testing.T) {
	got, err := BuildURL("https://jira.example.com", AgileAPI, "board", "42", "issue")
	if err != nil {
		t.Fatalf("BuildURL returned error: %v", err)
	}
	want := "https://jira.example.com/rest/agile/1.0/board/42/issue"
	if got != want {
		t.Fatalf("BuildURL = %q, want %q", got, want)
	}
}

func TestBuildURLRejectsInvalidBase(t *testing.T) {
	if _, err := BuildURL("jira.example.com", PlatformAPI); err == nil {
		t.Fatal("expected invalid base URL error")
	}
}

func TestBuildURLRejectsUnsafePathSegments(t *testing.T) {
	for _, segment := range []string{"..", ".", "", "JCLI-1/../../project", `JCLI-1\\evil`} {
		if _, err := BuildURL("https://jira.example.com", PlatformAPI, "issue", segment); err == nil {
			t.Fatalf("BuildURL accepted unsafe segment %q", segment)
		}
	}
}

func TestAddQuery(t *testing.T) {
	raw, err := AddQuery("https://jira.example.com/rest/api/2/search", url.Values{
		"jql":        {"project = JCLI"},
		"maxResults": {"10"},
	})
	if err != nil {
		t.Fatalf("AddQuery returned error: %v", err)
	}
	want := "https://jira.example.com/rest/api/2/search?jql=project+%3D+JCLI&maxResults=10"
	if raw != want {
		t.Fatalf("AddQuery = %q, want %q", raw, want)
	}
}

func TestBasicAuthHeader(t *testing.T) {
	got, err := BasicAuthHeader("jdoe", "secret")
	if err != nil {
		t.Fatalf("BasicAuthHeader returned error: %v", err)
	}
	want := "Basic amRvZTpzZWNyZXQ="
	if got != want {
		t.Fatalf("BasicAuthHeader = %q, want %q", got, want)
	}
}
