package cli

import (
	"context"
	"errors"
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
)

var Version = "1.0.0-dev"

type Options struct {
	Profile  string
	BaseURL  string
	Type     string
	User     string
	TokenEnv string

	Help    bool
	JSON    bool
	Raw     bool
	Compact bool
	DryRun  bool
	Yes     bool

	PageSize int
	Limit    int
	StartAt  int
	Timeout  time.Duration

	CommandValues map[string][]string
	CommandBools  map[string]bool
}

type Runtime = commands.Runtime

func DefaultOptions() Options {
	return Options{
		Type:     "server",
		Compact:  true,
		PageSize: 50,
		Limit:    50,
		Timeout:  30 * time.Second,

		CommandValues: map[string][]string{},
		CommandBools:  map[string]bool{},
	}
}

func Main(args []string, stdout, stderr io.Writer) int {
	return MainWithRuntime(args, stdout, stderr, DefaultRuntime())
}

func MainWithRuntime(args []string, stdout, stderr io.Writer, rt Runtime) int {
	opts, positionals, err := ParseArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "ERR usage %s\n", err)
		return 1
	}

	if opts.Help || len(positionals) == 0 || positionals[0] == "help" {
		fmt.Fprint(stdout, HelpText())
		return 0
	}

	switch positionals[0] {
	case "version":
		if err := output.WriteVersion(stdout, output.ModeFromOptions(opts.JSON, opts.Raw), Version); err != nil {
			fmt.Fprintf(stderr, "ERR output %s\n", err)
			return 1
		}
		return 0
	default:
		return commands.Execute(context.Background(), commandOptions(opts), positionals, stdout, stderr, rt)
	}
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

func commandOptions(opts Options) commands.Options {
	return commands.Options{
		Profile:  opts.Profile,
		BaseURL:  opts.BaseURL,
		Type:     opts.Type,
		User:     opts.User,
		TokenEnv: opts.TokenEnv,
		JSON:     opts.JSON,
		Raw:      opts.Raw,
		DryRun:   opts.DryRun,
		Yes:      opts.Yes,
		PageSize: opts.PageSize,
		Limit:    opts.Limit,
		StartAt:  opts.StartAt,
		Timeout:  opts.Timeout,

		CommandValues: opts.CommandValues,
		CommandBools:  opts.CommandBools,
	}
}

func ParseArgs(args []string) (Options, []string, error) {
	opts := DefaultOptions()
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if arg == "-h" {
			opts.Help = true
			continue
		}
		if !strings.HasPrefix(arg, "--") || arg == "--" {
			positionals = append(positionals, arg)
			continue
		}

		name, value, hasValue := strings.Cut(arg, "=")
		takeValue := func() (string, error) {
			if hasValue {
				return value, nil
			}
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return "", fmt.Errorf("%s requires a value", name)
			}
			i++
			return args[i], nil
		}

		switch name {
		case "--help":
			opts.Help = true
		case "--profile":
			v, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			opts.Profile = v
		case "--base-url":
			v, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			opts.BaseURL = v
		case "--type":
			v, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			if v != "server" && v != "cloud" {
				return opts, nil, errors.New("--type must be server or cloud")
			}
			opts.Type = v
		case "--user":
			v, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			opts.User = v
		case "--token-env":
			v, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			opts.TokenEnv = v
		case "--json":
			opts.JSON = true
			opts.Raw = false
			opts.Compact = false
		case "--raw":
			opts.Raw = true
			opts.JSON = false
			opts.Compact = false
		case "--compact":
			opts.Compact = true
			opts.JSON = false
			opts.Raw = false
		case "--page-size":
			v, err := parsePositiveIntFlag(name, takeValue)
			if err != nil {
				return opts, nil, err
			}
			opts.PageSize = v
		case "--limit":
			v, err := parsePositiveIntFlag(name, takeValue)
			if err != nil {
				return opts, nil, err
			}
			opts.Limit = v
		case "--start-at":
			raw, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			v, err := strconv.Atoi(raw)
			if err != nil || v < 0 {
				return opts, nil, fmt.Errorf("%s must be a non-negative integer", name)
			}
			opts.StartAt = v
		case "--timeout":
			raw, err := takeValue()
			if err != nil {
				return opts, nil, err
			}
			v, err := time.ParseDuration(raw)
			if err != nil || v <= 0 {
				return opts, nil, fmt.Errorf("%s must be a positive duration", name)
			}
			opts.Timeout = v
		case "--dry-run":
			opts.DryRun = true
		case "--yes":
			opts.Yes = true
		default:
			if isCommandStringFlag(name) {
				v, err := takeValue()
				if err != nil {
					return opts, nil, err
				}
				opts.CommandValues[name] = append(opts.CommandValues[name], v)
				continue
			}
			if isCommandBoolFlag(name) {
				opts.CommandBools[name] = true
				continue
			}
			return opts, nil, fmt.Errorf("unknown flag %s", name)
		}
	}

	return opts, positionals, nil
}

