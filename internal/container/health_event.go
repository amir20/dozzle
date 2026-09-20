package container

import "strings"

// HealthStatusOf returns the health-check result encoded on a container event.
// Docker puts it in the action itself ("health_status: healthy"). A bare
// "health_status" action (Podman) reads healthStatus / health_status /
// HealthStatus from actor attributes when a producer has filled them.
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
