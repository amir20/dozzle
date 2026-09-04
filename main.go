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
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/imagecheck"
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

	var instanceID, swarmClusterID string
	if h, err := hostService.LocalHost(); err == nil {
		instanceID = h.ID
		swarmClusterID = h.SwarmClusterID
	}

	cloudHostService := newCloudHostService(args.Mode, hostService)

	cloudClient := cloud.NewClient(apiKeyFunc, instanceID, args.Version(), cloud.ToolDeps{
		EnableActions:       args.EnableActions,
		HostService:         cloudHostService,
		Labels:              args.Filter,
		NotificationService: notificationService,
	})
	cloudClient.SetDeployment(args.Mode, swarmClusterID)
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
		GetAlerts:  cloudClient.GetAlerts,
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

	// Already validated in ParseArgs, so an error here is not reachable.
	imageCheckMode, err := imagecheck.ParseMode(args.ImageCheckMode)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not parse image check mode")
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
		EnableMCP:        args.EnableMCP,
		DisableAvatars:   args.DisableAvatars,
		ReleaseCheckMode: releaseCheckMode,
		ImageCheckMode:   imageCheckMode,
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

// cloudHostService is the view of the fleet the cloud client is given.
//
// In swarm mode every replica discovers every peer, so a replica reporting the
// whole fleet would multiply hosts — and duplicate log and stat ingestion — by
// replica count on the cloud side. There the view is scoped to this process's
// own docker daemon.
//
// Everywhere else there is exactly one Dozzle holding the fleet: --remote-host
// and --remote-agent endpoints are configured on it and on nothing else, so
// scoping them away simply hid them from the cloud. --remote-host survived only
// because it happens to be a *DockerClientService; agents did not appear at all.
//
// services is a func, not a slice, because an agent that is unreachable at boot
// joins later. Reading it per call means such an agent shows up as soon as it
// connects rather than at the next restart. Its retry argument asks for
// unreachable agents to be re-dialed, which costs a connection attempt each —
// only the periodic fan-out calls pay it.
type cloudHostService struct {
	services func(retry bool) []container_support.ClientService
	// hs is the underlying host service, used only to learn when a previously
	// unreachable host becomes available so log and stat subscriptions can be
	// extended to it.
	hs web.HostService

	mu      sync.Mutex
	hostIDs map[container_support.ClientService]string
}

func newCloudHostService(mode string, hs web.HostService) cloud.LogStreamHostService {
	services := func(bool) []container_support.ClientService { return hs.LocalClientServices() }
	if mode != "swarm" {
		if mhs, ok := hs.(*docker_support.MultiHostService); ok {
			services = mhs.ClientServices
		}
	}
	if len(services(false)) == 0 {
		// k8s has no docker client services but its HostService already
		// exposes only what this process can see, so use it directly.
		return hs
	}
	return &cloudHostService{
		services: services,
		hs:       hs,
		hostIDs:  make(map[container_support.ClientService]string),
	}
}

// reattachInterval is how often log and stat subscriptions re-scan the fleet.
// A var, not a const, so tests need not wait a real minute.
var reattachInterval = time.Minute

func (l *cloudHostService) hostTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// hostID caches a service's host ID. A host ID is stable for the lifetime of
// the client that serves it, so one lookup per service is enough.
//
// An agent answers Host over the wire and can fail transiently, but it returns
// its last known host alongside the error, and a service only enters the fleet
// after one successful Host call. So an id is taken whenever there is one,
// error or not; only a service that has never identified itself yields "".
//
// The lock is dropped across the lookup on purpose. Two callers racing a cold
// cache both dial and both store, which costs one redundant call and writes the
// same id twice; holding the lock instead would serialise every caller behind a
// network round trip, including callers asking about other hosts.
func (l *cloudHostService) hostID(s container_support.ClientService) string {
	l.mu.Lock()
	id, ok := l.hostIDs[s]
	l.mu.Unlock()
	if ok {
		return id
	}

	ctx, cancel := l.hostTimeout()
	h, _ := s.Host(ctx)
	cancel()
	if h.ID == "" {
		return ""
	}

	l.mu.Lock()
	l.hostIDs[s] = h.ID
	l.mu.Unlock()
	return h.ID
}

