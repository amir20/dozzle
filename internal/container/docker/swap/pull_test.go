package swap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
)

const rateLimited = "Error response from daemon: toomanyrequests: retry-after: 739.812µs, allowed: 44000/minute"

// stubWait records each wait instead of sleeping.
func stubWait(t *testing.T) *[]time.Duration {
	t.Helper()
	var waits []time.Duration
	orig := wait
	wait = func(_ context.Context, d time.Duration) error {
		waits = append(waits, d)
		return nil
	}
	t.Cleanup(func() { wait = orig })
	return &waits
}

// pulls returns a start func that answers each call with the next error, then
// succeeds with an empty stream.
func pulls(errs ...error) (func() (io.ReadCloser, error), *int) {
	calls := 0
	return func() (io.ReadCloser, error) {
		calls++
		if calls <= len(errs) && errs[calls-1] != nil {
			return nil, errs[calls-1]
		}
		return io.NopCloser(strings.NewReader("")), nil
	}, &calls
}

func noProgress(container.UpdateProgress) {}

func TestPullRetriesRateLimit(t *testing.T) {
	waits := stubWait(t)
	start, calls := pulls(errors.New(rateLimited))

	if err := Pull(context.Background(), start, noProgress); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if *calls != 2 {
		t.Fatalf("calls = %d, want 2", *calls)
	}
	if len(*waits) != 1 || (*waits)[0] != minRetryAfter {
		t.Fatalf("waits = %v, want [%v] (sub-second retry-after floored)", *waits, minRetryAfter)
	}
}

func TestPullHonoursRetryAfter(t *testing.T) {
	waits := stubWait(t)
	start, _ := pulls(errors.New("toomanyrequests: retry-after: 12s"))

	if err := Pull(context.Background(), start, noProgress); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if len(*waits) != 1 || (*waits)[0] != 12*time.Second {
		t.Fatalf("waits = %v, want [12s]", *waits)
	}
}

func TestPullRetriesRateLimitInStream(t *testing.T) {
	stubWait(t)
	calls := 0
	start := func() (io.ReadCloser, error) {
		calls++
		if calls == 1 {
			return io.NopCloser(strings.NewReader(`{"errorDetail":{"message":"toomanyrequests: retry-after: 2s"}}`)), nil
		}
		return io.NopCloser(strings.NewReader("")), nil
	}

	if err := Pull(context.Background(), start, noProgress); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestPullGivesUpAfterAttempts(t *testing.T) {
	waits := stubWait(t)
	errs := make([]error, pullAttempts)
	for i := range errs {
		errs[i] = errors.New(rateLimited)
	}
	start, calls := pulls(errs...)

	err := Pull(context.Background(), start, noProgress)
	if err == nil || !strings.Contains(err.Error(), "pull failed: ") {
		t.Fatalf("err = %v, want the rate-limit error", err)
	}
	if *calls != pullAttempts || len(*waits) != pullAttempts-1 {
		t.Fatalf("calls = %d, waits = %d", *calls, len(*waits))
	}
	// Sub-millisecond retry-afters back off exponentially, capped at a minute,
	// so the attempts span well over the minute the registry counts in.
	want := []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, time.Minute}
	if fmt.Sprint(*waits) != fmt.Sprint(want) {
		t.Fatalf("waits = %v, want %v", *waits, want)
	}
}

func TestPullFailsAtOnce(t *testing.T) {
	cases := map[string]string{
		"other error":          "manifest unknown",
		"retry-after too long": "toomanyrequests: retry-after: 6h0m0s",
	}
	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			waits := stubWait(t)
			start, calls := pulls(errors.New(msg))
			if err := Pull(context.Background(), start, noProgress); err == nil {
				t.Fatal("Pull succeeded, want error")
			}
			if *calls != 1 || len(*waits) != 0 {
				t.Fatalf("calls = %d, waits = %v, want one call and no wait", *calls, *waits)
			}
		})
	}
}

func TestPullStopsWhenCancelled(t *testing.T) {
	orig := wait
	wait = func(context.Context, time.Duration) error { return context.Canceled }
	t.Cleanup(func() { wait = orig })
	start, calls := pulls(errors.New(rateLimited))

	if err := Pull(context.Background(), start, noProgress); err == nil {
		t.Fatal("Pull succeeded, want error")
	}
	if *calls != 1 {
		t.Fatalf("calls = %d, want 1", *calls)
	}
}
