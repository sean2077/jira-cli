package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sean2077/jira-cli/internal/config"
	"github.com/sean2077/jira-cli/internal/jira"
	"github.com/sean2077/jira-cli/internal/output"
	"github.com/sean2077/jira-cli/internal/probe"
	"github.com/sean2077/jira-cli/internal/skillinstall"
)

type Options struct {
	Profile  string
	BaseURL  string
	Type     string
	User     string
	TokenEnv string

	JSON    bool
	Raw     bool
	DryRun  bool
	Yes     bool
	Compact bool

	PageSize int
	Limit    int
	StartAt  int
	Timeout  time.Duration

	Project     string
	IssueType   string
	Issues      []string
	Summary     string
	Body        string
	BodySet     bool
	Fields      []string
	Components  []string
	Versions    []string
	Due         string
	Priority    string
	Attachments []string
	Assignee    string
	ID          string
	Name        string
	Comment     string
	Time        string
	Target      string
	LinkType    string
	Queries     []string
	Days        string
	Status      string
	Sprint      string
	Filter      string
	File        string

	Force  bool
	Global bool
	Meta   bool
}

type Runtime struct {
	Env              map[string]string
	Profiles         config.ProfileFile
	ProfileLoadError error
	HTTPClient       *http.Client
}

type Action func(context.Context, Options, []string, io.Writer, io.Writer, Runtime, output.Mode) int

func RunConfigDoctor(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runConfig(opts, []string{"doctor"}, stdout, stderr, rt)
}

func RunSkillInstall(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runSkill(opts, []string{"install"}, stdout, stderr, rt, mode)
}

func RunProbe(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runProbe(ctx, opts, stdout, stderr, rt, mode)
}

func RunWhoami(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWhoami(ctx, opts, stdout, stderr, rt, mode)
}

func RunAPIGet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAPIPassThrough(ctx, opts, []string{"get", args[0]}, stdout, stderr, rt, mode)
}

func RunAPIPost(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAPIPassThrough(ctx, opts, []string{"post", args[0]}, stdout, stderr, rt, mode)
}

func RunAPIPut(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAPIPassThrough(ctx, opts, []string{"put", args[0]}, stdout, stderr, rt, mode)
}

func RunAPIDelete(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAPIPassThrough(ctx, opts, []string{"delete", args[0]}, stdout, stderr, rt, mode)
}

