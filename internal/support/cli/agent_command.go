package cli

import (
	"context"
	"embed"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/amir20/dozzle/internal/agent"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/amir20/dozzle/internal/docker"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	container_support "github.com/amir20/dozzle/internal/support/container"
	docker_support "github.com/amir20/dozzle/internal/support/docker"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

type AgentCmd struct {
	Addr string `arg:"--agent-addr,env:DOZZLE_AGENT_ADDR" default:":7007" help:"sets the host:port to bind for the agent"`
}

// persistingNotificationHandler wraps a notification manager and saves config to disk after updates
type persistingNotificationHandler struct {
	manager    *notification.Manager
	configPath string

	mu          sync.RWMutex
	cloudConfig *notification.CloudConfig
	onCloudSet  func()
}

// CloudConfig returns the agent's currently active cloud config, or nil if
// none has been pushed by the main server / loaded from disk.
func (h *persistingNotificationHandler) CloudConfig() *notification.CloudConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cloudConfig
}

func (h *persistingNotificationHandler) GetNotificationStats() []types.SubscriptionStats {
	return h.manager.GetNotificationStats()
}

func (h *persistingNotificationHandler) HandleNotificationConfig(subscriptions []types.SubscriptionConfig, dispatchers []types.DispatcherConfig) error {
	// Update the manager
	if err := h.manager.HandleNotificationConfig(subscriptions, dispatchers); err != nil {
		return err
	}

	// Save to disk
	if err := os.MkdirAll("./data", 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	file, err := os.Create(h.configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	if err := h.manager.WriteConfig(file); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	log.Debug().Str("path", h.configPath).Msg("Saved notification config to disk")
	return nil
}

func (h *persistingNotificationHandler) SetCloudDispatcher(d dispatcher.Dispatcher) {
	h.manager.SetCloudDispatcher(d)

	// Persist cloud config to disk so it survives agent restarts
	cd, ok := d.(*dispatcher.CloudDispatcher)
	if !ok {
		log.Warn().Str("type", fmt.Sprintf("%T", d)).Msg("Cloud dispatcher type assertion failed, cannot persist")
		return
	}
	cc := notification.CloudConfig{
		APIKey:    cd.APIKey,
		Prefix:    cd.Prefix,
		ExpiresAt: cd.ExpiresAt,
	}
	h.mu.Lock()
	h.cloudConfig = &cc
	notify := h.onCloudSet
	h.mu.Unlock()
	if notify != nil {
		notify()
	}
	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Error().Err(err).Msg("Could not create data directory for cloud config")
		return
	}
	file, err := os.Create("./data/cloud.yml")
	if err != nil {
		log.Error().Err(err).Msg("Could not create cloud.yml on agent")
		return
	}
	defer file.Close()
	if err := notification.WriteCloudConfig(file, cc); err != nil {
		log.Error().Err(err).Msg("Could not write cloud.yml on agent")
	} else {
		log.Debug().Msg("Persisted cloud.yml on agent")
	}
}

func (h *persistingNotificationHandler) ClearCloudDispatcher() {
	h.manager.ClearCloudDispatcher()
	h.mu.Lock()
	h.cloudConfig = nil
	notify := h.onCloudSet
	h.mu.Unlock()
	if notify != nil {
		notify()
	}
	if err := os.Remove("./data/cloud.yml"); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msg("Could not remove cloud.yml on agent")
	}
}

