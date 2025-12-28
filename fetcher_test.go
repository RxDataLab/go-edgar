package edgar_test

import (
	"testing"
	"time"

	"github.com/RxDataLab/go-edgar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchForm_RealSEC tests fetching from actual SEC (integration test)
// Skip in short mode to avoid rate limiting
func TestFetchForm_RealSEC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	url := "https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml"
	data, err := edgar.FetchForm(url)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Should be valid XML
	form, err := edgar.Parse(data)
	require.NoError(t, err)
	assert.Equal(t, "4", form.DocumentType)
	assert.Equal(t, "Wave Life Sciences Ltd.", form.Issuer.Name)
}

// TestFetchForm_RateLimit verifies rate limiting works
func TestFetchForm_RateLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	url := "https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml"

	// Make two requests - should take at least 100ms total due to rate limiting
	start := time.Now()
	_, err := edgar.FetchForm(url)
	require.NoError(t, err)

	_, err = edgar.FetchForm(url)
	require.NoError(t, err)
	elapsed := time.Since(start)

	// Should respect rate limit (at least 100ms between requests)
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(100))
}
