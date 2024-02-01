package docker

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type ContainerStore struct {
	containers     map[string]Container
	client         Client
	statsCollector *StatCollector
	subscribers    []chan ContainerEvent
}

func NewContainerStore(client Client) *ContainerStore {
	s := &ContainerStore{
		containers:     make(map[string]Container),
		client:         client,
		statsCollector: NewStatCollector(client),
	}

	go s.init(context.Background())
	go s.statsCollector.StartCollecting(context.Background())

	return s
}

func (s *ContainerStore) Get(id string) (Container, bool) {
	container, ok := s.containers[id]
	return container, ok
}

func (s *ContainerStore) List() []Container {
	containers := make([]Container, 0, len(s.containers))
	for _, c := range s.containers {
		containers = append(containers, c)
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
			close(toRemove)
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
		return
	}

	for _, c := range containers {
		s.containers[c.ID] = c
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
					s.containers[container.ID] = container
				}
			case "destroy":
				delete(s.containers, event.ActorID)
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
