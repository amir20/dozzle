package main

import (
	"context"
	"embed"
	"io/fs"
	_ "net/http/pprof"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/amir20/dozzle/docker"
	"github.com/amir20/dozzle/web"

	log "github.com/sirupsen/logrus"
)

var (
	filters map[string]string
	version = "dev"
)

type args struct {
	Addr     string `arg:"env:DOZZLE_ADDR" default:":8080"`
	Base     string `arg:"env:DOZZLE_BASE" default:"/"`
	Level    string `arg:"env:DOZZLE_LEVEL" default:"info"`
	TailSize int    `arg:"env:DOZZLE_TAILSIZE" default:300`
	// filters map[string]string
	Key      string `arg:"env:DOZZLE_KEY"`
	Username string `arg:"env:DOZZLE_USERNAME"`
	Password string `arg:"env:DOZZLE_PASSWORD"`
}

func (args) Version() string {
	return version
}

//go:embed static
var content embed.FS

func init() {
	// Until https://github.com/spf13/viper/issues/911 is fixed. We have to use this hacky way.
	// filters = viper.GetStringMapString("filter")
	if value, ok := os.LookupEnv("DOZZLE_FILTER"); ok {
		log.Infof("Parsing %s", value)
		urlValues, err := url.ParseQuery(strings.ReplaceAll(value, ",", "&"))
		if err != nil {
			log.Fatal(err)
		}
		filters = map[string]string{}
		for k, v := range urlValues {
			filters[k] = v[0]
		}
	}
}

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
	dockerClient := docker.NewClientWithFilters(filters)
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

	go func() {
		log.Infof("Accepting connections on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	<-c
	log.Info("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	os.Exit(0)
}
