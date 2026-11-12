package cli

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
	"syscall"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	dozzleconfig "github.com/amir20/dozzle/internal/config"
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/container/agent"
	"github.com/amir20/dozzle/internal/container/docker"
	"github.com/amir20/dozzle/internal/container/k8s"
	"github.com/amir20/dozzle/internal/hostservice"
	"github.com/amir20/dozzle/internal/imagecheck"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/web"
	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

// Main is the dozzle binary: a subcommand when one is given, the server
// otherwise. content holds the built frontend under dist/, certs the shared
// agent certificate pair.
func Main(content fs.FS, certs embed.FS) {
	ValidateEnvVars(Args{}, AgentCmd{})
	args, subcommand := ParseArgs()
	if subcommand != nil {
		runnable, ok := subcommand.(Runnable)
		if !ok {
			log.Fatal().Msg("Invalid command")
		}
		if err := runnable.Run(args, certs); err != nil {
			log.Fatal().Err(err).Msg("Failed to run command")
		}
		return
	}
	runServer(args, content, certs)
}

// runServer runs Dozzle in server, swarm or k8s mode until SIGINT or SIGTERM.
func runServer(args Args, content fs.FS, certs embed.FS) {
	// freshInstall is whether ./data held nothing from an earlier run when this
	// process started. Only a fresh install gets the no-login setup window.
	// Read before anything below writes to ./data (profiles, notification rules,
	// session secrets), so it reflects earlier runs only.
	freshInstall := dozzleconfig.FreshDataDir(filepath.Dir(dozzleconfig.Path))

	log.Info().Msgf("Dozzle version %s", args.Version())
	if args.NoAnalytics {
		analytics.Default.Disable()
	}
	dispatcher.UserAgent = fmt.Sprintf("Dozzle/%s", args.Version())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var hostService web.HostService
	var notificationService cloud.NotificationService
	// validateServer already limited the mode to these three.
	switch args.Mode {
	case "server":
		multiHostService := CreateMultiHostService(ctx, certs, args)
		if multiHostService.TotalClients() == 0 {
			log.Fatal().Msg("Could not connect to any Docker Engine")
		} else {
			log.Info().Int("clients", multiHostService.TotalClients()).Msg("Connected to Docker")
		}
		if err := multiHostService.StartNotificationManager(ctx, imagecheck.Mode(args.ImageCheckMode) == imagecheck.ModeAutomatic, args.ImageCheckInterval, args.Filter); err != nil {
			log.Fatal().Err(err).Msg("Could not start notification manager")
		}
		hostService = multiHostService
		notificationService = multiHostService
	case "swarm":
		// No host id override in swarm mode: identity here is the swarm node id,
		// which is already stable and is what every other node refers to this one by.
		localClient, err := docker.NewLocalClient("", container.DerivedHostID{})
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create docker client")
		}
		certs, err := ReadCertificates(certs, args.CertPath, args.KeyPath)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not read certificates")
		}
		// Only the operator's agents: dozzle.yml agents come from the server-mode UI,
		// and the private ones among them need a pair swarm mode never loads.
		agentManager := hostservice.NewRetriableClientManager(args.EnvAgents, nil, args.Timeout, certs)
		// Only these agents. Swarm nodes are found by a DNS lookup on each
		// container listing, which is not something to run on a timer.
		go agentManager.RetryFailedHosts(ctx)
		manager := hostservice.NewSwarmClientManager(localClient, certs, args.Timeout, agentManager, args.Filter)
		multiHostService := hostservice.NewMultiHostService(manager, args.Timeout)
		if err := multiHostService.StartNotificationManager(ctx, imagecheck.Mode(args.ImageCheckMode) == imagecheck.ModeAutomatic, args.ImageCheckInterval, args.Filter); err != nil {
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
		clientService := docker.NewService(localClient, args.Filter)
		server, err := agent.NewServer(clientService, certs, args.Version(), multiHostService.SwarmNotificationHandler())
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create agent")
		}
		go StartEvent(args, "swarm", localClient, "")
		go func() {
			log.Info().Msgf("Dozzle agent version in swarm mode %s", args.Version())
			if err := server.Serve(listener); err != nil {
				log.Error().Err(err).Msg("failed to serve")
			}
		}()
	case "k8s":
		localClient, err := k8s.NewClient(args.Namespace, args.Filter, container.NewHostIDResolver(args.HostID))
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create k8s client")
		}

		clusterService, err := hostservice.NewK8sClusterService(localClient, args.Timeout, args.Filter)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not create k8s cluster service")
		}

		if err := clusterService.StartNotificationManager(ctx, imagecheck.Mode(args.ImageCheckMode) == imagecheck.ModeAutomatic, args.ImageCheckInterval, args.Filter); err != nil {
			log.Fatal().Err(err).Msg("Could not start notification manager")
		}

		go StartEvent(args, "k8s", localClient, "")
		hostService = clusterService
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
		ImageCheckMode:      imagecheck.Mode(args.ImageCheckMode),
		HostService:         cloudHostService,
		Principal:           cloud.APIKeyPrincipal(args.Filter),
		NotificationService: notificationService,
		ConfigHostService:   newConfigHostService(args.Mode, hostService),
	})
	cloudClient.SetDeployment(args.Mode, swarmClusterID)
	cloudClient.SetStreamLogsFunc(func() bool {
		return hostService.CloudConfig().StreamLogsEnabled()
	})
	go cloudClient.Run(ctx)

	// In swarm mode, peer broadcasts of cloud config should kick this
	// replica's cloud client too, so every replica holds its own connection.
	if mhs, ok := hostService.(*hostservice.MultiHostService); ok {
		mhs.SetCloudNotifyFunc(cloudClient.Notify)
	}

	// If cloud is already configured at startup, start the client immediately
	if apiKeyFunc() != "" {
		cloudClient.Notify()
	}

	srv := createServer(args, content, freshInstall, hostService, web.CloudHooks{
		OnSetup:    cloudClient.Notify,
		OnUpdate:   cloudClient.Reconnect,
		SearchLogs: cloudClient.SearchLogs,
		GetAlerts:  cloudClient.GetAlerts,

		GetRecentAlerts:   cloudClient.GetRecentAlerts,
		GetRecentDeploys:  cloudClient.GetRecentDeploys,
		GetPatternContext: cloudClient.GetPatternContext,

		GetContainerMetrics: cloudClient.GetContainerMetrics,

		Chat: func(ctx context.Context, message string, view cloud.ViewContext, userRef string, principal cloud.Principal, emit func(cloud.ChatEvent)) error {
			// C10: credentials come from a resolver rather than the stored key.
			// It always answers with the instance key today. When a user can
			// attach a personal key, this is the only line that changes.
			return cloudClient.Chat(ctx, message, view, userRef, principal, apiKeyFunc, emit)
		},
	})
	go srv.RunUsageBeacon(ctx)

	if args.Mode == "server" {
		go web.RunAutoUpdateScheduler(ctx, hostService, web.Config{
			Mode:          args.Mode,
			EnableActions: args.EnableActions,
			Version:       args.Version(),
			Setup: web.SetupConfig{
				AutoUpdateMode: lockedValue(args.Locked.AutoUpdate, args.AutoUpdate),
				AutoUpdateTime: lockedValue(args.Locked.AutoUpdateTime, args.AutoUpdateTime),

				UpdateContainers: lockedValue(args.Locked.UpdateContainers, args.UpdateContainers),
			},
		}, srv.FlushUsage)
		go web.RunAutoPruneScheduler(ctx, hostService, web.Config{
			Mode:          args.Mode,
			EnableActions: args.EnableActions,
			Setup: web.SetupConfig{
				AutoPruneMode: lockedValue(args.Locked.AutoCleanup, args.AutoCleanup),
				AutoPruneTime: lockedValue(args.Locked.AutoCleanupTime, args.AutoCleanupTime),
			},
		})
	}

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
	// Usage is only sent once a day, so a restart would otherwise lose it.
	srv.FlushUsage()
	log.Debug().Msg("shut down complete")
}

