package main

import (
	"context"
	"crypto/tls"
	"embed"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/healthcheck"
	"github.com/amir20/dozzle/internal/support/cli"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/amir20/dozzle/internal/web"

	log "github.com/sirupsen/logrus"
)

var (
	version = "head"
)

type args struct {
	Addr            string              `arg:"env:DOZZLE_ADDR" default:":8080" help:"sets host:port to bind for server. This is rarely needed inside a docker container."`
	Base            string              `arg:"env:DOZZLE_BASE" default:"/" help:"sets the base for http router."`
	Hostname        string              `arg:"env:DOZZLE_HOSTNAME" help:"sets the hostname for display. This is useful with multiple Dozzle instances."`
	Level           string              `arg:"env:DOZZLE_LEVEL" default:"info" help:"set Dozzle log level. Use debug for more logging."`
	AuthProvider    string              `arg:"--auth-provider,env:DOZZLE_AUTH_PROVIDER" default:"none" help:"sets the auth provider to use. Currently only forward-proxy is supported."`
	AuthHeaderUser  string              `arg:"--auth-header-user,env:DOZZLE_AUTH_HEADER_USER" default:"Remote-User" help:"sets the HTTP Header to use for username in Forward Proxy configuration."`
	AuthHeaderEmail string              `arg:"--auth-header-email,env:DOZZLE_AUTH_HEADER_EMAIL" default:"Remote-Email" help:"sets the HTTP Header to use for email in Forward Proxy configuration."`
	AuthHeaderName  string              `arg:"--auth-header-name,env:DOZZLE_AUTH_HEADER_NAME" default:"Remote-Name" help:"sets the HTTP Header to use for name in Forward Proxy configuration."`
	EnableActions   bool                `arg:"--enable-actions,env:DOZZLE_ENABLE_ACTIONS" default:"false" help:"enables essential actions on containers from the web interface."`
	FilterStrings   []string            `arg:"env:DOZZLE_FILTER,--filter,separate" help:"filters docker containers using Docker syntax."`
	Filter          map[string][]string `arg:"-"`
	RemoteHost      []string            `arg:"env:DOZZLE_REMOTE_HOST,--remote-host,separate" help:"list of hosts to connect remotely"`
	RemoteAgent     []string            `arg:"env:DOZZLE_REMOTE_AGENT,--remote-agent,separate" help:"list of agents to connect remotely"`
	NoAnalytics     bool                `arg:"--no-analytics,env:DOZZLE_NO_ANALYTICS" help:"disables anonymous analytics"`
	Mode            string              `arg:"env:DOZZLE_MODE" default:"server" help:"sets the mode to run in (server, swarm)"`
	Healthcheck     *HealthcheckCmd     `arg:"subcommand:healthcheck" help:"checks if the server is running"`
	Generate        *GenerateCmd        `arg:"subcommand:generate" help:"generates a configuration file for simple auth"`
	Agent           *AgentCmd           `arg:"subcommand:agent" help:"starts the agent"`
}

type HealthcheckCmd struct {
}

type AgentCmd struct {
	Addr string `arg:"env:DOZZLE_AGENT_ADDR" default:":7007" help:"sets the host:port to bind for the agent"`
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

//go:embed shared_cert.pem shared_key.pem
var certs embed.FS

//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos ./protos/rpc.proto ./protos/types.proto
func main() {
	cli.ValidateEnvVars(args{}, AgentCmd{})
	args, subcommand := parseArgs()
	if subcommand != nil {
		switch subcommand.(type) {
		case *AgentCmd:
			client, err := docker.NewLocalClient(args.Filter, args.Hostname)
			if err != nil {
				log.Fatal(err)
			}
			certs, err := readCertificates()
			if err != nil {
				log.Fatalf("Could not read certificates: %v", err)
			}

			listener, err := net.Listen("tcp", args.Agent.Addr)
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}

			agent.RunServer(client, certs, listener)
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

	var multiHostService *docker_support.MultiHostService
	if args.Mode == "server" {
		multiHostService = createMultiHostService(args)
		if multiHostService.TotalClients() == 0 {
			log.Fatal("Could not connect to any Docker Engines")
		} else {
			log.Infof("Connected to %d Docker Engine(s)", multiHostService.TotalClients())
		}
	} else if args.Mode == "swarm" {
		localClient, err := docker.NewLocalClient(args.Filter, args.Hostname)
		if err != nil {
			log.Fatalf("Could not connect to local Docker Engine: %s", err)
		}
		certs, err := readCertificates()
		if err != nil {
			log.Fatalf("Could not read certificates: %v", err)
		}
		multiHostService = docker_support.NewSwarmService(localClient, certs)
		log.Infof("Connected to local Docker Engine")

		listener, err := net.Listen("tcp", ":7007")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		go agent.RunServer(localClient, certs, listener)
	} else {
		log.Fatalf("Invalid mode %s", args.Mode)
	}

	srv := createServer(args, multiHostService)
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

func readCertificates() (tls.Certificate, error) {
	cert, err := certs.ReadFile("shared_cert.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	key, err := certs.ReadFile("shared_key.pem")
	if err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(cert, key)
}

func createMultiHostService(args args) *docker_support.MultiHostService {
	var clients []docker_support.ClientService
	for _, remoteHost := range args.RemoteHost {
		host, err := docker.ParseConnection(remoteHost)
		if err != nil {
			log.Fatalf("Could not parse remote host %s: %s", remoteHost, err)
		}
		log.Debugf("creating remote client for %s with %+v", host.Name, host)
		log.Infof("Creating client for %s with %s", host.Name, host.URL.String())
		if client, err := docker.NewRemoteClient(args.Filter, host); err == nil {
			if _, err := client.ListContainers(); err == nil {
				log.Debugf("connected to local Docker Engine")
				clients = append(clients, docker_support.NewDockerClientService(client))
			} else {
				log.Warnf("Could not connect to remote host %s: %s", host.ID, err)
			}
		} else {
			log.Warnf("Could not create client for %s: %s", host.ID, err)
		}
	}
	certs, err := readCertificates()
	if err != nil {
		log.Fatalf("Could not read certificates: %v", err)
	}
	for _, remoteAgent := range args.RemoteAgent {
		client, err := agent.NewClient(remoteAgent, certs)
		if err != nil {
			log.Warnf("Could not connect to remote agent %s: %s", remoteAgent, err)
			continue
		}
		clients = append(clients, docker_support.NewAgentService(client))
	}

	localClient, err := docker.NewLocalClient(args.Filter, args.Hostname)
	if err == nil {
		_, err := localClient.ListContainers()
		if err != nil {
			log.Debugf("could not connect to local Docker Engine: %s", err)
			if !args.NoAnalytics {
				go cli.StartEvent(version, args.Mode, args.RemoteAgent, args.RemoteHost, nil)
			}
		} else {
			log.Debugf("connected to local Docker Engine")
			if !args.NoAnalytics {
				go cli.StartEvent(version, args.Mode, args.RemoteAgent, args.RemoteHost, localClient)
			}
			clients = append(clients, docker_support.NewDockerClientService(localClient))
		}
	}

	return docker_support.NewMultiHostService(clients)
}

func createServer(args args, multiHostService *docker_support.MultiHostService) *http.Server {
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

	return web.CreateServer(multiHostService, assets, config)
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
