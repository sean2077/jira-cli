package probe

import (
	"context"
	"net/url"

	"github.com/sean2077/jira-cli/internal/config"
	"github.com/sean2077/jira-cli/internal/jira"
)

type Result struct {
	OK            bool     `json:"ok"`
	Kind          string   `json:"kind"`
	BaseURL       string   `json:"baseUrl"`
	API           string   `json:"api"`
	ServerVersion string   `json:"serverVersion,omitempty"`
	User          string   `json:"user,omitempty"`
	Platform      bool     `json:"platform"`
	Agile         string   `json:"agile"`
	Search        bool     `json:"search"`
	Projects      bool     `json:"projects"`
	Fields        bool     `json:"fields"`
	Dashboards    bool     `json:"dashboards"`
	CreateMeta    bool     `json:"createMeta,omitempty"`
	EditMeta      bool     `json:"editMeta,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

type Options struct {
	ProjectKey    string
	IssueTypeName string
	IssueKey      string
}

func Run(ctx context.Context, client jira.Client, opts Options) (Result, error) {
	result := Result{
		OK:      true,
		Kind:    "probe",
		BaseURL: client.BaseURL,
		API:     "/rest/api/2",
		Agile:   "unavailable",
	}

	var server jira.ServerInfo
	if _, err := client.Get(ctx, jira.PlatformAPI, []string{"serverInfo"}, nil, &server); err != nil {
		return result, err
	}
	result.ServerVersion = server.Version
	result.Platform = true

	var user jira.User
	if _, err := client.Get(ctx, jira.PlatformAPI, []string{"myself"}, nil, &user); err != nil {
		return result, err
	}
	result.User = firstNonEmpty(user.Name, user.Key, user.DisplayName)

	result.Search = endpoint(ctx, client, jira.PlatformAPI, []string{"search"}, url.Values{"jql": {""}, "maxResults": {"0"}})
	result.Projects = endpoint(ctx, client, jira.PlatformAPI, []string{"project"}, nil)
	result.Fields = endpoint(ctx, client, jira.PlatformAPI, []string{"field"}, nil)
	result.Dashboards = endpoint(ctx, client, jira.PlatformAPI, []string{"dashboard"}, url.Values{"maxResults": {"1"}})
	if !result.Search {
		result.Warnings = append(result.Warnings, "search unavailable")
	}
	if !result.Projects {
		result.Warnings = append(result.Warnings, "projects unavailable")
	}
	if !result.Fields {
		result.Warnings = append(result.Warnings, "fields unavailable")
	}
	if !result.Dashboards {
		result.Warnings = append(result.Warnings, "dashboards unavailable")
	}
	if opts.ProjectKey != "" && opts.IssueTypeName != "" {
		result.CreateMeta = endpoint(ctx, client, jira.PlatformAPI, []string{"issue", "createmeta"}, url.Values{
			"projectKeys":    {opts.ProjectKey},
			"issuetypeNames": {opts.IssueTypeName},
			"expand":         {"projects.issuetypes.fields"},
		})
		if !result.CreateMeta {
			result.Warnings = append(result.Warnings, "createmeta unavailable")
		}
	}
	if opts.IssueKey != "" {
		result.EditMeta = endpoint(ctx, client, jira.PlatformAPI, []string{"issue", opts.IssueKey, "editmeta"}, nil)
		if !result.EditMeta {
			result.Warnings = append(result.Warnings, "editmeta unavailable")
		}
	}
	if endpoint(ctx, client, jira.AgileAPI, []string{"board"}, url.Values{"maxResults": {"1"}}) {
		result.Agile = "available"
	}
	// OK reflects whether every probed platform capability is available, so a
	// consumer checking .ok detects a degraded server instead of always seeing
	// true. Agile is optional and reported separately, so it does not flip OK.
	result.OK = len(result.Warnings) == 0
	return result, nil
}

func endpoint(ctx context.Context, client jira.Client, api jira.API, segments []string, query url.Values) bool {
	_, err := client.Get(ctx, api, segments, query, nil)
	return err == nil
}

// firstNonEmpty aliases config.FirstNonEmpty for a single shared implementation.
var firstNonEmpty = config.FirstNonEmpty
