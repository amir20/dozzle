package docker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/puzpuzpuz/xsync/v3"
	lop "github.com/samber/lo/parallel"
	log "github.com/sirupsen/logrus"
)

type ContainerStore struct {
	containers              *xsync.MapOf[string, *Container]
	subscribers             *xsync.MapOf[context.Context, chan<- ContainerEvent]
	newContainerSubscribers *xsync.MapOf[context.Context, chan<- Container]
	client                  Client
	statsCollector          *StatsCollector
	wg                      sync.WaitGroup
	connected               atomic.Bool
	events                  chan ContainerEvent
	ctx                     context.Context
}

func NewContainerStore(ctx context.Context, client Client) *ContainerStore {
	s := &ContainerStore{
		containers:              xsync.NewMapOf[string, *Container](),
		client:                  client,
		subscribers:             xsync.NewMapOf[context.Context, chan<- ContainerEvent](),
		newContainerSubscribers: xsync.NewMapOf[context.Context, chan<- Container](),
		statsCollector:          NewStatsCollector(client),
		wg:                      sync.WaitGroup{},
		events:                  make(chan ContainerEvent),
		ctx:                     ctx,
	}

	s.wg.Add(1)

	go s.init()

	return s
}

var ErrContainerNotFound = errors.New("container not found")

func (s *ContainerStore) checkConnectivity() error {
	if s.connected.CompareAndSwap(false, true) {
		go func() {
			log.Debugf("subscribing to docker events from container store %s", s.client.Host())
			err := s.client.ContainerEvents(s.ctx, s.events)
			if !errors.Is(err, context.Canceled) {
				log.Errorf("docker store unexpectedly disconnected from docker events from %s with %v", s.client.Host(), err)
			}
			s.connected.Store(false)
		}()

		if containers, err := s.client.ListContainers(); err != nil {
			return err
		} else {
			s.containers.Clear()
			lop.ForEach(containers, func(c Container, _ int) {
				container, _ := s.client.FindContainer(c.ID)
				s.containers.Store(c.ID, &container)
			})
		}
	}

	return nil
}

func (s *ContainerStore) ListContainers() ([]Container, error) {
	s.wg.Wait()

	if err := s.checkConnectivity(); err != nil {
		return nil, err
	}
	containers := make([]Container, 0)
	s.containers.Range(func(_ string, c *Container) bool {
		containers = append(containers, *c)
		return true
	})

	return containers, nil
}

func (s *ContainerStore) FindContainer(id string) (Container, error) {
	list, err := s.ListContainers()
	if err != nil {
		return Container{}, err
	}

	for _, c := range list {
		if c.ID == id {
			return c, nil
		}
	}

	log.Warnf("container %s not found in store", id)
	return Container{}, ErrContainerNotFound
}

func (s *ContainerStore) Client() Client {
	return s.client
}

func (s *ContainerStore) SubscribeEvents(ctx context.Context, events chan<- ContainerEvent) {
	go func() {
		if s.statsCollector.Start(s.ctx) {
			log.Debug("clearing container stats as stats collector has been stopped")
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
			log.Tracef("received event: %+v", event)
			switch event.Name {
			case "start":
				if container, err := s.client.FindContainer(event.ActorID); err == nil {
					log.Debugf("container %s started", container.ID)
					s.containers.Store(container.ID, &container)
					s.newContainerSubscribers.Range(func(c context.Context, containers chan<- Container) bool {
						select {
						case containers <- container:
						case <-c.Done():
						}
						return true
					})
				}
			case "destroy":
				log.Debugf("container %s destroyed", event.ActorID)
				s.containers.Delete(event.ActorID)

			case "die":
				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, bool) {
					if loaded {
						log.Debugf("container %s died", c.ID)
						c.State = "exited"
						return c, false
					} else {
						return c, true
					}
				})
			case "health_status: healthy", "health_status: unhealthy":
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}

				s.containers.Compute(event.ActorID, func(c *Container, loaded bool) (*Container, bool) {
					if loaded {
						log.Debugf("health status for container %s is %s", c.ID, healthy)
						c.Health = healthy
						return c, false
					} else {
						return c, true
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
