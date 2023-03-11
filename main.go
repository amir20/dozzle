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
	var args args
	var err error
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

	dockerClient := docker.NewClientWithFilters(args.Filter)
	for i := 1; ; i++ {
		_, err := dockerClient.ListContainers()
		if err == nil {
			break
		} else if args.WaitForDockerSeconds <= 0 {
			log.Fatalf("Could not connect to Docker Engine: %v", err)
		} else {
			log.Infof("Waiting for Docker Engine (attempt %d): %s", i, err)
			time.Sleep(5 * time.Second)
			args.WaitForDockerSeconds -= 5
		}
	}

	clients := make(map[string]docker.Client)
	clients["localhost"] = dockerClient

	for _, host := range args.RemoteHost {
		log.Infof("Creating client for %s", host)
		client := docker.NewClientWithTlsAndFilter(args.Filter, host)
		clients[host] = client
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

	config := web.Config{
		Addr:        args.Addr,
		Base:        args.Base,
		Version:     version,
		Username:    args.Username,
		Password:    args.Password,
		Hostname:    args.Hostname,
		NoAnalytics: args.NoAnalytics,
	}

	assets, err := fs.Sub(content, "dist")
	if err != nil {
		log.Fatalf("Could not open embedded dist folder: %v", err)
	}

	if _, ok := os.LookupEnv("LIVE_FS"); ok {
		log.Info("Using live filesystem at ./dist")
		assets = os.DirFS("./dist")
	}

	srv := web.CreateServer(clients, assets, config)
	go doStartEvent(args)
	go func() {
		log.Infof("Accepting connections on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	<-c
	log.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	os.Exit(0)
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
