package docker

// podmanRemoveAction is what Podman's Docker-compatible endpoint sends when a
// container is deleted. Docker sends "destroy" for the same thing and never
// sends "remove" for a container. Left as is, a container removed on Podman
// stayed listed as exited until a reload (#5662).
const podmanRemoveAction = "remove"

// containerAction turns Podman's container actions into Docker's, so the
// store, the SSE stream, the notification rules and the agent match a single
// encoding.
func containerAction(action string) string {
	if action == podmanRemoveAction {
		return "destroy"
	}
	return action
}
