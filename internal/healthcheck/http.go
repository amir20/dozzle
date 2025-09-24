package healthcheck

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

func HttpRequest(addr string, base string, useHttps bool) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	if base == "/" {
		base = ""
	}

	url := fmt.Sprintf("%s%s/healthcheck", addr, base)

	if !strings.HasPrefix(url, "http") {
		if useHttps {
			url = "https://" + url
		} else {
			url = "http://" + url
		}
	}

	log.Info().Str("url", url).Msg("performing healthcheck")

	var client *http.Client
	if useHttps || strings.HasPrefix(url, "https") {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true, // Set to true if the server's hostname does not match the certificate
		}
		client = &http.Client{Transport: &http.Transport{TLSClientConfig: tlsConfig}}
	} else {
		client = http.DefaultClient
	}

	resp, err := client.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("healthcheck failed with status code %d", resp.StatusCode)
}
