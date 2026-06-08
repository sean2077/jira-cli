package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sean2077/jira-cli/internal/commands"
	"github.com/sean2077/jira-cli/internal/config"
	"github.com/sean2077/jira-cli/internal/output"
	"github.com/spf13/cobra"
)

var Version = "1.1.0-dev"

type Options = commands.Options
type Runtime = commands.Runtime

func DefaultOptions() Options {
	return Options{
		Type:     "server",
		Compact:  true,
		PageSize: 50,
		Limit:    50,
		Timeout:  30 * time.Second,
	}
}

func Main(args []string, stdout, stderr io.Writer) int {
	return MainWithRuntime(args, stdout, stderr, DefaultRuntime())
}

func MainWithRuntime(args []string, stdout, stderr io.Writer, rt Runtime) int {
	opts := DefaultOptions()
	exitCode := 0
	root := newRootCommand(&opts, stdout, stderr, rt, &exitCode)
	root.SetArgs(args)
	cmd, err := root.ExecuteContextC(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		fmt.Fprintf(stderr, "HINT use %q\n", helpCommandPath(cmd))
		return 1
	}
	return exitCode
}

func DefaultRuntime() Runtime {
	rt := Runtime{
		Env:        config.EnvMap(os.Environ()),
		HTTPClient: http.DefaultClient,
	}
	home, err := os.UserHomeDir()
	if err != nil {
		rt.ProfileLoadError = err
		return rt
	}
	profiles, err := config.LoadProfileConfig(filepath.Join(home, ".config", "jira-cli", "config.toml"))
	if err != nil {
		rt.ProfileLoadError = err
		return rt
	}
	rt.Profiles = profiles
	return rt
}

