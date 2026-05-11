package cloud

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/amir20/dozzle/internal/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeNotificationService is a minimal NotificationService for tool tests.
type fakeNotificationService struct {
	mu        sync.Mutex
	subs      []*notification.Subscription
	nextID    int
	addedSubs []*notification.Subscription
}

func (f *fakeNotificationService) Subscriptions() []*notification.Subscription {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*notification.Subscription(nil), f.subs...)
}

func (f *fakeNotificationService) AddSubscription(sub *notification.Subscription) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	sub.ID = f.nextID
	f.subs = append(f.subs, sub)
	f.addedSubs = append(f.addedSubs, sub)
	return nil
}

func TestListNotifications_NotConfigured(t *testing.T) {
	resp := ExecuteTool(context.Background(), "list_notifications", `{"instance_id":"42"}`, ToolDeps{EnableActions: true})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "notifications are not configured")
}

func TestListNotifications_EmptyAndPopulated(t *testing.T) {
	svc := &fakeNotificationService{
		subs: []*notification.Subscription{
			{ID: 3, Name: "nginx errors", Enabled: true, DispatcherID: 0, ContainerExpression: `name contains "nginx"`, LogExpression: `level == "error"`},
			{ID: 4, Name: "cpu high", Enabled: true, DispatcherID: 0, ContainerExpression: `state == "running"`, MetricExpression: `cpu > 80`},
		},
	}

	resp := ExecuteTool(context.Background(), "list_notifications", `{"instance_id":"42"}`, ToolDeps{EnableActions: true, NotificationService: svc})
	require.True(t, resp.Success)

	msg := resp.GetNotification().Message
	assert.Contains(t, msg, "nginx errors")
	assert.Contains(t, msg, "cpu high")
	assert.Contains(t, msg, `log:       level == "error"`)
	assert.Contains(t, msg, `metric:    cpu > 80`)
}

func TestCreateLogNotification_Success(t *testing.T) {
	svc := &fakeNotificationService{}
	args := `{
		"name":"nginx 5xx",
		"instance_id":"42",
		"container_expression":"name contains \"nginx\"",
		"log_expression":"level == \"error\" && message contains \"5\""
	}`

	resp := ExecuteTool(context.Background(), "create_log_notification", args, ToolDeps{EnableActions: true, NotificationService: svc})
	require.True(t, resp.Success, "response: %s", resp.Error)

	require.Len(t, svc.addedSubs, 1)
	sub := svc.addedSubs[0]
	assert.Equal(t, "nginx 5xx", sub.Name)
	assert.Equal(t, cloudDispatcherID, sub.DispatcherID, "alerts must route to the cloud dispatcher")
	assert.Equal(t, `name contains "nginx"`, sub.ContainerExpression)
	assert.Equal(t, `level == "error" && message contains "5"`, sub.LogExpression)
	assert.Empty(t, sub.MetricExpression)
	assert.Empty(t, sub.EventExpression)
}

func TestCreateLogNotification_MissingFieldsError(t *testing.T) {
	svc := &fakeNotificationService{}
	// container_expression intentionally missing
	args := `{"name":"x","instance_id":"42","log_expression":"level == \"error\""}`

	resp := ExecuteTool(context.Background(), "create_log_notification", args, ToolDeps{EnableActions: true, NotificationService: svc})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "container_expression is required")
	assert.Empty(t, svc.addedSubs)
}

func TestCreateLogNotification_InvalidExpressionPropagates(t *testing.T) {
	// Route through the real Subscription.CompileExpressions so the test
	// actually exercises the expr grammar rather than a mocked error path.
	realSvc := &compilingSvc{base: &fakeNotificationService{}}
	args := `{"name":"bad","instance_id":"42","container_expression":"name ==","log_expression":"level == \"error\""}`

	resp := ExecuteTool(context.Background(), "create_log_notification", args, ToolDeps{EnableActions: true, NotificationService: realSvc})
	assert.False(t, resp.Success)
	assert.True(t, strings.HasPrefix(resp.Error, "creating subscription:"), "expected propagated compile error, got: %s", resp.Error)
}

func TestCreateMetricNotification_WithCooldownAndWindow(t *testing.T) {
	svc := &fakeNotificationService{}
	args := `{
		"name":"cpu burn",
		"instance_id":"42",
		"container_expression":"state == \"running\"",
		"metric_expression":"cpu > 80 || memory > 90",
		"cooldown_seconds":600,
		"sample_window_seconds":60
	}`

	resp := ExecuteTool(context.Background(), "create_metric_notification", args, ToolDeps{EnableActions: true, NotificationService: svc})
	require.True(t, resp.Success, "response: %s", resp.Error)

	require.Len(t, svc.addedSubs, 1)
	sub := svc.addedSubs[0]
	assert.Equal(t, cloudDispatcherID, sub.DispatcherID)
	assert.Equal(t, "cpu > 80 || memory > 90", sub.MetricExpression)
	assert.Equal(t, 600, sub.Cooldown)
	assert.Equal(t, 60, sub.SampleWindow)
}

func TestCreateEventNotification_Success(t *testing.T) {
	svc := &fakeNotificationService{}
	args := `{
		"name":"crashes",
		"instance_id":"42",
		"container_expression":"name contains \"api\"",
		"event_expression":"name in [\"die\", \"oom\"]"
	}`

	resp := ExecuteTool(context.Background(), "create_event_notification", args, ToolDeps{EnableActions: true, NotificationService: svc})
	require.True(t, resp.Success, "response: %s", resp.Error)

	require.Len(t, svc.addedSubs, 1)
	assert.Equal(t, cloudDispatcherID, svc.addedSubs[0].DispatcherID)
	assert.Equal(t, `name in ["die", "oom"]`, svc.addedSubs[0].EventExpression)
}

func TestCreateLogNotification_ActionsDisabled(t *testing.T) {
	svc := &fakeNotificationService{}
	args := `{"name":"x","instance_id":"42","container_expression":"name == \"a\"","log_expression":"level == \"error\""}`

	resp := ExecuteTool(context.Background(), "create_log_notification", args, ToolDeps{EnableActions: false, NotificationService: svc})
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Error, "container actions are not enabled")
}

// compilingSvc delegates to the real Subscription.CompileExpressions so tests
// can assert that genuine expr-lang syntax errors bubble back to the LLM.
type compilingSvc struct {
	base *fakeNotificationService
}

func (c *compilingSvc) Subscriptions() []*notification.Subscription {
	return c.base.Subscriptions()
}

func (c *compilingSvc) AddSubscription(sub *notification.Subscription) error {
	if err := sub.CompileExpressions(); err != nil {
		return err
	}
	return c.base.AddSubscription(sub)
}
