package notification

import (
	"testing"

	"github.com/amir20/dozzle/internal/container"
	"github.com/amir20/dozzle/types"
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestSubscription_MatchesLog(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		log        types.NotificationLog
		want       bool
	}{
		{
			name:       "matches message substring",
			expression: `message contains "error"`,
			log: types.NotificationLog{
				Message: "an error occurred",
			},
			want: true,
		},
		{
			name:       "does not match message substring",
			expression: `message contains "error"`,
			log: types.NotificationLog{
				Message: "everything is fine",
			},
			want: false,
		},
		{
			name:       "matches log level",
			expression: `level == "error"`,
			log: types.NotificationLog{
				Level: "error",
			},
			want: true,
		},
		{
			name:       "matches stream",
			expression: `stream == "stderr"`,
			log: types.NotificationLog{
				Stream: "stderr",
			},
			want: true,
		},
		{
			name:       "complex expression with AND",
			expression: `level == "error" && message contains "fatal"`,
			log: types.NotificationLog{
				Level:   "error",
				Message: "fatal exception",
			},
			want: true,
		},
		{
			name:       "complex expression fails AND",
			expression: `level == "error" && message contains "fatal"`,
			log: types.NotificationLog{
				Level:   "error",
				Message: "normal error",
			},
			want: false,
		},
		{
			name:       "complex expression with OR",
			expression: `level == "error" || message contains "warning"`,
			log: types.NotificationLog{
				Level:   "info",
				Message: "warning: something happened",
			},
			want: true,
		},
		{
			name:       "matches type",
			expression: `type == "stdout"`,
			log: types.NotificationLog{
				Type: "stdout",
			},
			want: true,
		},
		{
			name:       "matches complex message number field",
			expression: `message.number == 123`,
			log: types.NotificationLog{
				Message: map[string]any{
					"number": 123,
				},
			},
			want: true,
		},
		{
			name:       "matches complex message string field",
			expression: `message.status == "error"`,
			log: types.NotificationLog{
				Message: map[string]any{
					"status": "error",
				},
			},
			want: true,
		},
		{
			name:       "matches complex message nested field",
			expression: `message.user.name == "admin"`,
			log: types.NotificationLog{
				Message: map[string]any{
					"user": map[string]any{
						"name": "admin",
					},
				},
			},
			want: true,
		},
		{
			name:       "does not match complex message wrong value",
			expression: `message.number == 123`,
			log: types.NotificationLog{
				Message: map[string]any{
					"number": 456,
				},
			},
			want: false,
		},
		{
			name:       "complex message with multiple conditions",
			expression: `message.level == "error" && message.code >= 400`,
			log: types.NotificationLog{
				Message: map[string]any{
					"level": "error",
					"code":  500,
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := expr.Compile(tt.expression, expr.Env(types.NotificationLog{}), expr.AsBool())
			require.NoError(t, err, "failed to compile expression")

			sub := &Subscription{
				LogProgram: program,
			}

			got := sub.MatchesLog(tt.log)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSubscription_MatchesLog_NilProgram(t *testing.T) {
	sub := &Subscription{
		LogProgram: nil,
	}

	log := types.NotificationLog{
		Message: "test",
	}

	got := sub.MatchesLog(log)
	assert.False(t, got, "should return false when LogProgram is nil")
}

func TestSubscription_MatchesLog_InvalidExpression(t *testing.T) {
	// Create a program that will cause a runtime error by dividing by zero
	program, err := expr.Compile(`1 / 0 == 1`, expr.AsBool())
	require.NoError(t, err)

	sub := &Subscription{
		LogProgram: program,
	}

	log := types.NotificationLog{
		Message: "test",
	}

	got := sub.MatchesLog(log)
	assert.False(t, got, "should return false when expression evaluation fails")
}

func TestFromLogEvent_OrderedMapConversion(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		logEvent   container.LogEvent
		want       bool
	}{
		{
			name:       "orderedmap string any - matches number field",
			expression: `message.value > 0`,
			logEvent: container.LogEvent{
				Message: orderedmap.New[string, any](
					orderedmap.WithInitialData(
						orderedmap.Pair[string, any]{Key: "value", Value: 123},
						orderedmap.Pair[string, any]{Key: "name", Value: "test"},
					),
				),
			},
			want: true,
		},
		{
			name:       "orderedmap string any - matches string field",
			expression: `message.name == "test"`,
			logEvent: container.LogEvent{
				Message: orderedmap.New[string, any](
					orderedmap.WithInitialData(
						orderedmap.Pair[string, any]{Key: "value", Value: 123},
						orderedmap.Pair[string, any]{Key: "name", Value: "test"},
					),
				),
			},
			want: true,
		},
		{
			name:       "orderedmap string string - matches field",
			expression: `message.level == "error"`,
			logEvent: container.LogEvent{
				Message: orderedmap.New[string, string](
					orderedmap.WithInitialData(
						orderedmap.Pair[string, string]{Key: "level", Value: "error"},
						orderedmap.Pair[string, string]{Key: "msg", Value: "something failed"},
					),
				),
			},
			want: true,
		},
		{
			name:       "orderedmap string string - does not match",
			expression: `message.level == "error"`,
			logEvent: container.LogEvent{
				Message: orderedmap.New[string, string](
					orderedmap.WithInitialData(
						orderedmap.Pair[string, string]{Key: "level", Value: "info"},
						orderedmap.Pair[string, string]{Key: "msg", Value: "all good"},
					),
				),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert LogEvent to NotificationLog (this is where OrderedMap conversion happens)
			notificationLog := FromLogEvent(tt.logEvent)

			// Verify the message is now a regular map
			_, isMap := notificationLog.Message.(map[string]any)
			assert.True(t, isMap, "message should be converted to map[string]any")

			// Compile and run the expression
			program, err := expr.Compile(tt.expression, expr.Env(types.NotificationLog{}), expr.AsBool())
			require.NoError(t, err, "failed to compile expression")

			sub := &Subscription{
				LogProgram: program,
			}

			got := sub.MatchesLog(notificationLog)
			assert.Equal(t, tt.want, got)
		})
	}
}
