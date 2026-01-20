package web

import (
	"net/http"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/amir20/dozzle/graph"
)

func (h *handler) graphqlHandler() http.Handler {
	srv := gqlhandler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			HostService: h.hostService,
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
