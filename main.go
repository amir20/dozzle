package main

import (
	"context"
	"embed"
	"io"
	"io/fs"

	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/healthcheck"
	"github.com/amir20/dozzle/internal/support/cli"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/amir20/dozzle/internal/web"

	log "github.com/sirupsen/logrus"
)

//go:embed all:dist
var content embed.FS

//go:embed shared_cert.pem shared_key.pem
var certs embed.FS

//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos ./protos/rpc.proto ./protos/types.proto
func main() {
	cli.ValidateEnvVars(cli.Args{}, cli.AgentCmd{})
	args, subcommand := cli.ParseArgs()
	if subcommand != nil {
		switch subcommand.(type) {
		case *cli.AgentCmd:
			client, err := docker.NewLocalClient(args.Filter, args.Hostname)
			if err != nil {
				log.Fatalf("Could not create docker client: %v", err)
			}
			certs, err := cli.ReadCertificates(certs)
			if err != nil {
				log.Fatalf("Could not read certificates: %v", err)
			}

			listener, err := net.Listen("tcp", args.Agent.Addr)
			if err != nil {
				log.Fatalf("failed to listen: %v", err)
			}
			tempFile, err := os.CreateTemp("/", "agent-*.addr")
			if err != nil {
				log.Fatalf("failed to create temp file: %v", err)
			}
			defer os.Remove(tempFile.Name())
			io.WriteString(tempFile, listener.Addr().String())
			go cli.StartEvent(args.Version(), "", args.RemoteAgent, args.RemoteHost, client, "agent")
			agent.RunServer(client, certs, listener)
		case *cli.HealthcheckCmd:
			go cli.StartEvent(args.Version(), "", args.RemoteAgent, args.RemoteHost, nil, "healthcheck")
			files, err := os.ReadDir(".")
			if err != nil {
				log.Fatalf("Failed to read directory: %v", err)
			}

			agentAddress := ""
			for _, file := range files {
				if match, _ := filepath.Match("agent-*.addr", file.Name()); match {
					data, err := os.ReadFile(file.Name())
					if err != nil {
						log.Fatalf("Failed to read file: %v", err)
					}
					agentAddress = string(data)
					break
				}
			}
			if agentAddress == "" {
				if err := healthcheck.HttpRequest(args.Addr, args.Base); err != nil {
					log.Fatalf("Failed to make request: %v", err)
				}
			} else {
				certs, err := cli.ReadCertificates(certs)
				if err != nil {
					log.Fatalf("Could not read certificates: %v", err)
				}
				if err := healthcheck.RPCRequest(agentAddress, certs); err != nil {
					log.Fatalf("Failed to make request: %v", err)
				}
			}

		case *cli.GenerateCmd:
			go cli.StartEvent(args.Version(), "", args.RemoteAgent, args.RemoteHost, nil, "generate")
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
				log.Fatalf("Failed to write to stdout: %v", err)
			}
		}

		os.Exit(0)
	}

	if args.AuthProvider != "none" && args.AuthProvider != "forward-proxy" && args.AuthProvider != "simple" {
		log.Fatalf("Invalid auth provider %s", args.AuthProvider)
	}

	log.Infof("Dozzle version %s", args.Version())

	var multiHostService *docker_support.MultiHostService
	if args.Mode == "server" {
		multiHostService = cli.CreateMultiHostService(certs, args)
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
		certs, err := cli.ReadCertificates(certs)
		if err != nil {
			log.Fatalf("Could not read certificates: %v", err)
		}
		multiHostService = docker_support.NewSwarmService(localClient, certs)
		log.Infof("Starting in Swarm mode")
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

func createServer(args cli.Args, multiHostService *docker_support.MultiHostService) *http.Server {
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
		Version:     args.Version(),
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
