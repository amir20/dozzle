package analytics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"

	log "github.com/sirupsen/logrus"
)

func SendStartEvent(se StartEvent) error {
	postBody := map[string]interface{}{
		"client_id": se.ClientId,
		"events": []map[string]interface{}{
			{
				"name":   "start",
				"params": se,
			},
		},
	}

	return doRequest(postBody)
}

func SendRequestEvent(re RequestEvent) error {
	postBody := map[string]interface{}{
		"client_id": re.ClientId,
		"events": []map[string]interface{}{
			{
				"name":   "request",
				"params": re,
			},
		},
	}

	return doRequest(postBody)
}

func doRequest(body map[string]interface{}) error {
	jsonValue, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://www.google-analytics.com/mp/collect", bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("measurement_id", "G-S6NT05VXK9")
	q.Add("api_secret", "7FFhe65HQK-bXvujpQMquQ")
	req.URL.RawQuery = q.Encode()

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
