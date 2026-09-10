package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/cloud"
	"github.com/rs/zerolog/log"
)

// The turn carries the window the user is looking at, so the body is a few
// hundred lines of log rather than one sentence. The browser budgets what it
// sends; this is the backstop.
const maxChatRequestBytes = 1024 * 1024

type chatRequest struct {
	Message string            `json:"message"`
	View    cloud.ViewContext `json:"view"`
}

// cloudChat runs one assistant turn and streams it to the browser as SSE.
//
// This handler is the security boundary for the whole feature. It resolves the
// caller's principal from this request, then opens the chat stream itself, so
// every tool call cloud makes for this turn comes back down a stream this
// request owns and executes with these labels and these roles. There is no
// token to mint, forge or expire: the authority is the request, which is how
// every other route in Dozzle already works.
func (h *handler) cloudChat(w http.ResponseWriter, r *http.Request) {
	if h.config.Cloud.Chat == nil {
		writeError(w, http.StatusServiceUnavailable, "cloud not configured")
		return
	}

	var req chatRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxChatRequestBytes)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// The view arrives from the browser, so the containers it claims are
	// visible get run through the caller's scope before cloud sees them.
	// Otherwise a filtered user could name any container in the context and
	// have the assistant discuss it.
	if h.restrictedUser(r) {
		visible := h.visibleContainerIDs(r)
		allowed := make([]cloud.ViewContainer, 0, len(req.View.Containers))
		for _, c := range req.View.Containers {
			if _, ok := visible[c.ID]; ok {
				allowed = append(allowed, c)
			}
		}
		req.View.Containers = allowed

		// The lines are the user's own screen, so their text is theirs to quote.
		// The id attached to one is not: left alone it would tell the assistant
		// a container this user cannot see is on their screen.
		for i := range req.View.Lines {
			if _, ok := visible[req.View.Lines[i].ContainerID]; !ok {
				req.View.Lines[i].ContainerID = ""
			}
		}
		if req.View.Focused != nil {
			if _, ok := visible[req.View.Focused.ContainerID]; !ok {
				req.View.Focused.ContainerID = ""
			}
		}
	}

	principal := cloud.UserPrincipal(h.resolveLabels(r), h.userRoles(r))

	var ref string
	if user := auth.UserFromContext(r.Context()); user != nil {
		ref = userRef(user.Username)
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	encoder := json.NewEncoder(w)
	emit := func(event cloud.ChatEvent) {
		if _, err := w.Write([]byte("data: ")); err != nil {
			return
		}
		if err := encoder.Encode(event); err != nil {
			return
		}
		_, _ = w.Write([]byte("\n"))
		flusher.Flush()
	}

	// r.Context() cancels when the browser goes away, which closes the stream
	// and ends the turn rather than paying cloud to finish talking to nobody.
	err := h.config.Cloud.Chat(r.Context(), req.Message, req.View, ref, principal, emit)
	if err != nil {
		// The text is a fallback for anything reading this stream raw; the
		// browser renders the code in the reader's own language.
		if errors.Is(err, cloud.ErrNotConfigured) {
			emit(cloud.ChatEvent{Kind: "error", Code: "not_configured", Text: "cloud is not configured"})
			return
		}
		if r.Context().Err() != nil {
			return
		}
		log.Warn().Err(err).Msg("cloud chat failed")
		emit(cloud.ChatEvent{Kind: "error", Code: "unavailable", Text: "the assistant is unavailable right now"})
	}
}

// userRoles is the caller's roles, or every role when auth is off. Mirrors how
// the rest of the app treats an unauthenticated instance: there is nobody to
// restrict.
func (h *handler) userRoles(r *http.Request) auth.Role {
	if h.config.Authorization.Provider == NONE {
		return auth.All
	}
	if user := auth.UserFromContext(r.Context()); user != nil {
		return user.Roles
	}
	return auth.None
}
