package web

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/container"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *handler) attach(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to upgrade connection")
		return
	}
	defer conn.Close()

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
		conn.WriteMessage(websocket.TextMessage, []byte("⛔ Access denied: attaching to this container is forbidden\r\n"))
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		return
	}

	eventReader := &jsonEventReader{conn: conn}
	wsWriter := &webSocketWriter{conn: conn}
	if err = containerService.Attach(r.Context(), eventReader, wsWriter); err != nil {
		log.Error().Err(err).Msg("error while trying to attach to container")
		conn.WriteMessage(websocket.TextMessage, []byte("🚨 Error while trying to attach to container\r\n"))
		return
	}
}

func (h *handler) exec(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to upgrade connection")
		return
	}
	defer conn.Close()

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
		conn.WriteMessage(websocket.TextMessage, []byte("⛔ Access denied: attaching to this container is forbidden\r\n"))
		return
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		return
	}

	eventReader := &jsonEventReader{conn: conn}
	wsWriter := &webSocketWriter{conn: conn}
	if err = containerService.Exec(r.Context(), []string{"sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"}, eventReader, wsWriter); err != nil {
		log.Error().Err(err).Msg("error while trying to attach to container")
		conn.WriteMessage(websocket.TextMessage, []byte("🚨 Error while trying to attach to container\r\n"))
		return
	}
}

type webSocketWriter struct {
	conn *websocket.Conn
}

func (w *webSocketWriter) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

// jsonEventReader reads JSON-encoded ExecEvents from a websocket connection
type jsonEventReader struct {
	conn *websocket.Conn
}

func (r *jsonEventReader) ReadEvent() (*container.ExecEvent, error) {
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		return nil, io.EOF
	}

	var event container.ExecEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return nil, err
	}

	return &event, nil
}
