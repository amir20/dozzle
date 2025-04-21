package web

import (
	"io"
	"net/http"
	"strings"

	"github.com/andybalholm/brotli"
)

type brotliWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w brotliWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func Brotli(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if client supports Brotli
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "br") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Encoding", "br")
		w.Header().Add("Vary", "Accept-Encoding")

		brWriter := brotli.NewWriter(w)
		defer brWriter.Close()

		bw := brotliWriter{Writer: brWriter, ResponseWriter: w}
		next.ServeHTTP(bw, r)
	})
}
