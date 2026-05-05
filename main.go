package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/deploy"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/k8s"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/support/cli"
	container_support "github.com/amir20/dozzle/internal/support/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	k8s_support "github.com/amir20/dozzle/internal/support/k8s"
	"github.com/amir20/dozzle/internal/web"
	"github.com/rs/zerolog/log"
)

//go:embed all:dist
var content embed.FS

//go:embed shared_cert.pem shared_key.pem
var certs embed.FS

//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos ./protos/rpc.proto ./protos/types.proto
//go:generate protoc --go_out=. --go-grpc_out=. --proto_path=./protos --go_opt=module=github.com/amir20/dozzle --go-grpc_opt=module=github.com/amir20/dozzle ./protos/cloud.proto
func main() {
	cli.ValidateEnvVars(cli.Args{}, cli.AgentCmd{})
	args, subcommand := cli.ParseArgs()
	if subcommand != nil {
		runnable, ok := subcommand.(cli.Runnable)
		if !ok {
			log.Fatal().Msg("Invalid command")
		}
		err := runnable.Run(args, certs)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to run command")
		}

		os.Exit(0)
	}

	if args.AuthProvider != "none" && args.AuthProvider != "forward-proxy" && args.AuthProvider != "simple" {
		log.Fatal().Str("provider", args.AuthProvider).Msg("Invalid auth provider")
	}

	log.Info().Msgf("Dozzle version %s", args.Version())
	dispatcher.UserAgent = fmt.Sprintf("Dozzle/%s", args.Version())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var hostService web.HostService
	var notificationService cloud.NotificationService
	if args.Mode == "server" {
		multiHostService := cli.CreateMultiHostService(certs, args)
		if multiHostService.TotalClients() == 0 {
			log.Fatal().Msg("Could not connect to any Docker Engine")
		} else {
			log.Info().Int("clients", multiHostService.TotalClients()).Msg("Connected to Docker")
		}
		if err := multiHostService.StartNotificationManager(ctx); err != nil {
			log.Fatal().Err(err).Msg("Could not start notification manager")
		}
		hostService = multiHostService
		notificationService = multiHostService
	} else if args.Mode == "swarm" {
		localClient, err := docker.NewLocalClient("")
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create docker client")
		}
		certs, err := cli.ReadCertificates(certs, args.CertPath, args.KeyPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not read certificates")
		}
		agentManager := docker_support.NewRetriableClientManager(args.RemoteAgent, args.Timeout, certs)
		manager := docker_support.NewSwarmClientManager(localClient, certs, args.Timeout, agentManager, args.Filter)
		multiHostService := docker_support.NewMultiHostService(manager, args.Timeout)
		if err := multiHostService.StartNotificationManager(ctx); err != nil {
			log.Fatal().Err(err).Msg("Could not start notification manager")
		}
		hostService = multiHostService
		notificationService = multiHostService
		log.Info().Msg("Starting in swarm mode")
		listener, err := net.Listen("tcp", ":7007")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to listen")
		}
		// Create client service for agent server in swarm mode
		clientService := docker_support.NewDockerClientService(localClient, args.Filter)
		server, err := agent.NewServer(clientService, certs, args.Version(), multiHostService.SwarmNotificationHandler())
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create agent")
		}
		go cli.StartEvent(args, "swarm", localClient, "")
		go func() {
			log.Info().Msgf("Dozzle agent version in swarm mode %s", args.Version())
			if err := server.Serve(listener); err != nil {
				log.Error().Err(err).Msg("failed to serve")
			}
		}()
	} else if args.Mode == "k8s" {
		localClient, err := k8s.NewK8sClient(args.Namespace)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create k8s client")
		}

		clusterService, err := k8s_support.NewK8sClusterService(localClient, args.Timeout)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create k8s cluster service")
		}

		if err := clusterService.StartNotificationManager(ctx); err != nil {
			log.Fatal().Err(err).Msg("Could not start notification manager")
		}

		go cli.StartEvent(args, "k8s", localClient, "")
		hostService = clusterService
	} else {
		log.Fatal().Str("mode", args.Mode).Msg("Invalid mode")
	}

	// Create cloud tool client — does nothing until Notify() is called
	apiKeyFunc := func() string {
		if cc := hostService.CloudConfig(); cc != nil {
			return cc.APIKey
		}
		return ""
	}

	var deployManager *deploy.Manager
	if args.EnableActions && args.Mode != "k8s" {
		// TODO: route deploys through agents for remote hosts.
		localClient, err := docker.NewLocalClient("")
		if err != nil {
			log.Warn().Err(err).Msg("Compose deploy tools disabled: could not create local Docker client")
		} else if raw := localClient.RawClient(); raw != nil {
			deployManager = deploy.NewManager(raw, deploy.DefaultStacksDir)
		} else {
			log.Warn().Msg("Compose deploy tools disabled: local Docker client has no raw handle")
		}
	}

	var instanceID string
	if h, err := hostService.LocalHost(); err == nil {
		instanceID = h.ID
	}

	cloudHostService := newLocalCloudHostService(hostService)

	cloudClient := cloud.NewClient(apiKeyFunc, instanceID, args.Version(), cloud.ToolDeps{
		EnableActions:       args.EnableActions,
		HostService:         cloudHostService,
		Labels:              args.Filter,
		DeployManager:       deployManager,
		NotificationService: notificationService,
	})
	cloudClient.SetStreamLogsFunc(func() bool {
		return hostService.CloudConfig().StreamLogsEnabled()
	})
	go cloudClient.Run(ctx)

	// In swarm mode, peer broadcasts of cloud config should kick this
	// replica's cloud client too, so every replica holds its own connection.
	if mhs, ok := hostService.(*docker_support.MultiHostService); ok {
		mhs.SetCloudNotifyFunc(cloudClient.Notify)
	}

	// If cloud is already configured at startup, start the client immediately
	if apiKeyFunc() != "" {
		cloudClient.Notify()
	}

	srv := createServer(args, hostService, web.CloudHooks{
		OnSetup:    cloudClient.Notify,
		OnUpdate:   cloudClient.Reconnect,
		SearchLogs: cloudClient.SearchLogs,
	})

	go func() {
		log.Info().Msgf("Accepting connections on %s", args.Addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed to listen")
		}
	}()

	<-ctx.Done()
	stop()
	log.Info().Msg("shutting down gracefully, press Ctrl+C again to force")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("failed to shut down")
	}
	log.Debug().Msg("shut down complete")
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func createServer(args cli.Args, hostService web.HostService, cloudHooks web.CloudHooks) *http.Server {
	_, dev := os.LookupEnv("DEV")

	var releaseCheckMode web.ReleaseCheckMode = web.Automatic

	switch args.ReleaseCheckMode {
	case "automatic":
		releaseCheckMode = web.Automatic
	case "manual":
		releaseCheckMode = web.Manual
	default:
		log.Fatal().Str("releaseCheckMode", args.ReleaseCheckMode).Msg("Invalid release check mode")
	}

	var provider web.AuthProvider = web.NONE
	var authorizer web.Authorizer
	if args.AuthProvider == "forward-proxy" {
		log.Debug().Msg("Using forward proxy authentication")
		provider = web.FORWARD_PROXY
		authorizer = auth.NewForwardProxyAuth(args.AuthHeaderUser, args.AuthHeaderEmail, args.AuthHeaderName, args.AuthHeaderFilter, args.AuthHeaderRoles)
	} else if args.AuthProvider == "simple" {
		log.Debug().Msg("Using simple authentication")
		provider = web.SIMPLE

		userFilePath := "./data/users.yml"
		if !fileExists(userFilePath) {
			userFilePath = "./data/users.yaml"
			if !fileExists(userFilePath) {
				log.Fatal().Msg("No users.yaml or users.yml file found.")
			}
		}

		log.Debug().Msgf("Reading %s file", filepath.Base(userFilePath))

		db, err := auth.ReadUsersFromFile(userFilePath)
		if err != nil {
			log.Fatal().Err(err).Msgf("Could not read users file: %s", userFilePath)
		}

		log.Debug().Int("users", len(db.Users)).Msg("Loaded users")
		ttl := time.Duration(0)
		if args.AuthTTL != "session" {
			ttl, err = time.ParseDuration(args.AuthTTL)
			if err != nil {
				log.Fatal().Err(err).Msg("Could not parse auth ttl")
			}
		}
		authorizer = auth.NewSimpleAuth(db, ttl)
	}

	authTTL := time.Duration(0)

	if args.AuthTTL != "session" {
		ttl, err := time.ParseDuration(args.AuthTTL)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not parse auth ttl")
		}
		authTTL = ttl
	}

	config := web.Config{
		Addr:        args.Addr,
		Base:        args.Base,
		Version:     args.Version(),
		Hostname:    args.Hostname,
		NoAnalytics: args.NoAnalytics,
		Dev:         dev,
		Mode:        args.Mode,
		Authorization: web.Authorization{
			Provider:   provider,
			Authorizer: authorizer,
			TTL:        authTTL,
			LogoutUrl:  args.AuthLogoutUrl,
		},
		EnableActions:    args.EnableActions,
		EnableShell:      args.EnableShell,
		DisableAvatars:   args.DisableAvatars,
		ReleaseCheckMode: releaseCheckMode,
		Labels:           args.Filter,
		Cloud:            cloudHooks,
	}

	assets, err := fs.Sub(content, "dist")
	if err != nil {
		log.Fatal().Err(err).Msg("Could not get sub filesystem")
	}

	if _, ok := os.LookupEnv("LIVE_FS"); ok {
		if dev {
			log.Info().Msg("Using live filesystem at ./public")
			assets = os.DirFS("./public")
		} else {
			log.Info().Msg("Using live filesystem at ./dist")
			assets = os.DirFS("./dist")
		}
	}

	if !dev {
		if _, err := assets.Open(".vite/manifest.json"); err != nil {
			log.Fatal().Msg("manifest.json not found")
		}
		if _, err := assets.Open("index.html"); err != nil {
			log.Fatal().Msg("index.html not found")
		}
	}

	return web.CreateServer(hostService, assets, config)
}

