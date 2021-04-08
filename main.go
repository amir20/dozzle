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

	"github.com/amir20/dozzle/docker"
	"github.com/amir20/dozzle/web"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	addr     = ""
	base     = ""
	level    = ""
	tailSize = 300
	filters  map[string]string
	version  = "dev"
	key      string
	username string
	password string
)

//go:embed static
var content embed.FS

type handler struct {
	client docker.Client
}

func init() {
	pflag.String("addr", ":8080", "http service address")
	pflag.String("base", "/", "base address of the application to mount")
	pflag.String("level", "info", "logging level")
	pflag.Int("tailSize", 300, "Tail size to use for initial container logs")
	pflag.StringToStringVar(&filters, "filter", map[string]string{}, "Container filters to use for showing logs")
	pflag.String("key", "", "Dozzle secure key used for session encryption. Should be a random generated string. Use openssl rand -base64 32 to create one.")
	pflag.String("username", "", "Dozzle username to use for authentication. Requires key and password.")
	pflag.String("password", "", "Dozzle password for authentication. Requires username and key.")
	pflag.Parse()

	viper.AutomaticEnv()
	viper.SetEnvPrefix("DOZZLE")
	viper.BindPFlags(pflag.CommandLine)

	addr = viper.GetString("addr")
	base = viper.GetString("base")
	level = viper.GetString("level")
	tailSize = viper.GetInt("tailSize")
	key = viper.GetString("key")
	username = viper.GetString("username")
	password = viper.GetString("password")

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

	l, _ := log.ParseLevel(level)
	log.SetLevel(l)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
	})
}

func main() {
	log.Infof("Dozzle version %s", version)
	dockerClient := docker.NewClientWithFilters(filters)
	_, err := dockerClient.ListContainers()

	if err != nil {
		log.Fatalf("Could not connect to Docker Engine: %v", err)
	}

	if username != "" || password != "" {
		if username == "" || password == "" {
			log.Fatalf("Username AND password are required for authentication")
		}

		if key == "" {
			log.Fatalf("Key is required for authentication")
		}
	}

	config := web.Config{
		Addr:     addr,
		Base:     base,
		Version:  version,
		TailSize: tailSize,
		Key:      key,
		Username: username,
		Password: password,
	}

	static, err := fs.Sub(content, "static")
	if err != nil {
		log.Fatalf("Could not open embedded static folder: %v", err)
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
	log.Infof("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	os.Exit(0)
}
