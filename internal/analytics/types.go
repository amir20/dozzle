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
