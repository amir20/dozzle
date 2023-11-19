package analytics

type StartEvent struct {
	ClientId         string `json:"-"`
	Version          string `json:"version"`
	FilterLength     int    `json:"filterLength"`
	RemoteHostLength int    `json:"remoteHostLength"`
	CustomAddress    bool   `json:"customAddress"`
	CustomBase       bool   `json:"customBase"`
	Protected        bool   `json:"protected"`
	HasHostname      bool   `json:"hasHostname"`
}

type RequestEvent struct {
	ClientId          string `json:"-"`
	TotalContainers   int    `json:"totalContainers"`
	RunningContainers int    `json:"runningContainers"`
}

type BeaconEvent struct {
	Version           string `json:"version"`
	Browser           string `json:"browser"`
	AuthProvider      string `json:"authProvider"`
	FilterLength      int    `json:"filterLength"`
	RemoteHostLength  int    `json:"remoteHostLength"`
	HasDocumentation  bool   `json:"hasDocumentation"`
	HasCustomAddress  bool   `json:"hasCustomAddress"`
	HasCustomBase     bool   `json:"hasCustomBase"`
	HasHostname       bool   `json:"hasHostname"`
	RunningContainers int    `json:"runningContainers"`
}
