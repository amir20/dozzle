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
	container_support "github.com/amir20/dozzle/internal/support/container"
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

	// Validate all containers before starting to write response
	type containerInfo struct {
		hostId           string
		host             string
		id               string
		containerService *container_support.ContainerService
	}
	containers := make([]containerInfo, 0, len(hostIds))

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

		containers = append(containers, containerInfo{
			hostId:           hostId,
			host:             host,
			id:               id,
			containerService: containerService,
		})
	}

	// Set headers for zip file
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=container-logs-%s.zip", nowFmt))
	w.Header().Set("Content-Type", "application/zip")

	// Create zip writer
	zw := zip.NewWriter(w)
	defer zw.Close()

	// Process each container - errors after this point are logged only since response has started
	for _, c := range containers {
		// Create new file in zip for this container's logs
		fileName := fmt.Sprintf("%s-%s.log", c.containerService.Container.Name, nowFmt)
		f, err := zw.Create(fileName)
		if err != nil {
			log.Error().Err(err).Msgf("error creating zip entry for container %s", c.id)
			return
		}

		// Get container logs - use LogsBetweenDates if filtering is needed, otherwise use RawLogs
		if regex != nil || len(levels) > 0 {
			// Fetch parsed log events for filtering
			events, err := c.containerService.LogsBetweenDates(r.Context(), time.Time{}, now, stdTypes)
			if err != nil {
				log.Error().Err(err).Msgf("error getting logs for container %s", c.id)
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

				// Write timestamp followed by message
				_, err = fmt.Fprintf(f, "%s %s\n", timestamp, event.RawMessage)
				if err != nil {
					log.Error().Err(err).Msgf("error writing log for container %s", c.id)
					return
				}
			}
		} else {
			// No filtering needed, use raw logs for better performance
			reader, err := c.containerService.RawLogs(r.Context(), time.Time{}, now, stdTypes)
			if err != nil {
				log.Error().Err(err).Msgf("error getting logs for container %s", c.id)
				return
			}

			// Copy logs to zip file
			_, err = io.Copy(f, reader)
			if err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", c.id)
				return
			}
		}
	}
}
