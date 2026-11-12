package web

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/amir20/dozzle/internal/analytics"
	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

// upgrade accepts the websocket with coder/websocket's default same-origin
// check. Allowing arbitrary origins (OriginPatterns, InsecureSkipVerify)
// enables Cross-Site WebSocket Hijacking against /attach and /exec when the
// user is authenticated (the JWT cookie is SameSite=Lax, which still attaches
// on same-site cross-origin requests).
func upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return nil, err
	}
	// The default 32 KB limit would drop the shell on a large paste.
	conn.SetReadLimit(-1)
	return conn, nil
}

func (h *handler) attach(w http.ResponseWriter, r *http.Request) {
	analytics.Count("shell.attach")
	conn, err := upgrade(w, r)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to upgrade connection")
		return
	}
	defer conn.CloseNow()

	id := chi.URLParam(r, "id")
	userLabels := h.config.Labels
	permit := true
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
		permit = user.Roles.Has(auth.Shell)
	}

	if !permit {
		log.Warn().Msg("user is not permitted to attach to container")
		conn.Write(r.Context(), websocket.MessageText, []byte("⛔ Access denied: attaching to this container is forbidden\r\n"))
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		return
	}

	eventReader := &jsonEventReader{ctx: r.Context(), conn: conn}
	wsWriter := &webSocketWriter{ctx: r.Context(), conn: conn}
	if err = containerService.Attach(r.Context(), eventReader, wsWriter); err != nil {
		log.Error().Err(err).Msg("error while trying to attach to container")
		conn.Write(r.Context(), websocket.MessageText, []byte("🚨 Error while trying to attach to container\r\n"))
		return
	}
}

func (h *handler) exec(w http.ResponseWriter, r *http.Request) {
	analytics.Count("shell.exec")
	conn, err := upgrade(w, r)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to upgrade connection")
		return
	}
	defer conn.CloseNow()

	id := chi.URLParam(r, "id")
	userLabels := h.config.Labels
	permit := true
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
		permit = user.Roles.Has(auth.Shell)
	}

	if !permit {
		log.Warn().Msg("user is not permitted to exec into container")
		conn.Write(r.Context(), websocket.MessageText, []byte("⛔ Access denied: attaching to this container is forbidden\r\n"))
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		return
	}

	eventReader := &jsonEventReader{ctx: r.Context(), conn: conn}
	wsWriter := &webSocketWriter{ctx: r.Context(), conn: conn}
	if err = containerService.Exec(r.Context(), []string{"sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"}, eventReader, wsWriter); err != nil {
		log.Error().Err(err).Msg("error while trying to attach to container")
		conn.Write(r.Context(), websocket.MessageText, []byte("🚨 Error while trying to attach to container\r\n"))
		return
	}
}

type webSocketWriter struct {
	ctx  context.Context
	conn *websocket.Conn
}

func (w *webSocketWriter) Write(p []byte) (int, error) {
	err := w.conn.Write(w.ctx, websocket.MessageText, p)
	return len(p), err
}

// jsonEventReader reads JSON-encoded ExecEvents from a websocket connection
type jsonEventReader struct {
	ctx  context.Context
	conn *websocket.Conn
}

func (r *jsonEventReader) ReadEvent() (*container.ExecEvent, error) {
	_, message, err := r.conn.Read(r.ctx)
	if err != nil {
		return nil, io.EOF
	}

	var event container.ExecEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return nil, err
	}

	return &event, nil
}
