package edgar

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// VERSION should be updated with releases
	VERSION = "0.1.0"

	// DefaultUserAgent for SEC requests
	DefaultUserAgent = "go-edgar/" + VERSION + " (nicholas@rxdatalab.com)"

	// RateLimit delays between requests (SEC requires 10 requests/second max)
	RateLimit = 100 * time.Millisecond
)

var lastRequestTime time.Time

// FetchForm fetches a Form 4 XML from the SEC by URL
// Implements rate limiting and proper User-Agent header
func FetchForm(url string) ([]byte, error) {
	// Rate limiting
	if !lastRequestTime.IsZero() {
		elapsed := time.Since(lastRequestTime)
		if elapsed < RateLimit {
			time.Sleep(RateLimit - elapsed)
		}
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required User-Agent header
	req.Header.Set("User-Agent", DefaultUserAgent)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	lastRequestTime = time.Now()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SEC returned status %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return body, nil
}
