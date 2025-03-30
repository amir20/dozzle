package web

import (
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
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
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)

	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		return
	}

	wsReader := &webSocketReader{conn: conn}
	wsWriter := &webSocketWriter{conn: conn}
	if err = containerService.Attach(r.Context(), wsReader, wsWriter); err != nil {
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
	if h.config.Authorization.Provider != NONE {
		user := auth.UserFromContext(r.Context())
		if user.ContainerLabels.Exists() {
			userLabels = user.ContainerLabels
		}
	}

	containerService, err := h.hostService.FindContainer(hostKey(r), id, userLabels)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to find container")
		return
	}

	wsReader := &webSocketReader{conn: conn}
	wsWriter := &webSocketWriter{conn: conn}
	if err = containerService.Exec(r.Context(), []string{"sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"}, wsReader, wsWriter); err != nil {
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

type webSocketReader struct {
	conn   *websocket.Conn
	buffer []byte
}

func (r *webSocketReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}

	// Otherwise, read a new message
	_, message, err := r.conn.ReadMessage()
	if err != nil {
		return 0, err
	}

	n = copy(p, message)

	// If we couldn't copy the entire message, store the rest in our buffer
	if n < len(message) {
		r.buffer = message[n:]
	}

	return n, nil
}
