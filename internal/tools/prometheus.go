package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPPrometheusQuerier queries Prometheus via its HTTP query API.
type HTTPPrometheusQuerier struct {
	baseURL    string
	httpClient interface {
		Do(req *http.Request) (*http.Response, error)
	}
}

// NewHTTPPrometheusQuerier creates a querier pointing at the given Prometheus base URL.
func NewHTTPPrometheusQuerier(baseURL string) *HTTPPrometheusQuerier {
	return &HTTPPrometheusQuerier{
		baseURL:    baseURL,
		httpClient: http.DefaultClient,
	}
}

// Query executes a PromQL instant query and returns the raw JSON result.
func (q *HTTPPrometheusQuerier) Query(ctx context.Context, query string) (string, error) {
	u := fmt.Sprintf("%s/api/v1/query?query=%s", q.baseURL, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", fmt.Errorf("building prometheus request: %w", err)
	}

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("prometheus query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading prometheus response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("prometheus error %d: %s", resp.StatusCode, string(body))
	}
	return string(body), nil
}
