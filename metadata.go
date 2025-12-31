package edgar

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// FilingMetadata contains information extracted from SEC URLs or filings
type FilingMetadata struct {
	CIK       string
	Accession string
	FormType  string
}

// ExtractMetadataFromURL parses SEC EDGAR URLs to extract CIK and accession number
// Example URL: https://www.sec.gov/Archives/edgar/data/1631574/000119312525314736/ownership.xml
func ExtractMetadataFromURL(url string) (*FilingMetadata, error) {
	// Pattern: /edgar/data/{CIK}/{ACCESSION}/{filename}
	pattern := regexp.MustCompile(`/edgar/data/(\d+)/(\d+)/`)
	matches := pattern.FindStringSubmatch(url)

	if len(matches) < 3 {
		return nil, fmt.Errorf("could not extract CIK and accession from URL")
	}

	// Format accession number: 0001193125-25-314736
	accession := matches[2]
	if len(accession) == 18 {
		// Format: XXXXXXXXXX-XX-XXXXXX
		accession = accession[:10] + "-" + accession[10:12] + "-" + accession[12:]
	}

	return &FilingMetadata{
		CIK:       matches[1],
		Accession: accession,
	}, nil
}

// ExtractMetadataFromForm extracts metadata from a parsed form
func ExtractMetadataFromForm(form *ParsedForm) *FilingMetadata {
	meta := &FilingMetadata{
		FormType: form.FormType,
	}

	// Extract CIK based on form type
	switch form.FormType {
	case "4", "3", "5":
		if f4, ok := form.Data.(*Form4); ok {
			meta.CIK = f4.Issuer.CIK
		}
	}

	return meta
}

// MergeMetadata combines URL and form metadata, preferring URL data when available
func MergeMetadata(urlMeta, formMeta *FilingMetadata) *FilingMetadata {
	merged := &FilingMetadata{}

	if urlMeta != nil {
		merged.CIK = urlMeta.CIK
		merged.Accession = urlMeta.Accession
	}

	if formMeta != nil {
		if merged.CIK == "" {
			merged.CIK = formMeta.CIK
		}
		merged.FormType = formMeta.FormType
	}

	return merged
}

// GenerateFilename creates a smart filename based on metadata
// Format: {CIK}-{accession}_ownership.{ext}
// Falls back to ownership.{ext} if metadata is incomplete
func GenerateFilename(meta *FilingMetadata, ext string) string {
	if meta.CIK != "" && meta.Accession != "" {
		return fmt.Sprintf("%s-%s_ownership.%s", meta.CIK, meta.Accession, ext)
	}
	if meta.CIK != "" {
		return fmt.Sprintf("%s_ownership.%s", meta.CIK, ext)
	}
	return fmt.Sprintf("ownership.%s", ext)
}

// SaveOptions configures how files should be saved
type SaveOptions struct {
	SaveOriginal bool
	OriginalPath string // If empty, uses smart naming
	OutputPath   string // If empty, uses smart naming or stdout
	OutputDir    string // Directory for output files (default: current dir)
}

// SaveResult contains paths to saved files
type SaveResult struct {
	OriginalPath string
	OutputPath   string
}

// SaveFiles saves the original XML and/or JSON output based on options
func SaveFiles(xmlData []byte, form *ParsedForm, meta *FilingMetadata, opts SaveOptions) (*SaveResult, error) {
	result := &SaveResult{}

	// Ensure output directory exists
	if opts.OutputDir != "" {
		if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Save original XML if requested
	if opts.SaveOriginal {
		originalPath := opts.OriginalPath
		if originalPath == "" {
			originalPath = GenerateFilename(meta, "xml")
		}
		if opts.OutputDir != "" {
			originalPath = filepath.Join(opts.OutputDir, originalPath)
		}

		if err := os.WriteFile(originalPath, xmlData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save original XML: %w", err)
		}
		result.OriginalPath = originalPath
	}

	// Save JSON output if path is specified
	if opts.OutputPath != "" {
		outputPath := opts.OutputPath
		if opts.OutputDir != "" && !filepath.IsAbs(outputPath) {
			outputPath = filepath.Join(opts.OutputDir, outputPath)
		}

		jsonData, err := json.MarshalIndent(form, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}

		if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
			return nil, fmt.Errorf("failed to save JSON output: %w", err)
		}
		result.OutputPath = outputPath
	}

	return result, nil
}

// FormatJSON returns pretty-printed JSON for a parsed form
func FormatJSON(form *ParsedForm) ([]byte, error) {
	return json.MarshalIndent(form, "", "  ")
}

// FormatJSONBatch returns pretty-printed JSON for an array of ParsedForms
func FormatJSONBatch(filings []*ParsedForm) ([]byte, error) {
	// Extract just the data from each parsed form
	data := make([]interface{}, len(filings))
	for i, f := range filings {
		data[i] = f.Data
	}
	return json.MarshalIndent(data, "", "  ")
}
