package web

import (
	"net/http"
	"time"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/amir20/dozzle/graph"
	"github.com/amir20/dozzle/internal/cache"
	"github.com/amir20/dozzle/internal/releases"
)

func (h *handler) graphqlHandler() http.Handler {
	releasesCache := cache.New(func() ([]releases.Release, error) {
		return releases.Fetch(h.config.Version)
	}, time.Hour)

	srv := gqlhandler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			HostService:     h.hostService,
			ReleasesFetcher: releasesCache.Get,
		},
	}))

	return srv
}

func (h *handler) graphqlPlaygroundHandler() http.Handler {
	endpoint := h.config.Base + "/api/graphql"
	// Avoid double slashes when base is "/"
	if h.config.Base == "/" {
		endpoint = "/api/graphql"
	}
	return playground.Handler("Dozzle GraphQL", endpoint)
}
