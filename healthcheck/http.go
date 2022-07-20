package healthcheck

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
)

func HttpRequest(addr string, base string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	url := fmt.Sprintf("http://%s%s/healthcheck", addr, base)
	log.Info("Checking health of " + url)
	resp, err := http.Get(url)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		os.Exit(0)
	}

	os.Exit(1)

	return nil
}
