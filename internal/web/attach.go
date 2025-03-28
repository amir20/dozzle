package web

import (
	"context"
	"io"
	"net/http"
	"sync"

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	writer, reader, err := containerService.Attach(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error while trying to attach to container")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		wsReader := &WebSocketReader{conn: conn}
		if _, err := io.Copy(writer, wsReader); err != nil {
			log.Error().Err(err).Msg("error while reading from ws")
		}
		cancel()
		writer.Close()
	}()

	go func() {
		defer wg.Done()
		wsWriter := &WebSocketWriter{conn: conn}
		if _, err := io.Copy(wsWriter, reader); err != nil {
			log.Error().Err(err).Msg("error while writing to ws")
		}
		cancel()
	}()

	wg.Wait()
}

type WebSocketWriter struct {
	conn *websocket.Conn
}

func (w *WebSocketWriter) Write(p []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.TextMessage, p)
	return len(p), err
}

type WebSocketReader struct {
	conn   *websocket.Conn
	buffer []byte
}

func (r *WebSocketReader) Read(p []byte) (n int, err error) {
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
