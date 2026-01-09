package edgar

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	VERSION = "0.3.0"

	// RateLimit delays between requests (SEC requires 10 requests/second max)
	RateLimit = 100 * time.Millisecond

	// SecEmailEnvVar is the environment variable name for SEC email
	SecEmailEnvVar = "SEC_EMAIL"
)

var lastRequestTime time.Time

// GetSecEmail retrieves email from environment variable or returns error
func GetSecEmail() (string, error) {
	email := os.Getenv(SecEmailEnvVar)
	if email == "" {
		return "", fmt.Errorf("SEC email required: set %s environment variable or use --email flag", SecEmailEnvVar)
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return "", fmt.Errorf("invalid email format: %s", email)
	}
	if strings.HasSuffix(email, "example.com") {
		return "", fmt.Errorf("Use a real email address, not example.com: %s", email)
	}
	return email, nil
}

// BuildUserAgent creates a proper SEC User-Agent string
func BuildUserAgent(email string) string {
	return fmt.Sprintf("go-edgar/%s (%s)", VERSION, email)
}

// FetchForm fetches a form XML from the SEC by URL
// Implements rate limiting and proper User-Agent header
// Email is required by SEC - must be a valid email address
func FetchForm(url string, email string) ([]byte, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required for SEC requests")
	}

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

	// Set required User-Agent header with email
	req.Header.Set("User-Agent", BuildUserAgent(email))

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