func isCommandStringFlag(name string) bool {
	switch name {
	case "--project", "--issue-type", "--issue", "--summary", "--body", "--field", "--assignee", "--id", "--name", "--comment", "--time", "--target", "--link-type", "--query", "--days", "--status", "--sprint", "--filter", "--file":
		return true
	default:
		return false
	}
}

func isCommandBoolFlag(name string) bool {
	switch name {
	case "--force", "--global", "--meta":
		return true
	default:
		return false
	}
}

func parsePositiveIntFlag(name string, takeValue func() (string, error)) (int, error) {
	raw, err := takeValue()
	if err != nil {
		return 0, err
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return v, nil
}

func HelpText() string {
	return `jira - agent-first Jira Server CLI

Usage:
  jira <command> [flags]
  jira help

Read-only:
  jira probe | whoami
  jira search '<JQL>' [flags] | bulk search '<JQL>' ['<JQL>'...]
  jira issue <KEY> | comments <KEY> | worklogs <KEY> | attachments <KEY> | links <KEY> | remote-links <KEY>
  jira attachment <ID> | attachment download <ID> --file PATH
  jira transitions <KEY> | watchers <KEY>
  jira projects | project <KEY> | components <PROJECT> | versions <PROJECT> | roles <PROJECT>
  jira fields | issuetypes | priorities | statuses | resolutions | workflows
  jira filters | filter <ID> | users search --query TEXT | assignable --project KEY
  jira permissions | mypermissions [--project KEY|--issue KEY]
  jira create --project KEY --issue-type NAME --meta
  jira api get <PATH> [--query k=v] [--raw]

Agent views:
  jira mine [--days N]
  jira stale --project KEY [--days N --status NAME]
  jira blockers --issue KEY

Dashboards:
  jira dashboards [--filter favourite|my] | dashboard <ID>
  jira dashboard item properties <DASHBOARD_ID> <ITEM_ID>
  jira dashboard item property <DASHBOARD_ID> <ITEM_ID> <KEY>

Agile:
  jira boards | board <ID> | board issues <ID> | backlog <BOARD_ID>
  jira sprints <BOARD_ID> | sprint <ID> | sprint issues <ID>
  jira sprint summary --sprint ID | epic <KEY_OR_ID> | epic issues <KEY_OR_ID>

Writes:
  jira create --project KEY --issue-type NAME --summary TEXT [--body TEXT] [--field k=v] [--dry-run|--yes]
  jira update <KEY> --field k=v [--dry-run|--yes]
  jira comment <KEY> --body TEXT [--dry-run|--yes]
  jira issue property set <KEY> <PROPERTY> --body JSON [--dry-run|--yes]
  jira issue property delete <KEY> <PROPERTY> [--dry-run|--yes]
  jira assign <KEY> --assignee NAME [--dry-run|--yes]
  jira transition <KEY> --id ID [--comment TEXT] [--dry-run|--yes]
  jira watch <KEY> | unwatch <KEY> [--dry-run|--yes]
  jira worklog add <KEY> --time 1h [--comment TEXT] [--dry-run|--yes]
  jira attachment add <KEY> --file PATH [--dry-run|--yes]
  jira attachment delete <ID> [--dry-run|--yes]
  jira link create <SOURCE_KEY> --target KEY [--link-type NAME] [--dry-run|--yes]
  jira link delete <ID> [--dry-run|--yes]
  jira delete <KEY> --yes
  jira move sprint --sprint ID --issue KEY [--dry-run|--yes]
  jira move backlog --issue KEY [--dry-run|--yes]
  jira dashboard item property set <DASHBOARD_ID> <ITEM_ID> <KEY> --body JSON [--dry-run|--yes]
  jira dashboard item property delete <DASHBOARD_ID> <ITEM_ID> <KEY> [--dry-run|--yes]
  jira api post|put|delete <PATH> [--body JSON] [--dry-run|--yes] [--force for Agile writes]

Config:
  jira config doctor
  jira skill install [--global|--target DIR]
  jira version

Examples:
  jira probe --project T1 --issue-type Bug
  jira search 'project = T1 ORDER BY updated DESC' --limit 10
  jira mine --days 3
  jira create --project T1 --issue-type Bug --summary 'Fix login' --dry-run
  jira --json issue T1-123

Global:
  --profile NAME --base-url URL
  --type server|cloud
  --user NAME --token-env NAME
  --json --raw --compact
  --limit N --page-size N --start-at N
  --timeout DURATION
  --dry-run --yes --force

Safety:
  Use --dry-run before writes; add --yes only where the command supports confirmation.
  Run jira probe before assuming core API, dashboard, Agile, fields, or create/edit metadata support.
`
}
