package infra

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eichiarakaki/aegis/internals/logger"
)

const maxRetries = 5

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

// doGetWithRetry performs an HTTP GET with exponential backoff on transient
// errors (rate limits, 5xx). Returns the response body and status code.
func doGetWithRetry(reqURL string) ([]byte, int, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(attempt*attempt) * time.Second
			logger.Infof("RETRY %d/%d sleeping %s", attempt, maxRetries, wait)
			time.Sleep(wait)
		}

		resp, err := httpClient.Get(reqURL)
		if err != nil {
			lastErr = fmt.Errorf("GET: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}

		// Retry on rate limiting or server-side errors
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			preview := string(body)
			if len(preview) > 150 {
				preview = preview[:150]
			}
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(preview))
			continue
		}

		return body, resp.StatusCode, nil
	}

	return nil, 0, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}
