package edgar

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Helper function to download a 10-K for testing
// This is not part of the main library, just for setting up test data
func TestDownloadModerna10K(t *testing.T) {
	t.Skip("Only run manually to download test data")

	email := "nicholas@rxdatalab.com"

	// Fetch Moderna's submissions
	t.Log("Fetching Moderna submissions...")
	subs, err := FetchSubmissions("1682852", email)
	if err != nil {
		t.Fatalf("Failed to fetch submissions: %v", err)
	}

	t.Logf("Found company: %s (CIK: %s)", subs.Name, subs.CIK)

	// Get recent filings
	filings := subs.GetRecentFilings()

	// Find latest 10-K
	var latest10K *Filing
	for i := range filings {
		if filings[i].Form == "10-K" {
			latest10K = &filings[i]
			break
		}
	}

	if latest10K == nil {
		t.Fatal("No 10-K found")
	}

	t.Logf("Found 10-K: %s (filed: %s)", latest10K.AccessionNumber, latest10K.FilingDate)
	t.Logf("Primary document: %s", latest10K.PrimaryDocument)
	t.Logf("Is inline XBRL: %v", latest10K.IsInlineXBRL)
	t.Logf("URL: %s", latest10K.URL)

	// Download the primary document
	t.Log("Downloading...")
	req, err := http.NewRequest("GET", latest10K.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", fmt.Sprintf("go-edgar %s", email))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("SEC returned status %d", resp.StatusCode)
	}

	// Save to testdata
	outPath := "testdata/xbrl/moderna_10k/input.htm"
	outFile, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	t.Logf("Downloaded %d bytes to %s", written, outPath)

	// Also save a metadata file
	metaPath := "testdata/xbrl/moderna_10k/metadata.json"
	metaFile, err := os.Create(metaPath)
	if err != nil {
		t.Fatalf("Failed to create metadata file: %v", err)
	}
	defer metaFile.Close()

	metadata := fmt.Sprintf(`{
  "company": "%s",
  "cik": "%s",
  "form": "%s",
  "filing_date": "%s",
  "report_date": "%s",
  "accession": "%s",
  "source_url": "%s",
  "is_inline_xbrl": %v,
  "notes": "Moderna 10-K for testing XBRL parsing - biotech company with R&D focus"
}
`, subs.Name, subs.CIK, latest10K.Form, latest10K.FilingDate, latest10K.ReportDate, latest10K.AccessionNumber, latest10K.URL, latest10K.IsInlineXBRL)

	if _, err := metaFile.WriteString(metadata); err != nil {
		t.Fatalf("Failed to write metadata: %v", err)
	}

	t.Log("✅ Successfully downloaded Moderna 10-K test data")
}
