//go:generate go run github.com/99designs/gqlgen generate

package graph

import (
	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/amir20/dozzle/internal/releases"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/amir20/dozzle/types"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type HostService interface {
	// Notification rules (subscriptions)
	Subscriptions() []*notification.Subscription
	AddSubscription(sub *notification.Subscription) error
	ReplaceSubscription(sub *notification.Subscription) error
	UpdateSubscription(id int, updates map[string]any) error
	RemoveSubscription(id int)

	// Dispatchers
	Dispatchers() []types.DispatcherConfig
	AddDispatcher(d dispatcher.Dispatcher) int
	UpdateDispatcher(id int, d dispatcher.Dispatcher)
	RemoveDispatcher(id int)

	// Containers for preview
	ListAllContainers(labels container.ContainerLabels) ([]container.Container, []error)
	FindContainer(host string, id string, labels container.ContainerLabels) (*container_support.ContainerService, error)
}

type ReleasesFetcher func() ([]releases.Release, error)

type Resolver struct {
	HostService     HostService
	ReleasesFetcher ReleasesFetcher
}