// beaconBase is what every beacon from the web server repeats about how this
// process was started.
func beaconBase(args Args, simpleUsers int) types.BeaconEvent {
	b := BeaconBase(args, args.Mode)
	b.Users = analytics.BucketUsers(simpleUsers)
	return b
}

// lockedValue is value when a flag or env var set it, nil when dozzle.yml decides.
func lockedValue(locked bool, value string) *string {
	if !locked {
		return nil
	}
	return &value
}

// oauthProviders builds the external identity providers simple auth accepts.
// users.yml stays the allowlist either way: a provider configured here only adds
// a way to prove you are a user it already lists.
func oauthProviders(args Args) []auth.IdentityProvider {
	var providers []auth.IdentityProvider

	id, secret := args.AuthGithubClientID, args.AuthGithubClientSecret
	switch {
	case id != "" && secret != "":
		log.Debug().Msg("Enabling Sign in with GitHub")
		providers = append(providers, auth.NewGithubProvider(id, secret))
	case id != "" || secret != "":
		// Half-configured silently means no button at all, which reads as Dozzle
		// ignoring the setting.
		log.Warn().Msg("Both --auth-github-client-id and --auth-github-client-secret are required for Sign in with GitHub; ignoring GitHub configuration")
	}

	issuer, oidcID, oidcSecret := args.AuthOidcIssuer, args.AuthOidcClientID, args.AuthOidcClientSecret
	switch {
	case issuer != "" && oidcID != "" && oidcSecret != "":
		log.Debug().Str("issuer", issuer).Msg("Enabling OpenID Connect sign in")
		providers = append(providers, auth.NewOIDCProvider(issuer, oidcID, oidcSecret, args.AuthOidcName).AddScopes(args.AuthOidcScopes))
	case issuer != "" || oidcID != "" || oidcSecret != "":
		log.Warn().Msg("--auth-oidc-issuer, --auth-oidc-client-id and --auth-oidc-client-secret are all required for OpenID Connect; ignoring OIDC configuration")
	}

	return providers
}

