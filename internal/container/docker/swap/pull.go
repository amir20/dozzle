package swap

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/amir20/dozzle/internal/container"
)

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
