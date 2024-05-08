package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"
)

func SendBeacon(e BeaconEvent) error {
	log.Tracef("sending beacon: %+v", e)
	jsonValue, err := json.Marshal(e)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://b.dozzle.dev/event", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode/100 != 2 {
		dump, err := httputil.DumpResponse(response, true)
		if err != nil {
			return err
		}
		log.Debugf("%v", string(dump))
		return fmt.Errorf("google analytics returned non-2xx status code: %v", response.Status)
	}

	return nil
}
