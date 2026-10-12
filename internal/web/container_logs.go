package web

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// containerLogs is the whole log (or the view's time range) of any number of
// containers in one response, each named as `container=<host>/<id>` and filtered
// the way the log viewer filters. `format=zip` is the download: one file per container.
// `format=jsonl` is the SQL view's table: the JSON LogEvents only, as JSON lines,
// each carrying its container id in `c`. Otherwise it is plain text for the copy
// button, each container under a `── name ──` header when there is more than one.
// A container that is gone (a view keeps showing one the store has dropped) is
// left out; only a request where none is found is a 404.
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

	// Every container is found before anything is written, so a bad request is a
	// 4xx rather than a response that stops halfway.
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
		writeLogsJSON(writer, r, services, win.from, to, stdTypes, filter)
		return
	}

	analytics.Count("logs.copy")
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	writer, done := compressedWriter(w, r)
	defer done()
	writeLogsText(writer, r, services, win.from, to, stdTypes, filter, copyHeaders(services, h.hostService.Hosts))
}

// containerRef is one `<host>/<id>` a request names.
type containerRef struct{ host, id string }

// parseContainerRefs splits `<host>/<id>` pairs, keeping their order. Neither host
// ids nor container ids (k8s ids are `ns:pod:container`) can hold a `/`, so the
// first one splits them.
func parseContainerRefs(values []string) ([]containerRef, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one container is required")
	}
	refs := make([]containerRef, 0, len(values))
	for _, value := range values {
		host, id, ok := strings.Cut(value, "/")
		if !ok || host == "" || id == "" {
			return nil, fmt.Errorf("container must be <host>/<id>: %q", value)
		}
		refs = append(refs, containerRef{host, id})
	}
	return refs, nil
}

// findContainers resolves `<host>/<id>` pairs straight from each host's container
// store. A container the store no longer has is skipped, so one removed container
// does not fail the rest of a merged view.
func (h *handler) findContainers(values []string, labels container.ContainerLabels) ([]*container.ContainerService, int, error) {
	refs, err := parseContainerRefs(values)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	services, err := h.resolveContainers(refs, labels, nil)
	if err != nil {
		return nil, http.StatusNotFound, err
	}
	if len(services) == 0 {
		return nil, http.StatusNotFound, container.ErrContainerNotFound
	}
	return services, 0, nil
}

// resolveConcurrency bounds the lookups one request runs at once. For an agent
// host each one is a gRPC round-trip.
const resolveConcurrency = 8

// resolveContainers finds each ref in its host's store, several at a time so one
// slow host does not hold up the rest, and returns those found in request order.
// The lookup is also the ACL check. A container that is gone, or outside the
// user's labels, is skipped; the first other error is returned along with every
// container that did resolve. found, when set, is called from the lookup's
// goroutine as soon as its container resolves.
func (h *handler) resolveContainers(refs []containerRef, labels container.ContainerLabels, found func(*container.ContainerService)) ([]*container.ContainerService, error) {
	services := make([]*container.ContainerService, len(refs))
	errs := make([]error, len(refs))
	var group errgroup.Group
	group.SetLimit(resolveConcurrency)
	for i, ref := range refs {
		group.Go(func() error {
			service, err := h.hostService.FindContainer(ref.host, ref.id, labels)
			if err != nil {
				errs[i] = err
				return nil
			}
			services[i] = service
			if found != nil {
				found(service)
			}
			return nil
		})
	}
	group.Wait()

	var firstErr error
	for i, err := range errs {
		if errors.Is(err, container.ErrContainerNotFound) {
			log.Debug().Str("host", refs[i].host).Str("id", refs[i].id).Msg("skipping a container that is gone")
		} else if err != nil {
			log.Error().Err(err).Str("host", refs[i].host).Str("id", refs[i].id).Msg("error while finding container")
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	return slices.DeleteFunc(services, func(s *container.ContainerService) bool { return s == nil }), firstErr
}

// writeLogsJSON writes each container's JSON events as JSON lines, one container
// after another in request order. The SQL view builds its table with `unnest(m)`,
// which needs every `m` to be an object, so nothing else is sent. The response has started, so a container that fails is
// logged and skipped rather than cutting the table short.
func writeLogsJSON(w io.Writer, r *http.Request, services []*container.ContainerService, from, to time.Time, stdTypes container.StdType, filter logFilter) {
	for _, service := range services {
		// The client went away; the rest would only log one error each.
		if r.Context().Err() != nil {
			return
		}
		events, err := service.LogsBetweenDates(r.Context(), from, to, stdTypes)
		if err != nil {
			log.Error().Err(err).Str("container", service.Container.ID).Msg("error fetching logs")
			continue
		}
		writeAllLogs(w, events, filter, false)
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
		writeAllLogs(w, events, filter, true)
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

	// Errors from here on are logged only, since the response has started. A
	// container whose logs cannot be read is skipped; a failed write ends the zip.
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
				continue
			}
			_, err = io.Copy(f, reader)
			reader.Close()
			if err != nil {
				log.Error().Err(err).Msgf("error copying logs for container %s", id)
				return
			}
			continue
		}

		events, err := service.LogsBetweenDates(r.Context(), from, to, stdTypes)
		if err != nil {
			log.Error().Err(err).Msgf("error getting logs for container %s", id)
			continue
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