func (l *cloudHostService) Hosts() []container.Host {
	services := l.services(true)
	hosts := make([]container.Host, 0, len(services))
	for _, s := range services {
		ctx, cancel := l.hostTimeout()
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

func (l *cloudHostService) ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error) {
	var all []container.Container
	var errs []error
	for _, s := range l.services(true) {
		ctx, cancel := l.hostTimeout()
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

func (l *cloudHostService) FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error) {
	// No retry: this runs once per log reader, and a run of them against an
	// unreachable agent would each wait out the dial timeout.
	for _, s := range l.services(false) {
		if l.hostID(s) != host {
			continue
		}
		ctx, cancel := l.hostTimeout()
		cont, err := s.FindContainer(ctx, id, labels)
		cancel()
		if err != nil {
			return nil, err
		}
		return container_support.NewContainerService(s, cont), nil
	}
	return nil, fmt.Errorf("host %s is not served by this Dozzle instance", host)
}

// watchNewServices calls attach again whenever a host that was unreachable at
// startup joins, so a late agent gets subscribed without waiting for a restart.
//
// The ticker is the backstop. SubscribeAvailableHosts only fires on the
// unreachable-to-reachable edge, so anything attach skipped for another reason
// — a host whose id could not be resolved at the time — would otherwise stay
// skipped for the life of the connection. A re-attach that finds nothing new is
// a map lookup per service, so this is cheap enough to run unconditionally.
//
// attach is only ever called from this one goroutine, after the caller's
// initial synchronous call has returned, so it needs no locking of its own.
func (l *cloudHostService) watchNewServices(ctx context.Context, attach func()) {
	hosts := make(chan container.Host, 8)
	l.hs.SubscribeAvailableHosts(ctx, hosts)
	ticker := time.NewTicker(reattachInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-hosts:
				attach()
			case <-ticker.C:
				attach()
			}
		}
	}()
}

// SubscribeStats fans stats in from every client service, stamping the
// originating host on each sample. container.ContainerStat has no host field,
// so without this the cloud side could not tell two same-named containers on
// different hosts apart.
func (l *cloudHostService) SubscribeStats(ctx context.Context, samples chan<- cloud.StatSample) {
	// One inbound channel + forwarder goroutine per service, matching
	// SubscribeContainersStarted: a burst on one service must not stall the others.
	var dropWarn sync.Once
	subscribed := make(map[container_support.ClientService]bool)
	attach := func() {
		for _, s := range l.services(false) {
			if subscribed[s] {
				continue
			}
			hostID := l.hostID(s)
			if hostID == "" {
				// Unstamped samples can't be told apart on the cloud side.
				// Leave the service unsubscribed; watchNewServices re-attaches
				// on a timer, so this resolves itself once the host answers.
				continue
			}
			subscribed[s] = true
			ch := make(chan container.ContainerStat, 64)
			s.SubscribeStats(ctx, ch)
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case stat := <-ch:
						// Non-blocking on purpose. The stats collector dispatches to
						// each subscriber with a blocking send, so if cloud ingest
						// ever wedged, backpressure would travel all the way up and
						// starve the live UI's stats subscriber on this host. Losing
						// a sample only nudges a 30s average; stalling the UI is not
						// an acceptable trade for that.
						select {
						case samples <- cloud.StatSample{Stat: stat, HostID: hostID}:
						default:
							dropWarn.Do(func() {
								log.Warn().Msg("cloud stats: consumer is not keeping up, dropping samples (further drops are silent)")
							})
						}
					}
				}
			}()
		}
	}

	attach()
	l.watchNewServices(ctx, attach)
}

func (l *cloudHostService) SubscribeContainersStarted(ctx context.Context, containers chan<- container.Container, filter container_support.ContainerFilter) {
	// One inbound channel + forwarder goroutine per service so a slow consumer
	// or a burst on one service can't cause the others to drop events.
	subscribed := make(map[container_support.ClientService]bool)
	attach := func() {
		for _, s := range l.services(false) {
			if subscribed[s] {
				continue
			}
			subscribed[s] = true
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

	attach()
	l.watchNewServices(ctx, attach)
}
