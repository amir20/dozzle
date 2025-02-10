package web

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
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
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
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

		// Get container logs
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
