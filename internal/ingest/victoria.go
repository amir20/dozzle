package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/amir20/dozzle/internal/container"
	container_support "github.com/amir20/dozzle/internal/support/container"
	"github.com/rs/zerolog/log"
)

type VictoriaIngestor struct {
	service container_support.ClientService
	logs    chan *container.LogEvent
}

func NewVictoriaIngestor(service container_support.ClientService) *VictoriaIngestor {
	return &VictoriaIngestor{
		service: service,
		logs:    make(chan *container.LogEvent),
	}
}

func (v *VictoriaIngestor) consumeLogs(ctx context.Context) error {
	pr, pw := io.Pipe()

	req, err := http.NewRequestWithContext(ctx, "POST", "http://localhost:9428/insert/jsonline?_stream_fields=c&_time_field=ts&_msg_field=m", pr)
	if err != nil {
		return err
	}

	// Set headers as needed.
	req.Header.Set("Content-Type", "application/stream+json")

	go func() {
		defer pw.Close()
		writer := json.NewEncoder(pw)
		for event := range v.logs {
			log.Debug().Interface("event", event).Msg("Writing log to Victoria")
			err := writer.Encode(event)
			if err != nil {
				log.Error().Err(err).Msg("Error encoding log event")
			}
			log.Debug().Interface("event", event).Msg("Log written to Victoria")
		}
	}()

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (v *VictoriaIngestor) streamLogs(ctx context.Context, c container.Container) {
	err := v.service.StreamLogs(ctx, c, time.Now(), container.STDALL, v.logs)
	if err != nil {
		if errors.Is(err, io.EOF) {
			log.Debug().Str("container", c.ID).Msg("streaming ended")

		} else if !errors.Is(err, context.Canceled) {
			log.Error().Err(err).Str("container", c.ID).Msg("unknown error while streaming logs")
		}
	}
}

func (v *VictoriaIngestor) Start(ctx context.Context) error {
	go func() {
		err := v.consumeLogs(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Error consuming logs")
		}
	}()

	go func() {
		newContainers := make(chan container.Container)
		defer close(newContainers)

		v.service.SubscribeContainersStarted(ctx, newContainers)

		for {
			select {
			case c := <-newContainers:
				go v.streamLogs(ctx, c)
			case <-ctx.Done():
				return
			}
		}
	}()

	containers, err := v.service.ListContainers(ctx, container.ContainerLabels{})
	if err != nil {
		return err
	}

	for _, c := range containers {
		go v.streamLogs(ctx, c)
	}

	<-ctx.Done()

	return nil
}
