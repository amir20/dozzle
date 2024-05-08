package analytics

type BeaconEvent struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	Browser           string `json:"browser"`
	AuthProvider      string `json:"authProvider"`
	FilterLength      int    `json:"filterLength"`
	Clients           int    `json:"clients"`
	HasCustomAddress  bool   `json:"hasCustomAddress"`
	HasCustomBase     bool   `json:"hasCustomBase"`
	HasHostname       bool   `json:"hasHostname"`
	RunningContainers int    `json:"runningContainers"`
	HasActions        bool   `json:"hasActions"`
	IsSwarmMode       bool   `json:"isSwarmMode"`
	ServerVersion     string `json:"serverVersion"`
	ServerID          string `json:"serverID"`
}
