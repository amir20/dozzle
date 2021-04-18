package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
)

func main() {
	postBody := map[string]interface{}{
		"client_id": "XXXXXXXXXX.YYYYYYYYYY",
		"events": []map[string]interface{}{
			{
				"name":    "test_go",
				"parames": map[string]interface{}{},
			},
		},
	}

	jsonValue, err := json.Marshal(postBody)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(jsonValue))

	req, err := http.NewRequest("POST", "https://www.google-analytics.com/mp/collect", bytes.NewBuffer(jsonValue))
	if err != nil {
		log.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("measurement_id", "G-S6NT05VXK9")
	q.Add("api_secret", "7FFhe65HQK-bXvujpQMquQ")
	req.URL.RawQuery = q.Encode()
	dump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%v", string(dump))

	if response, err := http.DefaultClient.Do(req); err != nil {
		log.Fatal(err)
	} else {
		defer response.Body.Close()
		dump, err := httputil.DumpResponse(response, true)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%v", string(dump))
	}
}

/*
fetch(`https://www.google-analytics.com/mp/collect?measurement_id=G-S6NT05VXK9&api_secret=7FFhe65HQK-bXvujpQMquQ`, {
  method: "POST",
  body: JSON.stringify({
    client_id: 'XXXXXXXXXX.YYYYYYYYYY',
    events: [{
      name: 'test_fetch',
      params: {},
    }]
  })
});
*/