func RunSearch(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runSearch(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunIssue(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runIssue(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunIssuePropertyKeys(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runIssuePropertyKeys(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunIssuePropertyGet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runIssuePropertyGet(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunIssuePropertySet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runIssuePropertySet(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunIssuePropertyDelete(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runIssuePropertyDelete(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunComments(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runComments(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunCommentAdd(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runComment(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunCommentGet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runCommentGet(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunWorklogs(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWorklogs(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunWorklogAdd(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWorklogAdd(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunWorklogGet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWorklogGet(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunAttachments(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAttachments(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunAttachmentGet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAttachmentGet(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunAttachmentAdd(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAttachmentAdd(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunAttachmentDownload(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAttachmentDownload(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunAttachmentDelete(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAttachmentDelete(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunLinks(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runLinks(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunLinkCreate(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runLinkCreate(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunLinkDelete(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runLinkDelete(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunRemoteLinks(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runRemoteLinks(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunRemoteLink(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runRemoteLink(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunProjects(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runProjects(ctx, opts, stdout, stderr, rt, mode)
}

func RunProject(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runProject(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunComponents(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runProjectNamedList(ctx, opts, args[0], "components", "components", stdout, stderr, rt, mode)
}

func RunVersions(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runProjectVersions(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunRoles(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runProjectRoles(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunRole(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runRawGet(ctx, opts, "role", []string{"project", args[0], "role", args[1]}, nil, stdout, stderr, rt, mode, compactJSONBodyLine)
}

func RunProjectStatuses(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runRawGet(ctx, opts, "project_statuses", []string{"project", args[0], "statuses"}, nil, stdout, stderr, rt, mode, compactJSONArrayCount("project-statuses"))
}

func RunFields(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runFields(ctx, opts, stdout, stderr, rt, mode)
}

func RunUsersSearch(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runUsersSearch(ctx, opts, stdout, stderr, rt, mode)
}

func RunAssignableUsers(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAssignableUsers(ctx, opts, stdout, stderr, rt, mode)
}

func RunIssueTypes(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runNamedList(ctx, opts, stdout, stderr, rt, mode, "issuetypes", "issuetype")
}

func RunPriorities(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runNamedList(ctx, opts, stdout, stderr, rt, mode, "priorities", "priority")
}

func RunStatuses(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runNamedList(ctx, opts, stdout, stderr, rt, mode, "statuses", "status")
}

func RunResolutions(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runNamedList(ctx, opts, stdout, stderr, rt, mode, "resolutions", "resolution")
}

func RunResolution(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runNamedGet(ctx, opts, args[0], stdout, stderr, rt, mode, "resolution", "resolution")
}

func RunWorkflows(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runNamedList(ctx, opts, stdout, stderr, rt, mode, "workflows", "workflow")
}

func RunFilters(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runFilters(ctx, opts, stdout, stderr, rt, mode)
}

func RunFilter(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runFilter(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunPermissions(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runRawGet(ctx, opts, "permissions", []string{"permissions"}, nil, stdout, stderr, rt, mode, compactJSONBodyLine)
}

func RunMyPermissions(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runMyPermissions(ctx, opts, stdout, stderr, rt, mode)
}

func RunCreate(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runCreate(ctx, opts, stdout, stderr, rt, mode)
}

func RunUpdate(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runUpdate(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunAssign(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAssign(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunTransitions(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runTransitions(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunTransition(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runTransition(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunDelete(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDelete(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunWatchers(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWatchers(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunWatch(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWatch(ctx, opts, args[0], stdout, stderr, rt, mode, true)
}

func RunUnwatch(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runWatch(ctx, opts, args[0], stdout, stderr, rt, mode, false)
}

func RunBulkSearch(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runBulkSearch(ctx, opts, args, stdout, stderr, rt, mode)
}

func RunMoveSprint(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runMoveSprint(ctx, opts, args, stdout, stderr, rt, mode)
}

func RunMoveBacklog(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runMoveBacklog(ctx, opts, args, stdout, stderr, rt, mode)
}

func RunMine(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runMine(ctx, opts, stdout, stderr, rt, mode)
}

func RunStale(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runStale(ctx, opts, stdout, stderr, rt, mode)
}

func RunBlockers(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runBlockers(ctx, opts, stdout, stderr, rt, mode)
}

func RunDashboards(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDashboards(ctx, opts, stdout, stderr, rt, mode)
}

func RunDashboard(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDashboard(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunDashboardItemPropertyKeys(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDashboardItemPropertyKeys(ctx, opts, args[0], args[1], stdout, stderr, rt, mode)
}

func RunDashboardItemPropertyGet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDashboardItemPropertyGet(ctx, opts, args[0], args[1], args[2], stdout, stderr, rt, mode)
}

func RunDashboardItemPropertySet(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDashboardItemPropertySet(ctx, opts, args[0], args[1], args[2], stdout, stderr, rt, mode)
}

func RunDashboardItemPropertyDelete(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runDashboardItemPropertyDelete(ctx, opts, args[0], args[1], args[2], stdout, stderr, rt, mode)
}

func RunBoards(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileList[jira.Board](ctx, opts, stdout, stderr, rt, mode, "boards", []string{"board"}, compactBoard, func(board jira.Board) any {
		return toBoardView(board)
	})
}

func RunBoard(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileBoard(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunBoardIssues(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileIssues(ctx, opts, stdout, stderr, rt, mode, "board_issues", []string{"board", args[0], "issue"})
}

func RunBacklog(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileIssues(ctx, opts, stdout, stderr, rt, mode, "backlog", []string{"board", args[0], "backlog"})
}

func RunSprints(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileList[jira.Sprint](ctx, opts, stdout, stderr, rt, mode, "sprints", []string{"board", args[0], "sprint"}, compactSprint, func(sprint jira.Sprint) any {
		return toSprintView(sprint)
	})
}

func RunSprint(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runSprint(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunSprintIssues(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileIssues(ctx, opts, stdout, stderr, rt, mode, "sprint_issues", []string{"sprint", args[0], "issue"})
}

func RunSprintSummary(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runSprintSummary(ctx, opts, args, stdout, stderr, rt, mode)
}

func RunEpic(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runEpic(ctx, opts, args[0], stdout, stderr, rt, mode)
}

func RunEpicIssues(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runAgileIssues(ctx, opts, stdout, stderr, rt, mode, "epic_issues", []string{"epic", args[0], "issue"})
}

type userView struct {
	Name         string `json:"name,omitempty"`
	Key          string `json:"key,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
}

type namedValueView struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type projectView struct {
	ID   string    `json:"id,omitempty"`
	Key  string    `json:"key,omitempty"`
	Name string    `json:"name,omitempty"`
	Lead *userView `json:"lead,omitempty"`
}

type fieldView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Custom bool   `json:"custom"`
}

type issueView struct {
	ID          string `json:"id,omitempty"`
	Key         string `json:"key"`
	Summary     string `json:"summary,omitempty"`
	Status      string `json:"status,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Assignee    string `json:"assignee,omitempty"`
	Project     string `json:"project,omitempty"`
	IssueType   string `json:"issueType,omitempty"`
	Created     string `json:"created,omitempty"`
	Updated     string `json:"updated,omitempty"`
	Description string `json:"description,omitempty"`
}

type issueLinkView struct {
	ID        string     `json:"id,omitempty"`
	Type      string     `json:"type,omitempty"`
	Direction string     `json:"direction,omitempty"`
	Issue     *issueView `json:"issue,omitempty"`
}

type transitionView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   string `json:"to,omitempty"`
}

type watchersView struct {
	IsWatching bool       `json:"isWatching"`
	WatchCount int        `json:"watchCount"`
	Watchers   []userView `json:"watchers,omitempty"`
}

type boardView struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

type sprintView struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	State string `json:"state,omitempty"`
}

type epicView struct {
	ID      int    `json:"id,omitempty"`
	Key     string `json:"key,omitempty"`
	Name    string `json:"name,omitempty"`
	Summary string `json:"summary,omitempty"`
	Done    bool   `json:"done,omitempty"`
}

type dashboardView struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Self string `json:"self,omitempty"`
	View string `json:"view,omitempty"`
}

type entityPropertyKeyView struct {
	Self string `json:"self,omitempty"`
	Key  string `json:"key"`
}

type entityPropertyView struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type commentView struct {
	ID      string   `json:"id,omitempty"`
	Author  userView `json:"author,omitempty"`
	Body    string   `json:"body,omitempty"`
	Created string   `json:"created,omitempty"`
	Updated string   `json:"updated,omitempty"`
}

type worklogView struct {
	ID               string   `json:"id,omitempty"`
	Author           userView `json:"author,omitempty"`
	Comment          string   `json:"comment,omitempty"`
	TimeSpent        string   `json:"timeSpent,omitempty"`
	TimeSpentSeconds int      `json:"timeSpentSeconds,omitempty"`
	Started          string   `json:"started,omitempty"`
	Created          string   `json:"created,omitempty"`
	Updated          string   `json:"updated,omitempty"`
}

type attachmentView struct {
	ID       string   `json:"id,omitempty"`
	Filename string   `json:"filename,omitempty"`
	Size     int64    `json:"size,omitempty"`
	MimeType string   `json:"mimeType,omitempty"`
	Content  string   `json:"content,omitempty"`
	Self     string   `json:"self,omitempty"`
	Author   userView `json:"author,omitempty"`
	Created  string   `json:"created,omitempty"`
}

type createMetaSummary struct {
	Projects []createMetaProject `json:"projects"`
}

type createMetaProject struct {
	Key        string                `json:"key,omitempty"`
	Name       string                `json:"name,omitempty"`
	IssueTypes []createMetaIssueType `json:"issuetypes,omitempty"`
}

type createMetaIssueType struct {
	ID     string                     `json:"id,omitempty"`
	Name   string                     `json:"name,omitempty"`
	Fields map[string]createMetaField `json:"fields,omitempty"`
}

type createMetaField struct {
	Name         string           `json:"name,omitempty"`
	Required     bool             `json:"required,omitempty"`
	Schema       createMetaSchema `json:"schema,omitempty"`
	AllowedValue []map[string]any `json:"allowedValues,omitempty"`
	DefaultValue map[string]any   `json:"defaultValue,omitempty"`
}

type createMetaSchema struct {
	Type   string `json:"type,omitempty"`
	Items  string `json:"items,omitempty"`
	System string `json:"system,omitempty"`
	Custom string `json:"custom,omitempty"`
}

type filterView struct {
	ID        string   `json:"id,omitempty"`
	Name      string   `json:"name,omitempty"`
	JQL       string   `json:"jql,omitempty"`
	Owner     userView `json:"owner,omitempty"`
	Favourite bool     `json:"favourite,omitempty"`
	ViewURL   string   `json:"viewUrl,omitempty"`
}

type remoteIssueLinkView struct {
	ID           int    `json:"id,omitempty"`
	GlobalID     string `json:"globalId,omitempty"`
	Relationship string `json:"relationship,omitempty"`
	Title        string `json:"title,omitempty"`
	URL          string `json:"url,omitempty"`
}

type versionView struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Archived    bool   `json:"archived,omitempty"`
	Released    bool   `json:"released,omitempty"`
}

func runConfig(opts Options, args []string, stdout, stderr io.Writer, rt Runtime) int {
	if len(args) != 1 || args[0] != "doctor" {
		fmt.Fprintln(stderr, "ERR usage expected config doctor")
		return 1
	}
	values, missing, err := resolveConfig(opts, rt, true)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}

	result := struct {
		OK          bool     `json:"ok"`
		Kind        string   `json:"kind"`
		Profile     string   `json:"profile,omitempty"`
		Type        string   `json:"type"`
		BaseURL     bool     `json:"baseUrl"`
		User        bool     `json:"user"`
		TokenSource string   `json:"tokenSource,omitempty"`
		TokenEnv    string   `json:"tokenEnv,omitempty"`
		Token       bool     `json:"token"`
		Missing     []string `json:"missing,omitempty"`
	}{
		OK:          len(missing) == 0,
		Kind:        "config_doctor",
		Profile:     values.Profile,
		Type:        values.Type,
		BaseURL:     values.BaseURL != "",
		User:        values.User != "",
		TokenSource: tokenSource(values),
		TokenEnv:    tokenEnvName(values),
		Token:       tokenAvailable(values, rt.Env),
		Missing:     missing,
	}
	if opts.JSON {
		if err := output.WriteJSON(stdout, result); err != nil {
			fmt.Fprintf(stderr, "ERR output %s\n", err)
			return 1
		}
		if result.OK {
			return 0
		}
		return 1
	}
	if !result.OK {
		fmt.Fprintf(stderr, "ERR config missing=%s\n", strings.Join(missing, ","))
		return 1
	}
	line := fmt.Sprintf("OK config type=%s base-url=set user=set token-source=%s token=set", values.Type, result.TokenSource)
	if result.TokenEnv != "" {
		line += " token-env=" + result.TokenEnv
	}
	if err := output.WriteCompact(stdout, line); err != nil {
		fmt.Fprintf(stderr, "ERR output %s\n", err)
		return 1
	}
	return 0
}

func runSkill(opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	if len(args) != 1 || args[0] != "install" {
		fmt.Fprintln(stderr, "ERR usage expected skill install")
		return 1
	}
	if mode == output.Raw {
		fmt.Fprintln(stderr, "ERR usage --raw is not supported for skill install; use --json for structured output")
		return 1
	}
	if opts.Global && commandValue(opts, "--target") != "" {
		fmt.Fprintln(stderr, "ERR usage --global and --target cannot be combined")
		return 1
	}

	scope, root, err := skillInstallRoot(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR skill %s\n", err)
		return 1
	}
	result, err := skillinstall.Install(root, opts.Force, opts.DryRun)
	if err != nil {
		if errors.Is(err, skillinstall.ErrTargetExists) {
			fmt.Fprintf(stderr, "ERR skill target exists %s; use --force to replace it\n", result.Target)
			return 1
		}
		fmt.Fprintf(stderr, "ERR skill %s\n", err)
		return 1
	}

	body := struct {
		OK      bool   `json:"ok"`
		Kind    string `json:"kind"`
		Skill   string `json:"skill"`
		Scope   string `json:"scope"`
		Root    string `json:"root"`
		Target  string `json:"target"`
		DryRun  bool   `json:"dryRun"`
		Force   bool   `json:"force"`
		Exists  bool   `json:"exists"`
		Changed bool   `json:"changed"`
	}{
		OK:      true,
		Kind:    "skill_install",
		Skill:   result.Skill,
		Scope:   scope,
		Root:    result.Root,
		Target:  result.Target,
		DryRun:  result.DryRun,
		Force:   result.Force,
		Exists:  result.Exists,
		Changed: result.Changed,
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, body)
	}
	if result.DryRun {
		return writeCompact(stdout, stderr, fmt.Sprintf("OK skill install dry-run target=%s scope=%s exists=%t", result.Target, scope, result.Exists))
	}
	return writeCompact(stdout, stderr, fmt.Sprintf("OK skill install target=%s scope=%s", result.Target, scope))
}

func skillInstallRoot(opts Options, rt Runtime) (string, string, error) {
	if target := commandValue(opts, "--target"); target != "" {
		root, err := filepath.Abs(target)
		if err != nil {
			return "", "", err
		}
		return "custom", root, nil
	}
	if opts.Global {
		home := firstNonEmpty(rt.Env["HOME"], rt.Env["USERPROFILE"])
		if home == "" && rt.Env["HOMEDRIVE"] != "" && rt.Env["HOMEPATH"] != "" {
			home = rt.Env["HOMEDRIVE"] + rt.Env["HOMEPATH"]
		}
		if home == "" {
			var err error
			home, err = os.UserHomeDir()
			if err != nil {
				return "", "", fmt.Errorf("resolve home directory: %w", err)
			}
		}
		root, err := filepath.Abs(filepath.Join(home, ".agents", "skills"))
		if err != nil {
			return "", "", err
		}
		return "global", root, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	root, err := filepath.Abs(filepath.Join(wd, ".agents", "skills"))
	if err != nil {
		return "", "", err
	}
	return "project", root, nil
}

func runProbe(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	if mode == output.Raw {
		fmt.Fprintln(stderr, "ERR usage --raw is not supported for probe; use --json for structured probe results")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}

	result, err := probe.Run(ctx, client, probe.Options{
		ProjectKey:    commandValue(opts, "--project"),
		IssueTypeName: commandValue(opts, "--issue-type"),
		IssueKey:      commandValue(opts, "--issue"),
	})
	if err != nil {
		return writeCommandError(stderr, err)
	}

	return writeProbe(stdout, stderr, mode, result)
}

func runWhoami(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var user jira.User
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"myself"}, nil, decodeOut(mode, &user))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK   bool     `json:"ok"`
		Kind string   `json:"kind"`
		User userView `json:"user"`
	}{OK: true, Kind: "whoami", User: toUserView(user)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	name := firstNonEmpty(user.Name, user.Key, "-")
	return writeCompact(stdout, stderr, fmt.Sprintf("%s key=%s email=%s display=%q", name, emptyDash(user.Key), emptyDash(user.EmailAddress), user.DisplayName))
}

func runAPIPassThrough(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "ERR usage expected api get|post|put|delete <PATH>")
		return 1
	}
	methodName := strings.ToLower(args[0])
	method := ""
	switch methodName {
	case "get":
		method = http.MethodGet
	case "post":
		method = http.MethodPost
	case "put":
		method = http.MethodPut
	case "delete":
		method = http.MethodDelete
	default:
		fmt.Fprintln(stderr, "ERR usage expected api get|post|put|delete <PATH>")
		return 1
	}
	api, segments, normalizedPath, err := parsePublicRESTPath(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	if err := validateAPIPassThroughEndpoint(api, method, segments, opts.DryRun || opts.Force); err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	query, err := commandQuery(opts)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}

	var payload any
	var rawBody string
	if len(commandValues(opts, "--body")) > 0 {
		value, compacted, err := commandJSONBody(opts)
		if err != nil {
			fmt.Fprintf(stderr, "ERR usage %s\n", err)
			return 1
		}
		payload = value
		rawBody = compacted
	}
	if method == http.MethodGet && payload != nil {
		fmt.Fprintln(stderr, "ERR usage api get does not accept --body")
		return 1
	}

	kind := "api_" + methodName
	plan := map[string]any{"method": method, "path": normalizedPath}
	if len(query) > 0 {
		plan["query"] = query.Encode()
	}
	if rawBody != "" {
		plan["body"] = rawBody
	}
	if method != http.MethodGet {
		if opts.DryRun {
			return writeDryRun(stdout, stderr, mode, kind, plan)
		}
		if !opts.Yes {
			fmt.Fprintf(stderr, "ERR usage api %s requires --dry-run or --yes\n", methodName)
			return 1
		}
	}

	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, method, api, segments, query, payload, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK         bool            `json:"ok"`
		Kind       string          `json:"kind"`
		Method     string          `json:"method"`
		Path       string          `json:"path"`
		StatusCode int             `json:"statusCode"`
		Body       json.RawMessage `json:"body,omitempty"`
	}{OK: true, Kind: kind, Method: method, Path: normalizedPath, StatusCode: resp.StatusCode, Body: rawJSONBody(resp.Body)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	if method == http.MethodGet {
		body := compactJSONRaw(result.Body)
		if len(body) > 2000 {
			body = fmt.Sprintf("body-bytes=%d", len(resp.Body))
		} else {
			body = "body=" + body
		}
		return writeCompact(stdout, stderr, fmt.Sprintf("OK %s status=%d path=%s %s", kind, resp.StatusCode, normalizedPath, body))
	}
	return writeOK(stdout, stderr, mode, kind, map[string]any{"path": normalizedPath, "status": resp.StatusCode})
}

func runSearch(ctx context.Context, opts Options, jql string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	return runSearchWithKind(ctx, opts, jql, "issue_search", stdout, stderr, rt, mode)
}

func runSearchWithKind(ctx context.Context, opts Options, jql, kind string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if mode == output.Raw {
		resp, err := client.Get(ctx, jira.PlatformAPI, []string{"search"}, jira.SearchQuery(jql, opts.StartAt, minPositive(opts.PageSize, opts.Limit)), nil)
		if err != nil {
			return writeCommandError(stderr, err)
		}
		return writeRaw(stdout, stderr, resp.Body)
	}

	items, page, err := collectSearch(ctx, client, jql, opts.StartAt, opts.PageSize, opts.Limit)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	result := struct {
		OK    bool        `json:"ok"`
		Kind  string      `json:"kind"`
		Items []issueView `json:"items"`
		Page  jira.Page   `json:"page"`
	}{OK: true, Kind: kind, Items: toIssueViews(items), Page: page}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}

	lines := make([]string, 0, len(items)+1)
	for _, issue := range items {
		lines = append(lines, compactIssue(issue))
	}
	summary := fmt.Sprintf("%d issues total=%d", len(items), page.Total)
	if page.NextStartAt > 0 {
		summary += fmt.Sprintf(" next=\"--start-at %d\"", page.NextStartAt)
	}
	lines = append(lines, summary)
	return writeCompact(stdout, stderr, lines...)
}

func runMine(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	days, err := commandDays(opts, 7)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	jql := fmt.Sprintf("assignee = currentUser() OR updated >= -%dd ORDER BY updated DESC", days)
	return runSearchWithKind(ctx, opts, jql, "mine", stdout, stderr, rt, mode)
}

func runStale(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	project, err := requiredCommandValue(opts, "--project")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	days, err := commandDays(opts, 30)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	clauses := []string{
		fmt.Sprintf("project = %s", jqlValue(project)),
		"resolution = EMPTY",
		fmt.Sprintf("updated <= -%dd", days),
	}
	if status := commandValue(opts, "--status"); status != "" {
		clauses = append(clauses, fmt.Sprintf("status = %s", jqlValue(status)))
	}
	jql := strings.Join(clauses, " AND ") + " ORDER BY updated ASC"
	return runSearchWithKind(ctx, opts, jql, "stale", stdout, stderr, rt, mode)
}

func runBulkSearch(ctx context.Context, opts Options, jqls []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	if mode == output.Raw {
		fmt.Fprintln(stderr, "ERR usage --raw is not supported for bulk search")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	type group struct {
		OK     bool        `json:"ok"`
		JQL    string      `json:"jql"`
		Items  []issueView `json:"items,omitempty"`
		Page   jira.Page   `json:"page,omitempty"`
		Error  string      `json:"error,omitempty"`
		issues []jira.Issue
	}
	groups := make([]group, 0, len(jqls))
	successes := 0
	failures := 0
	var firstErr error
	for _, jql := range jqls {
		items, page, err := collectSearch(ctx, client, jql, opts.StartAt, opts.PageSize, opts.Limit)
		if err != nil {
			failures++
			if firstErr == nil {
				firstErr = err
			}
			groups = append(groups, group{OK: false, JQL: jql, Error: err.Error()})
			continue
		}
		successes++
		groups = append(groups, group{OK: true, JQL: jql, Items: toIssueViews(items), Page: page, issues: items})
	}
	exitCode := 0
	if failures > 0 && successes > 0 {
		exitCode = 6
	} else if failures > 0 {
		return writeCommandError(stderr, firstErr)
	}
	if mode == output.JSON {
		code := writeJSON(stdout, stderr, struct {
			OK       bool    `json:"ok"`
			Kind     string  `json:"kind"`
			Groups   []group `json:"groups"`
			Success  int     `json:"success"`
			Failures int     `json:"failures"`
		}{OK: failures == 0, Kind: "bulk_search", Groups: groups, Success: successes, Failures: failures})
		if code != 0 {
			return code
		}
		return exitCode
	}
	lines := make([]string, 0, len(groups)*3)
	for i, group := range groups {
		if !group.OK {
			lines = append(lines, fmt.Sprintf("query=%d err %s", i+1, clean(group.Error)))
			continue
		}
		lines = append(lines, fmt.Sprintf("query=%d ok jql=%q", i+1, group.JQL))
		for _, issue := range group.issues {
			lines = append(lines, compactIssue(issue))
		}
		summary := fmt.Sprintf("%d issues total=%d", len(group.issues), group.Page.Total)
		if group.Page.NextStartAt > 0 {
			summary += fmt.Sprintf(" next=\"--start-at %d\"", group.Page.NextStartAt)
		}
		lines = append(lines, summary)
	}
	if err := output.WriteCompact(stdout, lines...); err != nil {
		fmt.Fprintf(stderr, "ERR output %s\n", err)
		return 1
	}
	return exitCode
}

func runIssue(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{"fields": {"summary,status,priority,assignee,project,issuetype,created,updated,description"}}
	var issue jira.Issue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key}, query, decodeOut(mode, &issue))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK    bool      `json:"ok"`
		Kind  string    `json:"kind"`
		Issue issueView `json:"issue"`
	}{OK: true, Kind: "issue", Issue: toIssueView(issue)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	return writeCompact(stdout, stderr, compactIssue(issue))
}

func runLinks(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var issue jira.Issue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key}, issueLinksQuery(), decodeOut(mode, &issue))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK    bool            `json:"ok"`
			Kind  string          `json:"kind"`
			Issue string          `json:"issue"`
			Links []issueLinkView `json:"links"`
		}{OK: true, Kind: "links", Issue: key, Links: toIssueLinkViews(issue.Fields.IssueLinks)})
	}
	if len(issue.Fields.IssueLinks) == 0 {
		return writeCompact(stdout, stderr, "0 links")
	}
	lines := make([]string, 0, len(issue.Fields.IssueLinks))
	for _, link := range issue.Fields.IssueLinks {
		lines = append(lines, compactIssueLink(link))
	}
	return writeCompact(stdout, stderr, lines...)
}

func runBlockers(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	key, err := requiredCommandValue(opts, "--issue")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var issue jira.Issue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key}, issueLinksQuery(), decodeOut(mode, &issue))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	blockers := make([]jira.IssueLink, 0, len(issue.Fields.IssueLinks))
	for _, link := range issue.Fields.IssueLinks {
		if isBlockingLink(link) {
			blockers = append(blockers, link)
		}
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool            `json:"ok"`
			Kind     string          `json:"kind"`
			Issue    string          `json:"issue"`
			Blockers []issueLinkView `json:"blockers"`
		}{OK: true, Kind: "blockers", Issue: key, Blockers: toIssueLinkViews(blockers)})
	}
	if len(blockers) == 0 {
		return writeCompact(stdout, stderr, "0 blockers")
	}
	lines := make([]string, 0, len(blockers))
	for _, link := range blockers {
		lines = append(lines, compactIssueLink(link))
	}
	return writeCompact(stdout, stderr, lines...)
}

func runComments(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
	}
	var page jira.CommentsResult
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "comment"}, query, decodeOut(mode, &page))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	pageInfo := jira.Page{StartAt: page.StartAt, MaxResults: page.MaxResults, Total: page.Total}
	if next, ok := jira.NextStartAt(page.StartAt, page.Total, len(page.Comments)); ok {
		pageInfo.NextStartAt = next
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool          `json:"ok"`
			Kind     string        `json:"kind"`
			Issue    string        `json:"issue"`
			Comments []commentView `json:"comments"`
			Page     jira.Page     `json:"page"`
		}{OK: true, Kind: "comments", Issue: key, Comments: toCommentViews(page.Comments), Page: pageInfo})
	}
	lines := make([]string, 0, len(page.Comments)+1)
	for _, comment := range page.Comments {
		lines = append(lines, compactComment(comment))
	}
	summary := fmt.Sprintf("%d comments total=%d", len(page.Comments), page.Total)
	if pageInfo.NextStartAt > 0 {
		summary += fmt.Sprintf(" next=\"--start-at %d\"", pageInfo.NextStartAt)
	}
	lines = append(lines, summary)
	return writeCompact(stdout, stderr, lines...)
}

func runCommentGet(ctx context.Context, opts Options, key, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var comment jira.Comment
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "comment", id}, nil, decodeOut(mode, &comment))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK      bool        `json:"ok"`
			Kind    string      `json:"kind"`
			Issue   string      `json:"issue"`
			Comment commentView `json:"comment"`
		}{OK: true, Kind: "comment", Issue: key, Comment: toCommentView(comment)})
	}
	return writeCompact(stdout, stderr, compactComment(comment))
}

func runWorklogs(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
	}
	var page jira.WorklogsResult
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "worklog"}, query, decodeOut(mode, &page))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	pageInfo := jira.Page{StartAt: page.StartAt, MaxResults: page.MaxResults, Total: page.Total}
	if next, ok := jira.NextStartAt(page.StartAt, page.Total, len(page.Worklogs)); ok {
		pageInfo.NextStartAt = next
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool          `json:"ok"`
			Kind     string        `json:"kind"`
			Issue    string        `json:"issue"`
			Worklogs []worklogView `json:"worklogs"`
			Page     jira.Page     `json:"page"`
		}{OK: true, Kind: "worklogs", Issue: key, Worklogs: toWorklogViews(page.Worklogs), Page: pageInfo})
	}
	lines := make([]string, 0, len(page.Worklogs)+1)
	for _, worklog := range page.Worklogs {
		lines = append(lines, compactWorklog(worklog))
	}
	summary := fmt.Sprintf("%d worklogs total=%d", len(page.Worklogs), page.Total)
	if pageInfo.NextStartAt > 0 {
		summary += fmt.Sprintf(" next=\"--start-at %d\"", pageInfo.NextStartAt)
	}
	lines = append(lines, summary)
	return writeCompact(stdout, stderr, lines...)
}

func runWorklogGet(ctx context.Context, opts Options, key, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var worklog jira.Worklog
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "worklog", id}, nil, decodeOut(mode, &worklog))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK      bool        `json:"ok"`
			Kind    string      `json:"kind"`
			Issue   string      `json:"issue"`
			Worklog worklogView `json:"worklog"`
		}{OK: true, Kind: "worklog", Issue: key, Worklog: toWorklogView(worklog)})
	}
	return writeCompact(stdout, stderr, compactWorklog(worklog))
}

func runAttachments(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{"fields": {"attachment"}}
	var issue jira.Issue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key}, query, decodeOut(mode, &issue))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	attachments := issue.Fields.Attachments
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK          bool             `json:"ok"`
			Kind        string           `json:"kind"`
			Issue       string           `json:"issue"`
			Attachments []attachmentView `json:"attachments"`
		}{OK: true, Kind: "attachments", Issue: key, Attachments: toAttachmentViews(attachments)})
	}
	lines := make([]string, 0, len(attachments)+1)
	for _, attachment := range attachments {
		lines = append(lines, compactAttachment(attachment))
	}
	lines = append(lines, fmt.Sprintf("%d attachments", len(attachments)))
	return writeCompact(stdout, stderr, lines...)
}

func runAttachmentGet(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var attachment jira.Attachment
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"attachment", id}, nil, decodeOut(mode, &attachment))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK         bool           `json:"ok"`
			Kind       string         `json:"kind"`
			Attachment attachmentView `json:"attachment"`
		}{OK: true, Kind: "attachment", Attachment: toAttachmentView(attachment)})
	}
	return writeCompact(stdout, stderr, compactAttachment(attachment))
}

func runAttachmentAdd(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	filePath, err := requiredCommandValue(opts, "--file")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	plan := map[string]any{"issue": key, "file": filePath}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "attachment_add", plan)
	}
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage attachment add requires --dry-run or --yes")
		return 1
	}
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage open attachment file: %s\n", err)
		return 1
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage stat attachment file: %s\n", err)
		return 1
	}
	if info.IsDir() {
		fmt.Fprintln(stderr, "ERR usage attachment file must not be a directory")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var attachments []jira.Attachment
	resp, err := client.PostMultipartFile(ctx, jira.PlatformAPI, []string{"issue", key, "attachments"}, "file", filepath.Base(filePath), file, decodeOut(mode, &attachments))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK          bool             `json:"ok"`
			Kind        string           `json:"kind"`
			Issue       string           `json:"issue"`
			Attachments []attachmentView `json:"attachments"`
		}{OK: true, Kind: "attachment_add", Issue: key, Attachments: toAttachmentViews(attachments)})
	}
	lines := make([]string, 0, len(attachments)+1)
	for _, attachment := range attachments {
		lines = append(lines, compactAttachment(attachment))
	}
	lines = append(lines, fmt.Sprintf("%d attachments uploaded", len(attachments)))
	return writeCompact(stdout, stderr, lines...)
}

func runAttachmentDownload(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	filePath, err := requiredCommandValue(opts, "--file")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	plan := map[string]any{"id": id, "file": filePath}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "attachment_download", plan)
	}
	if info, err := os.Stat(filePath); err == nil {
		if info.IsDir() {
			fmt.Fprintln(stderr, "ERR usage attachment download output must not be a directory")
			return 1
		}
		if !opts.Force {
			fmt.Fprintln(stderr, "ERR usage attachment download output exists; use --force")
			return 1
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "ERR usage stat output file: %s\n", err)
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var attachment jira.Attachment
	if _, err := client.Get(ctx, jira.PlatformAPI, []string{"attachment", id}, nil, &attachment); err != nil {
		return writeCommandError(stderr, err)
	}
	contentURL, err := safeAttachmentContentURL(client.BaseURL, attachment.Content)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	dir := filepath.Dir(filePath)
	temp, err := os.CreateTemp(dir, "."+filepath.Base(filePath)+".tmp-")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage create temp output file: %s\n", err)
		return 1
	}
	tempPath := temp.Name()
	var bytesWritten int64
	downloadOK := false
	defer func() {
		if !downloadOK {
			_ = os.Remove(tempPath)
		}
	}()
	_, bytesWritten, err = client.DownloadURL(ctx, contentURL, temp)
	closeErr := temp.Close()
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if closeErr != nil {
		fmt.Fprintf(stderr, "ERR usage close output file: %s\n", closeErr)
		return 1
	}
	if opts.Force {
		_ = os.Remove(filePath)
	}
	if err := os.Rename(tempPath, filePath); err != nil {
		fmt.Fprintf(stderr, "ERR usage move output file: %s\n", err)
		return 1
	}
	downloadOK = true
	return writeOK(stdout, stderr, mode, "attachment_download", map[string]any{"id": id, "file": filePath, "bytes": bytesWritten})
}

func runAttachmentDelete(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	plan := map[string]any{"id": id}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "attachment_delete", plan)
	}
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage attachment delete requires --dry-run or --yes")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodDelete, jira.PlatformAPI, []string{"attachment", id}, nil, nil, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "attachment_delete", plan)
}

func runProjects(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var projects []jira.Project
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"project"}, nil, decodeOut(mode, &projects))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK       bool          `json:"ok"`
		Kind     string        `json:"kind"`
		Projects []projectView `json:"projects"`
	}{OK: true, Kind: "projects", Projects: toProjectViews(projects)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	lines := make([]string, 0, len(projects))
	for _, project := range projects {
		lines = append(lines, fmt.Sprintf("%s %s", emptyDash(project.Key), clean(project.Name)))
	}
	return writeCompact(stdout, stderr, lines...)
}

func runProject(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var project jira.Project
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"project", key}, nil, decodeOut(mode, &project))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK      bool        `json:"ok"`
		Kind    string      `json:"kind"`
		Project projectView `json:"project"`
	}{OK: true, Kind: "project", Project: toProjectView(project)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	lead := "-"
	if project.Lead != nil {
		lead = firstNonEmpty(project.Lead.Name, project.Lead.Key, "-")
	}
	return writeCompact(stdout, stderr, fmt.Sprintf("%s %s lead=%s", emptyDash(project.Key), clean(project.Name), lead))
}

