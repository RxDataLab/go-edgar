package edgar

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Submissions represents the complete SEC submissions data for a CIK
type Submissions struct {
	CIK                               string      `json:"cik"`
	EntityType                        string      `json:"entityType"`
	SIC                               string      `json:"sic"`
	SICDescription                    string      `json:"sicDescription"`
	Name                              string      `json:"name"`
	Ticker                            []string    `json:"tickers"`
	Exchanges                         []string    `json:"exchanges"`
	Ein                               string      `json:"ein"`
	Description                       string      `json:"description"`
	Category                          string      `json:"category"`
	FiscalYearEnd                     string      `json:"fiscalYearEnd"`
	Filings                           FilingsData `json:"filings"`
	InsiderTransactionForOwnerExists  int         `json:"insiderTransactionForOwnerExists"`  // 0 or 1
	InsiderTransactionForIssuerExists int         `json:"insiderTransactionForIssuerExists"` // 0 or 1
}

// FilingsData contains recent and paginated filings information
type FilingsData struct {
	Recent FilingArrays `json:"recent"`
	Files  []FilingFile `json:"files"`
}

// FilingFile represents a paginated file containing older filings
type FilingFile struct {
	Name        string `json:"name"`
	FilingCount int    `json:"filingCount"`
	FilingFrom  string `json:"filingFrom"`
	FilingTo    string `json:"filingTo"`
}

// FilingArrays contains parallel arrays of filing data
// Each index in the arrays represents one filing
type FilingArrays struct {
	AccessionNumber       []string `json:"accessionNumber"`
	FilingDate            []string `json:"filingDate"`
	ReportDate            []string `json:"reportDate"`
	AcceptanceDateTime    []string `json:"acceptanceDateTime"`
	Act                   []string `json:"act"`
	Form                  []string `json:"form"`
	FileNumber            []string `json:"fileNumber"`
	FilmNumber            []string `json:"filmNumber"`
	Items                 []string `json:"items"`
	Size                  []int    `json:"size"`
	IsXBRL                []int    `json:"isXBRL"`
	IsInlineXBRL          []int    `json:"isInlineXBRL"`
	PrimaryDocument       []string `json:"primaryDocument"`
	PrimaryDocDescription []string `json:"primaryDocDescription"`
}

// Filing represents a single filing with all its metadata
type Filing struct {
	AccessionNumber       string
	FilingDate            string
	ReportDate            string
	AcceptanceDateTime    string
	Act                   string
	Form                  string
	FileNumber            string
	FilmNumber            string
	Items                 string
	Size                  int
	IsXBRL                bool
	IsInlineXBRL          bool
	PrimaryDocument       string
	PrimaryDocDescription string
	// Derived fields
	CIK string
	URL string // Full URL to the filing
}

// FetchSubmissions fetches and parses the CIK submissions JSON from SEC
func FetchSubmissions(cik string, email string) (*Submissions, error) {
	// Pad CIK to 10 digits
	paddedCIK := fmt.Sprintf("%010s", cik)

	// Construct URL
	url := fmt.Sprintf("https://data.sec.gov/submissions/CIK%s.json", paddedCIK)

	// Create request with User-Agent header
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := fmt.Sprintf("go-edgar %s", email)
	req.Header.Set("User-Agent", userAgent)

	// Execute request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch submissions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SEC returned status %d", resp.StatusCode)
	}

	// Parse JSON
	var subs Submissions
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		return nil, fmt.Errorf("failed to parse submissions JSON: %w", err)
	}

	return &subs, nil
}

// ParseSubmissions parses a submissions JSON from a reader (for local files or testing)
func ParseSubmissions(r io.Reader) (*Submissions, error) {
	var subs Submissions
	if err := json.NewDecoder(r).Decode(&subs); err != nil {
		return nil, fmt.Errorf("failed to parse submissions JSON: %w", err)
	}
	return &subs, nil
}

