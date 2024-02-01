package docker

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type ContainerStore struct {
	containers     map[string]*Container
	client         Client
	statsCollector *StatsCollector
	subscribers    []chan ContainerEvent
}

func NewContainerStore(client Client) *ContainerStore {
	s := &ContainerStore{
		containers:     make(map[string]*Container),
		client:         client,
		statsCollector: NewStatsCollector(client),
	}

	go s.init(context.Background())
	go s.statsCollector.StartCollecting(context.Background())

	return s
}

func (s *ContainerStore) List() []Container {
	containers := make([]Container, 0, len(s.containers))
	for _, c := range s.containers {
		containers = append(containers, *c)
	}

	return containers
}

func (s *ContainerStore) Client() Client {
	return s.client
}

func (s *ContainerStore) Subscribe(events chan ContainerEvent) {
	s.subscribers = append(s.subscribers, events)
}

func (s *ContainerStore) Unsubscribe(toRemove chan ContainerEvent) {
	for i, sub := range s.subscribers {
		if sub == toRemove {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			break
		}
	}
}

func (s *ContainerStore) SubscribeStats(stats chan ContainerStat) {
	s.statsCollector.Subscribe(stats)
}

func (s *ContainerStore) UnsubscribeStats(toRemove chan ContainerStat) {
	s.statsCollector.Unsubscribe(toRemove)
}

func (s *ContainerStore) init(ctx context.Context) {
	containers, err := s.client.ListContainers()
	if err != nil {
		log.Fatalf("error while listing containers: %v", err)
	}

	for _, c := range containers {
		c := c // create a new variable to avoid capturing the loop variable
		s.containers[c.ID] = &c
	}

	events := make(chan ContainerEvent)
	s.client.Events(ctx, events)

	stats := make(chan ContainerStat)
	s.statsCollector.Subscribe(stats)
	defer s.statsCollector.Unsubscribe(stats)

	for {
		select {
		case event := <-events:
			log.Debugf("received event: %+v", event)
			switch event.Name {
			case "start":
				if container, err := s.client.FindContainer(event.ActorID); err == nil {
					s.containers[container.ID] = &container
				}
			case "destroy":
				log.Debugf("container %s destroyed", event.ActorID)
				delete(s.containers, event.ActorID)

			case "die":
				if container, ok := s.containers[event.ActorID]; ok {
					log.Debugf("container %s died", container.ID)
					container.State = "exited"
				}
			case "health_status: healthy", "health_status: unhealthy":
				healthy := "unhealthy"
				if event.Name == "health_status: healthy" {
					healthy = "healthy"
				}
				if container, ok := s.containers[event.ActorID]; ok {
					log.Debugf("container %s is %s", container.ID, healthy)
					container.Health = healthy
				}
			}

			for _, sub := range s.subscribers {
				sub <- event
			}
		case stat := <-stats:
			if container, ok := s.containers[stat.ID]; ok {
				container.Stats.Push(stat)
			}
		case <-ctx.Done():
			return
		}
	}
}
