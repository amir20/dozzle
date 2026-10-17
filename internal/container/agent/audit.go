package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

// audit logs one line for each call that changes something on this host, so an
// incident can be traced back to the hub that made it. Commands and stdin are
// never logged.
func audit(ctx context.Context, op, containerID string, err error) {
	ev := log.Info()
	if err != nil {
		ev = log.Warn().Err(err)
	}
	if p, ok := peer.FromContext(ctx); ok {
		ev = ev.Str("peer", p.Addr.String())
		if info, ok := p.AuthInfo.(credentials.TLSInfo); ok && len(info.State.PeerCertificates) > 0 {
			sum := sha256.Sum256(info.State.PeerCertificates[0].Raw)
			ev = ev.Str("cert", hex.EncodeToString(sum[:]))
		}
	}
	if containerID != "" {
		ev = ev.Str("container", containerID)
	}
	ev.Str("op", op).Msg("agent audit")
}
