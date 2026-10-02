package web

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/amir20/dozzle/internal/container"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestHistogramWidth(t *testing.T) {
	cases := []struct {
		span    time.Duration
		buckets int
		want    time.Duration
	}{
		{time.Minute, 60, time.Second},
		{26 * time.Minute, 60, 30 * time.Second},
		{time.Hour, 60, time.Minute},
		{time.Hour, 120, 30 * time.Second},
		{24 * time.Hour, 60, 30 * time.Minute},
		{7 * 24 * time.Hour, 60, 3 * time.Hour},
		{10 * 365 * 24 * time.Hour, 60, 24 * time.Hour},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, histogramWidth(c.span, c.buckets), "%s in %d", c.span, c.buckets)
	}
}

func histogramMock(id string, data []byte) *MockedClient {
	m := new(MockedClient)
	m.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Tty: false}, nil)
	m.On("ContainerLogsTail", mock.Anything, id, mock.Anything).Return(io.NopCloser(bytes.NewReader(data)), nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(time.Second)
	})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{{ID: id, Name: "test", State: "running"}}, nil)
	return m
}

func Test_handler_log_histogram(t *testing.T) {
	id := "123456"
	now := time.Now().UTC()
	var data []byte
	for i := 1; i <= 5; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339Nano)
		data = append(data, makeMessage(ts+" INFO all good\n", container.STDOUT)...)
		if i%2 == 0 {
			data = append(data, makeMessage(ts+" ERROR boom\n", container.STDERR)...)
		}
	}

	q := url.Values{}
	q.Set("from", now.Add(-10*time.Minute).Format(time.RFC3339Nano))
	q.Set("to", now.Format(time.RFC3339Nano))
	q.Set("buckets", "10")
	req, err := http.NewRequest("GET", "/api/hosts/localhost/containers/"+id+"/logs/histogram?"+q.Encode(), nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	createDefaultHandler(histogramMock(id, data)).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp histogramResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.EqualValues(t, 60, resp.Width)
	assert.Nil(t, resp.ScannedFrom)
	assert.True(t, resp.Start.Equal(now.Add(-10*time.Minute).Truncate(time.Minute)))

	var total, errs uint32
	for i := range resp.Total {
		total += resp.Total[i]
		errs += resp.Errors[i]
	}
	assert.EqualValues(t, 7, total)
	assert.EqualValues(t, 2, errs)
}

func Test_handler_log_histogram_rejects_bad_range(t *testing.T) {
	for _, query := range []string{"", "from=yesterday&to=now", "from=2026-10-02T15:30:00Z&to=2026-10-02T15:04:00Z"} {
		req, err := http.NewRequest("GET", "/api/hosts/localhost/containers/123456/logs/histogram?"+query, nil)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		createDefaultHandler(histogramMock("123456", nil)).ServeHTTP(rr, req)
		assert.Equal(t, http.StatusBadRequest, rr.Code, query)
	}
}

func Test_handler_download_logs_in_range(t *testing.T) {
	id := "123456"
	from := time.Date(2026, 10, 2, 15, 4, 0, 0, time.UTC)
	to := time.Date(2026, 10, 2, 15, 30, 0, 0, time.UTC)

	m := new(MockedClient)
	m.On("FindContainer", mock.Anything, id).Return(container.Container{ID: id, Tty: false}, nil)
	m.On("ContainerLogsBetweenDates", mock.Anything, id, from, to, container.STDOUT).Return(io.NopCloser(bytes.NewReader(nil)), nil)
	m.On("Host").Return(container.Host{ID: "localhost"})
	m.On("ContainerEvents", mock.Anything, mock.AnythingOfType("chan<- container.ContainerEvent")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(time.Second)
	})
	m.On("ListContainers", mock.Anything, mock.Anything).Return([]container.Container{{ID: id, Name: "test", State: "running"}}, nil)

	q := url.Values{}
	q.Set("stdout", "1")
	q.Set("from", from.Format(time.RFC3339))
	q.Set("to", to.Format(time.RFC3339))
	req, err := http.NewRequest("GET", "/api/containers/localhost~"+id+"/download?"+q.Encode(), nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	createDefaultHandler(m).ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)
	m.AssertExpectations(t)
}