func runFields(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var fields []jira.Field
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"field"}, nil, decodeOut(mode, &fields))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK     bool        `json:"ok"`
		Kind   string      `json:"kind"`
		Fields []fieldView `json:"fields"`
	}{OK: true, Kind: "fields", Fields: toFieldViews(fields)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	lines := make([]string, 0, len(fields))
	for _, field := range fields {
		lines = append(lines, fmt.Sprintf("%s %s custom=%t", emptyDash(field.ID), clean(field.Name), field.Custom))
	}
	return writeCompact(stdout, stderr, lines...)
}

func runUsersSearch(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	queryText, err := requiredCommandValue(opts, "--query")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{
		"username":   {queryText},
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
	}
	var users []jira.User
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"user", "search"}, query, decodeOut(mode, &users))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK    bool       `json:"ok"`
			Kind  string     `json:"kind"`
			Users []userView `json:"users"`
		}{OK: true, Kind: "users_search", Users: toUserViews(users)})
	}
	lines := make([]string, 0, len(users)+1)
	for _, user := range users {
		lines = append(lines, compactUser(user))
	}
	lines = append(lines, fmt.Sprintf("%d users", len(users)))
	return writeCompact(stdout, stderr, lines...)
}

func runAssignableUsers(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	project := commandValue(opts, "--project")
	issue := commandValue(opts, "--issue")
	if project == "" && issue == "" {
		fmt.Fprintln(stderr, "ERR usage assignable requires --project KEY or --issue KEY")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
	}
	if project != "" {
		query.Set("project", project)
	}
	if issue != "" {
		query.Set("issueKey", issue)
	}
	if text := commandValue(opts, "--query"); text != "" {
		query.Set("username", text)
	}
	var users []jira.User
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"user", "assignable", "search"}, query, decodeOut(mode, &users))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK      bool       `json:"ok"`
			Kind    string     `json:"kind"`
			Project string     `json:"project,omitempty"`
			Issue   string     `json:"issue,omitempty"`
			Users   []userView `json:"users"`
		}{OK: true, Kind: "assignable", Project: project, Issue: issue, Users: toUserViews(users)})
	}
	lines := make([]string, 0, len(users)+1)
	for _, user := range users {
		lines = append(lines, compactUser(user))
	}
	lines = append(lines, fmt.Sprintf("%d assignable", len(users)))
	return writeCompact(stdout, stderr, lines...)
}

func runFilters(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var filters []jira.Filter
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"filter", "favourite"}, nil, decodeOut(mode, &filters))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK      bool         `json:"ok"`
			Kind    string       `json:"kind"`
			Filters []filterView `json:"filters"`
		}{OK: true, Kind: "filters", Filters: toFilterViews(filters)})
	}
	lines := make([]string, 0, len(filters)+1)
	for _, filter := range filters {
		lines = append(lines, compactFilter(filter))
	}
	lines = append(lines, fmt.Sprintf("%d filters", len(filters)))
	return writeCompact(stdout, stderr, lines...)
}

func runFilter(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var filter jira.Filter
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"filter", id}, nil, decodeOut(mode, &filter))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK     bool       `json:"ok"`
			Kind   string     `json:"kind"`
			Filter filterView `json:"filter"`
		}{OK: true, Kind: "filter", Filter: toFilterView(filter)})
	}
	return writeCompact(stdout, stderr, compactFilter(filter))
}

func runRemoteLinks(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var links []jira.RemoteIssueLink
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "remotelink"}, nil, decodeOut(mode, &links))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK    bool                  `json:"ok"`
			Kind  string                `json:"kind"`
			Issue string                `json:"issue"`
			Links []remoteIssueLinkView `json:"links"`
		}{OK: true, Kind: "remote_links", Issue: key, Links: toRemoteIssueLinkViews(links)})
	}
	lines := make([]string, 0, len(links)+1)
	for _, link := range links {
		lines = append(lines, compactRemoteIssueLink(link))
	}
	lines = append(lines, fmt.Sprintf("%d remote-links", len(links)))
	return writeCompact(stdout, stderr, lines...)
}

func runRemoteLink(ctx context.Context, opts Options, key, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var link jira.RemoteIssueLink
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "remotelink", id}, nil, decodeOut(mode, &link))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK    bool                `json:"ok"`
			Kind  string              `json:"kind"`
			Issue string              `json:"issue"`
			Link  remoteIssueLinkView `json:"link"`
		}{OK: true, Kind: "remote_link", Issue: key, Link: toRemoteIssueLinkView(link)})
	}
	return writeCompact(stdout, stderr, compactRemoteIssueLink(link))
}