// oidcAuth builds the oidc provider, where the token is the user database.
//
// It shares the --auth-oidc-* flags with simple auth, so the split between the
// two has to show up in behavior: this mode never opens users.yml, refuses the
// GitHub flags, and says at startup which claims it will read roles from.
func oidcAuth(args Args) web.OAuthAuthorizer {
	dataDir := "./data"
	for _, name := range []string{"users.yml", "users.yaml"} {
		if fileExists(filepath.Join(dataDir, name)) {
			log.Warn().Str("file", name).Msg("Ignoring the user database: with --auth-provider oidc every user comes from the token")
		}
	}

	authorizer := auth.NewOIDCAuth(auth.OIDCConfig{
		Issuer:       args.AuthOidcIssuer,
		ClientID:     args.AuthOidcClientID,
		ClientSecret: args.AuthOidcClientSecret,
		DisplayName:  args.AuthOidcName,
		RolesClaim:   args.AuthOidcRolesClaim,
		FiltersClaim: args.AuthOidcFiltersClaim,
		Scopes:       args.AuthOidcScopes,
		LogoutURL:    args.AuthLogoutUrl,
		DataDir:      dataDir,
	}, args.Base, args.SessionTTL, auth.SessionSecret(dataDir))

	log.Info().
		Str("issuer", args.AuthOidcIssuer).
		Str("rolesClaim", authorizer.RolesClaims()).
		Msg("Using OpenID Connect authentication; users and roles come from the token")

	return authorizer
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func createServer(args Args, content fs.FS, freshInstall bool, hostService web.HostService, cloudHooks web.CloudHooks) *web.Server {
	_, dev := os.LookupEnv("DEV")

	var provider web.AuthProvider = web.NONE
	var authorizer web.Authorizer
	simpleUsers := 0
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
		simpleUsers = len(db.Users)
		// Sits next to users.yml so it lands on the same volume operators already
		// persist, and so a multi-replica setup sharing that volume signs with the
		// same key.
		simpleAuth := auth.NewSimpleAuth(db, args.SessionTTL, auth.SessionSecret(filepath.Dir(userFilePath)))
		authorizer = simpleAuth

		if providers := oauthProviders(args); len(providers) > 0 {
			authorizer = auth.NewOAuthAuth(simpleAuth, args.Base, providers...)
		}
	} else if args.AuthProvider == "oidc" {
		provider = web.OIDC
		authorizer = oidcAuth(args)
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
		Beacon:      beaconBase(args, simpleUsers),
		Dev:         dev,
		Mode:        args.Mode,
		Authorization: web.Authorization{
			Provider:   provider,
			Authorizer: authorizer,
			TTL:        args.SessionTTL,
			LogoutUrl:  args.AuthLogoutUrl,
		},
		EnableActions:      args.EnableActions,
		EnableShell:        args.EnableShell,
		EnableMCP:          args.EnableMCP,
		MCPToken:           mcpToken(args, provider),
		DisableAvatars:     args.DisableAvatars,
		DisableSetupWizard: args.DisableSetupWizard,
		ReleaseCheckMode:   web.ReleaseCheckMode(args.ReleaseCheckMode),
		ImageCheckMode:     imageCheckMode,
		Labels:             args.Filter,
		Cloud:              cloudHooks,
		Setup: web.SetupConfig{
			LockedAuthProvider:  args.Locked.AuthProvider,
			LockedEnableActions: args.Locked.EnableActions,
			LockedEnableShell:   args.Locked.EnableShell,
			LockedAutoUpdate:    args.Locked.AutoUpdate || args.Locked.AutoUpdateTime,
			AutoUpdateMode:      lockedValue(args.Locked.AutoUpdate, args.AutoUpdate),
			AutoUpdateTime:      lockedValue(args.Locked.AutoUpdateTime, args.AutoUpdateTime),
			UpdateContainers:    lockedValue(args.Locked.UpdateContainers, args.UpdateContainers),
			AutoPruneMode:       lockedValue(args.Locked.AutoCleanup, args.AutoCleanup),
			AutoPruneTime:       lockedValue(args.Locked.AutoCleanupTime, args.AutoCleanupTime),
			StartedAt:           web.SetupWindowStart(time.Now(), freshInstall),
			EnvAgents:           args.EnvAgents,
			CustomCert:          CustomCert(args.CertPath, args.KeyPath),
		},
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

// mcpToken decides whether --mcp-token applies (ParseArgs already checked its
// shape). The token only means something where Dozzle runs its own MCP
// authorization server: with MCP on, under simple or oidc auth. Anywhere else
// it is ignored out loud rather than silently.
func mcpToken(args Args, provider web.AuthProvider) string {
	token := args.MCPToken
	if token == "" {
		return ""
	}
	if !args.EnableMCP {
		log.Warn().Msg("Ignoring --mcp-token: it needs --enable-mcp")
		return ""
	}
	if provider != web.SIMPLE && provider != web.OIDC {
		log.Warn().Str("provider", string(provider)).Msg("Ignoring --mcp-token: it only applies with --auth-provider simple or oidc")
		return ""
	}

	log.Debug().Msg("Accepting --mcp-token on /api/mcp")
	return token
}
