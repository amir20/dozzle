package healthcheck

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

func HttpRequest(addr string, base string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	if base == "/" {
		base = ""
	}

	url := fmt.Sprintf("%s%s/healthcheck", addr, base)

	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	log.Info("Checking health of " + url)
	resp, err := http.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("healthcheck failed with status code %d", resp.StatusCode)
}