func newRootCommand(opts *Options, stdout, stderr io.Writer, rt Runtime, exitCode *int) *cobra.Command {
	actions := commandActions()
	run := func(path ...string) func(*cobra.Command, []string) error {
		return func(cmd *cobra.Command, args []string) error {
			if err := validateOptions(*opts); err != nil {
				return err
			}
			action, ok := actions[strings.Join(path, " ")]
			if !ok {
				return fmt.Errorf("internal command action is not wired for %q", strings.Join(path, " "))
			}
			opts.BodySet = cmd.Flags().Changed("body")
			mode := outputMode(*opts)
			*exitCode = action(cmd.Context(), *opts, args, stdout, stderr, rt, mode)
			return nil
		}
	}

	root := &cobra.Command{
		Use:           "jira",
		Short:         "agent-first Jira Server CLI",
		Long:          "jira is a compact Jira Server 8.1 oriented CLI for agent workflows.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.DisableDefaultCmd = true
	root.SuggestionsMinimumDistance = 2
	root.SetHelpTemplate(helpTemplate())
	addGlobalFlags(root, opts)

	root.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print jira-cli version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOptions(*opts); err != nil {
				return err
			}
			if err := output.WriteVersion(stdout, outputMode(*opts), Version); err != nil {
				fmt.Fprintf(stderr, "ERR output %s\n", err)
				*exitCode = 1
			}
			return nil
		},
	})

	configCmd := parent("config", "Manage local Jira CLI configuration")
	configCmd.AddCommand(leaf("doctor", "Validate resolved configuration", cobra.NoArgs, run("config", "doctor")))
	root.AddCommand(configCmd)

	skillCmd := parent("skill", "Install bundled agent skills")
	skillInstall := leaf("install", "Install the bundled jira-cli skill", cobra.NoArgs, run("skill", "install"))
	addStringFlag(skillInstall, &opts.Target, "target", "install root directory")
	addBoolFlag(skillInstall, &opts.Global, "global", "install under the user skill directory")
	root.PersistentFlags().BoolVar(&opts.Force, "force", false, "force replacement where supported")
	skillCmd.AddCommand(skillInstall)
	root.AddCommand(skillCmd)

	probe := leaf("probe", "Probe Jira API capabilities", cobra.NoArgs, run("probe"))
	addStringFlag(probe, &opts.Project, "project", "project key for create metadata probe")
	addStringFlag(probe, &opts.IssueType, "issue-type", "issue type name for create metadata probe")
	addStringArrayFlag(probe, &opts.Issues, "issue", "issue key for edit metadata probe")
	root.AddCommand(probe)
	root.AddCommand(leaf("whoami", "Show authenticated Jira user", cobra.NoArgs, run("whoami")))

	api := parent("api", "Call supported public Jira REST endpoints")
	for _, method := range []string{"get", "post", "put", "delete"} {
		cmd := leaf(method+" PATH", "Call Jira REST "+strings.ToUpper(method), cobra.ExactArgs(1), run("api", method))
		addStringArrayFlag(cmd, &opts.Queries, "query", "query parameter as name=value")
		if method != "get" {
			addStringFlag(cmd, &opts.Body, "body", "JSON request body")
		}
		api.AddCommand(cmd)
	}
	root.AddCommand(api)

	root.AddCommand(leaf("search JQL", "Search issues by JQL", cobra.ExactArgs(1), run("search")))
	root.AddCommand(issueCommand(run, opts))
	root.AddCommand(leaf("comments KEY", "List issue comments", cobra.ExactArgs(1), run("comments")))
	root.AddCommand(commentCommand(run, opts))
	root.AddCommand(leaf("worklogs KEY", "List issue worklogs", cobra.ExactArgs(1), run("worklogs")))
	root.AddCommand(worklogCommand(run, opts))
	root.AddCommand(leaf("attachments KEY", "List issue attachments", cobra.ExactArgs(1), run("attachments")))
	root.AddCommand(attachmentCommand(run, opts))
	root.AddCommand(leaf("links KEY", "List issue links", cobra.ExactArgs(1), run("links")))
	root.AddCommand(linkCommand(run, opts))
	root.AddCommand(leaf("remote-links KEY", "List remote issue links", cobra.ExactArgs(1), run("remote-links")))
	root.AddCommand(leaf("remote-link KEY ID", "Show one remote issue link", cobra.ExactArgs(2), run("remote-link")))

	root.AddCommand(leaf("projects", "List projects", cobra.NoArgs, run("projects")))
	root.AddCommand(leaf("project KEY", "Show a project", cobra.ExactArgs(1), run("project")))
	root.AddCommand(leaf("components PROJECT", "List project components", cobra.ExactArgs(1), run("components")))
	root.AddCommand(leaf("versions PROJECT", "List project versions", cobra.ExactArgs(1), run("versions")))
	root.AddCommand(leaf("roles PROJECT", "List project roles", cobra.ExactArgs(1), run("roles")))
	root.AddCommand(leaf("role PROJECT ROLE_ID", "Show a project role", cobra.ExactArgs(2), run("role")))
	root.AddCommand(leaf("project-statuses PROJECT", "List project statuses", cobra.ExactArgs(1), run("project-statuses")))
	root.AddCommand(leaf("fields", "List fields", cobra.NoArgs, run("fields")))

	users := parent("users", "User lookup commands")
	usersSearch := leaf("search", "Search users", cobra.NoArgs, run("users", "search"))
	addStringArrayFlag(usersSearch, &opts.Queries, "query", "user search query")
	users.AddCommand(usersSearch)
	root.AddCommand(users)

	assignable := leaf("assignable", "List assignable users", cobra.NoArgs, run("assignable"))
	addStringFlag(assignable, &opts.Project, "project", "project key")
	addStringArrayFlag(assignable, &opts.Queries, "query", "username query")
	root.AddCommand(assignable)

	root.AddCommand(leaf("issuetypes", "List issue types", cobra.NoArgs, run("issuetypes")))
	root.AddCommand(leaf("priorities", "List priorities", cobra.NoArgs, run("priorities")))
	root.AddCommand(leaf("statuses", "List statuses", cobra.NoArgs, run("statuses")))
	root.AddCommand(leaf("resolutions", "List resolutions", cobra.NoArgs, run("resolutions")))
	root.AddCommand(leaf("resolution ID", "Show a resolution", cobra.ExactArgs(1), run("resolution")))
	root.AddCommand(leaf("workflows", "List workflows", cobra.NoArgs, run("workflows")))
	root.AddCommand(leaf("filters", "List favourite filters", cobra.NoArgs, run("filters")))
	root.AddCommand(leaf("filter ID", "Show a filter", cobra.ExactArgs(1), run("filter")))
	root.AddCommand(leaf("permissions", "List global permissions", cobra.NoArgs, run("permissions")))
	mypermissions := leaf("mypermissions", "List current user permissions", cobra.NoArgs, run("mypermissions"))
	addStringFlag(mypermissions, &opts.Project, "project", "project key")
	addStringArrayFlag(mypermissions, &opts.Issues, "issue", "issue key")
	root.AddCommand(mypermissions)

	create := leaf("create", "Create an issue or inspect create metadata", cobra.NoArgs, run("create"))
	addCreateFlags(create, opts)
	root.AddCommand(create)
	update := leaf("update KEY", "Update issue fields", cobra.ExactArgs(1), run("update"))
	addIssueEditFlags(update, opts)
	root.AddCommand(update)
	assign := leaf("assign KEY", "Assign an issue", cobra.ExactArgs(1), run("assign"))
	addStringFlag(assign, &opts.Assignee, "assignee", "assignee username")
	root.AddCommand(assign)
	root.AddCommand(leaf("transitions KEY", "List available issue transitions", cobra.ExactArgs(1), run("transitions")))
	transition := leaf("transition KEY", "Transition an issue", cobra.ExactArgs(1), run("transition"))
	addStringFlag(transition, &opts.ID, "id", "transition id")
	addStringFlag(transition, &opts.Name, "name", "transition name")
	addStringFlag(transition, &opts.Comment, "comment", "transition comment")
	root.AddCommand(transition)
	root.AddCommand(leaf("delete KEY", "Delete an issue", cobra.ExactArgs(1), run("delete")))
	root.AddCommand(leaf("watchers KEY", "Show issue watchers", cobra.ExactArgs(1), run("watchers")))
	root.AddCommand(leaf("watch KEY", "Watch an issue", cobra.ExactArgs(1), run("watch")))
	root.AddCommand(leaf("unwatch KEY", "Stop watching an issue", cobra.ExactArgs(1), run("unwatch")))

	bulk := parent("bulk", "Bulk operations")
	bulk.AddCommand(leaf("search JQL [JQL...]", "Run multiple searches", cobra.MinimumNArgs(1), run("bulk", "search")))
	root.AddCommand(bulk)

	root.AddCommand(moveCommand(run, opts))

	mine := leaf("mine", "Search work relevant to the current user", cobra.NoArgs, run("mine"))
	addStringFlag(mine, &opts.Days, "days", "lookback days")
	root.AddCommand(mine)
	stale := leaf("stale", "Search stale unresolved project issues", cobra.NoArgs, run("stale"))
	addStringFlag(stale, &opts.Project, "project", "project key")
	addStringFlag(stale, &opts.Days, "days", "age threshold in days")
	addStringFlag(stale, &opts.Status, "status", "status name")
	root.AddCommand(stale)
	blockers := leaf("blockers", "List blocking links for an issue", cobra.NoArgs, run("blockers"))
	addStringArrayFlag(blockers, &opts.Issues, "issue", "issue key")
	root.AddCommand(blockers)

	dashboards := leaf("dashboards", "List dashboards", cobra.NoArgs, run("dashboards"))
	addStringFlag(dashboards, &opts.Filter, "filter", "dashboard filter: favourite or my")
	root.AddCommand(dashboards)
	root.AddCommand(dashboardCommand(run, opts))

	root.AddCommand(leaf("boards", "List Agile boards", cobra.NoArgs, run("boards")))
	root.AddCommand(boardCommand(run))
	root.AddCommand(leaf("backlog BOARD_ID", "List board backlog issues", cobra.ExactArgs(1), run("backlog")))
	root.AddCommand(leaf("sprints BOARD_ID", "List board sprints", cobra.ExactArgs(1), run("sprints")))
	root.AddCommand(sprintCommand(run, opts))
	root.AddCommand(epicCommand(run))

	return root
}