func (a *AgentCmd) Run(args Args, embeddedCerts embed.FS) error {
	if args.Mode != "server" {
		return fmt.Errorf("agent command is only available in server mode")
	}
	client, err := docker.NewLocalClient(args.Hostname)
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	certs, err := ReadCertificates(embeddedCerts, args.CertPath, args.KeyPath)
	if err != nil {
		return fmt.Errorf("failed to read certificates: %w", err)
	}

	listener, err := net.Listen("tcp", args.Agent.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	const agentAddrFile = "/tmp/dozzle-agent.addr"
	if err := os.WriteFile(agentAddrFile, []byte(args.Agent.Addr), 0644); err != nil {
		return fmt.Errorf("failed to write agent address file: %w", err)
	}
	go StartEvent(args, "", client, "agent")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Create shared client service (single ContainerStore for both agent server and notifications)
	clientService := docker_support.NewDockerClientService(client, args.Filter)

	// Create notification manager using the shared client service
	const notificationConfigPath = "./data/notifications.yml"
	clients := []container_support.ClientService{clientService}
	notificationManager := notification.NewManager(
		notification.NewContainerLogListener(ctx, clients),
		notification.NewContainerStatsListener(ctx, clients),
		notification.NewContainerEventListener(ctx, clients),
	)

	// Start first so matcher is available for LoadConfig
	if err := notificationManager.Start(); err != nil {
		return fmt.Errorf("failed to start notification manager: %w", err)
	}

	// Load existing notification config if available
	if file, err := os.Open(notificationConfigPath); err == nil {
		if err := notificationManager.LoadConfig(file); err != nil {
			log.Warn().Err(err).Msg("Failed to load notification config, starting fresh")
		} else {
			log.Info().Str("path", notificationConfigPath).Msg("Loaded notification config from disk")
		}
		file.Close()
	}

	// Create handler that wraps manager and persists config to disk
	notificationHandler := &persistingNotificationHandler{
		manager:    notificationManager,
		configPath: notificationConfigPath,
	}

	// Load cloud config if available
	if file, err := os.Open("./data/cloud.yml"); err == nil {
		cc, err := notification.LoadCloudConfig(file)
		file.Close()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to load cloud config on agent")
		} else {
			d, err := dispatcher.NewCloudDispatcher("Dozzle Cloud", cc.APIKey, cc.Prefix, cc.ExpiresAt)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create cloud dispatcher on agent")
			} else {
				notificationManager.SetCloudDispatcher(d)
				notificationHandler.cloudConfig = &cc
				log.Info().Msg("Loaded cloud config from disk")
			}
		}
	}

	// Create a single-host MultiHostService so the cloud client has a
	// HostService for tool execution (list_containers, fetch_logs, etc.).
	agentManager := docker_support.NewRetriableClientManager(nil, args.Timeout, certs, clientService)
	agentHostService := docker_support.NewMultiHostService(agentManager, args.Timeout)

	// Cloud gRPC client — connects directly to Dozzle Cloud with this agent's
	// own host ID as instance_id, so log streaming and tool dispatch happen
	// here instead of funneling through the main server.
	var instanceID string
	if h, err := agentHostService.LocalHost(); err == nil {
		instanceID = h.ID
	}
	apiKeyFunc := func() string {
		if cc := notificationHandler.CloudConfig(); cc != nil {
			return cc.APIKey
		}
		return ""
	}
	cloudClient := cloud.NewClient(apiKeyFunc, instanceID, args.Version(), cloud.ToolDeps{
		EnableActions: false, // agents don't host action tools today
		HostService:   agentHostService,
		Labels:        args.Filter,
	})
	cloudClient.SetStreamLogsFunc(func() bool {
		cc := notificationHandler.CloudConfig()
		return cc != nil && cc.StreamLogsEnabled()
	})
	notificationHandler.onCloudSet = cloudClient.Notify
	go cloudClient.Run(ctx)
	if apiKeyFunc() != "" {
		cloudClient.Notify()
	}

	// Create agent server using the same shared client service
	server, err := agent.NewServer(clientService, certs, args.Version(), notificationHandler)
	if err != nil {
		return fmt.Errorf("failed to create agent server: %w", err)
	}
	go func() {
		log.Info().Msgf("Dozzle agent version %s", args.Version())
		log.Info().Msgf("Agent listening on %s", listener.Addr().String())

		if err := server.Serve(listener); err != nil {
			log.Error().Err(err).Msg("failed to serve")
		}
	}()
	<-ctx.Done()
	stop()
	log.Info().Msg("Shutting down agent")
	server.Stop()
	log.Debug().Str("file", agentAddrFile).Msg("Removing agent address file")
	os.Remove(agentAddrFile)
	return nil
}
