package container

import "strings"

// HealthStatusOf returns the health-check result encoded on a container event.
// Docker puts it in the action itself ("health_status: healthy"). Podman's
// Docker-compatible events endpoint uses the bare action "health_status" and
// puts the value in actor attributes (health_status / healthStatus / HealthStatus).
func HealthStatusOf(event ContainerEvent) (string, bool) {
	if after, ok := strings.CutPrefix(event.Name, "health_status: "); ok {
		after = strings.TrimSpace(after)
		if after != "" {
			return after, true
		}
	}
	if event.Name != "health_status" {
		return "", false
	}
	for _, key := range []string{"healthStatus", "health_status", "HealthStatus"} {
		if v := strings.TrimSpace(event.ActorAttributes[key]); v != "" {
			return v, true
		}
	}
	return "", false
}
