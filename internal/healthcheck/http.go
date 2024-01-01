package healthcheck

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

func HttpRequest(addr string, base string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	url := "http://" + strings.Replace(fmt.Sprintf("%s%s/healthcheck", addr, base), "//", "/", 1)
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