// GetFilings converts the parallel arrays in FilingArrays into a slice of Filing structs
func (fa *FilingArrays) GetFilings(cik string) []Filing {
	count := len(fa.AccessionNumber)
	filings := make([]Filing, count)

	for i := 0; i < count; i++ {
		filing := Filing{
			CIK:             cik,
			AccessionNumber: fa.AccessionNumber[i],
			FilingDate:      fa.FilingDate[i],
			Form:            fa.Form[i],
			PrimaryDocument: fa.PrimaryDocument[i],
		}

		// Handle optional fields with bounds checking
		if i < len(fa.ReportDate) {
			filing.ReportDate = fa.ReportDate[i]
		}
		if i < len(fa.AcceptanceDateTime) {
			filing.AcceptanceDateTime = fa.AcceptanceDateTime[i]
		}
		if i < len(fa.Act) {
			filing.Act = fa.Act[i]
		}
		if i < len(fa.FileNumber) {
			filing.FileNumber = fa.FileNumber[i]
		}
		if i < len(fa.FilmNumber) {
			filing.FilmNumber = fa.FilmNumber[i]
		}
		if i < len(fa.Items) {
			filing.Items = fa.Items[i]
		}
		if i < len(fa.Size) {
			filing.Size = fa.Size[i]
		}
		if i < len(fa.IsXBRL) {
			filing.IsXBRL = fa.IsXBRL[i] != 0
		}
		if i < len(fa.IsInlineXBRL) {
			filing.IsInlineXBRL = fa.IsInlineXBRL[i] != 0
		}
		if i < len(fa.PrimaryDocDescription) {
			filing.PrimaryDocDescription = fa.PrimaryDocDescription[i]
		}

		// Build URL
		filing.URL = filing.BuildURL()

		filings[i] = filing
	}

	return filings
}

// BuildURL constructs the full SEC EDGAR URL for this filing
func (f *Filing) BuildURL() string {
	// Remove hyphens from accession number for URL path
	accessionPath := strings.ReplaceAll(f.AccessionNumber, "-", "")

	// For Form 4, the primaryDocument often points to HTML rendering (xslF345X05/doc4.xml)
	// Strip the xsl path prefix to get the actual document name
	doc := f.PrimaryDocument
	if strings.Contains(doc, "/") {
		// Extract filename from path like "xslF345X05/doc4.xml" -> "doc4.xml"
		parts := strings.Split(doc, "/")
		doc = parts[len(parts)-1]
	}

	// https://www.sec.gov/Archives/edgar/data/{CIK}/{ACCESSION}/{PRIMARY_DOCUMENT}
	return fmt.Sprintf("https://www.sec.gov/Archives/edgar/data/%s/%s/%s",
		strings.TrimLeft(f.CIK, "0"), // Remove leading zeros from CIK
		accessionPath,
		doc,
	)
}

// GetRecentFilings returns all recent filings as a slice
func (s *Submissions) GetRecentFilings() []Filing {
	return s.Filings.Recent.GetFilings(s.CIK)
}