func runMyPermissions(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	query := url.Values{}
	if project := commandValue(opts, "--project"); project != "" {
		query.Set("projectKey", project)
	}
	if issue := commandValue(opts, "--issue"); issue != "" {
		query.Set("issueKey", issue)
	}
	return runRawGet(ctx, opts, "mypermissions", []string{"mypermissions"}, query, stdout, stderr, rt, mode, compactJSONBodyLine)
}

func runRawGet(ctx context.Context, opts Options, kind string, segments []string, query url.Values, stdout, stderr io.Writer, rt Runtime, mode output.Mode, compact func([]byte) string) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Get(ctx, jira.PlatformAPI, segments, query, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	raw := rawJSONBody(resp.Body)
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK   bool            `json:"ok"`
			Kind string          `json:"kind"`
			Raw  json.RawMessage `json:"raw,omitempty"`
		}{OK: true, Kind: kind, Raw: raw})
	}
	return writeCompact(stdout, stderr, compact(resp.Body))
}

func runNamedList(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode, commandName, endpoint string) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var items []jira.NamedValue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{endpoint}, nil, decodeOut(mode, &items))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK    bool             `json:"ok"`
		Kind  string           `json:"kind"`
		Items []namedValueView `json:"items"`
	}{OK: true, Kind: commandName, Items: toNamedValueViews(items)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lines = append(lines, compactNamedValue(item))
	}
	return writeCompact(stdout, stderr, lines...)
}

func runNamedGet(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode, commandName, endpoint string) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var item jira.NamedValue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{endpoint, id}, nil, decodeOut(mode, &item))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK   bool           `json:"ok"`
		Kind string         `json:"kind"`
		Item namedValueView `json:"item"`
	}{OK: true, Kind: commandName, Item: toNamedValueView(item)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	return writeCompact(stdout, stderr, compactNamedValue(item))
}

func runProjectNamedList(ctx context.Context, opts Options, project, commandName, endpoint string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var items []jira.NamedValue
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"project", project, endpoint}, nil, decodeOut(mode, &items))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK      bool             `json:"ok"`
			Kind    string           `json:"kind"`
			Project string           `json:"project"`
			Items   []namedValueView `json:"items"`
		}{OK: true, Kind: commandName, Project: project, Items: toNamedValueViews(items)})
	}
	lines := make([]string, 0, len(items)+1)
	for _, item := range items {
		lines = append(lines, compactNamedValue(item))
	}
	lines = append(lines, fmt.Sprintf("%d %s", len(items), commandName))
	return writeCompact(stdout, stderr, lines...)
}

func runProjectVersions(ctx context.Context, opts Options, project string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var versions []jira.Version
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"project", project, "versions"}, nil, decodeOut(mode, &versions))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool          `json:"ok"`
			Kind     string        `json:"kind"`
			Project  string        `json:"project"`
			Versions []versionView `json:"versions"`
		}{OK: true, Kind: "versions", Project: project, Versions: toVersionViews(versions)})
	}
	lines := make([]string, 0, len(versions)+1)
	for _, version := range versions {
		lines = append(lines, compactVersion(version))
	}
	lines = append(lines, fmt.Sprintf("%d versions", len(versions)))
	return writeCompact(stdout, stderr, lines...)
}

func runProjectRoles(ctx context.Context, opts Options, project string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var roles map[string]string
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"project", project, "role"}, nil, decodeOut(mode, &roles))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK      bool              `json:"ok"`
			Kind    string            `json:"kind"`
			Project string            `json:"project"`
			Roles   map[string]string `json:"roles"`
		}{OK: true, Kind: "roles", Project: project, Roles: roles})
	}
	keys := make([]string, 0, len(roles))
	for key := range roles {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s %s", clean(key), roles[key]))
	}
	lines = append(lines, fmt.Sprintf("%d roles", len(keys)))
	return writeCompact(stdout, stderr, lines...)
}

func runDashboards(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	filter := commandValue(opts, "--filter")
	if filter != "" && filter != "favourite" && filter != "my" {
		fmt.Fprintln(stderr, "ERR usage --filter must be favourite or my")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
	}
	if filter != "" {
		query.Set("filter", filter)
	}
	var result jira.DashboardsResult
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"dashboard"}, query, decodeOut(mode, &result))
	if err != nil {
		return writeCommandError(stderr, jira.AsCapabilityError(err))
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	page := jira.Page{StartAt: result.StartAt, MaxResults: result.MaxResults, Total: result.Total}
	if next, ok := jira.NextStartAt(result.StartAt, result.Total, len(result.Dashboards)); ok {
		page.NextStartAt = next
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK            bool              `json:"ok"`
			Kind          string            `json:"kind"`
			Filter        string            `json:"filter,omitempty"`
			Items         []dashboardView   `json:"items"`
			RawDashboards []json.RawMessage `json:"rawDashboards,omitempty"`
			Page          jira.Page         `json:"page"`
		}{OK: true, Kind: "dashboards", Filter: filter, Items: toDashboardViews(result.Dashboards), RawDashboards: rawDashboardList(resp.Body), Page: page})
	}
	lines := make([]string, 0, len(result.Dashboards)+1)
	for _, dashboard := range result.Dashboards {
		lines = append(lines, compactDashboard(dashboard))
	}
	summary := fmt.Sprintf("%d dashboards total=%d", len(result.Dashboards), result.Total)
	if page.NextStartAt > 0 {
		summary += fmt.Sprintf(" next=\"--start-at %d\"", page.NextStartAt)
	}
	lines = append(lines, summary)
	return writeCompact(stdout, stderr, lines...)
}

func runDashboard(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var dashboard jira.Dashboard
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"dashboard", id}, nil, decodeOut(mode, &dashboard))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK           bool            `json:"ok"`
			Kind         string          `json:"kind"`
			Dashboard    dashboardView   `json:"dashboard"`
			RawDashboard json.RawMessage `json:"rawDashboard,omitempty"`
		}{OK: true, Kind: "dashboard", Dashboard: toDashboardView(dashboard), RawDashboard: rawJSONBody(resp.Body)})
	}
	return writeCompact(stdout, stderr, compactDashboard(dashboard))
}

func runDashboardItemPropertyKeys(ctx context.Context, opts Options, dashboardID, itemID string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var result jira.EntityPropertyKeys
	resp, err := client.Get(ctx, jira.PlatformAPI, dashboardItemPropertySegments(dashboardID, itemID), nil, decodeOut(mode, &result))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK        bool                    `json:"ok"`
			Kind      string                  `json:"kind"`
			Dashboard string                  `json:"dashboard"`
			Item      string                  `json:"item"`
			Keys      []entityPropertyKeyView `json:"keys"`
		}{OK: true, Kind: "dashboard_item_properties", Dashboard: dashboardID, Item: itemID, Keys: toEntityPropertyKeyViews(result.Keys)})
	}
	lines := make([]string, 0, len(result.Keys)+1)
	for _, key := range result.Keys {
		lines = append(lines, compactEntityPropertyKey(key))
	}
	lines = append(lines, fmt.Sprintf("%d properties", len(result.Keys)))
	return writeCompact(stdout, stderr, lines...)
}

func runDashboardItemPropertyGet(ctx context.Context, opts Options, dashboardID, itemID, propertyKey string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var property jira.EntityProperty
	resp, err := client.Get(ctx, jira.PlatformAPI, append(dashboardItemPropertySegments(dashboardID, itemID), propertyKey), nil, decodeOut(mode, &property))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if property.Key == "" {
		property.Key = propertyKey
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK        bool               `json:"ok"`
			Kind      string             `json:"kind"`
			Dashboard string             `json:"dashboard"`
			Item      string             `json:"item"`
			Property  entityPropertyView `json:"property"`
		}{OK: true, Kind: "dashboard_item_property", Dashboard: dashboardID, Item: itemID, Property: toEntityPropertyView(property)})
	}
	return writeCompact(stdout, stderr, compactEntityProperty(property))
}

func runDashboardItemPropertySet(ctx context.Context, opts Options, dashboardID, itemID, propertyKey string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	value, raw, err := commandJSONBody(opts)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	plan := map[string]any{"dashboard": dashboardID, "item": itemID, "key": propertyKey, "body": raw}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "dashboard_item_property_set", plan)
	}
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage dashboard item property set requires --dry-run or --yes")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodPut, jira.PlatformAPI, append(dashboardItemPropertySegments(dashboardID, itemID), propertyKey), nil, value, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "dashboard_item_property_set", map[string]any{"dashboard": dashboardID, "item": itemID, "key": propertyKey})
}

func runDashboardItemPropertyDelete(ctx context.Context, opts Options, dashboardID, itemID, propertyKey string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	plan := map[string]any{"dashboard": dashboardID, "item": itemID, "key": propertyKey}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "dashboard_item_property_delete", plan)
	}
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage dashboard item property delete requires --dry-run or --yes")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodDelete, jira.PlatformAPI, append(dashboardItemPropertySegments(dashboardID, itemID), propertyKey), nil, nil, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "dashboard_item_property_delete", plan)
}

func runIssuePropertyKeys(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var result jira.EntityPropertyKeys
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "properties"}, nil, decodeOut(mode, &result))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK    bool                    `json:"ok"`
			Kind  string                  `json:"kind"`
			Issue string                  `json:"issue"`
			Keys  []entityPropertyKeyView `json:"keys"`
		}{OK: true, Kind: "issue_properties", Issue: key, Keys: toEntityPropertyKeyViews(result.Keys)})
	}
	lines := make([]string, 0, len(result.Keys)+1)
	for _, propertyKey := range result.Keys {
		lines = append(lines, compactEntityPropertyKey(propertyKey))
	}
	lines = append(lines, fmt.Sprintf("%d properties", len(result.Keys)))
	return writeCompact(stdout, stderr, lines...)
}

func runIssuePropertyGet(ctx context.Context, opts Options, key, propertyKey string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var property jira.EntityProperty
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "properties", propertyKey}, nil, decodeOut(mode, &property))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if property.Key == "" {
		property.Key = propertyKey
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool               `json:"ok"`
			Kind     string             `json:"kind"`
			Issue    string             `json:"issue"`
			Property entityPropertyView `json:"property"`
		}{OK: true, Kind: "issue_property", Issue: key, Property: toEntityPropertyView(property)})
	}
	return writeCompact(stdout, stderr, compactEntityProperty(property))
}

func runIssuePropertySet(ctx context.Context, opts Options, key, propertyKey string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	value, raw, err := commandJSONBody(opts)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	plan := map[string]any{"issue": key, "key": propertyKey, "body": raw}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "issue_property_set", plan)
	}
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage issue property set requires --dry-run or --yes")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodPut, jira.PlatformAPI, []string{"issue", key, "properties", propertyKey}, nil, value, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "issue_property_set", map[string]any{"issue": key, "key": propertyKey})
}

func runIssuePropertyDelete(ctx context.Context, opts Options, key, propertyKey string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	plan := map[string]any{"issue": key, "key": propertyKey}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "issue_property_delete", plan)
	}
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage issue property delete requires --dry-run or --yes")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodDelete, jira.PlatformAPI, []string{"issue", key, "properties", propertyKey}, nil, nil, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "issue_property_delete", plan)
}

func runCreate(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	project, err := requiredCommandValue(opts, "--project")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	issueType, err := requiredCommandValue(opts, "--issue-type")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	if opts.Meta {
		return runCreateMeta(ctx, opts, project, issueType, stdout, stderr, rt, mode)
	}
	summary, err := requiredCommandValue(opts, "--summary")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	fields, err := fieldMap(opts)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	fields["project"] = map[string]string{"key": project}
	fields["issuetype"] = map[string]string{"name": issueType}
	fields["summary"] = summary
	if body := commandValue(opts, "--body"); body != "" {
		fields["description"] = body
	}
	if err := applyCommonIssueFieldFlags(fields, opts); err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	payload := map[string]any{"fields": fields}
	attachFiles := commandValues(opts, "--attach")
	if opts.DryRun {
		if len(attachFiles) > 0 {
			payload["attachments"] = attachFiles
		}
		return writeDryRun(stdout, stderr, mode, "create", payload)
	}
	if !requireWriteConfirmation(opts, stderr, "create") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var result jira.WriteResult
	resp, err := client.Do(ctx, http.MethodPost, jira.PlatformAPI, []string{"issue"}, nil, payload, decodeOut(mode, &result))
	if err != nil {
		return writeCommandErrorWithHint(stderr, err, fmt.Sprintf("run jira probe --project %s --issue-type %q to inspect createmeta", project, issueType))
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	issueKey := firstNonEmpty(result.Key, result.ID)
	uploaded, ok := uploadCreateAttachments(ctx, client, issueKey, attachFiles, stderr, mode)
	if !ok {
		return 1
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK          bool             `json:"ok"`
			Kind        string           `json:"kind"`
			Issue       string           `json:"issue"`
			Attachments []attachmentView `json:"attachments,omitempty"`
		}{OK: true, Kind: "create", Issue: issueKey, Attachments: uploaded})
	}
	resultFields := map[string]any{"issue": issueKey}
	if len(attachFiles) > 0 {
		resultFields["attachments"] = len(uploaded)
	}
	return writeOK(stdout, stderr, mode, "create", resultFields)
}

func runCreateMeta(ctx context.Context, opts Options, project, issueType string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	query := url.Values{
		"projectKeys":    {project},
		"issuetypeNames": {issueType},
		"expand":         {"projects.issuetypes.fields"},
	}
	var metadata map[string]any
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", "createmeta"}, query, decodeOut(mode, &metadata))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK        bool           `json:"ok"`
			Kind      string         `json:"kind"`
			Project   string         `json:"project"`
			IssueType string         `json:"issueType"`
			Metadata  map[string]any `json:"metadata"`
		}{OK: true, Kind: "createmeta", Project: project, IssueType: issueType, Metadata: metadata})
	}
	lines := compactCreateMeta(resp.Body, project, issueType)
	if createMetaIssueTypeCount(resp.Body) == 0 {
		query := url.Values{
			"projectKeys": {project},
			"expand":      {"projects.issuetypes"},
		}
		if allTypesResp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", "createmeta"}, query, nil); err == nil {
			if names := createMetaIssueTypeNames(allTypesResp.Body); len(names) > 0 {
				lines = append(lines, "available issue-types: "+strings.Join(names, ", "))
			}
		}
	}
	return writeCompact(stdout, stderr, lines...)
}

