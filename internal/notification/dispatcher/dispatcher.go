package dispatcher

import (
	"context"

	"github.com/amir20/dozzle/types"
)

// Dispatcher is responsible for sending notifications to external systems
type Dispatcher interface {
	// Send sends a notification to the configured destination
	Send(ctx context.Context, notification types.Notification) error
}
