package web

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall"
	"time"

	"github.com/amir20/dozzle/internal/auth"
	"github.com/amir20/dozzle/internal/profile"
	"github.com/rs/zerolog/log"
)

func (h *handler) updateProfile(w http.ResponseWriter, r *http.Request) {
	username := profile.DefaultUsername
	if user := auth.UserFromContext(r.Context()); user != nil {
		username = user.Username
	}

	if err := profile.UpdateFromReader(username, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to update profile")
		return
	}

	w.WriteHeader(http.StatusOK)
}

// maxAvatarBytes caps a proxied avatar. Provider pictures are thumbnails, so
// anything larger is not an avatar.
const maxAvatarBytes = 2 << 20

var errPrivateAddress = errors.New("refusing to fetch an avatar from a non-public address")

// pictureClient fetches provider-asserted pictures. At many providers a user
// can edit their own picture, so it only ever connects to public addresses:
// the check runs on the resolved IP at dial time, which also covers redirects
// and DNS that answers differently the second time. No proxy, because a proxy
// would be the address checked instead of the real target.
var pictureClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
			Control: refuseNonPublicAddress,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" {
			return errors.New("refusing a non-https redirect")
		}
		return nil
	},
}

var avatarClient = &http.Client{Timeout: 10 * time.Second}

var cgnat = netip.MustParsePrefix("100.64.0.0/10")

func refuseNonPublicAddress(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || cgnat.Contains(ip) {
		return errPrivateAddress
	}
	return nil
}

func (h *handler) avatar(w http.ResponseWriter, r *http.Request) {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		http.Error(w, "Unable to find user", http.StatusInternalServerError)
		return
	}

	// The CSP only allows same-origin images, so the provider picture is
	// proxied. When it cannot be fetched the gravatar is served instead.
	if picture := user.PictureURL(); picture != "" {
		err := serveAvatar(w, r, pictureClient, picture)
		if err == nil {
			return
		}
		log.Debug().Err(err).Str("url", picture).Msg("Failed to fetch provider picture, falling back to gravatar")
	}

	url := user.AvatarURL()
	if err := serveAvatar(w, r, avatarClient, url); err != nil {
		log.Error().Err(err).Str("url", url).Msg("Failed to fetch avatar")
		http.Error(w, "Unable to fetch avatar", http.StatusBadGateway)
	}
}

// serveAvatar writes the image at url, or returns an error having written nothing.
func serveAvatar(w http.ResponseWriter, r *http.Request, client *http.Client, url string) error {
	log.Trace().Str("url", url).Msg("Fetching avatar")
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	// svg is refused because it can carry script and would be served from
	// Dozzle's own origin.
	contentType := response.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") || strings.Contains(contentType, "svg") {
		return fmt.Errorf("unexpected content type %q", contentType)
	}

	var body bytes.Buffer
	if n, err := io.Copy(&body, io.LimitReader(response.Body, maxAvatarBytes+1)); err != nil {
		return err
	} else if n > maxAvatarBytes {
		return errors.New("avatar is too large")
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, err = w.Write(body.Bytes())
	if err != nil {
		log.Debug().Err(err).Msg("Failed to write avatar")
	}
	return nil
}
