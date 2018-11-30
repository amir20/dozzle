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