// FilterByForm filters filings by form type (e.g., "4", "3", "5")
// Supports exact matching and prefix matching for amendments
// Examples:
//   - "4" matches "4" exactly
//   - "13D" matches "13D", "13D/A", "13D/A/A", etc.
//   - "13G" matches "13G", "13G/A", etc.
func FilterByForm(filings []Filing, formType string) []Filing {
	var filtered []Filing
	for _, f := range filings {
		if matchesFormType(f.Form, formType) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// matchesFormType checks if a filing form matches the requested form type
// Handles Schedule 13 form normalization and amendment matching:
//   - "13D" → "SC 13D" (matches "SC 13D", "SC 13D/A", etc.)
//   - "13G" → "SC 13G" (matches "SC 13G", "SC 13G/A", etc.)
//   - "13" → matches ALL Schedule 13 forms ("SC 13D", "SC 13D/A", "SC 13G", "SC 13G/A", etc.)
//   - "4" → matches "4" only (exact match, NO amendments)
//   - "3", "5" → exact match only
//
// Note: Form 4/3/5 do NOT include amendments by default (use "4/A" explicitly to match amendments).
// Schedule 13 forms DO include amendments when filtering by base type.
func matchesFormType(filingForm, requestedForm string) bool {
	// Normalize requested form: add "SC" prefix for Schedule 13 forms
	normalizedRequest := normalizeFormType(requestedForm)

	// Special case: "13" as wildcard for all Schedule 13 forms
	if requestedForm == "13" {
		// Match any Schedule 13 form (SC 13D, SC 13G, and amendments)
		return strings.HasPrefix(filingForm, "SC 13D") || strings.HasPrefix(filingForm, "SC 13G")
	}

	// Exact match
	if filingForm == normalizedRequest {
		return true
	}

	// Amendment matching: Only include amendments for Schedule 13 forms
	// Form 4/3/5 require exact match (no implicit amendment inclusion)
	isSchedule13Request := strings.HasPrefix(normalizedRequest, "SC 13")
	if isSchedule13Request {
		// Schedule 13: include amendments (e.g., "SC 13D/A" matches request for "SC 13D")
		if strings.HasPrefix(filingForm, normalizedRequest+"/") {
			return true
		}
	}

	return false
}

// normalizeFormType converts user-friendly form names to SEC form names
// Examples:
//   - "13D" → "SC 13D"
//   - "13G" → "SC 13G"
//   - "4" → "4" (unchanged)
//   - "SC 13D" → "SC 13D" (already normalized)
func normalizeFormType(formType string) string {
	// Trim whitespace
	formType = strings.TrimSpace(formType)

	// If already has "SC" prefix, return as-is
	if strings.HasPrefix(formType, "SC ") {
		return formType
	}

	// Add "SC" prefix for Schedule 13 forms
	if formType == "13D" || formType == "13G" {
		return "SC " + formType
	}

	// Handle amendments: "13D/A" → "SC 13D/A"
	if strings.HasPrefix(formType, "13D/") {
		return "SC " + formType
	}
	if strings.HasPrefix(formType, "13G/") {
		return "SC " + formType
	}

	// Return unchanged for other forms (4, 3, 5, 10-K, etc.)
	return formType
}

// FilterByDateRange filters filings by date range (inclusive)
// Dates should be in YYYY-MM-DD format
func FilterByDateRange(filings []Filing, from, to string) []Filing {
	var filtered []Filing
	for _, f := range filings {
		// Use filing date for filtering
		if f.FilingDate >= from && f.FilingDate <= to {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// FetchPaginatedFilings fetches and parses a paginated filings file
func FetchPaginatedFilings(cik string, filename string, email string) (*FilingArrays, error) {
	// Construct URL
	url := fmt.Sprintf("https://data.sec.gov/submissions/%s", filename)

	// Create request with User-Agent header
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	userAgent := fmt.Sprintf("go-edgar %s", email)
	req.Header.Set("User-Agent", userAgent)

	// Execute request with rate limiting
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch paginated filings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SEC returned status %d for %s", resp.StatusCode, filename)
	}

	// Parse JSON - paginated files only contain the FilingArrays
	var filings FilingArrays
	if err := json.NewDecoder(resp.Body).Decode(&filings); err != nil {
		return nil, fmt.Errorf("failed to parse paginated filings JSON: %w", err)
	}

	return &filings, nil
}

// GetAllFilings returns all filings including paginated results
// This fetches all paginated files if they exist
func (s *Submissions) GetAllFilings(email string) ([]Filing, error) {
	// Start with recent filings
	allFilings := s.GetRecentFilings()

	// Fetch paginated files if they exist
	for _, fileInfo := range s.Filings.Files {
		filings, err := FetchPaginatedFilings(s.CIK, fileInfo.Name, email)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch %s: %w", fileInfo.Name, err)
		}

		// Convert to Filing structs and append
		pageFilings := filings.GetFilings(s.CIK)
		allFilings = append(allFilings, pageFilings...)

		// Rate limiting: sleep 100ms between requests (10 req/sec max)
		time.Sleep(100 * time.Millisecond)
	}

	return allFilings, nil
}
