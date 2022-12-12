package analytics

type StartEvent struct {
	ClientId      string `json:"-"`
	Version       string `json:"version"`
	FilterLength  int    `json:"filterLength"`
	CustomAddress bool   `json:"customAddress"`
	CustomBase    bool   `json:"customBase"`
	Protected     bool   `json:"protected"`
}
