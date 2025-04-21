package container

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"golang.org/x/sync/semaphore"
)

type StatsCollector interface {
	Start(parentCtx context.Context) bool
	Subscribe(ctx context.Context, stats chan<- ContainerStat)
	Stop()
}

type ContainerStore struct {
	containers              *xsync.Map[string, *Container]
	subscribers             *xsync.Map[context.Context, chan<- ContainerEvent]
	newContainerSubscribers *xsync.Map[context.Context, chan<- Container]
	client                  Client
	statsCollector          StatsCollector
	wg                      sync.WaitGroup
	connected               atomic.Bool
	events                  chan ContainerEvent
	ctx                     context.Context
	labels                  ContainerLabels
}

const defaultTimeout = 10 * time.Second

func NewContainerStore(ctx context.Context, client Client, statsCollect StatsCollector, labels ContainerLabels) *ContainerStore {
	log.Debug().Str("host", client.Host().Name).Interface("labels", labels).Msg("initializing container store")

	s := &ContainerStore{
		containers:              xsync.NewMap[string, *Container](),
		client:                  client,
		subscribers:             xsync.NewMap[context.Context, chan<- ContainerEvent](),
		newContainerSubscribers: xsync.NewMap[context.Context, chan<- Container](),
		statsCollector:          statsCollect,
		wg:                      sync.WaitGroup{},
		events:                  make(chan ContainerEvent),
		ctx:                     ctx,
		labels:                  labels,
	}

	s.wg.Add(1)

	go s.init()

	return s
}

var (
	ErrContainerNotFound = errors.New("container not found")
	maxFetchParallelism  = int64(30)
)

func (s *ContainerStore) checkConnectivity() error {
	if s.connected.CompareAndSwap(false, true) {
		go func() {
			log.Debug().Str("host", s.client.Host().Name).Msg("docker store subscribing docker events")
			err := s.client.ContainerEvents(s.ctx, s.events)
			if !errors.Is(err, context.Canceled) {
				log.Error().Err(err).Str("host", s.client.Host().Name).Msg("docker store unexpectedly disconnected from docker events")
			}
			s.connected.Store(false)
		}()

		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
		if containers, err := s.client.ListContainers(ctx, s.labels); err != nil {
			return err
		} else {
			s.containers.Clear()

			for _, c := range containers {
				s.containers.Store(c.ID, &c)
			}

			running := lo.Filter(containers, func(item Container, index int) bool {
				return item.State != "exited" && !item.FullyLoaded
			})

			sem := semaphore.NewWeighted(maxFetchParallelism)

			for i, c := range running {
				if err := sem.Acquire(s.ctx, 1); err != nil {
					log.Error().Err(err).Msg("failed to acquire semaphore")
					break
				}
				go func(c Container, i int) {
					defer sem.Release(1)
					ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
					defer cancel()
					if container, err := s.client.FindContainer(ctx, c.ID); err == nil {
						s.containers.Store(c.ID, &container)
					}
				}(c, i)
			}

			if err := sem.Acquire(s.ctx, maxFetchParallelism); err != nil {
				log.Error().Err(err).Msg("failed to acquire semaphore")
			}

			log.Debug().Int("containers", len(containers)).Msg("finished initializing container store")
		}
	}

	return nil
}

func (s *ContainerStore) ListContainers(labels ContainerLabels) ([]Container, error) {
	s.wg.Wait()

	if err := s.checkConnectivity(); err != nil {
		return nil, err
	}

	containers := make([]Container, 0)
	if labels.Exists() {
		validContainers, err := s.client.ListContainers(s.ctx, labels)
		if err != nil {
			return nil, err
		}

		if len(validContainers) == 0 {
			log.Warn().Interface("userLabels", labels).Msg("no containers found with user labels")
		}

		validIDMap := lo.KeyBy(validContainers, func(item Container) string {
			return item.ID
		})

		s.containers.Range(func(_ string, c *Container) bool {
			if _, ok := validIDMap[c.ID]; ok {
				containers = append(containers, *c)
			}
			return true
		})
	} else {
		s.containers.Range(func(_ string, c *Container) bool {
			containers = append(containers, *c)
			return true
		})
	}

	return containers, nil
}

func (s *ContainerStore) FindContainer(id string, labels ContainerLabels) (Container, error) {
	s.wg.Wait()
	if labels.Exists() {
		validContainers, err := s.client.ListContainers(s.ctx, labels)
		if err != nil {
			return Container{}, err
		}

		validIDMap := lo.KeyBy(validContainers, func(item Container) string {
			return item.ID
		})

		if _, ok := validIDMap[id]; !ok {
			log.Warn().Str("id", id).Msg("user doesn't have access to container")
			return Container{}, ErrContainerNotFound
		}
	}

	if container, ok := s.containers.Load(id); ok {
		if !container.FullyLoaded {
			log.Debug().Str("id", id).Msg("container is not fully loaded, fetching it")
			if newContainer, ok := s.containers.Compute(id, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
				if loaded {
					ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
					defer cancel()
					if newContainer, err := s.client.FindContainer(ctx, id); err == nil {
						return &newContainer, xsync.UpdateOp
					} else {
						log.Error().Err(err).Msg("failed to fetch container")
						return c, xsync.CancelOp
					}
				}
				return c, xsync.CancelOp
			}); ok {
				go func() {
					event := ContainerEvent{
						Name:      "update",
						Host:      newContainer.Host,
						ActorID:   id,
						Container: newContainer,
					}

					s.subscribers.Range(func(c context.Context, events chan<- ContainerEvent) bool {
						select {
						case events <- event:
						case <-c.Done():
							s.subscribers.Delete(c)
						}
						return true
					})
				}()
				return *newContainer, nil
			}
		}
		return *container, nil
	} else {
		log.Warn().Str("id", id).Msg("container not found")
		return Container{}, ErrContainerNotFound
	}
}