func uploadCreateAttachments(ctx context.Context, client jira.Client, issueKey string, files []string, stderr io.Writer, mode output.Mode) ([]attachmentView, bool) {
	if len(files) == 0 {
		return nil, true
	}
	uploaded := []attachmentView{}
	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintf(stderr, "ERR usage create succeeded issue=%s but open attachment file failed: %s\n", issueKey, err)
			return nil, false
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			fmt.Fprintf(stderr, "ERR usage create succeeded issue=%s but stat attachment file failed: %s\n", issueKey, err)
			return nil, false
		}
		if info.IsDir() {
			_ = file.Close()
			fmt.Fprintf(stderr, "ERR usage create succeeded issue=%s but attachment file must not be a directory\n", issueKey)
			return nil, false
		}
		var attachments []jira.Attachment
		_, err = client.PostMultipartFile(ctx, jira.PlatformAPI, []string{"issue", issueKey, "attachments"}, "file", filepath.Base(filePath), file, decodeOut(mode, &attachments))
		_ = file.Close()
		if err != nil {
			fmt.Fprintf(stderr, "ERR application create succeeded issue=%s but attachment upload failed: %s\n", issueKey, err)
			return nil, false
		}
		uploaded = append(uploaded, toAttachmentViews(attachments)...)
	}
	return uploaded, true
}

func runUpdate(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	fields, err := fieldMap(opts)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	if err := applyCommonIssueFieldFlags(fields, opts); err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	if len(fields) == 0 {
		fmt.Fprintln(stderr, "ERR usage update requires at least one field flag")
		return 1
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "update", map[string]any{"issue": key, "fields": fields})
	}
	if !requireWriteConfirmation(opts, stderr, "update") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodPut, jira.PlatformAPI, []string{"issue", key}, nil, map[string]any{"fields": fields}, nil)
	if err != nil {
		return writeCommandErrorWithHint(stderr, err, fmt.Sprintf("run jira probe --issue %s to inspect editmeta", key))
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "update", map[string]any{"issue": key})
}

func runComment(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	body, err := requiredCommandValue(opts, "--body")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "comment", map[string]any{"issue": key})
	}
	if !requireWriteConfirmation(opts, stderr, "comment") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var result jira.WriteResult
	resp, err := client.Do(ctx, http.MethodPost, jira.PlatformAPI, []string{"issue", key, "comment"}, nil, map[string]string{"body": body}, decodeOut(mode, &result))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "comment", map[string]any{"issue": key, "id": result.ID})
}

func runAssign(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	assignee, err := requiredCommandValue(opts, "--assignee")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "assign", map[string]any{"issue": key, "assignee": assignee})
	}
	if !requireWriteConfirmation(opts, stderr, "assign") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodPut, jira.PlatformAPI, []string{"issue", key, "assignee"}, nil, map[string]string{"name": assignee}, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "assign", map[string]any{"issue": key, "assignee": assignee})
}

func runTransitions(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var result jira.TransitionsResult
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "transitions"}, nil, decodeOut(mode, &result))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK          bool             `json:"ok"`
			Kind        string           `json:"kind"`
			Transitions []transitionView `json:"transitions"`
		}{OK: true, Kind: "transitions", Transitions: toTransitionViews(result.Transitions)})
	}
	lines := make([]string, 0, len(result.Transitions))
	for _, transition := range result.Transitions {
		to := "-"
		if transition.To != nil {
			to = namedName(transition.To)
		}
		lines = append(lines, fmt.Sprintf("%s %s to=%s", emptyDash(transition.ID), clean(transition.Name), emptyDash(to)))
	}
	return writeCompact(stdout, stderr, lines...)
}

func runTransition(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	id := commandValue(opts, "--id")
	name := commandValue(opts, "--name")
	if id == "" && name == "" {
		fmt.Fprintln(stderr, "ERR usage transition requires --id or --name")
		return 1
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "transition", map[string]any{"issue": key, "id": id, "name": name})
	}
	if !requireWriteConfirmation(opts, stderr, "transition") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if id == "" {
		resolved, err := resolveTransitionName(ctx, client, key, name)
		if err != nil {
			fmt.Fprintf(stderr, "ERR usage %s\n", err)
			return 1
		}
		id = resolved
	}
	payload := map[string]any{"transition": map[string]string{"id": id}}
	if comment := commandValue(opts, "--comment"); comment != "" {
		payload["update"] = map[string]any{"comment": []map[string]any{{"add": map[string]string{"body": comment}}}}
	}
	resp, err := client.Do(ctx, http.MethodPost, jira.PlatformAPI, []string{"issue", key, "transitions"}, nil, payload, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "transition", map[string]any{"issue": key, "id": id})
}

func runDelete(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	if !opts.Yes {
		fmt.Fprintln(stderr, "ERR usage delete requires --yes")
		return 1
	}
	if !validIssueKey(key) {
		fmt.Fprintln(stderr, "ERR usage delete requires an explicit issue key")
		return 1
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "delete", map[string]any{"issue": key})
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodDelete, jira.PlatformAPI, []string{"issue", key}, nil, nil, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "delete", map[string]any{"issue": key})
}

func runWatchers(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var watchers jira.Watchers
	resp, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "watchers"}, nil, decodeOut(mode, &watchers))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool         `json:"ok"`
			Kind     string       `json:"kind"`
			Watchers watchersView `json:"watchers"`
		}{OK: true, Kind: "watchers", Watchers: toWatchersView(watchers)})
	}
	return writeCompact(stdout, stderr, fmt.Sprintf("watching=%t count=%d", watchers.IsWatching, watchers.WatchCount))
}

func runWatch(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode, add bool) int {
	kind := "watch"
	method := http.MethodPost
	query := url.Values(nil)
	body := any(nil)
	if !add {
		kind = "unwatch"
		method = http.MethodDelete
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, kind, map[string]any{"issue": key})
	}
	if !requireWriteConfirmation(opts, stderr, kind) {
		return 1
	}
	client, values, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	user := values.User
	if add {
		body = any(user)
	} else {
		query = url.Values{"username": {user}}
	}
	resp, err := client.Do(ctx, method, jira.PlatformAPI, []string{"issue", key, "watchers"}, query, body, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, kind, map[string]any{"issue": key, "user": user})
}

func runWorklogAdd(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	timeSpent, err := requiredCommandValue(opts, "--time")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	payload := map[string]string{"timeSpent": timeSpent}
	if comment := commandValue(opts, "--comment"); comment != "" {
		payload["comment"] = comment
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "worklog_add", map[string]any{"issue": key, "time": timeSpent})
	}
	if !requireWriteConfirmation(opts, stderr, "worklog add") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	var result jira.WriteResult
	resp, err := client.Do(ctx, http.MethodPost, jira.PlatformAPI, []string{"issue", key, "worklog"}, nil, payload, decodeOut(mode, &result))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "worklog_add", map[string]any{"issue": key, "id": result.ID})
}

func runLinkCreate(ctx context.Context, opts Options, source string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	target, err := requiredCommandValue(opts, "--target")
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	linkType := commandValue(opts, "--link-type")
	if linkType == "" {
		linkType = "Blocks"
	}
	payload := map[string]any{
		"type":         map[string]string{"name": linkType},
		"inwardIssue":  map[string]string{"key": source},
		"outwardIssue": map[string]string{"key": target},
	}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "link_create", map[string]any{"source": source, "target": target, "type": linkType})
	}
	if !requireWriteConfirmation(opts, stderr, "link create") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodPost, jira.PlatformAPI, []string{"issueLink"}, nil, payload, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "link_create", map[string]any{"source": source, "target": target, "type": linkType})
}

func runLinkDelete(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "link_delete", map[string]any{"id": id})
	}
	if !requireWriteConfirmation(opts, stderr, "link delete") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	resp, err := client.Do(ctx, http.MethodDelete, jira.PlatformAPI, []string{"issueLink", id}, nil, nil, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "link_delete", map[string]any{"id": id})
}

func runMoveSprint(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	sprintID := commandValue(opts, "--sprint")
	positionalIssues := args
	if sprintID == "" && len(args) > 0 {
		sprintID = args[0]
		positionalIssues = args[1:]
	}
	if sprintID == "" {
		fmt.Fprintln(stderr, "ERR usage move sprint requires --sprint ID")
		return 1
	}
	issues, err := explicitIssueKeys(opts, positionalIssues)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	plan := map[string]any{"sprint": sprintID, "issues": issues}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "move_sprint", plan)
	}
	if !requireWriteConfirmation(opts, stderr, "move sprint") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	resp, err := client.Do(ctx, http.MethodPost, jira.AgileAPI, []string{"sprint", sprintID, "issue"}, nil, map[string]any{"issues": issues}, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "move_sprint", plan)
}

func runMoveBacklog(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	issues, err := explicitIssueKeys(opts, args)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}
	plan := map[string]any{"issues": issues}
	if opts.DryRun {
		return writeDryRun(stdout, stderr, mode, "move_backlog", plan)
	}
	if !requireWriteConfirmation(opts, stderr, "move backlog") {
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	resp, err := client.Do(ctx, http.MethodPost, jira.AgileAPI, []string{"backlog", "issue"}, nil, map[string]any{"issues": issues}, nil)
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	return writeOK(stdout, stderr, mode, "move_backlog", plan)
}

func runAgileBoard(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	var board jira.Board
	resp, err := client.Get(ctx, jira.AgileAPI, []string{"board", id}, nil, decodeOut(mode, &board))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK    bool      `json:"ok"`
		Kind  string    `json:"kind"`
		Board boardView `json:"board"`
	}{OK: true, Kind: "board", Board: toBoardView(board)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	return writeCompact(stdout, stderr, compactBoard(board))
}

func runSprint(ctx context.Context, opts Options, id string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	var sprint jira.Sprint
	resp, err := client.Get(ctx, jira.AgileAPI, []string{"sprint", id}, nil, decodeOut(mode, &sprint))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK     bool       `json:"ok"`
		Kind   string     `json:"kind"`
		Sprint sprintView `json:"sprint"`
	}{OK: true, Kind: "sprint", Sprint: toSprintView(sprint)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	return writeCompact(stdout, stderr, compactSprint(sprint))
}

func runEpic(ctx context.Context, opts Options, key string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	var epic jira.Epic
	resp, err := client.Get(ctx, jira.AgileAPI, []string{"epic", key}, nil, decodeOut(mode, &epic))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	result := struct {
		OK   bool     `json:"ok"`
		Kind string   `json:"kind"`
		Epic epicView `json:"epic"`
	}{OK: true, Kind: "epic", Epic: toEpicView(epic)}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	return writeCompact(stdout, stderr, compactEpic(epic))
}

func runSprintSummary(ctx context.Context, opts Options, args []string, stdout, stderr io.Writer, rt Runtime, mode output.Mode) int {
	sprintID := commandValue(opts, "--sprint")
	if sprintID == "" && len(args) > 0 {
		sprintID = args[0]
	}
	if sprintID == "" {
		fmt.Fprintln(stderr, "ERR usage sprint summary requires --sprint ID")
		return 1
	}
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
		"fields":     {"summary,status,priority,assignee,project,created,updated"},
	}
	var page jira.SearchResult
	resp, err := client.Get(ctx, jira.AgileAPI, []string{"sprint", sprintID, "issue"}, query, decodeOut(mode, &page))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	statuses := map[string]int{}
	for _, issue := range page.Issues {
		statuses[firstNonEmpty(namedName(issue.Fields.Status), "-")]++
	}
	pageInfo := jira.Page{StartAt: page.StartAt, MaxResults: page.MaxResults, Total: page.Total}
	if next, ok := jira.NextStartAt(page.StartAt, page.Total, len(page.Issues)); ok {
		pageInfo.NextStartAt = next
	}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK       bool           `json:"ok"`
			Kind     string         `json:"kind"`
			Sprint   string         `json:"sprint"`
			Statuses map[string]int `json:"statuses"`
			Page     jira.Page      `json:"page"`
		}{OK: true, Kind: "sprint_summary", Sprint: sprintID, Statuses: statuses, Page: pageInfo})
	}
	line := fmt.Sprintf("sprint=%s issues=%d total=%d statuses=%s", sprintID, len(page.Issues), page.Total, compactCounts(statuses))
	if pageInfo.NextStartAt > 0 {
		line += fmt.Sprintf(" next=\"--start-at %d\"", pageInfo.NextStartAt)
	}
	return writeCompact(stdout, stderr, line)
}

func runAgileList[T any](ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode, kind string, segments []string, compact func(T) string, view func(T) any) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
	}
	var page jira.AgilePage[T]
	resp, err := client.Get(ctx, jira.AgileAPI, segments, query, decodeOut(mode, &page))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	pageInfo := jira.Page{StartAt: page.StartAt, MaxResults: page.MaxResults, Total: page.Total}
	if next, ok := jira.NextStartAt(page.StartAt, page.Total, len(page.Values)); ok && !page.IsLast {
		pageInfo.NextStartAt = next
	}
	result := struct {
		OK    bool      `json:"ok"`
		Kind  string    `json:"kind"`
		Items []any     `json:"items"`
		Page  jira.Page `json:"page"`
	}{OK: true, Kind: kind, Items: agileViews(page.Values, view), Page: pageInfo}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	lines := make([]string, 0, len(page.Values)+1)
	for _, item := range page.Values {
		lines = append(lines, compact(item))
	}
	summary := fmt.Sprintf("%d %s total=%d", len(page.Values), kind, page.Total)
	if pageInfo.NextStartAt > 0 {
		summary += fmt.Sprintf(" next=\"--start-at %d\"", pageInfo.NextStartAt)
	}
	lines = append(lines, summary)
	return writeCompact(stdout, stderr, lines...)
}

