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
	m := unsavableService(t)

	d, err := dispatcher.NewWebhookDispatcher("hook", "https://example.com/hook", "", nil)
	require.NoError(t, err)

	_, err = m.AddDispatcher(d)
	assert.ErrorIs(t, err, notification.ErrSave)
	assert.Empty(t, m.Dispatchers(), "a failed add must not stay live, or every retry leaves a duplicate")
}

// PUT and PATCH on a rule go through these two, so editing or toggling a rule
// must not answer 200 when nothing reached disk.
func TestMultiHostService_SubscriptionEditsReturnSaveError(t *testing.T) {
	m := unsavableService(t)

	sub := &notification.Subscription{Name: "rule", ContainerExpression: "true", LogExpression: "true"}
	assert.ErrorIs(t, m.AddSubscription(sub), notification.ErrSave)
	assert.Empty(t, m.Subscriptions(), "a failed add must not stay live, or every retry leaves a duplicate")

	// Edits need a rule that is live, which a failed add no longer leaves behind.
	require.NoError(t, m.notificationManager.AddSubscription(sub))

	assert.ErrorIs(t, m.UpdateSubscription(sub.ID, map[string]any{"enabled": false}), notification.ErrSave)

	replaced := &notification.Subscription{ID: sub.ID, Name: "rule", ContainerExpression: "true", LogExpression: "true"}
	assert.ErrorIs(t, m.ReplaceSubscription(replaced), notification.ErrSave)
}

// unsavableService puts a file where the data directory should be, so every save fails.
func unsavableService(t *testing.T) *MultiHostService {
	ctx := t.Context()
	blocker := filepath.Join(t.TempDir(), "data")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	manager := notification.NewManager(
		notification.NewContainerLogListener(ctx, nil),
		notification.NewContainerStatsListener(ctx, nil),
		notification.NewContainerEventListener(ctx, nil),
	)
	return &MultiHostService{
		manager:             &listManager{},
		notificationManager: manager,
		persister: &notification.Persister{
			Manager:          manager,
			NotificationPath: filepath.Join(blocker, "notifications.yml"),
		},
	}
}
