package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpRequest(t *testing.T) {
	// Test server that always responds with a status code of 200
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test server that always responds with a status code of 500
	errorServer := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	tests := []struct {
		name    string
		addr    string
		base    string
		wantErr bool
	}{
		{
			name:    "Healthcheck OK",
			addr:    server.URL,
			base:    "/",
			wantErr: false,
		},
		{
			name:    "Healthcheck Fail",
			addr:    errorServer.URL,
			base:    "/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HttpRequest(tt.addr, tt.base)

			if (err != nil) != tt.wantErr {
				t.Errorf("HttpRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
