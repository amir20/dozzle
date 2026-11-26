package cloud

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// installIDPath keeps an id this Dozzle picks once, sent to Dozzle Cloud as
// x-dozzle-install-id beside the host id. The host id cannot tell two Dozzles
// apart when they share an engine (a server and an agent on one machine) or a
// cloned engine id (a copied VM), and Cloud then cannot keep both connected.
// A var so tests can point it elsewhere.
var installIDPath = "./data/install-id"

// loadInstallID reads the saved install id, or picks and saves a new one. An id
// that cannot be saved still tells this process apart until it restarts.
func loadInstallID(path string) string {
	if b, err := os.ReadFile(path); err == nil {
		if id := strings.TrimSpace(string(b)); id != "" {
			return id
		}
	}
	id := uuid.NewString()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		log.Warn().Err(err).Msg("Could not save the install id; using a new one until restart")
		return id
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o644); err != nil {
		log.Warn().Err(err).Msg("Could not save the install id; using a new one until restart")
	}
	return id
}
