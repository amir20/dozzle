package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/amir20/dozzle/analytics"
	"github.com/amir20/dozzle/docker"
	"github.com/amir20/dozzle/healthcheck"
	"github.com/amir20/dozzle/web"

	log "github.com/sirupsen/logrus"
)

var (
	version = "head"
)

type DockerSecret struct {
	Value string
}

func (s *DockerSecret) UnmarshalText(b []byte) error {
	v, err := os.ReadFile(string(b))
	s.Value = strings.Trim(string(v), "\r\n")
	return err
}

type args struct {
	Addr                 string              `arg:"env:DOZZLE_ADDR" default:":8080" help:"sets host:port to bind for server. This is rarely needed inside a docker container."`
	Base                 string              `arg:"env:DOZZLE_BASE" default:"/" help:"sets the base for http router."`
	Hostname             string              `arg:"env:DOZZLE_HOSTNAME" help:"sets the hostname for display. This is useful with multiple Dozzle instances."`
	Level                string              `arg:"env:DOZZLE_LEVEL" default:"info" help:"set Dozzle log level. Use debug for more logging."`
	Username             string              `arg:"env:DOZZLE_USERNAME" help:"sets the username for auth."`
	Password             string              `arg:"env:DOZZLE_PASSWORD" help:"sets password for auth"`
	UsernameFile         *DockerSecret       `arg:"env:DOZZLE_USERNAME_FILE" help:"sets the secret path read username for auth."`
	PasswordFile         *DockerSecret       `arg:"env:DOZZLE_PASSWORD_FILE" help:"sets the secret path read password for auth"`
	NoAnalytics          bool                `arg:"--no-analytics,env:DOZZLE_NO_ANALYTICS" help:"disables anonymous analytics"`
	WaitForDockerSeconds int                 `arg:"--wait-for-docker-seconds,env:DOZZLE_WAIT_FOR_DOCKER_SECONDS" help:"wait for docker to be available for at most this many seconds before starting the server."`
	FilterStrings        []string            `arg:"env:DOZZLE_FILTER,--filter,separate" help:"filters docker containers using Docker syntax."`
	Filter               map[string][]string `arg:"-"`
	Healthcheck          *HealthcheckCmd     `arg:"subcommand:healthcheck" help:"checks if the server is running."`
	RemoteHost           []string            `arg:"env:DOZZLE_REMOTE_HOST,--remote-host,separate" help:"list of hosts to connect remotely"`
}

type HealthcheckCmd struct {
}

func (args) Version() string {
	return version
}

//go:embed dist
var content embed.FS

func main() {
	args := parseArgs()

	level, _ := log.ParseLevel(args.Level)
	log.SetLevel(level)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})

	if args.Healthcheck != nil {
		if err := healthcheck.HttpRequest(args.Addr, args.Base); err != nil {
			log.Fatal(err)
		}
	}

	log.Infof("Dozzle version %s", version)

	clients := createClients(args, docker.NewClientWithFilters, docker.NewClientWithTlsAndFilter)

	if len(clients) == 0 {
		log.Fatal("Could not connect to any Docker Engines")
	} else {
		log.Infof("Connected to %d Docker Engine(s)", len(clients))
	}

	srv := createServer(args, clients)
	go doStartEvent(args)
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

func doStartEvent(arg args) {
	if arg.NoAnalytics {
		log.Debug("Analytics disabled.")
		return
	}
	host, err := os.Hostname()
	if err != nil {
		log.Debug(err)
		return
	}

	event := analytics.StartEvent{
		ClientId:         host,
		Version:          version,
		FilterLength:     len(arg.Filter),
		CustomAddress:    arg.Addr != ":8080",
		CustomBase:       arg.Base != "/",
		RemoteHostLength: len(arg.RemoteHost),
		Protected:        arg.Username != "",
		HasHostname:      arg.Hostname != "",
	}

	if err := analytics.SendStartEvent(event); err != nil {
		log.Debug(err)
	}
}

func createClients(args args, localClientFactory func(map[string][]string) (docker.Client, error), remoteClientFactory func(map[string][]string, docker.Host) (docker.Client, error)) map[string]docker.Client {
	clients := make(map[string]docker.Client)

	if localClient := createLocalClient(args, localClientFactory); localClient != nil {
		clients[localClient.Host().Host] = localClient
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
				clients[client.Host().Host] = client
			} else {
				log.Warnf("Could not connect to remote host %s: %s", host.Host, err)
			}
		} else {
			log.Warnf("Could not create client for %s: %s", host.Host, err)
		}
	}

	return clients
}

func createServer(args args, clients map[string]docker.Client) *http.Server {
	_, dev := os.LookupEnv("DEV")
	config := web.Config{
		Addr:        args.Addr,
		Base:        args.Base,
		Version:     version,
		Username:    args.Username,
		Password:    args.Password,
		Hostname:    args.Hostname,
		NoAnalytics: args.NoAnalytics,
		Dev:         dev,
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
		if _, err := assets.Open("manifest.json"); err != nil {
			log.Fatal("manifest.json not found")
		}
		if _, err := assets.Open("index.html"); err != nil {
			log.Fatal("index.html not found")
		}
	}

	return web.CreateServer(clients, assets, config)
}

func createLocalClient(args args, localClientFactory func(map[string][]string) (docker.Client, error)) docker.Client {
	for i := 1; ; i++ {
		dockerClient, err := localClientFactory(args.Filter)
		if err == nil {
			_, err := dockerClient.ListContainers()

			if err == nil {
				log.Debugf("Connected to local Docker Engine")
				return dockerClient
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
	return nil
}

func parseArgs() args {
	var args args
	parser := arg.MustParse(&args)
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

	if args.Username == "" && args.UsernameFile != nil {
		args.Username = args.UsernameFile.Value
	}

	if args.Password == "" && args.PasswordFile != nil {
		args.Password = args.PasswordFile.Value
	}

	if args.Username != "" || args.Password != "" {
		if args.Username == "" || args.Password == "" {
			log.Fatalf("Username AND password are required for authentication")
		}
	}
	return args
}
