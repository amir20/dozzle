package types

// BeaconEvent is everything Dozzle's analytics beacon can send, and the one place
// that lists it. Counts and flags only: never a container, image, host or user
// name, a path, or log content. --no-analytics turns every beacon off.
//
// Name is "start" when the process starts, "events" each time the UI opens, and
// "usage" once a day with the counters below.
type BeaconEvent struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	Browser           string `json:"browser"`           // user agent, events beacon only
	AuthProvider      string `json:"authProvider"`      // none, simple, forward-proxy, oidc
	FilterLength      int    `json:"filterLength"`      // number of DOZZLE_FILTER entries
	Clients           int    `json:"clients"`           // hosts this Dozzle shows
	HasCustomAddress  bool   `json:"hasCustomAddress"`  // --addr changed
	HasCustomBase     bool   `json:"hasCustomBase"`     // --base changed
	HasHostname       bool   `json:"hasHostname"`       // --hostname set
	RunningContainers int    `json:"runningContainers"` // containers listed when the UI opened
	HasActions        bool   `json:"hasActions"`
	HasShell          bool   `json:"hasShell"`
	IsSwarmMode       bool   `json:"isSwarmMode"`
	ServerVersion     string `json:"serverVersion"` // Docker Engine version
	ServerID          string `json:"serverID"`      // Docker Engine id, to count an install once
	Mode              string `json:"mode"`          // server, swarm, k8s
	RemoteAgents      int    `json:"remoteAgents"`
	RemoteClients     int    `json:"remoteClients"`
	SubCommand        string `json:"subCommand"`
	FileAgents        int    `json:"fileAgents"` // RemoteAgents added from the UI

	HostsByType     map[string]int `json:"hostsByType,omitempty"`     // local, agent, remote, swarm, k8s
	AgentsDown      int            `json:"agentsDown"`                // hosts not answering
	PrivateAgents   int            `json:"privateAgents"`             // UI agents on the hub's private pair
	CloudLinked     bool           `json:"cloudLinked"`               // Dozzle Cloud configured
	AlertRules      map[string]int `json:"alertRules,omitempty"`      // rules by kind: log, event, metric
	Destinations    map[string]int `json:"destinations,omitempty"`    // notification destinations by type
	AutoUpdate      string         `json:"autoUpdate,omitempty"`      // off, daily, weekly
	ContainersTotal *int           `json:"containersTotal,omitempty"` // containers of any state; left out when a host failed to list
	Labels          map[string]int `json:"labels,omitempty"`          // containers using dev.dozzle.name, group, url, icon; left out like containersTotal
	Users           string         `json:"users,omitempty"`           // simple-auth users as a range: 1, 2-5, 6-20, 21+
	Usage           map[string]int `json:"usage,omitempty"`           // usage beacon: counts since the last one, keys in analytics.UsageKeys
	Locales         map[string]int `json:"locales,omitempty"`         // usage beacon: UI sessions by UI language
	ActiveMinutes   string         `json:"activeMinutes,omitempty"`   // usage beacon: minutes the UI was in use, as a range
}