// localCloudHostService scopes the cloud client to local docker only;
// otherwise every connection re-reports its peers, multiplying hosts
// by connection count on the cloud side.
type localCloudHostService struct {
	services    []container_support.ClientService
	hostIDByIdx []string // parallel to services, populated lazily
	hostIDOnce  sync.Once
}

func newLocalCloudHostService(hs web.HostService) cloud.LogStreamHostService {
	services := hs.LocalClientServices()
	if len(services) == 0 {
		// k8s has no docker LocalClientServices but its HostService already
		// exposes only what this process can see, so use it directly.
		return hs
	}
	return &localCloudHostService{services: services}
}

func (l *localCloudHostService) localHostTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// resolveHostIDs caches each service's host ID once. Local docker host IDs
// are stable for the process lifetime, so a one-time lookup is enough.
func (l *localCloudHostService) resolveHostIDs() []string {
	l.hostIDOnce.Do(func() {
		l.hostIDByIdx = make([]string, len(l.services))
		for i, s := range l.services {
			ctx, cancel := l.localHostTimeout()
			h, err := s.Host(ctx)
			cancel()
			if err == nil {
				l.hostIDByIdx[i] = h.ID
			}
		}
	})
	return l.hostIDByIdx
}

func (l *localCloudHostService) Hosts() []container.Host {
	hosts := make([]container.Host, 0, len(l.services))
	for _, s := range l.services {
		ctx, cancel := l.localHostTimeout()
		h, err := s.Host(ctx)
		cancel()
		if err != nil {
			continue
		}
		h.Available = true
		hosts = append(hosts, h)
	}
	return hosts
}

func (l *localCloudHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	var all []container.Container
	var errs []error
	for _, s := range l.services {
		ctx, cancel := l.localHostTimeout()
		list, err := s.ListContainers(ctx, labels)
		cancel()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		all = append(all, list...)
	}
	return all, errs
}

func (l *localCloudHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	hostIDs := l.resolveHostIDs()
	for i, s := range l.services {
		if hostIDs[i] != host {
			continue
		}
		ctx, cancel := l.localHostTimeout()
		cont, err := s.FindContainer(ctx, id, labels)
		cancel()
		if err != nil {
			return nil, err
		}
		return container_support.NewContainerService(s, cont), nil
	}
	return nil, fmt.Errorf("host %s not local to this process", host)
}

func (l *localCloudHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter) {
	// One inbound channel + forwarder goroutine per service so a slow consumer
	// or a burst on one service can't cause the others to drop events.
	for _, s := range l.services {
		ch := make(chan container.Container, 64)
		s.SubscribeContainersStarted(ctx, ch)
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case c := <-ch:
					if filter(&c) {
						select {
						case containers <- c:
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}()
	}
}