func runAgileIssues(ctx context.Context, opts Options, stdout, stderr io.Writer, rt Runtime, mode output.Mode, kind string, segments []string) int {
	client, _, err := clientFromOptions(opts, rt)
	if err != nil {
		fmt.Fprintf(stderr, "ERR config %s\n", err)
		return 1
	}
	if err := ensureAgileAvailable(ctx, client); err != nil {
		return writeCommandError(stderr, err)
	}
	query := url.Values{
		"startAt":    {strconv.Itoa(opts.StartAt)},
		"maxResults": {strconv.Itoa(minPositive(opts.PageSize, opts.Limit))},
		"fields":     {"summary,status,priority,assignee,project,created,updated"},
	}
	var page jira.SearchResult
	resp, err := client.Get(ctx, jira.AgileAPI, segments, query, decodeOut(mode, &page))
	if err != nil {
		return writeCommandError(stderr, err)
	}
	if mode == output.Raw {
		return writeRaw(stdout, stderr, resp.Body)
	}
	pageInfo := jira.Page{StartAt: page.StartAt, MaxResults: page.MaxResults, Total: page.Total}
	if next, ok := jira.NextStartAt(page.StartAt, page.Total, len(page.Issues)); ok {
		pageInfo.NextStartAt = next
	}
	result := struct {
		OK    bool        `json:"ok"`
		Kind  string      `json:"kind"`
		Items []issueView `json:"items"`
		Page  jira.Page   `json:"page"`
	}{OK: true, Kind: kind, Items: toIssueViews(page.Issues), Page: pageInfo}
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	lines := make([]string, 0, len(page.Issues)+1)
	for _, issue := range page.Issues {
		lines = append(lines, compactIssue(issue))
	}
	summary := fmt.Sprintf("%d issues total=%d", len(page.Issues), page.Total)
	if pageInfo.NextStartAt > 0 {
		summary += fmt.Sprintf(" next=\"--start-at %d\"", pageInfo.NextStartAt)
	}
	lines = append(lines, summary)
	return writeCompact(stdout, stderr, lines...)
}

func collectSearch(ctx context.Context, client jira.Client, jql string, startAt, pageSize, limit int) ([]jira.Issue, jira.Page, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	if limit <= 0 {
		limit = pageSize
	}

	items := make([]jira.Issue, 0, minPositive(pageSize, limit))
	nextStart := startAt
	total := -1
	maxResults := pageSize
	for len(items) < limit {
		requestSize := minPositive(pageSize, limit-len(items))
		var result jira.SearchResult
		if _, err := client.Get(ctx, jira.PlatformAPI, []string{"search"}, jira.SearchQuery(jql, nextStart, requestSize), &result); err != nil {
			return nil, jira.Page{}, err
		}
		total = result.Total
		maxResults = result.MaxResults
		if maxResults == 0 {
			maxResults = requestSize
		}
		items = append(items, result.Issues...)
		next, ok := jira.NextStartAt(result.StartAt, result.Total, len(result.Issues))
		if !ok {
			nextStart = 0
			break
		}
		nextStart = next
	}

	page := jira.Page{
		StartAt:    startAt,
		MaxResults: maxResults,
		Total:      total,
	}
	if nextStart > 0 && (total < 0 || nextStart < total) {
		page.NextStartAt = nextStart
	}
	return items, page, nil
}

func issueLinksQuery() url.Values {
	return url.Values{"fields": {"issuelinks,summary,status,priority,assignee"}}
}

func decodeOut(mode output.Mode, out any) any {
	if mode == output.Raw {
		return nil
	}
	return out
}

func ensureAgileAvailable(ctx context.Context, client jira.Client) error {
	_, err := client.Get(ctx, jira.AgileAPI, []string{"board"}, url.Values{"maxResults": {"1"}}, nil)
	return jira.AsCapabilityError(err)
}

func resolveTransitionName(ctx context.Context, client jira.Client, key, name string) (string, error) {
	var result jira.TransitionsResult
	if _, err := client.Get(ctx, jira.PlatformAPI, []string{"issue", key, "transitions"}, nil, &result); err != nil {
		return "", err
	}
	var matches []jira.Transition
	for _, transition := range result.Transitions {
		if transition.Name == name {
			matches = append(matches, transition)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("transition name %q was not found", name)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("transition name %q is ambiguous", name)
	}
	return matches[0].ID, nil
}

func clientFromOptions(opts Options, rt Runtime) (jira.Client, config.Values, error) {
	values, missing, err := resolveConfig(opts, rt, true)
	if err != nil {
		return jira.Client{}, values, err
	}
	if len(missing) > 0 {
		return jira.Client{}, values, fmt.Errorf("missing %s", strings.Join(missing, ","))
	}
	return jira.Client{
		BaseURL:    values.BaseURL,
		User:       values.User,
		Secret:     resolvedSecret(values, rt.Env),
		HTTPClient: rt.HTTPClient,
		Timeout:    opts.Timeout,
	}, values, nil
}

func resolveConfig(opts Options, rt Runtime, requireProfile bool) (config.Values, []string, error) {
	if rt.Env == nil {
		rt.Env = map[string]string{}
	}
	if requireProfile && rt.ProfileLoadError != nil {
		return config.Values{}, nil, rt.ProfileLoadError
	}
	values := config.Resolve(config.Values{
		Profile:  opts.Profile,
		Type:     opts.Type,
		BaseURL:  opts.BaseURL,
		User:     opts.User,
		TokenEnv: opts.TokenEnv,
	}, rt.Env, rt.Profiles)
	missing := config.Validate(values, rt.Env)
	sort.Strings(missing)
	return values, missing, nil
}

func resolvedSecret(values config.Values, env map[string]string) string {
	if values.Secret != "" {
		return values.Secret
	}
	if values.TokenEnv != "" {
		return env[values.TokenEnv]
	}
	return ""
}

func tokenAvailable(values config.Values, env map[string]string) bool {
	return resolvedSecret(values, env) != ""
}

func tokenSource(values config.Values) string {
	if values.SecretSource.Kind != "" {
		return values.SecretSource.Kind
	}
	if values.TokenEnv != "" {
		return config.SecretSourceEnv
	}
	return ""
}

func tokenEnvName(values config.Values) string {
	if values.TokenEnv != "" && tokenSource(values) == config.SecretSourceEnv {
		return values.TokenEnv
	}
	return ""
}

func writeProbe(stdout, stderr io.Writer, mode output.Mode, result probe.Result) int {
	if mode == output.JSON {
		return writeJSON(stdout, stderr, result)
	}
	dashboard := "unavailable"
	if result.Dashboards {
		dashboard = "available"
	}
	line := fmt.Sprintf("OK probe server=%s api=%s dashboard=%s agile=%s user=%s", emptyDash(result.ServerVersion), result.API, dashboard, result.Agile, emptyDash(result.User))
	if len(result.Warnings) > 0 {
		line += " warnings=" + strconv.Itoa(len(result.Warnings))
	}
	return writeCompact(stdout, stderr, line)
}

func writeCommandError(stderr io.Writer, err error) int {
	if jiraErr, ok := err.(*jira.Error); ok {
		fmt.Fprintf(stderr, "ERR jira %s\n", jiraErr.Error())
		return jiraErr.ExitCode()
	}
	fmt.Fprintf(stderr, "ERR %s\n", err)
	return 1
}

func writeCommandErrorWithHint(stderr io.Writer, err error, hint string) int {
	code := writeCommandError(stderr, err)
	if code == 3 && hint != "" {
		fmt.Fprintf(stderr, "HINT %s\n", hint)
	}
	return code
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	if err := output.WriteJSON(stdout, value); err != nil {
		fmt.Fprintf(stderr, "ERR output %s\n", err)
		return 1
	}
	return 0
}

func writeRaw(stdout, stderr io.Writer, body []byte) int {
	if err := output.WriteRaw(stdout, body); err != nil {
		fmt.Fprintf(stderr, "ERR output %s\n", err)
		return 1
	}
	return 0
}

func writeCompact(stdout, stderr io.Writer, lines ...string) int {
	if err := output.WriteCompact(stdout, lines...); err != nil {
		fmt.Fprintf(stderr, "ERR output %s\n", err)
		return 1
	}
	return 0
}

func writeDryRun(stdout, stderr io.Writer, mode output.Mode, kind string, plan map[string]any) int {
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK     bool           `json:"ok"`
			Kind   string         `json:"kind"`
			DryRun bool           `json:"dryRun"`
			Plan   map[string]any `json:"plan"`
		}{OK: true, Kind: kind, DryRun: true, Plan: plan})
	}
	parts := []string{"DRY-RUN", kind}
	for _, key := range sortedKeys(plan) {
		parts = append(parts, fmt.Sprintf("%s=%s", key, compactPlanValue(plan[key])))
	}
	return writeCompact(stdout, stderr, strings.Join(parts, " "))
}

func writeOK(stdout, stderr io.Writer, mode output.Mode, kind string, fields map[string]any) int {
	if mode == output.JSON {
		return writeJSON(stdout, stderr, struct {
			OK     bool           `json:"ok"`
			Kind   string         `json:"kind"`
			Fields map[string]any `json:"fields,omitempty"`
		}{OK: true, Kind: kind, Fields: fields})
	}
	parts := []string{"OK", kind}
	for _, key := range sortedKeys(fields) {
		if fields[key] == "" || fields[key] == nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%v", key, fields[key]))
	}
	return writeCompact(stdout, stderr, strings.Join(parts, " "))
}

func requireWriteConfirmation(opts Options, stderr io.Writer, command string) bool {
	if opts.Yes {
		return true
	}
	fmt.Fprintf(stderr, "ERR usage %s requires --dry-run or --yes\n", command)
	return false
}

func toIssueView(issue jira.Issue) issueView {
	return issueView{
		ID:          issue.ID,
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Status:      namedName(issue.Fields.Status),
		Priority:    namedName(issue.Fields.Priority),
		Assignee:    userName(issue.Fields.Assignee),
		Project:     projectKey(issue.Fields.Project),
		IssueType:   namedName(issue.Fields.IssueType),
		Created:     issue.Fields.Created,
		Updated:     issue.Fields.Updated,
		Description: issue.Fields.Description,
	}
}

func toIssueViews(issues []jira.Issue) []issueView {
	views := make([]issueView, 0, len(issues))
	for _, issue := range issues {
		views = append(views, toIssueView(issue))
	}
	return views
}

func toIssueLinkView(link jira.IssueLink) issueLinkView {
	view := issueLinkView{ID: link.ID}
	if link.Type != nil {
		view.Type = firstNonEmpty(link.Type.Name, link.Type.ID)
		view.Direction = firstNonEmpty(link.Type.Outward, link.Type.Inward, link.Type.Name)
	}
	target := link.OutwardIssue
	if target == nil {
		target = link.InwardIssue
		if link.Type != nil {
			view.Direction = firstNonEmpty(link.Type.Inward, link.Type.Outward, link.Type.Name)
		}
	}
	if target != nil {
		issue := toIssueView(*target)
		view.Issue = &issue
	}
	return view
}

func toIssueLinkViews(links []jira.IssueLink) []issueLinkView {
	views := make([]issueLinkView, 0, len(links))
	for _, link := range links {
		views = append(views, toIssueLinkView(link))
	}
	return views
}

func toUserView(user jira.User) userView {
	return userView{
		Name:         user.Name,
		Key:          user.Key,
		DisplayName:  user.DisplayName,
		EmailAddress: user.EmailAddress,
	}
}

func toUserViews(users []jira.User) []userView {
	views := make([]userView, 0, len(users))
	for _, user := range users {
		views = append(views, toUserView(user))
	}
	return views
}

func toProjectView(project jira.Project) projectView {
	view := projectView{ID: project.ID, Key: project.Key, Name: project.Name}
	if project.Lead != nil {
		lead := toUserView(*project.Lead)
		view.Lead = &lead
	}
	return view
}

func toProjectViews(projects []jira.Project) []projectView {
	views := make([]projectView, 0, len(projects))
	for _, project := range projects {
		views = append(views, toProjectView(project))
	}
	return views
}

func toFieldViews(fields []jira.Field) []fieldView {
	views := make([]fieldView, 0, len(fields))
	for _, field := range fields {
		views = append(views, fieldView{ID: field.ID, Name: field.Name, Custom: field.Custom})
	}
	return views
}

func toNamedValueViews(items []jira.NamedValue) []namedValueView {
	views := make([]namedValueView, 0, len(items))
	for _, item := range items {
		views = append(views, toNamedValueView(item))
	}
	return views
}

func toNamedValueView(item jira.NamedValue) namedValueView {
	return namedValueView{ID: item.ID, Key: item.Key, Name: item.Name, Description: item.Description}
}

func toTransitionViews(transitions []jira.Transition) []transitionView {
	views := make([]transitionView, 0, len(transitions))
	for _, transition := range transitions {
		views = append(views, transitionView{ID: transition.ID, Name: transition.Name, To: namedName(transition.To)})
	}
	return views
}

func toWatchersView(watchers jira.Watchers) watchersView {
	return watchersView{
		IsWatching: watchers.IsWatching,
		WatchCount: watchers.WatchCount,
		Watchers:   toUserViews(watchers.Watchers),
	}
}

func toBoardView(board jira.Board) boardView {
	return boardView{ID: board.ID, Name: board.Name, Type: board.Type}
}

func toSprintView(sprint jira.Sprint) sprintView {
	return sprintView{ID: sprint.ID, Name: sprint.Name, State: sprint.State}
}

func toEpicView(epic jira.Epic) epicView {
	return epicView{ID: epic.ID, Key: epic.Key, Name: epic.Name, Summary: epic.Summary, Done: epic.Done}
}

func toDashboardView(dashboard jira.Dashboard) dashboardView {
	return dashboardView{ID: dashboard.ID, Name: dashboard.Name, Self: dashboard.Self, View: dashboard.View}
}

func toDashboardViews(dashboards []jira.Dashboard) []dashboardView {
	views := make([]dashboardView, 0, len(dashboards))
	for _, dashboard := range dashboards {
		views = append(views, toDashboardView(dashboard))
	}
	return views
}

