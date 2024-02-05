package docker

import (
	"context"

	"github.com/puzpuzpuz/xsync/v3"
	log "github.com/sirupsen/logrus"
)

type ContainerStore struct {
	containers     *xsync.MapOf[string, *Container]
	subscribers    *xsync.MapOf[context.Context, chan ContainerEvent]
	client         Client
	statsCollector *StatsCollector
}

func NewContainerStore(ctx context.Context, client Client) *ContainerStore {
	s := &ContainerStore{
		containers:     xsync.NewMapOf[string, *Container](),
		client:         client,
		subscribers:    xsync.NewMapOf[context.Context, chan ContainerEvent](),
		statsCollector: NewStatsCollector(client),
	}

	s.init(ctx)
	s.statsCollector.StartCollecting(ctx)

	return s
}

func (s *ContainerStore) List() []Container {
	containers := make([]Container, 0)
	s.containers.Range(func(_ string, c *Container) bool {
		containers = append(containers, *c)
		return true
	})

	return containers
}

func (s *ContainerStore) Client() Client {
	return s.client
}

func (s *ContainerStore) Subscribe(ctx context.Context, events chan ContainerEvent) {
	s.subscribers.Store(ctx, events)
}

func (s *ContainerStore) SubscribeStats(ctx context.Context, stats chan ContainerStat) {
	s.statsCollector.Subscribe(ctx, stats)
}

func (s *ContainerStore) init(ctx context.Context) {
	events := make(chan ContainerEvent)
	s.client.Events(ctx, events)

	containers, err := s.client.ListContainers()
	if err != nil {
		log.Fatalf("error while listing containers: %v", err)
	}

	for _, c := range containers {
		c := c // create a new variable to avoid capturing the loop variable
		s.containers.Store(c.ID, &c)
	}

	stats := make(chan ContainerStat)
	s.statsCollector.Subscribe(ctx, stats)

	go func() {
		for {
			select {
			case event := <-events:
				log.Debugf("received event: %+v", event)
				switch event.Name {
				case "start":
					if container, err := s.client.FindContainer(event.ActorID); err == nil {
						log.Debugf("container %s started", container.ID)
						s.containers.Store(container.ID, &container)
					}
				case "destroy":
					log.Debugf("container %s destroyed", event.ActorID)
					s.containers.Delete(event.ActorID)

				case "die":
					if container, ok := s.containers.Load(event.ActorID); ok {
						log.Debugf("container %s died", container.ID)
						container.State = "exited"
					}
				case "health_status: healthy", "health_status: unhealthy":
					healthy := "unhealthy"
					if event.Name == "health_status: healthy" {
						healthy = "healthy"
					}
					if container, ok := s.containers.Load(event.ActorID); ok {
						log.Debugf("container %s is %s", container.ID, healthy)
						container.Health = healthy
					}
				}
				s.subscribers.Range(func(c context.Context, events chan ContainerEvent) bool {
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
			case <-ctx.Done():
				return
			}
		}
	}()
}