func commandActions() map[string]commands.Action {
	return map[string]commands.Action{
		"config doctor":                  commands.RunConfigDoctor,
		"skill install":                  commands.RunSkillInstall,
		"probe":                          commands.RunProbe,
		"whoami":                         commands.RunWhoami,
		"api get":                        commands.RunAPIGet,
		"api post":                       commands.RunAPIPost,
		"api put":                        commands.RunAPIPut,
		"api delete":                     commands.RunAPIDelete,
		"search":                         commands.RunSearch,
		"issue":                          commands.RunIssue,
		"issue properties":               commands.RunIssuePropertyKeys,
		"issue property":                 commands.RunIssuePropertyGet,
		"issue property set":             commands.RunIssuePropertySet,
		"issue property delete":          commands.RunIssuePropertyDelete,
		"comments":                       commands.RunComments,
		"comment":                        commands.RunCommentAdd,
		"comment get":                    commands.RunCommentGet,
		"worklogs":                       commands.RunWorklogs,
		"worklog add":                    commands.RunWorklogAdd,
		"worklog get":                    commands.RunWorklogGet,
		"attachments":                    commands.RunAttachments,
		"attachment":                     commands.RunAttachmentGet,
		"attachment get":                 commands.RunAttachmentGet,
		"attachment add":                 commands.RunAttachmentAdd,
		"attachment download":            commands.RunAttachmentDownload,
		"attachment delete":              commands.RunAttachmentDelete,
		"links":                          commands.RunLinks,
		"link create":                    commands.RunLinkCreate,
		"link delete":                    commands.RunLinkDelete,
		"remote-links":                   commands.RunRemoteLinks,
		"remote-link":                    commands.RunRemoteLink,
		"projects":                       commands.RunProjects,
		"project":                        commands.RunProject,
		"components":                     commands.RunComponents,
		"versions":                       commands.RunVersions,
		"roles":                          commands.RunRoles,
		"role":                           commands.RunRole,
		"project-statuses":               commands.RunProjectStatuses,
		"fields":                         commands.RunFields,
		"users search":                   commands.RunUsersSearch,
		"assignable":                     commands.RunAssignableUsers,
		"issuetypes":                     commands.RunIssueTypes,
		"priorities":                     commands.RunPriorities,
		"statuses":                       commands.RunStatuses,
		"resolutions":                    commands.RunResolutions,
		"resolution":                     commands.RunResolution,
		"workflows":                      commands.RunWorkflows,
		"filters":                        commands.RunFilters,
		"filter":                         commands.RunFilter,
		"permissions":                    commands.RunPermissions,
		"mypermissions":                  commands.RunMyPermissions,
		"create":                         commands.RunCreate,
		"update":                         commands.RunUpdate,
		"assign":                         commands.RunAssign,
		"transitions":                    commands.RunTransitions,
		"transition":                     commands.RunTransition,
		"delete":                         commands.RunDelete,
		"watchers":                       commands.RunWatchers,
		"watch":                          commands.RunWatch,
		"unwatch":                        commands.RunUnwatch,
		"bulk search":                    commands.RunBulkSearch,
		"move sprint":                    commands.RunMoveSprint,
		"move backlog":                   commands.RunMoveBacklog,
		"mine":                           commands.RunMine,
		"stale":                          commands.RunStale,
		"blockers":                       commands.RunBlockers,
		"dashboards":                     commands.RunDashboards,
		"dashboard":                      commands.RunDashboard,
		"dashboard item properties":      commands.RunDashboardItemPropertyKeys,
		"dashboard item property":        commands.RunDashboardItemPropertyGet,
		"dashboard item property set":    commands.RunDashboardItemPropertySet,
		"dashboard item property delete": commands.RunDashboardItemPropertyDelete,
		"boards":                         commands.RunBoards,
		"board":                          commands.RunBoard,
		"board issues":                   commands.RunBoardIssues,
		"backlog":                        commands.RunBacklog,
		"sprints":                        commands.RunSprints,
		"sprint":                         commands.RunSprint,
		"sprint issues":                  commands.RunSprintIssues,
		"sprint summary":                 commands.RunSprintSummary,
		"epic":                           commands.RunEpic,
		"epic issues":                    commands.RunEpicIssues,
	}
}

