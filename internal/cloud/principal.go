package cloud

import (
	"fmt"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
)

// PrincipalKind is who a tool call runs as. Cloud reaches back into Dozzle from
// more than one direction and they do not carry the same authority, so every
// call names which one it is rather than inferring it from what happens to be
// on the connection.
type PrincipalKind int

const (
	// PrincipalAPIKey is a turn Cloud started on its own behalf — a question
	// asked in Telegram or Discord, or triage acting on an alert. There is no
	// Dozzle user behind it; the API key holder is whoever linked the
	// instance, so --enable-actions is the only gate. This is the zero value
	// because it is what every tool call meant before principals existed.
	PrincipalAPIKey PrincipalKind = iota
	// PrincipalInstance is background work owned by the process: log ingest,
	// stats collection, scans. Nobody asked for it, so it reads and never acts.
	PrincipalInstance
	// PrincipalUser is a turn a signed-in Dozzle user started from the app.
	// Their container filter and their roles both apply.
	PrincipalUser
)

// Principal carries what a caller may see and what a caller may do. Labels
// confine visibility the same way they do on every HTTP route; Roles are only
// consulted for PrincipalUser, because the other two kinds have no user whose
// roles could be read.
type Principal struct {
	Kind   PrincipalKind
	Labels container.ContainerLabels
	Roles  auth.Role
}

// UserPrincipal builds the principal for a signed-in user's request. Callers
// resolve labels exactly as the rest of the web layer does, so a user reaches
// the same containers through the assistant as they do through the UI.
func UserPrincipal(labels container.ContainerLabels, roles auth.Role) Principal {
	return Principal{Kind: PrincipalUser, Labels: labels, Roles: roles}
}

// InstancePrincipal builds the read-only principal for background work.
func InstancePrincipal(labels container.ContainerLabels) Principal {
	return Principal{Kind: PrincipalInstance, Labels: labels}
}

// APIKeyPrincipal builds the principal for Cloud-originated turns.
func APIKeyPrincipal(labels container.ContainerLabels) Principal {
	return Principal{Kind: PrincipalAPIKey, Labels: labels}
}

// mutatingTools maps every tool that changes something to the role a user needs
// to call it. Membership here also means the tool is gated behind
// --enable-actions, which is how these tools have always been gated.
var mutatingTools = map[string]auth.Role{
	toolStartContainer:           auth.Actions,
	toolStopContainer:            auth.Actions,
	toolRestartContainer:         auth.Actions,
	toolRemoveContainer:          auth.Actions,
	toolUpdateContainer:          auth.Actions,
	toolCreateLogNotification:    auth.Notifications,
	toolCreateMetricNotification: auth.Notifications,
	toolCreateEventNotification:  auth.Notifications,
}

// readTools maps reads that still need a role. Notification rules are instance
// wide and match containers by expression rather than by filter, so listing
// them can name containers a filtered user cannot otherwise see.
var readTools = map[string]auth.Role{
	toolListNotifications: auth.Notifications,
}

// mayCall reports why a principal cannot invoke a tool, or nil when it can.
func (p Principal) mayCall(name string, enableActions bool) error {
	if role, mutating := mutatingTools[name]; mutating {
		if !enableActions {
			return fmt.Errorf("container actions are not enabled")
		}
		switch p.Kind {
		case PrincipalInstance:
			return fmt.Errorf("%s is not available to background work", name)
		case PrincipalUser:
			if !p.Roles.Has(role) {
				return fmt.Errorf("you do not have permission to run %s", name)
			}
		}
		return nil
	}

	if role, gated := readTools[name]; gated && p.Kind == PrincipalUser && !p.Roles.Has(role) {
		return fmt.Errorf("you do not have permission to run %s", name)
	}
	return nil
}

// scopedHost is a host service already bound to a principal's labels. Tools go
// through this rather than the bare ToolHostService so that there is no call
// left for a tool to make with the wrong scope, or with none.
type scopedHost struct {
	hosts  ToolHostService
	labels container.ContainerLabels
}

func (s scopedHost) ListAllContainers() ([]container.Container, []error) {
	return s.hosts.ListAllContainers(s.labels)
}

func (s scopedHost) FindContainer(host string, id string) (*container_support.ContainerService, error) {
	return s.hosts.FindContainer(host, id, s.labels)
}

// Hosts is not label-scoped. A host is not a container, and the rest of the app
// shows every host to every user regardless of filter.
func (s scopedHost) Hosts() []container.Host {
	return s.hosts.Hosts()
}
