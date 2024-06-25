package main

import (
	"context"
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/healthcheck"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/amir20/dozzle/internal/web"
	"google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

var (
	version = "head"
)

type args struct {
	Addr                 string              `arg:"env:DOZZLE_ADDR" default:":8080" help:"sets host:port to bind for server. This is rarely needed inside a docker container."`
	Base                 string              `arg:"env:DOZZLE_BASE" default:"/" help:"sets the base for http router."`
	Hostname             string              `arg:"env:DOZZLE_HOSTNAME" help:"sets the hostname for display. This is useful with multiple Dozzle instances."`
	Level                string              `arg:"env:DOZZLE_LEVEL" default:"info" help:"set Dozzle log level. Use debug for more logging."`
	AuthProvider         string              `arg:"--auth-provider,env:DOZZLE_AUTH_PROVIDER" default:"none" help:"sets the auth provider to use. Currently only forward-proxy is supported."`
	AuthHeaderUser       string              `arg:"--auth-header-user,env:DOZZLE_AUTH_HEADER_USER" default:"Remote-User" help:"sets the HTTP Header to use for username in Forward Proxy configuration."`
	AuthHeaderEmail      string              `arg:"--auth-header-email,env:DOZZLE_AUTH_HEADER_EMAIL" default:"Remote-Email" help:"sets the HTTP Header to use for email in Forward Proxy configuration."`
	AuthHeaderName       string              `arg:"--auth-header-name,env:DOZZLE_AUTH_HEADER_NAME" default:"Remote-Name" help:"sets the HTTP Header to use for name in Forward Proxy configuration."`
	WaitForDockerSeconds int                 `arg:"--wait-for-docker-seconds,env:DOZZLE_WAIT_FOR_DOCKER_SECONDS" help:"wait for docker to be available for at most this many seconds before starting the server."`
	EnableActions        bool                `arg:"--enable-actions,env:DOZZLE_ENABLE_ACTIONS" default:"false" help:"enables essential actions on containers from the web interface."`
	FilterStrings        []string            `arg:"env:DOZZLE_FILTER,--filter,separate" help:"filters docker containers using Docker syntax."`
	Filter               map[string][]string `arg:"-"`
	RemoteHost           []string            `arg:"env:DOZZLE_REMOTE_HOST,--remote-host,separate" help:"list of hosts to connect remotely"`
	RemoteAgents         []string            `arg:"env:DOZZLE_REMOTE_AGENT,--remote-agent,separate" help:"list of agents to connect remotely"`
	NoAnalytics          bool                `arg:"--no-analytics,env:DOZZLE_NO_ANALYTICS" help:"disables anonymous analytics"`

	Healthcheck *HealthcheckCmd `arg:"subcommand:healthcheck" help:"checks if the server is running"`
	Generate    *GenerateCmd    `arg:"subcommand:generate" help:"generates a configuration file for simple auth"`
	Agent       *AgentCmd       `arg:"subcommand:agent" help:"starts the agent"`
	Test        *TestCmd        `arg:"subcommand:test" help:"runs tests"` // TODO remove this
}

type HealthcheckCmd struct {
}

type AgentCmd struct {
}

type TestCmd struct {
}

type GenerateCmd struct {
	Username string `arg:"positional"`
	Password string `arg:"--password, -p" help:"sets the password for the user"`
	Name     string `arg:"--name, -n" help:"sets the display name for the user"`
	Email    string `arg:"--email, -e" help:"sets the email for the user"`
}

func (args) Version() string {
	return version
}

//go:embed all:dist
var content embed.FS

//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos ./protos/rpc.proto ./protos/types.proto
func main() {
	args, subcommand := parseArgs()
	validateEnvVars()
	if subcommand != nil {
		switch subcommand.(type) {
		case *TestCmd:
			client := agent.NewClient()
			service := docker_support.NewAgentService(client)
			events := make(chan *docker.LogEvent)
			go func() {
				for event := range events {
					log.Infof("Event: %+v", event)
				}
			}()
			// err := service.StreamLogs(context.Background(), docker.Container{ID: "57dbe50682eb"}, time.Now(), docker.STDALL, events)
			// log.Infof("Error: %v", err)

			reader, err := service.RawLogs(context.Background(), docker.Container{ID: "57dbe50682eb"}, time.Time{}, time.Now(), docker.STDALL)

			if err != nil {
				log.Fatal(err)
			}

			status.FromError()

			io.Copy(os.Stdout, reader)

		case *AgentCmd:
			client, err := docker.NewClientWithFilters(map[string][]string{}, "")
			if err != nil {
				log.Fatal(err)
			}
			agent.RunServer(client)
		case *HealthcheckCmd:
			if err := healthcheck.HttpRequest(args.Addr, args.Base); err != nil {
				log.Fatal(err)
			}

		case *GenerateCmd:
			if args.Generate.Username == "" || args.Generate.Password == "" {
				log.Fatal("Username and password are required")
			}

			buffer := auth.GenerateUsers(auth.User{
				Username: args.Generate.Username,
				Password: args.Generate.Password,
				Name:     args.Generate.Name,
				Email:    args.Generate.Email,
			}, true)

			if _, err := os.Stdout.Write(buffer.Bytes()); err != nil {
				log.Fatal(err)
			}
		}

		os.Exit(0)
	}

	if args.AuthProvider != "none" && args.AuthProvider != "forward-proxy" && args.AuthProvider != "simple" {
		log.Fatalf("Invalid auth provider %s", args.AuthProvider)
	}

	log.Infof("Dozzle version %s", version)

	clients := createClients(args, docker.NewClientWithFilters, docker.NewClientWithTlsAndFilter, args.Hostname)

	if len(clients) == 0 {
		log.Fatal("Could not connect to any Docker Engines")
	} else {
		log.Infof("Connected to %d Docker Engine(s)", len(clients))
	}

	srv := createServer(args, clients)
	go doStartEvent(args, clients)
	go func() {
		log.Infof("Accepting connections on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	stop()
	log.Info("shutting down gracefully, press Ctrl+C again to force")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	log.Debug("shutdown complete")
}

func doStartEvent(arg args, clients map[string]docker.Client) {
	if arg.NoAnalytics {
		log.Debug("Analytics disabled.")
		return
	}

	event := analytics.BeaconEvent{
		Name:    "start",
		Version: version,
	}

	if client, ok := clients["localhost"]; ok {
		event.ServerID = client.SystemInfo().ID
		event.ServerVersion = client.SystemInfo().ServerVersion
	} else {
		for _, client := range clients {
			event.ServerID = client.SystemInfo().ID
			event.ServerVersion = client.SystemInfo().ServerVersion
			break
		}
	}

	if err := analytics.SendBeacon(event); err != nil {
		log.Debug(err)
	}
}

func createClients(args args,
	localClientFactory func(map[string][]string, string) (docker.Client, error),
	remoteClientFactory func(map[string][]string, docker.Host) (docker.Client, error),
	hostname string) map[string]docker.Client {
	clients := make(map[string]docker.Client)

	if localClient, err := createLocalClient(args, localClientFactory); err == nil {
		clients[localClient.Host().ID] = localClient
	}

	for _, remoteHost := range args.RemoteHost {
		host, err := docker.ParseConnection(remoteHost)
		if err != nil {
			log.Fatalf("Could not parse remote host %s: %s", remoteHost, err)
		}
		log.Debugf("Creating remote client for %s with %+v", host.Name, host)
		log.Infof("Creating client for %s with %s", host.Name, host.URL.String())
		if client, err := remoteClientFactory(args.Filter, host); err == nil {
			if _, err := client.ListContainers(); err == nil {
				log.Debugf("Connected to local Docker Engine")
				clients[client.Host().ID] = client
			} else {
				log.Warnf("Could not connect to remote host %s: %s", host.ID, err)
			}
		} else {
			log.Warnf("Could not create client for %s: %s", host.ID, err)
		}
	}

	return clients
}

func createServer(args args, clients map[string]docker.Client) *http.Server {
	_, dev := os.LookupEnv("DEV")

	var provider web.AuthProvider = web.NONE
	var authorizer web.Authorizer
	if args.AuthProvider == "forward-proxy" {
		provider = web.FORWARD_PROXY
		authorizer = auth.NewForwardProxyAuth(args.AuthHeaderUser, args.AuthHeaderEmail, args.AuthHeaderName)
	} else if args.AuthProvider == "simple" {
		provider = web.SIMPLE

		path, err := filepath.Abs("./data/users.yml")
		if err != nil {
			log.Fatalf("Could not find absolute path to users.yml file: %s", err)
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Fatalf("Could not find users.yml file at %s", path)
		}

		users, err := auth.ReadUsersFromFile(path)
		if err != nil {
			log.Fatalf("Could not read users.yml file at %s: %s", path, err)
		}
		authorizer = auth.NewSimpleAuth(users)
	}

	config := web.Config{
		Addr:        args.Addr,
		Base:        args.Base,
		Version:     version,
		Hostname:    args.Hostname,
		NoAnalytics: args.NoAnalytics,
		Dev:         dev,
		Authorization: web.Authorization{
			Provider:   provider,
			Authorizer: authorizer,
		},
		EnableActions: args.EnableActions,
	}

	assets, err := fs.Sub(content, "dist")
	if err != nil {
		log.Fatalf("Could not open embedded dist folder: %v", err)
	}

	if _, ok := os.LookupEnv("LIVE_FS"); ok {
		if dev {
			log.Info("Using live filesystem at ./public")
			assets = os.DirFS("./public")
		} else {
			log.Info("Using live filesystem at ./dist")
			assets = os.DirFS("./dist")
		}
	}

	if !dev {
		if _, err := assets.Open(".vite/manifest.json"); err != nil {
			log.Fatal(".vite/manifest.json not found")
		}
		if _, err := assets.Open("index.html"); err != nil {
			log.Fatal("index.html not found")
		}
	}

	return web.CreateServer(clients, assets, config)
}

func createLocalClient(args args, localClientFactory func(map[string][]string, string) (docker.Client, error)) (docker.Client, error) {
	for i := 1; ; i++ {
		dockerClient, err := localClientFactory(args.Filter, args.Hostname)
		if err == nil {
			_, err := dockerClient.ListContainers()
			if err != nil {
				log.Debugf("Could not connect to local Docker Engine: %s", err)
			} else {
				log.Debugf("Connected to local Docker Engine")
				return dockerClient, nil
			}
		}
		if args.WaitForDockerSeconds > 0 {
			log.Infof("Waiting for Docker Engine (attempt %d): %s", i, err)
			time.Sleep(5 * time.Second)
			args.WaitForDockerSeconds -= 5
		} else {
			log.Debugf("Local Docker Engine not found")
			break
		}
	}
	return nil, errors.New("could not connect to local Docker Engine")
}

func parseArgs() (args, interface{}) {
	var args args
	parser := arg.MustParse(&args)

	configureLogger(args.Level)

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

	return args, parser.Subcommand()
}

func configureLogger(level string) {
	if l, err := log.ParseLevel(level); err == nil {
		log.SetLevel(l)
	} else {
		panic(err)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})

}

func validateEnvVars() {
	argsType := reflect.TypeOf(args{})
	expectedEnvs := make(map[string]bool)
	for i := 0; i < argsType.NumField(); i++ {
		field := argsType.Field(i)
		for _, tag := range strings.Split(field.Tag.Get("arg"), ",") {
			if strings.HasPrefix(tag, "env:") {
				expectedEnvs[strings.TrimPrefix(tag, "env:")] = true
			}
		}
	}

	for _, env := range os.Environ() {
		actual := strings.Split(env, "=")[0]
		if strings.HasPrefix(actual, "DOZZLE_") && !expectedEnvs[actual] {
			log.Warnf("Unexpected environment variable %s", actual)
		}
	}
}