func toEntityPropertyKeyViews(keys []jira.EntityPropertyKey) []entityPropertyKeyView {
	views := make([]entityPropertyKeyView, 0, len(keys))
	for _, key := range keys {
		views = append(views, entityPropertyKeyView{Self: key.Self, Key: key.Key})
	}
	return views
}

func toEntityPropertyView(property jira.EntityProperty) entityPropertyView {
	return entityPropertyView{Key: property.Key, Value: property.Value}
}

func toCommentView(comment jira.Comment) commentView {
	view := commentView{ID: comment.ID, Body: comment.Body, Created: comment.Created, Updated: comment.Updated}
	if comment.Author != nil {
		view.Author = toUserView(*comment.Author)
	}
	return view
}

func toCommentViews(comments []jira.Comment) []commentView {
	views := make([]commentView, 0, len(comments))
	for _, comment := range comments {
		views = append(views, toCommentView(comment))
	}
	return views
}

func toWorklogView(worklog jira.Worklog) worklogView {
	view := worklogView{
		ID:               worklog.ID,
		Comment:          worklog.Comment,
		TimeSpent:        worklog.TimeSpent,
		TimeSpentSeconds: worklog.TimeSpentSeconds,
		Started:          worklog.Started,
		Created:          worklog.Created,
		Updated:          worklog.Updated,
	}
	if worklog.Author != nil {
		view.Author = toUserView(*worklog.Author)
	}
	return view
}

func toWorklogViews(worklogs []jira.Worklog) []worklogView {
	views := make([]worklogView, 0, len(worklogs))
	for _, worklog := range worklogs {
		views = append(views, toWorklogView(worklog))
	}
	return views
}

func toAttachmentView(attachment jira.Attachment) attachmentView {
	view := attachmentView{
		ID:       attachment.ID,
		Filename: attachment.Filename,
		Size:     attachment.Size,
		MimeType: attachment.MimeType,
		Content:  attachment.Content,
		Self:     attachment.Self,
		Created:  attachment.Created,
	}
	if attachment.Author != nil {
		view.Author = toUserView(*attachment.Author)
	}
	return view
}

func toAttachmentViews(attachments []jira.Attachment) []attachmentView {
	views := make([]attachmentView, 0, len(attachments))
	for _, attachment := range attachments {
		views = append(views, toAttachmentView(attachment))
	}
	return views
}

func toFilterView(filter jira.Filter) filterView {
	view := filterView{ID: filter.ID, Name: filter.Name, JQL: filter.JQL, Favourite: filter.Favourite, ViewURL: filter.ViewURL}
	if filter.Owner != nil {
		view.Owner = toUserView(*filter.Owner)
	}
	return view
}

func toFilterViews(filters []jira.Filter) []filterView {
	views := make([]filterView, 0, len(filters))
	for _, filter := range filters {
		views = append(views, toFilterView(filter))
	}
	return views
}

func toRemoteIssueLinkView(link jira.RemoteIssueLink) remoteIssueLinkView {
	return remoteIssueLinkView{
		ID:           link.ID,
		GlobalID:     link.GlobalID,
		Relationship: link.Relationship,
		Title:        link.Object.Title,
		URL:          link.Object.URL,
	}
}

func toRemoteIssueLinkViews(links []jira.RemoteIssueLink) []remoteIssueLinkView {
	views := make([]remoteIssueLinkView, 0, len(links))
	for _, link := range links {
		views = append(views, toRemoteIssueLinkView(link))
	}
	return views
}

func toVersionViews(versions []jira.Version) []versionView {
	views := make([]versionView, 0, len(versions))
	for _, version := range versions {
		views = append(views, versionView{ID: version.ID, Name: version.Name, Description: version.Description, Archived: version.Archived, Released: version.Released})
	}
	return views
}

func agileViews[T any](items []T, view func(T) any) []any {
	views := make([]any, 0, len(items))
	for _, item := range items {
		views = append(views, view(item))
	}
	return views
}

func compactIssue(issue jira.Issue) string {
	return strings.Join([]string{
		emptyDash(issue.Key),
		emptyDash(namedName(issue.Fields.Status)),
		emptyDash(namedName(issue.Fields.Priority)),
		emptyDash(userName(issue.Fields.Assignee)),
		clean(issue.Fields.Summary),
	}, " ")
}

func compactIssueLink(link jira.IssueLink) string {
	direction := "linked"
	if link.Type != nil {
		direction = firstNonEmpty(link.Type.Outward, link.Type.Inward, link.Type.Name, direction)
	}
	target := link.OutwardIssue
	if target == nil {
		target = link.InwardIssue
		if link.Type != nil {
			direction = firstNonEmpty(link.Type.Inward, link.Type.Outward, link.Type.Name, direction)
		}
	}
	targetText := "-"
	if target != nil {
		targetText = compactIssue(*target)
	}
	linkType := "-"
	if link.Type != nil {
		linkType = firstNonEmpty(link.Type.Name, link.Type.ID, "-")
	}
	return fmt.Sprintf("%s %s %s %s", emptyDash(link.ID), emptyDash(linkType), clean(direction), targetText)
}

func compactUser(user jira.User) string {
	name := firstNonEmpty(user.Name, user.Key, "-")
	return fmt.Sprintf("%s key=%s email=%s display=%q", name, emptyDash(user.Key), emptyDash(user.EmailAddress), user.DisplayName)
}

func compactNamedValue(item jira.NamedValue) string {
	id := firstNonEmpty(item.ID, item.Key, "-")
	name := firstNonEmpty(item.Name, item.Description, "-")
	return fmt.Sprintf("%s %s", id, clean(name))
}

func compactCounts(values map[string]int) string {
	if len(values) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s:%d", clean(key), values[key]))
	}
	return strings.Join(parts, ",")
}

func compactBoard(board jira.Board) string {
	return fmt.Sprintf("%d %s %s", board.ID, emptyDash(board.Type), clean(board.Name))
}

func compactSprint(sprint jira.Sprint) string {
	return fmt.Sprintf("%d %s %s", sprint.ID, emptyDash(sprint.State), clean(sprint.Name))
}

func compactEpic(epic jira.Epic) string {
	id := firstNonEmpty(epic.Key, strconv.Itoa(epic.ID))
	name := firstNonEmpty(epic.Name, epic.Summary, "-")
	return fmt.Sprintf("%s %s", id, clean(name))
}

func compactDashboard(dashboard jira.Dashboard) string {
	line := fmt.Sprintf("%s %s", emptyDash(dashboard.ID), clean(dashboard.Name))
	if dashboard.View != "" {
		line += " view=" + dashboard.View
	}
	return line
}

func compactEntityPropertyKey(key jira.EntityPropertyKey) string {
	line := emptyDash(key.Key)
	if key.Self != "" {
		line += " self=" + key.Self
	}
	return line
}

func compactEntityProperty(property jira.EntityProperty) string {
	return fmt.Sprintf("%s value=%s", emptyDash(property.Key), compactJSONRaw(property.Value))
}

func compactComment(comment jira.Comment) string {
	return fmt.Sprintf("%s %s %s", emptyDash(comment.ID), emptyDash(userName(comment.Author)), clean(comment.Body))
}

func compactWorklog(worklog jira.Worklog) string {
	timeSpent := firstNonEmpty(worklog.TimeSpent, strconv.Itoa(worklog.TimeSpentSeconds)+"s", "-")
	return fmt.Sprintf("%s %s %s %s", emptyDash(worklog.ID), emptyDash(userName(worklog.Author)), emptyDash(timeSpent), clean(worklog.Comment))
}

func compactAttachment(attachment jira.Attachment) string {
	return fmt.Sprintf("%s %s size=%d mime=%s", emptyDash(attachment.ID), clean(attachment.Filename), attachment.Size, emptyDash(attachment.MimeType))
}

func compactFilter(filter jira.Filter) string {
	id := firstNonEmpty(filter.ID, "-")
	name := firstNonEmpty(filter.Name, "-")
	owner := "-"
	if filter.Owner != nil {
		owner = userName(filter.Owner)
	}
	return fmt.Sprintf("%s %s owner=%s favourite=%t jql=%q", id, clean(name), emptyDash(owner), filter.Favourite, filter.JQL)
}

func compactRemoteIssueLink(link jira.RemoteIssueLink) string {
	return fmt.Sprintf("%d %s %s %s", link.ID, emptyDash(link.Relationship), clean(link.Object.Title), emptyDash(link.Object.URL))
}

func compactVersion(version jira.Version) string {
	return fmt.Sprintf("%s %s released=%t archived=%t", emptyDash(version.ID), clean(version.Name), version.Released, version.Archived)
}

func compactJSONBodyLine(body []byte) string {
	raw := rawJSONBody(body)
	if raw == nil {
		return clean(string(body))
	}
	return compactJSONRaw(raw)
}

func compactJSONArrayCount(noun string) func([]byte) string {
	return func(body []byte) string {
		var items []json.RawMessage
		if err := json.Unmarshal(body, &items); err != nil {
			return compactJSONBodyLine(body)
		}
		return fmt.Sprintf("%d %s", len(items), noun)
	}
}

func requiredCommandValue(opts Options, name string) (string, error) {
	value := commandValue(opts, name)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func commandValue(opts Options, name string) string {
	values := commandValues(opts, name)
	if len(values) == 0 {
		return ""
	}
	return values[len(values)-1]
}

func commandValues(opts Options, name string) []string {
	switch name {
	case "--project":
		return singleValue(opts.Project)
	case "--issue-type":
		return singleValue(opts.IssueType)
	case "--issue":
		return opts.Issues
	case "--summary":
		return singleValue(opts.Summary)
	case "--body":
		if opts.BodySet {
			return []string{opts.Body}
		}
		return singleValue(opts.Body)
	case "--field":
		return opts.Fields
	case "--component":
		return opts.Components
	case "--version":
		return opts.Versions
	case "--due":
		return singleValue(opts.Due)
	case "--priority":
		return singleValue(opts.Priority)
	case "--attach":
		return opts.Attachments
	case "--assignee":
		return singleValue(opts.Assignee)
	case "--id":
		return singleValue(opts.ID)
	case "--name":
		return singleValue(opts.Name)
	case "--comment":
		return singleValue(opts.Comment)
	case "--time":
		return singleValue(opts.Time)
	case "--target":
		return singleValue(opts.Target)
	case "--link-type":
		return singleValue(opts.LinkType)
	case "--query":
		return opts.Queries
	case "--days":
		return singleValue(opts.Days)
	case "--status":
		return singleValue(opts.Status)
	case "--sprint":
		return singleValue(opts.Sprint)
	case "--filter":
		return singleValue(opts.Filter)
	case "--file":
		return singleValue(opts.File)
	default:
		return nil
	}
}

func singleValue(value string) []string {
	if value == "" {
		return nil
	}
	return []string{value}
}

func fieldMap(opts Options) (map[string]any, error) {
	fields := map[string]any{}
	for _, item := range commandValues(opts, "--field") {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("--field must be name=value")
		}
		parsed, err := fieldValue(value)
		if err != nil {
			return nil, fmt.Errorf("--field %s: %w", strings.TrimSpace(key), err)
		}
		fields[strings.TrimSpace(key)] = parsed
	}
	return fields, nil
}

func applyCommonIssueFieldFlags(fields map[string]any, opts Options) error {
	if components := commandValues(opts, "--component"); len(components) > 0 {
		values, err := jiraReferenceList(components)
		if err != nil {
			return fmt.Errorf("--component %w", err)
		}
		fields["components"] = values
	}
	if versions := commandValues(opts, "--version"); len(versions) > 0 {
		values, err := jiraReferenceList(versions)
		if err != nil {
			return fmt.Errorf("--version %w", err)
		}
		fields["versions"] = values
	}
	if due := strings.TrimSpace(commandValue(opts, "--due")); due != "" {
		if _, err := time.Parse("2006-01-02", due); err != nil {
			return fmt.Errorf("--due must be YYYY-MM-DD")
		}
		fields["duedate"] = due
	}
	if priority := commandValue(opts, "--priority"); priority != "" {
		value, err := jiraReference(priority)
		if err != nil {
			return fmt.Errorf("--priority %w", err)
		}
		fields["priority"] = value
	}
	return nil
}