func outputMode(opts Options) output.Mode {
	switch {
	case opts.JSON:
		return output.JSON
	case opts.Raw:
		return output.Raw
	case opts.Compact:
		return output.Compact
	default:
		return output.Compact
	}
}

func parent(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Annotations:   map[string]string{"jira-cli-parent": "true"},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("expected a subcommand; use %q", cmd.CommandPath()+" --help")
		},
	}
}

func leaf(use, short string, args cobra.PositionalArgs, run func(*cobra.Command, []string) error) *cobra.Command {
	return &cobra.Command{
		Use:           use,
		Short:         short,
		Args:          args,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          run,
	}
}

func issueCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	issue := leaf("issue KEY", "Show an issue", cobra.ExactArgs(1), run("issue"))
	properties := leaf("properties KEY", "List issue property keys", cobra.ExactArgs(1), run("issue", "properties"))
	property := leaf("property KEY PROPERTY", "Show an issue property", cobra.ExactArgs(2), run("issue", "property"))
	set := leaf("set KEY PROPERTY", "Set an issue property", cobra.ExactArgs(2), run("issue", "property", "set"))
	deleteCmd := leaf("delete KEY PROPERTY", "Delete an issue property", cobra.ExactArgs(2), run("issue", "property", "delete"))
	addStringFlag(set, &opts.Body, "body", "JSON property body")
	property.AddCommand(set, deleteCmd)
	issue.AddCommand(properties, property)
	return issue
}

func commentCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	comment := leaf("comment KEY", "Add an issue comment", cobra.ExactArgs(1), run("comment"))
	addStringFlag(comment, &opts.Body, "body", "comment body")
	comment.AddCommand(leaf("get KEY ID", "Show one issue comment", cobra.ExactArgs(2), run("comment", "get")))
	return comment
}

func worklogCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	worklog := parent("worklog", "Issue worklog commands")
	add := leaf("add KEY", "Add an issue worklog", cobra.ExactArgs(1), run("worklog", "add"))
	addStringFlag(add, &opts.Time, "time", "time spent, for example 1h")
	addStringFlag(add, &opts.Comment, "comment", "worklog comment")
	worklog.AddCommand(add, leaf("get KEY ID", "Show one worklog", cobra.ExactArgs(2), run("worklog", "get")))
	return worklog
}

func attachmentCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	attachment := leaf("attachment ID", "Show attachment metadata", cobra.ExactArgs(1), run("attachment"))
	add := leaf("add KEY", "Upload an issue attachment", cobra.ExactArgs(1), run("attachment", "add"))
	addStringFlag(add, &opts.File, "file", "file path")
	get := leaf("get ID", "Show attachment metadata", cobra.ExactArgs(1), run("attachment", "get"))
	download := leaf("download ID", "Download attachment content", cobra.ExactArgs(1), run("attachment", "download"))
	addStringFlag(download, &opts.File, "file", "output file path")
	deleteCmd := leaf("delete ID", "Delete an attachment", cobra.ExactArgs(1), run("attachment", "delete"))
	attachment.AddCommand(add, get, download, deleteCmd)
	return attachment
}

func linkCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	link := parent("link", "Issue link commands")
	create := leaf("create SOURCE_KEY", "Create an issue link", cobra.ExactArgs(1), run("link", "create"))
	addStringFlag(create, &opts.Target, "target", "target issue key")
	addStringFlag(create, &opts.LinkType, "link-type", "Jira link type")
	link.AddCommand(create, leaf("delete ID", "Delete an issue link", cobra.ExactArgs(1), run("link", "delete")))
	return link
}

func moveCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	move := parent("move", "Agile issue move commands")
	sprint := leaf("sprint [SPRINT_ID] [ISSUE...]", "Move issues to a sprint", cobra.ArbitraryArgs, run("move", "sprint"))
	addStringFlag(sprint, &opts.Sprint, "sprint", "sprint id")
	addStringArrayFlag(sprint, &opts.Issues, "issue", "issue key")
	backlog := leaf("backlog [ISSUE...]", "Move issues to the backlog", cobra.ArbitraryArgs, run("move", "backlog"))
	addStringArrayFlag(backlog, &opts.Issues, "issue", "issue key")
	move.AddCommand(sprint, backlog)
	return move
}

func dashboardCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	dashboard := leaf("dashboard ID", "Show a dashboard", cobra.ExactArgs(1), run("dashboard"))
	item := parent("item", "Dashboard item commands")
	item.AddCommand(leaf("properties DASHBOARD_ID ITEM_ID", "List dashboard item property keys", cobra.ExactArgs(2), run("dashboard", "item", "properties")))
	property := leaf("property DASHBOARD_ID ITEM_ID KEY", "Show a dashboard item property", cobra.ExactArgs(3), run("dashboard", "item", "property"))
	set := leaf("set DASHBOARD_ID ITEM_ID KEY", "Set a dashboard item property", cobra.ExactArgs(3), run("dashboard", "item", "property", "set"))
	addStringFlag(set, &opts.Body, "body", "JSON property body")
	property.AddCommand(set, leaf("delete DASHBOARD_ID ITEM_ID KEY", "Delete a dashboard item property", cobra.ExactArgs(3), run("dashboard", "item", "property", "delete")))
	item.AddCommand(property)
	dashboard.AddCommand(item)
	return dashboard
}

func boardCommand(run func(...string) func(*cobra.Command, []string) error) *cobra.Command {
	board := leaf("board ID", "Show an Agile board", cobra.ExactArgs(1), run("board"))
	board.AddCommand(leaf("issues ID", "List board issues", cobra.ExactArgs(1), run("board", "issues")))
	return board
}

func sprintCommand(run func(...string) func(*cobra.Command, []string) error, opts *Options) *cobra.Command {
	sprint := leaf("sprint ID", "Show an Agile sprint", cobra.ExactArgs(1), run("sprint"))
	sprint.AddCommand(leaf("issues ID", "List sprint issues", cobra.ExactArgs(1), run("sprint", "issues")))
	summary := leaf("summary [ID]", "Summarize sprint issues", cobra.RangeArgs(0, 1), run("sprint", "summary"))
	addStringFlag(summary, &opts.Sprint, "sprint", "sprint id")
	sprint.AddCommand(summary)
	return sprint
}

func epicCommand(run func(...string) func(*cobra.Command, []string) error) *cobra.Command {
	epic := leaf("epic KEY_OR_ID", "Show an Agile epic", cobra.ExactArgs(1), run("epic"))
	epic.AddCommand(leaf("issues KEY_OR_ID", "List epic issues", cobra.ExactArgs(1), run("epic", "issues")))
	return epic
}

func addGlobalFlags(cmd *cobra.Command, opts *Options) {
	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.Profile, "profile", "", "select named profile")
	flags.StringVar(&opts.BaseURL, "base-url", "", "override Jira base URL")
	flags.StringVar(&opts.Type, "type", "server", "Jira type: server or cloud")
	flags.StringVar(&opts.User, "user", "", "override username")
	flags.StringVar(&opts.TokenEnv, "token-env", "", "token/password environment variable")
	flags.Var(outputModeFlag{opts: opts, mode: "json"}, "json", "write stable JSON output")
	flags.Var(outputModeFlag{opts: opts, mode: "raw"}, "raw", "write unmodified Jira response output")
	flags.Var(outputModeFlag{opts: opts, mode: "compact"}, "compact", "write compact output")
	flags.Lookup("json").NoOptDefVal = "true"
	flags.Lookup("raw").NoOptDefVal = "true"
	flags.Lookup("compact").NoOptDefVal = "true"
	flags.IntVar(&opts.Limit, "limit", 50, "max items across pages")
	flags.IntVar(&opts.PageSize, "page-size", 50, "requested page size")
	flags.IntVar(&opts.StartAt, "start-at", 0, "start offset for paged endpoints")
	flags.DurationVar(&opts.Timeout, "timeout", 30*time.Second, "HTTP timeout")
	flags.BoolVar(&opts.DryRun, "dry-run", false, "show intended write without executing")
	flags.BoolVar(&opts.Yes, "yes", false, "confirm write/destructive execution")
}

type outputModeFlag struct {
	opts *Options
	mode string
}

func (flag outputModeFlag) Set(value string) error {
	enabled, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	switch flag.mode {
	case "json":
		flag.opts.JSON = enabled
		if enabled {
			flag.opts.Raw = false
			flag.opts.Compact = false
		}
	case "raw":
		flag.opts.Raw = enabled
		if enabled {
			flag.opts.JSON = false
			flag.opts.Compact = false
		}
	case "compact":
		flag.opts.Compact = enabled
		if enabled {
			flag.opts.JSON = false
			flag.opts.Raw = false
		}
	default:
		return fmt.Errorf("unknown output mode flag %q", flag.mode)
	}
	if !flag.opts.JSON && !flag.opts.Raw {
		flag.opts.Compact = true
	}
	return nil
}

