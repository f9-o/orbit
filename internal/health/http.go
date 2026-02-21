// Package health: HTTP probe implementation.
package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// CheckHTTP performs an HTTP GET to url and verifies the response code.
// If expectedCode is 0, any 2xx is accepted.
func CheckHTTP(ctx context.Context, url string, expectedCode int, timeout time.Duration) error {
	if url == "" {
		return fmt.Errorf("http health check: url is required")
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "orbit-health/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http get %q: %w", url, err)
	}
	defer resp.Body.Close()

	if expectedCode != 0 {
		if resp.StatusCode != expectedCode {
			return fmt.Errorf("expected status %d, got %d", expectedCode, resp.StatusCode)
		}
	} else {
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("non-2xx status: %d", resp.StatusCode)
		}
	}
	return nil
}
