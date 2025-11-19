package web

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	support_web "github.com/amir20/dozzle/internal/support/web"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func (h *handler) downloadLogs(w http.ResponseWriter, r *http.Request) {
	hostIds := strings.Split(chi.URLParam(r, "hostIds"), ",")
	if len(hostIds) == 0 {
		log.Error().Msg("no container ids provided")
		http.Error(w, "no container ids provided", http.StatusBadRequest)
		return
	}

	userLabels := h.config.Labels
	permit := true
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
		permit = user.Roles.Has(auth.Download)
	}

	if !permit {
		log.Warn().Msg("user is not permitted to download logs from container")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	now := time.Now()
	nowFmt := now.Format("2006-01-02T15-04-05")

	var stdTypes container.StdType
	if r.URL.Query().Has("stdout") {
		stdTypes |= container.STDOUT
	}
	if r.URL.Query().Has("stderr") {
		stdTypes |= container.STDERR
	}

	if stdTypes == 0 {
		http.Error(w, "stdout or stderr is required", http.StatusBadRequest)
		return
	}

	// Parse filter regex if provided
	var regex *regexp.Regexp
	var err error
	if r.URL.Query().Has("filter") {
		regex, err = support_web.ParseRegex(r.URL.Query().Get("filter"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Parse level filters if provided
	levels := make(map[string]struct{})
	if r.URL.Query().Has("levels") {
		for _, level := range r.URL.Query()["levels"] {
			levels[level] = struct{}{}
		}
	}

	// Set headers for zip file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=container-logs-%s.zip", nowFmt))
	w.Header().Set("Content-Type", "application/zip")

	// Create zip writer
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Process each container
	for _, hostId := range hostIds {
		parts := strings.Split(hostId, "~")
		if len(parts) != 2 {
			log.Error().Msgf("invalid host id: %s", hostId)
			http.Error(w, fmt.Sprintf("invalid host id: %s", hostId), http.StatusBadRequest)
			return
		}

		host := parts[0]
		id := parts[1]
		containerService, err := h.hostService.FindContainer(host, id, userLabels)
		if err != nil {
			log.Error().Err(err).Msgf("error finding container %s", id)
			http.Error(w, fmt.Sprintf("error finding container %s: %v", id, err), http.StatusBadRequest)
			return
		}

		// Create new file in zip for this container's logs
		fileName := fmt.Sprintf("%s-%s.log", containerService.Container.Name, nowFmt)
		f, err := zw.Create(fileName)
		if err != nil {
			log.Error().Err(err).Msgf("error creating zip entry for container %s", id)
			http.Error(w, fmt.Sprintf("error creating zip entry: %v", err), http.StatusInternalServerError)
			return
		}

		// Get container logs - use LogsBetweenDates if filtering is needed, otherwise use RawLogs
		if regex != nil || len(levels) > 0 {
			// Fetch parsed log events for filtering
			events, err := containerService.LogsBetweenDates(r.Context(), time.Time{}, now, stdTypes)
			if err != nil {
				log.Error().Err(err).Msgf("error getting logs for container %s", id)
				http.Error(w, fmt.Sprintf("error getting logs for container %s: %v", id, err), http.StatusInternalServerError)
				return
			}

			// Filter and write events
			for event := range events {
				// Apply regex filter if provided
				if regex != nil && !support_web.Search(regex, event) {
					continue
				}

				// Apply level filter if provided
				if len(levels) > 0 {
					if _, ok := levels[event.Level]; !ok {
						continue
					}
				}

				// Format timestamp in UTC
				timestamp := time.UnixMilli(event.Timestamp).UTC().Format(time.RFC3339Nano)

				// Write the log message to the file
				message := event.RawMessage
				if message == "" {
					// Fallback to formatted message if RawMessage is empty
					if msg, ok := event.Message.(string); ok {
						message = msg
					} else {
						// For complex messages, use a simple string representation
						message = fmt.Sprintf("%v", event.Message)
					}
				}

				// Write timestamp followed by message
				_, err = fmt.Fprintf(f, "%s %s\n", timestamp, message)
				if err != nil {
					log.Error().Err(err).Msgf("error writing log for container %s", id)
					http.Error(w, fmt.Sprintf("error writing logs for container %s: %v", id, err), http.StatusInternalServerError)
					return
				}
			}
		} else {
			// No filtering needed, use raw logs for better performance
			reader, err := containerService.RawLogs(r.Context(), time.Time{}, now, stdTypes)
			if err != nil {
				log.Error().Err(err).Msgf("error getting logs for container %s", id)
				http.Error(w, fmt.Sprintf("error getting logs for container %s: %v", id, err), http.StatusInternalServerError)
				return
			}

			// Copy logs to zip file
			_, err = io.Copy(f, reader)
			if err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", id)
				http.Error(w, fmt.Sprintf("error copying logs for container %s: %v", id, err), http.StatusInternalServerError)
				return
			}
		}
	}
}
