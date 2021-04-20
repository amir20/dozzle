package main

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/amir20/dozzle/analytics"
	"github.com/amir20/dozzle/docker"
	"github.com/amir20/dozzle/web"

	log "github.com/sirupsen/logrus"
)

var (
	version = "dev"
)

type args struct {
	Addr        string            `arg:"env:DOZZLE_ADDR" default:":8080"`
	Base        string            `arg:"env:DOZZLE_BASE" default:"/"`
	Level       string            `arg:"env:DOZZLE_LEVEL" default:"info"`
	TailSize    int               `arg:"env:DOZZLE_TAILSIZE" default:"300"`
	Filter      map[string]string `arg:"env:DOZZLE_FILTER"`
	Key         string            `arg:"env:DOZZLE_KEY"`
	Username    string            `arg:"env:DOZZLE_USERNAME"`
	Password    string            `arg:"env:DOZZLE_PASSWORD"`
	NoAnalytics bool              `arg:"--no-analytics,env:DOZZLE_NO_ANALYTICS"`
}

func (args) Version() string {
	return version
}

//go:embed static
var content embed.FS

func main() {
	var args args
	arg.MustParse(&args)
	level, _ := log.ParseLevel(args.Level)
	log.SetLevel(level)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})

	log.Infof("Dozzle version %s", version)
	dockerClient := docker.NewClientWithFilters(args.Filter)
	_, err := dockerClient.ListContainers()

	if err != nil {
		log.Fatalf("Could not connect to Docker Engine: %v", err)
	}

	if args.Username != "" || args.Password != "" {
		if args.Username == "" || args.Password == "" {
			log.Fatalf("Username AND password are required for authentication")
		}

		if args.Key == "" {
			log.Fatalf("Key is required for authentication")
		}
	}

	config := web.Config{
		Addr:     args.Addr,
		Base:     args.Base,
		Version:  version,
		TailSize: args.TailSize,
		Key:      args.Key,
		Username: args.Username,
		Password: args.Password,
	}

	static, err := fs.Sub(content, "static")
	if err != nil {
		log.Fatalf("Could not open embedded static folder: %v", err)
	}

	if _, ok := os.LookupEnv("LIVE_FS"); ok {
		log.Info("Using live filesystem at ./static")
		static = os.DirFS("./static")
	}

	srv := web.CreateServer(dockerClient, static, config)
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
		ClientId:      host,
		Version:       version,
		FilterLength:  len(arg.Filter),
		CustomAddress: arg.Addr != ":8080",
		CustomBase:    arg.Base != "/",
		TailSize:      arg.TailSize,
		Protected:     arg.Username != "",
	}

	if err := analytics.SendStartEvent(event); err != nil {
		log.Debug(err)
	}
}
