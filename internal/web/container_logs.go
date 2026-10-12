package web

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
)

// containerLogs is the whole log (or the view's time range) of any number of
// containers in one response, each named as `container=<host>/<id>` and filtered
// the way the log viewer filters. `format=zip` is the download: one file per container.
// `format=jsonl` is the SQL view's table: LogEvents as JSON lines, each carrying its
// container id in `c`, and only the JSON ones with `jsonOnly`. Otherwise it is plain
// text for the copy button, each container under a `── name ──` header when there is
// more than one.
func (h *handler) containerLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	format := q.Get("format")
	zipped := format == "zip"

	if zipped && h.config.Authorization.Provider != NONE && !auth.UserFromContext(r.Context()).Roles.Has(auth.Download) {
		log.Warn().Msg("user is not permitted to download logs from container")
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	stdTypes := parseStdTypes(r)
	if stdTypes == 0 {
		http.Error(w, "stdout or stderr is required", http.StatusBadRequest)
		return
	}

	filter, err := parseLogFilter(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	win, err := parseFetchWindow(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	to := win.to
	if to.IsZero() {
		to = time.Now()
	}

	// Every container is found before anything is written, so a bad id is a 4xx
	// rather than a response that stops halfway.
	services, status, err := h.findContainers(q["container"], h.resolveLabels(r))
	if err != nil {
		http.Error(w, err.Error(), status)
		return
	}

	if zipped {
		analytics.Count("logs.download")
		writeLogsZip(w, r, services, win.from, to, stdTypes, filter)
		return
	}

	if format == "jsonl" {
		w.Header().Set("Content-Type", "application/x-jsonl; charset=UTF-8")
		writer, done := compressedWriter(w, r)
		defer done()
		writeLogsJSON(writer, r, services, win.from, to, stdTypes, filter, q.Has("jsonOnly"))
		return
	}

	analytics.Count("logs.copy")
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	writer, done := compressedWriter(w, r)
	defer done()
	writeLogsText(writer, r, services, win.from, to, stdTypes, filter, copyHeaders(services, h.hostService.Hosts))
}

// findContainers resolves `<host>/<id>` pairs straight from each host's container
// store. Neither host ids nor container ids (k8s ids are `ns:pod:container`) can
// hold a `/`, so the first one splits them.
func (h *handler) findContainers(refs []string, labels container.ContainerLabels) ([]*container.ContainerService, int, error) {
	if len(refs) == 0 {
		return nil, http.StatusBadRequest, fmt.Errorf("at least one container is required")
	}
	services := make([]*container.ContainerService, 0, len(refs))
	for _, ref := range refs {
		host, id, ok := strings.Cut(ref, "/")
		if !ok || host == "" || id == "" {
			return nil, http.StatusBadRequest, fmt.Errorf("container must be <host>/<id>: %q", ref)
		}
		service, err := h.hostService.FindContainer(host, id, labels)
		if err != nil {
			return nil, http.StatusNotFound, err
		}
		services = append(services, service)
	}
	return services, 0, nil
}

// writeLogsJSON writes each container's events as JSON lines, one container after
// another in request order. The response has started, so a container that fails is
// logged and skipped rather than cutting the table short.
func writeLogsJSON(w io.Writer, r *http.Request, services []*container.ContainerService, from, to time.Time, stdTypes container.StdType, filter logFilter, onlyComplex bool) {
	for _, service := range services {
		events, err := service.LogsBetweenDates(r.Context(), from, to, stdTypes)
		if err != nil {
			log.Error().Err(err).Str("container", service.Container.ID).Msg("error fetching logs")
			continue
		}
		writeAllLogs(w, events, filter, onlyComplex, false)
	}
}

// writeLogsText writes each container's lines as the single-container text fetch
// does, one container at a time so a big stack holds one log stream open.
func writeLogsText(w io.Writer, r *http.Request, services []*container.ContainerService, from, to time.Time, stdTypes container.StdType, filter logFilter, headers []string) {
	for i, service := range services {
		if len(services) > 1 {
			if i > 0 {
				fmt.Fprint(w, "\n")
			}
			fmt.Fprintf(w, "── %s ──\n", headers[i])
		}
		if err := writeContainerText(w, r, service, from, to, stdTypes, filter); err != nil {
			// The response has started, so the failure goes into the text itself
			// rather than leaving a block silently empty.
			log.Error().Err(err).Str("container", service.Container.ID).Msg("error fetching logs")
			fmt.Fprintf(w, "error: %v\n", err)
		}
	}
}

// writeContainerText writes one container's lines. With nothing to filter on, the
// raw stream is copied a line at a time without parsing, which is several times
// faster on a big log. A search or a level filter needs each line parsed.
func writeContainerText(w io.Writer, r *http.Request, service *container.ContainerService, from, to time.Time, stdTypes container.StdType, filter logFilter) error {
	if !filter.empty() {
		events, err := service.LogsBetweenDates(r.Context(), from, to, stdTypes)
		if err != nil {
			return err
		}
		writeAllLogs(w, events, filter, false, true)
		return nil
	}
	reader, err := service.RawLogs(r.Context(), from, to, stdTypes)
	if err != nil {
		return err
	}
	defer reader.Close()
	return writeRawText(w, reader)
}

// writeRawText copies a raw log stream as copy text: the engine's timestamp in
// front of each line is dropped and ANSI escapes are stripped. Unlike the parsed
// path it keeps a logfmt or `prefix {json}` line as the app printed it, rather
// than re-encoded as JSON.
func writeRawText(w io.Writer, r io.Reader) error {
	br := bufio.NewReaderSize(r, 64*1024)
	var long []byte
	for {
		chunk, err := br.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			// A line longer than the buffer: gather it before writing it whole.
			long = append(long, chunk...)
			continue
		}
		line := chunk
		if long != nil {
			line = append(long, chunk...)
			long = long[:0:0]
		}
		if len(line) > 0 {
			if _, werr := io.WriteString(w, rawCopyLine(line)); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// rawCopyLine turns `2026-10-12T10:00:00.000000000Z message\n` into `message\n`.
func rawCopyLine(line []byte) string {
	text := string(line)
	if i := strings.IndexByte(text, ' '); i >= 20 && text[4] == '-' && text[10] == 'T' {
		text = text[i+1:]
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return container.StripANSI(text)
}

// copyHeaders names each container in a copy: its name, plus its host's name when
// another container in the same copy shares the name (the same service on two
// hosts). Asking for host names reaches every agent, so it only happens on a clash.
func copyHeaders(services []*container.ContainerService, hosts func() []container.Host) []string {
	counts := make(map[string]int, len(services))
	for _, service := range services {
		counts[service.Container.Name]++
	}
	headers := make([]string, len(services))
	var hostNames map[string]string
	for i, service := range services {
		c := service.Container
		headers[i] = c.Name
		if counts[c.Name] < 2 {
			continue
		}
		if hostNames == nil {
			hostNames = make(map[string]string)
			for _, host := range hosts() {
				hostNames[host.ID] = host.Name
			}
		}
		name := hostNames[c.Host]
		if name == "" {
			name = c.Host
		}
		headers[i] = fmt.Sprintf("%s (%s)", c.Name, name)
	}
	return headers
}

// writeLogsZip is the download: one timestamped .log file per container. With no
// search or level filter it copies the raw stream, which is faster than parsing it.
func writeLogsZip(w http.ResponseWriter, r *http.Request, services []*container.ContainerService, from, to time.Time, stdTypes container.StdType, filter logFilter) {
	now := time.Now()
	nowFmt := now.Format("2006-01-02T15-04-05")

	zipName := "container-logs"
	if name := sanitizeFileName(r.URL.Query().Get("name")); name != "" {
		zipName = name
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.zip", zipName, nowFmt))
	w.Header().Set("Content-Type", "application/zip")

	zw := zip.NewWriter(w)
	defer zw.Close()

	filtered := !filter.empty()

	// Errors from here on are logged only, since the response has started.
	for _, service := range services {
		id := service.Container.ID
		// Container names come from the engine or an agent, so they can carry path
		// separators or "..". Entry names must stay a plain file name.
		baseName := sanitizeFileName(service.Container.Name)
		if baseName == "" {
			baseName = sanitizeFileName(id)
		}
		f, err := zw.CreateHeader(&zip.FileHeader{
			Name:     fmt.Sprintf("%s-%s.log", baseName, nowFmt),
			Modified: now,
			Method:   zip.Deflate,
		})
		if err != nil {
			log.Error().Err(err).Msgf("error creating zip entry for container %s", id)
			return
		}

		if !filtered {
			reader, err := service.RawLogs(r.Context(), from, to, stdTypes)
			if err != nil {
				log.Error().Err(err).Msgf("error getting logs for container %s", id)
				return
			}
			if _, err := io.Copy(f, reader); err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", id)
				return
			}
			continue
		}

		events, err := service.LogsBetweenDates(r.Context(), from, to, stdTypes)
		if err != nil {
			log.Error().Err(err).Msgf("error getting logs for container %s", id)
			return
		}
		for event := range events {
			if !filter.matchesText(event) || (len(filter.levels) > 0 && !filter.matchesLevel(event)) {
				continue
			}
			timestamp := time.UnixMilli(event.Timestamp).UTC().Format(time.RFC3339Nano)
			if event.Type == container.LogTypeGroup {
				if fragments, ok := event.Message.([]container.LogFragment); ok {
					for _, fragment := range fragments {
						// The regex filter marks matches inside grouped fragments for
						// highlighting; strip them so they don't leak into the file.
						if _, err := fmt.Fprintf(f, "%s %s\n", timestamp, container.StripHighlightMarkers(fragment.Message)); err != nil {
							log.Error().Err(err).Msgf("error writing log for container %s", id)
							return
						}
					}
				}
				continue
			}
			if _, err := fmt.Fprintf(f, "%s %s\n", timestamp, event.RawMessage); err != nil {
				log.Error().Err(err).Msgf("error writing log for container %s", id)
				return
			}
		}
	}
}

// sanitizeFileName keeps alphanumerics, hyphens, underscores and dots, replacing
// everything else with a hyphen, and trims leading dots so the result is never
// "." / ".." or a hidden file.
func sanitizeFileName(name string) string {
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '-'
	}, name)
	return strings.TrimLeft(sanitized, ".")
}