func (flag outputModeFlag) String() string {
	if flag.opts == nil {
		return "false"
	}
	switch flag.mode {
	case "json":
		return strconv.FormatBool(flag.opts.JSON)
	case "raw":
		return strconv.FormatBool(flag.opts.Raw)
	case "compact":
		return strconv.FormatBool(flag.opts.Compact)
	default:
		return "false"
	}
}

func (outputModeFlag) Type() string {
	return "bool"
}

func (outputModeFlag) IsBoolFlag() bool {
	return true
}

func addCreateFlags(cmd *cobra.Command, opts *Options) {
	addStringFlag(cmd, &opts.Project, "project", "project key")
	addStringFlag(cmd, &opts.IssueType, "issue-type", "issue type name")
	addStringFlag(cmd, &opts.Summary, "summary", "issue summary")
	addStringFlag(cmd, &opts.Body, "body", "description text or JSON body, depending on command")
	addIssueEditFlags(cmd, opts)
	addStringArrayFlag(cmd, &opts.Attachments, "attach", "attachment path")
	addBoolFlag(cmd, &opts.Meta, "meta", "show create metadata")
}

func addIssueEditFlags(cmd *cobra.Command, opts *Options) {
	addStringArrayFlag(cmd, &opts.Fields, "field", "field assignment as name=value")
	addStringArrayFlag(cmd, &opts.Components, "component", "component id or name")
	addStringArrayFlag(cmd, &opts.Versions, "version", "version id or name")
	addStringFlag(cmd, &opts.Due, "due", "due date YYYY-MM-DD")
	addStringFlag(cmd, &opts.Priority, "priority", "priority id or name")
}

func addStringFlag(cmd *cobra.Command, target *string, name, usage string) {
	cmd.Flags().StringVar(target, name, "", usage)
}

func addStringArrayFlag(cmd *cobra.Command, target *[]string, name, usage string) {
	cmd.Flags().StringArrayVar(target, name, nil, usage)
}

func addBoolFlag(cmd *cobra.Command, target *bool, name, usage string) {
	cmd.Flags().BoolVar(target, name, false, usage)
}

func validateOptions(opts Options) error {
	if opts.Type != "server" && opts.Type != "cloud" {
		return fmt.Errorf("--type must be server or cloud")
	}
	if opts.Limit <= 0 {
		return fmt.Errorf("--limit must be a positive integer")
	}
	if opts.PageSize <= 0 {
		return fmt.Errorf("--page-size must be a positive integer")
	}
	if opts.StartAt < 0 {
		return fmt.Errorf("--start-at must be a non-negative integer")
	}
	if opts.Timeout <= 0 {
		return fmt.Errorf("--timeout must be a positive duration")
	}
	return nil
}

func helpCommandPath(cmd *cobra.Command) string {
	if cmd == nil {
		return "jira --help"
	}
	return cmd.CommandPath() + " --help"
}

func helpTemplate() string {
	return `{{if eq .CommandPath "jira"}}{{.Short}}

Usage:
  jira <command> [flags]
  jira help <command>

Command Groups:
  Core: probe, whoami, search, issue, comments, worklogs, attachments, links
  Projects: projects, project, components, versions, roles, fields, users, assignable
  Metadata: filters, permissions, mypermissions, issuetypes, priorities, statuses, resolutions, workflows
  Writes: create, update, comment, assign, transition, watch, unwatch, delete
  Attachments: attachment add|get|download|delete
  Properties: issue property, dashboard item property
  Dashboards: dashboards, dashboard
  Agile: boards, board, backlog, sprints, sprint, epic, move
  Utilities: api, bulk, mine, stale, blockers, config, skill, version

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Use "jira help <command>" or "jira <command> --help" for command-specific help.
{{else}}{{.Short}}

Usage:
  {{.UseLine}}{{if .HasAvailableSubCommands}}

Commands:{{range .Commands}}{{if (or .IsAvailableCommand .IsAdditionalHelpTopicCommand)}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}

Use "jira help <command>" or "jira <command> --help" for command-specific help.
{{end}}`
}
