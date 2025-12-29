package edgar

import (
	"bytes"
	"fmt"
	"time"
)

// BatchOptions configures batch download and parsing
type BatchOptions struct {
	CIK              string // Required: CIK to fetch filings for
	FormType         string // Required: Form type to filter (e.g., "4", "3", "5")
	DateFrom         string // Optional: Start date (YYYY-MM-DD), empty = no limit
	DateTo           string // Optional: End date (YYYY-MM-DD), empty = no limit
	Email            string // Required: Email for SEC User-Agent header
	IncludePaginated bool   // If true, fetch all paginated filings (can be slow)
}

// BatchResult contains the results of a batch operation
type BatchResult struct {
	Filings    []*Form4Output
	TotalFound int     // Total filings matching criteria
	Fetched    int     // Number actually downloaded and parsed
	Errors     []error // Any errors encountered during processing
}

// FetchAndParseBatch fetches all filings for a CIK matching the criteria and parses them
func FetchAndParseBatch(opts BatchOptions) (*BatchResult, error) {
	result := &BatchResult{
		Filings: make([]*Form4Output, 0),
		Errors:  make([]error, 0),
	}

	// Validate options
	if opts.CIK == "" {
		return nil, fmt.Errorf("CIK is required")
	}
	if opts.FormType == "" {
		return nil, fmt.Errorf("FormType is required")
	}
	if opts.Email == "" {
		return nil, fmt.Errorf("Email is required")
	}

	// Fetch submissions
	fmt.Printf("Fetching submissions for CIK %s...\n", opts.CIK)
	subs, err := FetchSubmissions(opts.CIK, opts.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submissions: %w", err)
	}

	// Get all filings (recent + paginated if requested)
	var allFilings []Filing
	if opts.IncludePaginated {
		fmt.Println("Fetching paginated filings (this may take a while)...")
		allFilings, err = subs.GetAllFilings(opts.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch paginated filings: %w", err)
		}
	} else {
		allFilings = subs.GetRecentFilings()
	}

	// Filter by form type
	filings := FilterByForm(allFilings, opts.FormType)
	fmt.Printf("Found %d Form %s filings\n", len(filings), opts.FormType)

	// Filter by date range if specified
	if opts.DateFrom != "" || opts.DateTo != "" {
		from := opts.DateFrom
		to := opts.DateTo

		// Use reasonable defaults if not specified
		if from == "" {
			from = "1900-01-01"
		}
		if to == "" {
			to = "2099-12-31"
		}

		filings = FilterByDateRange(filings, from, to)
		fmt.Printf("Filtered to %d filings in date range %s to %s\n", len(filings), from, to)
	}

	result.TotalFound = len(filings)

	// Download and parse each filing
	fmt.Printf("Downloading and parsing %d filings...\n", len(filings))

	rateLimiter := time.NewTicker(100 * time.Millisecond) // 10 req/sec
	defer rateLimiter.Stop()

	for i, filing := range filings {
		// Rate limiting
		<-rateLimiter.C

		// Progress indicator
		if (i+1)%10 == 0 || i == 0 {
			fmt.Printf("  Progress: %d/%d\n", i+1, len(filings))
		}

		// Fetch the XML
		xmlData, err := FetchForm(filing.URL, opts.Email)
		if err != nil {
			errMsg := fmt.Errorf("failed to fetch %s: %w", filing.AccessionNumber, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		// Parse the form
		parsed, err := ParseAny(bytes.NewReader(xmlData))
		if err != nil {
			errMsg := fmt.Errorf("failed to parse %s: %w", filing.AccessionNumber, err)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		// Ensure it's a Form 4
		if parsed.FormType != "4" {
			continue
		}

		// Get the Form4Output
		form4Output, ok := parsed.Data.(*Form4Output)
		if !ok {
			errMsg := fmt.Errorf("unexpected data type for %s", filing.AccessionNumber)
			result.Errors = append(result.Errors, errMsg)
			continue
		}

		// Populate metadata from the filing index
		form4Output.SetSource(filing.URL)
		form4Output.SetFilingMetadata(filing.AccessionNumber, filing.FilingDate, filing.ReportDate)

		result.Filings = append(result.Filings, form4Output)
		result.Fetched++
	}

	fmt.Printf("Successfully parsed %d/%d filings\n", result.Fetched, result.TotalFound)
	if len(result.Errors) > 0 {
		fmt.Printf("Encountered %d errors during processing\n", len(result.Errors))
	}

	return result, nil
}