func jiraReferenceList(values []string) ([]map[string]string, error) {
	refs := make([]map[string]string, 0, len(values))
	for _, value := range values {
		ref, err := jiraReference(value)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func jiraReference(value string) (map[string]string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, fmt.Errorf("must not be empty")
	}
	if allDigits(trimmed) {
		return map[string]string{"id": trimmed}, nil
	}
	return map[string]string{"name": trimmed}, nil
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func fieldValue(value string) (any, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return trimmed, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("object/array values must be valid JSON")
	}
	return parsed, nil
}

func commandJSONBody(opts Options) (json.RawMessage, string, error) {
	values := commandValues(opts, "--body")
	if len(values) == 0 {
		return nil, "", fmt.Errorf("--body is required")
	}
	body := values[len(values)-1]
	raw := []byte(body)
	if !json.Valid(raw) {
		return nil, "", fmt.Errorf("--body must be valid JSON")
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err != nil {
		return nil, "", fmt.Errorf("--body must be valid JSON")
	}
	return append(json.RawMessage(nil), compacted.Bytes()...), compacted.String(), nil
}

func commandQuery(opts Options) (url.Values, error) {
	query := url.Values{}
	for _, item := range commandValues(opts, "--query") {
		key, value, ok := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("--query must be name=value")
		}
		query.Add(key, value)
	}
	return query, nil
}

func parsePublicRESTPath(rawPath string) (jira.API, []string, string, error) {
	if rawPath == "" {
		return "", nil, "", fmt.Errorf("api path is required")
	}
	if strings.Contains(rawPath, "?") || strings.Contains(rawPath, "#") {
		return "", nil, "", fmt.Errorf("api path must not include query or fragment; use --query")
	}
	if strings.HasPrefix(rawPath, "//") {
		return "", nil, "", fmt.Errorf("api path must be a Jira REST path, not a scheme-relative URL")
	}
	if parsed, err := url.Parse(rawPath); err == nil && parsed.Scheme != "" {
		return "", nil, "", fmt.Errorf("api path must be a Jira REST path, not an absolute URL")
	}
	lower := strings.ToLower(rawPath)
	if strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c") {
		return "", nil, "", fmt.Errorf("api path must not contain encoded slash")
	}

	trimmed := strings.Trim(rawPath, "/")
	parts := strings.Split(trimmed, "/")
	api := jira.PlatformAPI
	switch {
	case len(parts) >= 3 && parts[0] == "rest" && parts[1] == "api" && parts[2] == "2":
		parts = parts[3:]
	case len(parts) >= 3 && parts[0] == "rest" && parts[1] == "api" && parts[2] == "latest":
		return "", nil, "", fmt.Errorf("api path must use /rest/api/2, not /rest/api/latest")
	case len(parts) >= 4 && parts[0] == "rest" && parts[1] == "agile" && parts[2] == "1.0":
		api = jira.AgileAPI
		parts = parts[3:]
	case len(parts) >= 2 && parts[0] == "api" && parts[1] == "2":
		parts = parts[2:]
	case len(parts) >= 2 && parts[0] == "api" && parts[1] == "latest":
		return "", nil, "", fmt.Errorf("api path must use api/2, not api/latest")
	case len(parts) >= 2 && parts[0] == "agile" && parts[1] == "1.0":
		api = jira.AgileAPI
		parts = parts[2:]
	case len(parts) >= 2 && parts[0] == "rest" && parts[1] == "auth":
		return "", nil, "", fmt.Errorf("api pass-through does not support auth-session endpoints")
	case len(parts) >= 1 && parts[0] == "rest":
		return "", nil, "", fmt.Errorf("api path must be under /rest/api/2 or /rest/agile/1.0")
	}
	if len(parts) == 0 {
		return "", nil, "", fmt.Errorf("api path must include a resource")
	}
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		decoded, err := url.PathUnescape(part)
		if err != nil {
			return "", nil, "", fmt.Errorf("api path contains invalid escape")
		}
		if decoded == "" || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, `/\`) {
			return "", nil, "", fmt.Errorf("unsafe api path segment %q", decoded)
		}
		segments = append(segments, decoded)
	}
	prefix := "/rest/api/2"
	if api == jira.AgileAPI {
		prefix = "/rest/agile/1.0"
	}
	return api, segments, prefix + "/" + strings.Join(segments, "/"), nil
}

func validateAPIPassThroughEndpoint(api jira.API, method string, segments []string, riskyWriteAllowed bool) error {
	if api == jira.AgileAPI {
		if method != http.MethodGet && !riskyWriteAllowed {
			return fmt.Errorf("api pass-through Agile writes require --force in addition to --yes; prefer typed Agile commands")
		}
		return nil
	}
	if api != jira.PlatformAPI {
		return nil
	}
	lower := make([]string, 0, len(segments))
	for _, segment := range segments {
		lower = append(lower, strings.ToLower(segment))
	}
	if method != http.MethodGet && isAvatarBinaryEndpoint(lower) {
		return fmt.Errorf("api pass-through does not support avatar upload endpoints")
	}
	if len(lower) >= 3 && lower[0] == "issue" && lower[2] == "attachments" {
		return fmt.Errorf("api pass-through does not support attachment upload endpoints")
	}
	if len(lower) >= 1 && lower[0] == "attachment" {
		if len(lower) >= 2 && lower[1] == "content" {
			return fmt.Errorf("api pass-through does not support attachment content endpoints; use jira attachment download")
		}
		if method != http.MethodGet {
			return fmt.Errorf("api pass-through does not support attachment write endpoints; use jira attachment")
		}
		return fmt.Errorf("api pass-through does not support attachment endpoints; use jira attachment")
	}
	return nil
}

func isAvatarBinaryEndpoint(segments []string) bool {
	for _, segment := range segments {
		switch segment {
		case "temporary", "temp", "upload", "crop", "cropping":
			return true
		}
	}
	return false
}

func safeAttachmentContentURL(baseURL, contentURL string) (string, error) {
	if strings.TrimSpace(contentURL) == "" {
		return "", fmt.Errorf("attachment metadata does not include a content URL")
	}
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return "", fmt.Errorf("invalid Jira base URL")
	}
	content, err := url.Parse(contentURL)
	if err != nil {
		return "", fmt.Errorf("invalid attachment content URL")
	}
	if content.User != nil {
		return "", fmt.Errorf("attachment content URL must not include credentials")
	}
	if !content.IsAbs() {
		content = base.ResolveReference(content)
	}
	if !strings.EqualFold(content.Scheme, base.Scheme) || !strings.EqualFold(content.Host, base.Host) {
		return "", fmt.Errorf("attachment content URL must stay on the Jira host")
	}
	if content.Fragment != "" {
		return "", fmt.Errorf("attachment content URL must not include a fragment")
	}
	basePath := "/" + strings.Trim(base.EscapedPath(), "/")
	if basePath != "/" {
		contentPath := content.EscapedPath()
		if contentPath != basePath && !strings.HasPrefix(contentPath, basePath+"/") {
			return "", fmt.Errorf("attachment content URL must stay under the Jira base path")
		}
	}
	return content.String(), nil
}

func dashboardItemPropertySegments(dashboardID, itemID string) []string {
	return []string{"dashboard", dashboardID, "items", itemID, "properties"}
}

func rawDashboardList(body []byte) []json.RawMessage {
	var payload struct {
		Dashboards []json.RawMessage `json:"dashboards"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return payload.Dashboards
}

func rawJSONBody(body []byte) json.RawMessage {
	if !json.Valid(body) {
		return nil
	}
	return append(json.RawMessage(nil), body...)
}

func compactJSONRaw(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "null"
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, raw); err != nil {
		return clean(string(raw))
	}
	return compacted.String()
}

func compactPlanValue(value any) string {
	switch typed := value.(type) {
	case json.RawMessage:
		return compactJSONRaw(typed)
	case map[string]any, []any:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", value)
		}
		return compactJSONRaw(encoded)
	default:
		return fmt.Sprintf("%v", value)
	}
}

func compactCreateMeta(body []byte, project, issueType string) []string {
	lines := []string{fmt.Sprintf("OK createmeta project=%s issue-type=%s", project, issueType)}
	meta, ok := parseCreateMeta(body)
	if !ok {
		return lines
	}
	var selected *createMetaIssueType
	for i := range meta.Projects {
		for j := range meta.Projects[i].IssueTypes {
			if strings.EqualFold(meta.Projects[i].IssueTypes[j].Name, issueType) || meta.Projects[i].IssueTypes[j].ID == issueType {
				selected = &meta.Projects[i].IssueTypes[j]
				break
			}
		}
		if selected != nil {
			break
		}
	}
	if selected == nil {
		lines = append(lines, "0 issue-types matched")
		return lines
	}
	required := requiredCreateFields(*selected)
	if len(required) == 0 {
		lines = append(lines, "0 additional required fields")
		return lines
	}
	lines = append(lines, fmt.Sprintf("%d additional required fields", len(required)))
	for _, id := range required {
		field := selected.Fields[id]
		line := fmt.Sprintf("%s %s", id, clean(firstNonEmpty(field.Name, id)))
		if allowed := compactAllowedValues(field); allowed != "" {
			line += " allowed=" + allowed
		}
		if example := createFieldExample(id, field); example != "" {
			line += " example=" + example
		}
		lines = append(lines, line)
	}
	return lines
}

func createMetaIssueTypeCount(body []byte) int {
	meta, ok := parseCreateMeta(body)
	if !ok {
		return -1
	}
	count := 0
	for _, project := range meta.Projects {
		count += len(project.IssueTypes)
	}
	return count
}

func createMetaIssueTypeNames(body []byte) []string {
	meta, ok := parseCreateMeta(body)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	var names []string
	for _, project := range meta.Projects {
		for _, issueType := range project.IssueTypes {
			name := firstNonEmpty(issueType.Name, issueType.ID)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func parseCreateMeta(body []byte) (createMetaSummary, bool) {
	var meta createMetaSummary
	if err := json.Unmarshal(body, &meta); err != nil {
		return createMetaSummary{}, false
	}
	return meta, true
}

func requiredCreateFields(issueType createMetaIssueType) []string {
	keys := make([]string, 0, len(issueType.Fields))
	for id, field := range issueType.Fields {
		if !field.Required {
			continue
		}
		switch id {
		case "project", "issuetype", "summary":
			continue
		}
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

func compactAllowedValues(field createMetaField) string {
	values := field.AllowedValue
	if len(values) == 0 {
		return ""
	}
	values = recommendedFirstAllowedValues(field)
	limit := minPositive(3, len(values))
	parts := make([]string, 0, limit+1)
	for _, value := range values[:limit] {
		id, name := allowedValueIDName(value)
		parts = append(parts, firstNonEmpty(joinIDName(id, name), id, name, "-"))
	}
	if len(values) > limit {
		parts = append(parts, fmt.Sprintf("+%d more", len(values)-limit))
	}
	return strings.Join(parts, ",")
}

func recommendedFirstAllowedValues(field createMetaField) []map[string]any {
	recommended, ok := recommendedAllowedValue(field)
	if !ok {
		return field.AllowedValue
	}
	values := []map[string]any{recommended}
	recID, recName := allowedValueIDName(recommended)
	for _, value := range field.AllowedValue {
		id, name := allowedValueIDName(value)
		if id == recID && name == recName {
			continue
		}
		values = append(values, value)
	}
	return values
}

func createFieldExample(id string, field createMetaField) string {
	if id == "description" || field.Schema.System == "description" {
		return "--body '...'"
	}
	if field.Schema.System == "duedate" || id == "duedate" {
		return "--due YYYY-MM-DD"
	}
	if value, ok := recommendedAllowedValue(field); ok {
		ref := recommendedReferenceValue(value)
		switch id {
		case "components":
			return "--component " + ref
		case "versions":
			return "--version " + ref
		case "priority":
			return "--priority " + ref
		default:
			return "--field " + id + "=" + shellJSONExample(createFieldJSONValue(field, value))
		}
	}
	switch field.Schema.Type {
	case "array":
		return "--field " + id + "=" + shellJSONExample([]any{})
	case "number", "integer":
		return "--field " + id + "=0"
	default:
		return "--field " + id + "=VALUE"
	}
}

func recommendedAllowedValue(field createMetaField) (map[string]any, bool) {
	if len(field.DefaultValue) > 0 {
		return field.DefaultValue, true
	}
	var fallback map[string]any
	for _, value := range field.AllowedValue {
		if fallback == nil {
			fallback = value
		}
		archived, _ := value["archived"].(bool)
		released, hasReleased := value["released"].(bool)
		overdue, _ := value["overdue"].(bool)
		if !archived && !overdue && (!hasReleased || !released) {
			return value, true
		}
	}
	for _, value := range field.AllowedValue {
		archived, _ := value["archived"].(bool)
		released, hasReleased := value["released"].(bool)
		if !archived && (!hasReleased || !released) {
			return value, true
		}
	}
	if fallback != nil {
		return fallback, true
	}
	return nil, false
}

func createFieldJSONValue(field createMetaField, value map[string]any) any {
	id, name := allowedValueIDName(value)
	item := map[string]string{}
	if id != "" {
		item["id"] = id
	} else if name != "" {
		item["name"] = name
	}
	if field.Schema.Type == "array" {
		return []map[string]string{item}
	}
	return item
}

func recommendedReferenceValue(value map[string]any) string {
	id, name := allowedValueIDName(value)
	return firstNonEmpty(id, name, "VALUE")
}

func shellJSONExample(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "VALUE"
	}
	return "'" + string(encoded) + "'"
}

func allowedValueIDName(value map[string]any) (string, string) {
	id, _ := value["id"].(string)
	name, _ := value["name"].(string)
	key, _ := value["key"].(string)
	return firstNonEmpty(id, key), name
}

func joinIDName(id, name string) string {
	switch {
	case id != "" && name != "":
		return id + ":" + clean(name)
	case id != "":
		return id
	case name != "":
		return clean(name)
	default:
		return ""
	}
}

func commandDays(opts Options, defaultDays int) (int, error) {
	raw := commandValue(opts, "--days")
	if raw == "" {
		return defaultDays, nil
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days <= 0 {
		return 0, fmt.Errorf("--days must be a positive integer")
	}
	return days, nil
}

func explicitIssueKeys(opts Options, positional []string) ([]string, error) {
	issues := append([]string{}, commandValues(opts, "--issue")...)
	issues = append(issues, positional...)
	if len(issues) == 0 {
		return nil, fmt.Errorf("at least one --issue KEY is required")
	}
	for _, issue := range issues {
		if !validIssueKey(issue) {
			return nil, fmt.Errorf("issue key %q must be explicit", issue)
		}
	}
	return issues, nil
}

func validIssueKey(key string) bool {
	if key == "" || strings.ContainsAny(key, " *\t\n\r") {
		return false
	}
	project, number, ok := strings.Cut(key, "-")
	if !ok || project == "" || number == "" {
		return false
	}
	for _, r := range project {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	for _, r := range number {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isBlockingLink(link jira.IssueLink) bool {
	if link.Type == nil {
		return false
	}
	value := strings.ToLower(strings.Join([]string{link.Type.Name, link.Type.Inward, link.Type.Outward}, " "))
	return strings.Contains(value, "block")
}

func jqlValue(value string) string {
	if value == "" {
		return `""`
	}
	simple := true
	for _, r := range value {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '-' {
			simple = false
			break
		}
	}
	if simple {
		return value
	}
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func namedName(value *jira.NamedValue) string {
	if value == nil {
		return ""
	}
	return firstNonEmpty(value.Name, value.Key, value.ID)
}

func userName(user *jira.User) string {
	if user == nil {
		return ""
	}
	return firstNonEmpty(user.Name, user.Key, user.DisplayName)
}

func projectKey(project *jira.Project) string {
	if project == nil {
		return ""
	}
	return firstNonEmpty(project.Key, project.ID, project.Name)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return clean(value)
}

func clean(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	if value == "" {
		return "-"
	}
	return strings.Join(strings.Fields(value), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func minPositive(a, b int) int {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}
