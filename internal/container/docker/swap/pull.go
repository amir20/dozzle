package swap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

const (
	// pullAttempts bounds a pull the registry keeps rate-limiting.
	pullAttempts = 4
	// minRetryAfter floors the registry's retry-after. A registry can answer
	// with well under a millisecond, and retrying that fast meets the same limit.
	minRetryAfter = time.Second
	// maxRetryAfter is the longest wait honoured. A longer one fails the pull
	// rather than holding up the rest of the run.
	maxRetryAfter = time.Minute
)

var retryAfterPattern = regexp.MustCompile(`retry-after:\s*([0-9.]+[a-zµμ]+)`)

// wait sleeps for d or until ctx is done. A variable so tests need not sleep.
var wait = func(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// Pull starts a pull with start and reads it to the end like ReadPull. When
// the registry answers toomanyrequests it waits the retry-after the error
// gives, floored at minRetryAfter, and pulls again, up to pullAttempts times.
// Any other error, or a retry-after over maxRetryAfter, fails at once.
func Pull(ctx context.Context, start func() (io.ReadCloser, error), progress func(container.UpdateProgress)) error {
	for attempt := 1; ; attempt++ {
		err := pullOnce(start, progress)
		if err == nil {
			return nil
		}
		delay, ok := retryAfter(err)
		if !ok || attempt == pullAttempts || delay > maxRetryAfter {
			return err
		}
		log.Warn().Err(err).Int("attempt", attempt).Dur("retry_in", delay).Msg("pull rate-limited, retrying")
		if werr := wait(ctx, delay); werr != nil {
			return err
		}
	}
}

func pullOnce(start func() (io.ReadCloser, error), progress func(container.UpdateProgress)) error {
	reader, err := start()
	if err != nil {
		return fmt.Errorf("pull failed: %w", err)
	}
	defer reader.Close()
	return ReadPull(reader, progress)
}

// retryAfter reports whether err is a registry rate limit and how long to wait.
// A rate limit without a parseable retry-after waits minRetryAfter.
func retryAfter(err error) (time.Duration, bool) {
	msg := err.Error()
	if !strings.Contains(msg, "toomanyrequests") {
		return 0, false
	}
	if m := retryAfterPattern.FindStringSubmatch(msg); m != nil {
		if d, perr := time.ParseDuration(m[1]); perr == nil {
			return max(d, minRetryAfter), true
		}
	}
	return minRetryAfter, true
}

type pullMessage struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	// ErrorDetail is how the engine reports a pull that failed after the
	// stream started: a missing tag, a registry that refused, a full disk.
	ErrorDetail *struct {
		Message string `json:"message"`
	} `json:"errorDetail"`
}

// ReadPull reads an image pull's stream to its end, reporting each message as
// UpdatePulling progress. The engine ends the stream cleanly even when the
// pull failed, so an error message in it is returned as the error: without
// that a failed pull reads as "already up to date".
func ReadPull(r io.Reader, progress func(container.UpdateProgress)) error {
	decoder := json.NewDecoder(r)
	for {
		var msg pullMessage
		if err := decoder.Decode(&msg); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("pull decode failed: %w", err)
		}
		if msg.ErrorDetail != nil {
			return fmt.Errorf("pull failed: %s", msg.ErrorDetail.Message)
		}
		progress(container.UpdateProgress{
			Status:  container.UpdatePulling,
			Layer:   msg.ID,
			Current: msg.ProgressDetail.Current,
			Total:   msg.ProgressDetail.Total,
		})
	}
}
