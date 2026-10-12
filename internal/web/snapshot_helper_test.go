package web

import (
	"io"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"testing"

	"github.com/gkampitakis/go-snaps/snaps"
)

// All snapshots in this package go to __snapshots__/web.snap. Without an explicit
// Filename go-snaps would name the file after the caller, which is always this helper.
var webSnaps = snaps.WithConfig(snaps.Filename("web"), snaps.Raw())

func TestMain(m *testing.M) {
	v := m.Run()
	// Fails the run when a snapshot no longer has a test asserting it.
	if dirty, err := snaps.Clean(m); dirty || err != nil {
		v = 1
	}
	os.Exit(v)
}

// matchHTTPSnapshot snapshots the full response (status line, headers, body) as
// httputil.DumpResponse renders it.
func matchHTTPSnapshot(t *testing.T, res *http.Response) {
	t.Helper()
	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		t.Fatal(err)
	}
	webSnaps.MatchSnapshot(t, normalizeSnapshot(string(dump)))
}

// matchReaderSnapshot snapshots everything read from r.
func matchReaderSnapshot(t *testing.T, r io.Reader) {
	t.Helper()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	webSnaps.MatchSnapshot(t, normalizeSnapshot(string(data)))
}

// normalizeSnapshot trims surrounding whitespace (the old abide snapshots compared
// trimmed output) and converts the CRLF that DumpResponse writes after each header
// to LF, since go-snaps does not handle CRLF.
func normalizeSnapshot(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}
