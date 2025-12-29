package edgar

import (
	"os"
	"testing"
)

func TestParseSubmissions(t *testing.T) {
	// Parse the Pfizer CIK JSON we downloaded
	f, err := os.Open("testdata/cik/CIK0000078003.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	subs, err := ParseSubmissions(f)
	if err != nil {
		t.Fatalf("Failed to parse submissions: %v", err)
	}

	// Verify basic fields
	if subs.CIK != "0000078003" {
		t.Errorf("Expected CIK 0000078003, got %s", subs.CIK)
	}

	if subs.Name != "PFIZER INC" {
		t.Errorf("Expected name PFIZER INC, got %s", subs.Name)
	}

	// Verify filings exist
	if len(subs.Filings.Recent.AccessionNumber) == 0 {
		t.Error("Expected recent filings, got none")
	}

	// Verify pagination files exist
	if len(subs.Filings.Files) == 0 {
		t.Error("Expected pagination files, got none")
	}

	t.Logf("Parsed %d recent filings", len(subs.Filings.Recent.AccessionNumber))
	t.Logf("Found %d pagination files", len(subs.Filings.Files))
}

func TestGetRecentFilings(t *testing.T) {
	f, err := os.Open("testdata/cik/CIK0000078003.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	subs, err := ParseSubmissions(f)
	if err != nil {
		t.Fatalf("Failed to parse submissions: %v", err)
	}

	filings := subs.GetRecentFilings()

	if len(filings) == 0 {
		t.Fatal("Expected filings, got none")
	}

	// Check first filing has required fields
	first := filings[0]
	if first.AccessionNumber == "" {
		t.Error("First filing missing accession number")
	}
	if first.Form == "" {
		t.Error("First filing missing form type")
	}
	if first.FilingDate == "" {
		t.Error("First filing missing filing date")
	}
	if first.URL == "" {
		t.Error("First filing missing URL")
	}

	t.Logf("First filing: %s %s filed %s", first.Form, first.AccessionNumber, first.FilingDate)
	t.Logf("URL: %s", first.URL)
}

func TestFilterByForm(t *testing.T) {
	f, err := os.Open("testdata/cik/CIK0000078003.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	subs, err := ParseSubmissions(f)
	if err != nil {
		t.Fatalf("Failed to parse submissions: %v", err)
	}

	allFilings := subs.GetRecentFilings()
	form4Filings := FilterByForm(allFilings, "4")

	if len(form4Filings) == 0 {
		t.Error("Expected Form 4 filings, got none")
	}

	// Verify all returned filings are Form 4
	for _, filing := range form4Filings {
		if filing.Form != "4" {
			t.Errorf("Expected Form 4, got %s", filing.Form)
		}
	}

	t.Logf("Found %d Form 4 filings out of %d total", len(form4Filings), len(allFilings))
}

func TestFilterByDateRange(t *testing.T) {
	f, err := os.Open("testdata/cik/CIK0000078003.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	subs, err := ParseSubmissions(f)
	if err != nil {
		t.Fatalf("Failed to parse submissions: %v", err)
	}

	allFilings := subs.GetRecentFilings()

	// Filter for December 2025
	filtered := FilterByDateRange(allFilings, "2025-12-01", "2025-12-31")

	if len(filtered) == 0 {
		t.Log("No filings in December 2025 range")
	}

	// Verify all returned filings are in range
	for _, filing := range filtered {
		if filing.FilingDate < "2025-12-01" || filing.FilingDate > "2025-12-31" {
			t.Errorf("Filing date %s outside of range", filing.FilingDate)
		}
	}

	t.Logf("Found %d filings in December 2025 out of %d total", len(filtered), len(allFilings))
}

func TestCombinedFiltering(t *testing.T) {
	f, err := os.Open("testdata/cik/CIK0000078003.json")
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	subs, err := ParseSubmissions(f)
	if err != nil {
		t.Fatalf("Failed to parse submissions: %v", err)
	}

	// Get recent filings
	allFilings := subs.GetRecentFilings()

	// Filter for Form 4 in December 2025
	form4s := FilterByForm(allFilings, "4")
	form4sDec := FilterByDateRange(form4s, "2025-12-01", "2025-12-31")

	t.Logf("Found %d Form 4 filings in December 2025", len(form4sDec))

	// Verify all match criteria
	for _, filing := range form4sDec {
		if filing.Form != "4" {
			t.Errorf("Expected Form 4, got %s", filing.Form)
		}
		if filing.FilingDate < "2025-12-01" || filing.FilingDate > "2025-12-31" {
			t.Errorf("Filing date %s outside of range", filing.FilingDate)
		}
	}
}

func TestBuildURL(t *testing.T) {
	filing := Filing{
		CIK:             "0000078003",
		AccessionNumber: "0001225208-25-010078",
		PrimaryDocument: "ownership.xml",
	}

	url := filing.BuildURL()
	expected := "https://www.sec.gov/Archives/edgar/data/78003/000122520825010078/ownership.xml"

	if url != expected {
		t.Errorf("Expected URL:\n%s\nGot:\n%s", expected, url)
	}
}
