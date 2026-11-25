package hostservice

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/amir20/dozzle/internal/notification"
	"github.com/amir20/dozzle/internal/notification/dispatcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A webhook that never reached disk is gone after a restart, so the caller has
// to hear about it instead of getting a success.
func TestMultiHostService_AddDispatcherReturnsSaveError(t *testing.T) {
	ctx := t.Context()
	// A file where the data directory should be makes the save fail.
	blocker := filepath.Join(t.TempDir(), "data")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	manager := notification.NewManager(
		notification.NewContainerLogListener(ctx, nil),
		notification.NewContainerStatsListener(ctx, nil),
		notification.NewContainerEventListener(ctx, nil),
	)
	m := &MultiHostService{
		manager:             &listManager{},
		notificationManager: manager,
		persister: &notification.Persister{
			Manager:          manager,
			NotificationPath: filepath.Join(blocker, "notifications.yml"),
		},
	}

	d, err := dispatcher.NewWebhookDispatcher("hook", "https://example.com/hook", "", nil)
	require.NoError(t, err)

	_, err = m.AddDispatcher(d)
	assert.ErrorIs(t, err, notification.ErrSave)
}
