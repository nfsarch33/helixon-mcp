package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPPrometheusQuerier(t *testing.T) {
	q := NewHTTPPrometheusQuerier("http://localhost:9090")
	assert.NotNil(t, q)
	assert.Equal(t, "http://localhost:9090", q.baseURL)
}

func TestHTTPPrometheusQuerier_Query_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/query", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "query=up")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up"},"value":[1,"1"]}]}}`))
	}))
	defer server.Close()

	q := &HTTPPrometheusQuerier{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
	result, err := q.Query(context.Background(), "up")
	require.NoError(t, err)
	assert.Contains(t, result, "success")
	assert.Contains(t, result, "up")
}

func TestHTTPPrometheusQuerier_Query_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`internal server error`))
	}))
	defer server.Close()

	q := &HTTPPrometheusQuerier{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
	_, err := q.Query(context.Background(), "up")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus error 500")
}

func TestHTTPPrometheusQuerier_Query_ConnectionRefused(t *testing.T) {
	q := NewHTTPPrometheusQuerier("http://127.0.0.1:1")
	_, err := q.Query(context.Background(), "up")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "prometheus query")
}

func TestHTTPPrometheusQuerier_Query_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`ok`))
	}))
	defer server.Close()

	q := &HTTPPrometheusQuerier{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := q.Query(ctx, "up")
	assert.Error(t, err)
}

func TestHTTPPrometheusQuerier_Query_SpecialCharacters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "query=rate")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[]}}`))
	}))
	defer server.Close()

	q := &HTTPPrometheusQuerier{
		baseURL:    server.URL,
		httpClient: server.Client(),
	}
	result, err := q.Query(context.Background(), `rate(http_requests_total[5m])`)
	require.NoError(t, err)
	assert.Contains(t, result, "success")
}