func (s *ContainerStore) Client() Client {
	return s.client
}

func (s *ContainerStore) SubscribeEvents(ctx context.Context, events chan<- ContainerEvent) {
	go func() {
		if s.statsCollector.Start(s.ctx) {
			s.containers.Range(func(_ string, c *Container) bool {
				c.Stats.Clear()
				return true
			})
		}
	}()

	s.subscribers.Store(ctx, events)
	go func() {
		<-ctx.Done()
		s.subscribers.Delete(ctx)
		s.statsCollector.Stop()
	}()
}

func (s *ContainerStore) SubscribeStats(ctx context.Context, stats chan<- ContainerStat) {
	s.statsCollector.Subscribe(ctx, stats)
}

func (s *ContainerStore) SubscribeNewContainers(ctx context.Context, containers chan<- Container) {
	s.newContainerSubscribers.Store(ctx, containers)
	go func() {
		<-ctx.Done()
		s.newContainerSubscribers.Delete(ctx)
	}()
}

func (s *ContainerStore) init() {
	stats := make(chan ContainerStat)
	s.statsCollector.Subscribe(s.ctx, stats)

	s.checkConnectivity()

	s.wg.Done()

	for {
		select {
		case event := <-s.events:
			log.Debug().Str("event", event.Name).Str("id", event.ActorID).Msg("received container event")
			switch event.Name {
			case "create":
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

				if container, err := s.client.FindContainer(ctx, event.ActorID); err == nil {
					list, _ := s.client.ListContainers(ctx, s.labels)

					// make sure the container is in the list of containers when using filter
					valid := lo.ContainsBy(list, func(item Container) bool {
						return item.ID == container.ID
					})

					if valid {
						s.containers.Store(container.ID, &container)
						s.newContainerSubscribers.Range(func(c context.Context, containers chan<- Container) bool {
							select {
							case containers <- container:
							case <-c.Done():
							}
							return true
						})
					}
				}
				cancel()

			case "start":
				ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)

				if container, err := s.client.FindContainer(ctx, event.ActorID); err == nil {
					list, _ := s.client.ListContainers(ctx, s.labels)

					// make sure the container is in the list of containers when using filter
					valid := lo.ContainsBy(list, func(item Container) bool {
						return item.ID == container.ID
					})

					if valid {
						s.containers.Store(container.ID, &container)
						s.newContainerSubscribers.Range(func(c context.Context, containers chan<- Container) bool {
							select {
							case containers <- container:
							case <-c.Done():
							}
							return true
						})
					}
				}
				cancel()
			case "destroy":
				log.Debug().Str("id", event.ActorID).Msg("container destroyed")
				s.containers.Delete(event.ActorID)

			case "update":
				started := false
				updatedContainer, _ := s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded && event.Container != nil {
						newContainer := event.Container
						if newContainer.State == "running" && c.State != "running" {
							started = true
						}
						c.Name = newContainer.Name
						c.State = newContainer.State
						c.Labels = newContainer.Labels
						c.StartedAt = newContainer.StartedAt
						c.FinishedAt = newContainer.FinishedAt
						c.Created = newContainer.Created
						c.Host = newContainer.Host
						return c, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})

				if started {
					s.subscribers.Range(func(ctx context.Context, events chan<- ContainerEvent) bool {
						select {
						case events <- ContainerEvent{
							Name:    "start",
							ActorID: updatedContainer.ID,
							Host:    updatedContainer.Host,
						}:
						case <-ctx.Done():
							s.subscribers.Delete(ctx)
						}
						return true
					})
				}

			case "die":
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded {
						log.Debug().Str("id", c.ID).Msg("container died")
						c.State = "exited"
						c.FinishedAt = time.Now()
						return c, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})
			case "health_status: healthy", "health_status: unhealthy":
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}

				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded {
						log.Debug().Str("id", c.ID).Str("health", healthy).Msg("container health status changed")
						c.Health = healthy
						return c, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})

			case "rename":
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, xsync.ComputeOp) {
					if loaded {
						log.Debug().Str("id", event.ActorID).Str("name", event.ActorAttributes["name"]).Msg("container renamed")
						c.Name = event.ActorAttributes["name"]
						return c, xsync.UpdateOp
					} else {
						return c, xsync.CancelOp
					}
				})
			}
			s.subscribers.Range(func(c context.Context, events chan<- ContainerEvent) bool {
				select {
				case events <- event:
				case <-c.Done():
					s.subscribers.Delete(c)
				}
				return true
			})

		case stat := <-stats:
			if container, ok := s.containers.Load(stat.ID); ok {
				stat.ID = ""
				container.Stats.Push(stat)
			}
		case <-s.ctx.Done():
			return
		}
	}
}
