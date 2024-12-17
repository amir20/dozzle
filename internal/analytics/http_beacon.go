package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/amir20/dozzle/types"
	"github.com/rs/zerolog/log"
)

func SendBeacon(e types.BeaconEvent) error {
	log.Trace().Interface("event", e).Msg("sending beacon")
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
		log.Debug().Str("response", string(dump)).Msg("google analytics returned non-2xx status code")
		return fmt.Errorf("google analytics returned non-2xx status code: %v", response.Status)
	}

	return nil
}
