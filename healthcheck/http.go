package healthcheck

import "net/http"

func HttpRequest() (int, error) {
	resp, err := http.Get("http://localhost:8080/healthcheck")

	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
