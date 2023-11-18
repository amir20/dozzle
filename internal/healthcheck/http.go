package healthcheck

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

func HttpRequest(addr string, base string) error {
	if strings.HasPrefix(addr, ":") {
		addr = "localhost" + addr
	}

	url := fmt.Sprintf("http://"+path.Clean("%s%s/healthcheck"), addr, base)
	log.Info("Checking health of " + url)
	resp, err := http.Get(url)

	if err != nil {
		log.Error("Healthcheck failed with error: " + err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		log.Info("Healthcheck passed")
		os.Exit(0)
	}

	log.Errorf("Healthcheck failed with status code %d", resp.StatusCode)
	os.Exit(1)

	return nil
}
