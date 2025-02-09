package cli

import (
	"strings"
	"time"

	"github.com/alexflint/go-arg"
)

var Version = "head"

type Args struct {
	Addr             string              `arg:"env:DOZZLE_ADDR" default:":8080" help:"sets host:port to bind for server. This is rarely needed inside a docker container."`
	Base             string              `arg:"env:DOZZLE_BASE" default:"/" help:"sets the base for http router."`
	Hostname         string              `arg:"env:DOZZLE_HOSTNAME" help:"sets the hostname for display. This is useful with multiple Dozzle instances."`
	Level            string              `arg:"env:DOZZLE_LEVEL" default:"info" help:"set Dozzle log level. Use debug for more logging."`
	AuthProvider     string              `arg:"--auth-provider,env:DOZZLE_AUTH_PROVIDER" default:"none" help:"sets the auth provider to use. Currently only forward-proxy is supported."`
	AuthTTL          string              `arg:"--auth-ttl,env:DOZZLE_AUTH_TTL" default:"session" help:"sets the TTL for the auth token. Accepts duration values like 12h. Valid time units are s, m, h"`
	AuthHeaderUser   string              `arg:"--auth-header-user,env:DOZZLE_AUTH_HEADER_USER" default:"Remote-User" help:"sets the HTTP Header to use for username in Forward Proxy configuration."`
	AuthHeaderEmail  string              `arg:"--auth-header-email,env:DOZZLE_AUTH_HEADER_EMAIL" default:"Remote-Email" help:"sets the HTTP Header to use for email in Forward Proxy configuration."`
	AuthHeaderName   string              `arg:"--auth-header-name,env:DOZZLE_AUTH_HEADER_NAME" default:"Remote-Name" help:"sets the HTTP Header to use for name in Forward Proxy configuration."`
	AuthHeaderFilter string              `arg:"--auth-header-filter,env:DOZZLE_AUTH_HEADER_FILTER" default:"Remote-Filter" help:"sets the HTTP Header to use for filtering in Forward Proxy configuration."`
	EnableActions    bool                `arg:"--enable-actions,env:DOZZLE_ENABLE_ACTIONS" default:"false" help:"enables essential actions on containers from the web interface."`
	FilterStrings    []string            `arg:"env:DOZZLE_FILTER,--filter,separate" help:"filters docker containers using Docker syntax."`
	Filter           map[string][]string `arg:"-"`
	RemoteHost       []string            `arg:"env:DOZZLE_REMOTE_HOST,--remote-host,separate" help:"list of hosts to connect remotely"`
	RemoteAgent      []string            `arg:"env:DOZZLE_REMOTE_AGENT,--remote-agent,separate" help:"list of agents to connect remotely"`
	NoAnalytics      bool                `arg:"--no-analytics,env:DOZZLE_NO_ANALYTICS" help:"disables anonymous analytics"`
	Mode             string              `arg:"env:DOZZLE_MODE" default:"server" help:"sets the mode to run in (server, swarm)"`
	TimeoutString    string              `arg:"--timeout,env:DOZZLE_TIMEOUT" default:"10s" help:"sets the timeout for docker client"`
	Timeout          time.Duration       `arg:"-"`
	Healthcheck      *HealthcheckCmd     `arg:"subcommand:healthcheck" help:"checks if the server is running"`
	Generate         *GenerateCmd        `arg:"subcommand:generate" help:"generates a configuration file for simple auth"`
	Agent            *AgentCmd           `arg:"subcommand:agent" help:"starts the agent"`
	AgentTest        *AgentTestCmd       `arg:"subcommand:agent-test" help:"tests an agent"`
}

type HealthcheckCmd struct {
}

type AgentCmd struct {
	Addr string `arg:"--agent-addr,env:DOZZLE_AGENT_ADDR" default:":7007" help:"sets the host:port to bind for the agent"`
}

type AgentTestCmd struct {
	Address string `arg:"positional"`
}

type GenerateCmd struct {
	Username string `arg:"positional"`
	Password string `arg:"--password, -p" help:"sets the password for the user"`
	Name     string `arg:"--name, -n" help:"sets the display name for the user"`
	Email    string `arg:"--email, -e" help:"sets the email for the user"`
	Filter   string `arg:"--user-filter" help:"sets the filter for the user. This can be a comma separated list of filters."`
}

func (Args) Version() string {
	return Version
}

func ParseArgs() (Args, interface{}) {
	var args Args
	parser := arg.MustParse(&args)

	ConfigureLogger(args.Level)

	args.Filter = make(map[string][]string)

	for _, filter := range args.FilterStrings {
		pos := strings.Index(filter, "=")
		if pos == -1 {
			parser.Fail("each filter should be of the form key=value")
		}
		key := filter[:pos]
		val := filter[pos+1:]
		args.Filter[key] = append(args.Filter[key], val)
	}

	if args.TimeoutString != "" {
		timeout, err := time.ParseDuration(args.TimeoutString)
		if err != nil {
			parser.Fail("timeout should be a valid duration")
		}
		args.Timeout = timeout
	}

	return args, parser.Subcommand()
}
