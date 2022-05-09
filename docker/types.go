package docker

// Container represents an internal representation of docker containers
type Container struct {
	ID      string   `json:"id"`
	Names   []string `json:"names"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	ImageID string   `json:"imageId"`
	Command string   `json:"command"`
	Created int64    `json:"created"`
	State   string   `json:"state"`
	Status  string   `json:"status"`
}

// ContainerStat represent stats instant for a container
type ContainerStat struct {
	ID            string `json:"id"`
	CPUPercent    int64  `json:"cpu"`
	MemoryPercent int64  `json:"memory"`
	MemoryUsage   int64  `json:"memoryUsage"`
}

// ContainerEvent represents events that are triggered
type ContainerEvent struct {
	ActorID string `json:"actorId"`
	Name    string `json:"name"`
}

type LogEvent struct {
	Message   string                 `json:"m,omitempty"`
	Timestamp int64                  `json:"ts"`
	Data      map[string]interface{} `json:"d,omitempty"`
}
