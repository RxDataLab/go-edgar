package edgar

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// ParsedForm represents any parsed SEC form with its type
type ParsedForm struct {
	FormType string      `json:"formType"`
	Data     interface{} `json:"data"`
}

// ParseAny auto-detects the form type and parses accordingly
func ParseAny(r io.Reader) (*ParsedForm, error) {
	// Read all data
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	// First check if it's XBRL (10-K, 10-Q, etc.)
	// IMPORTANT: Check XBRL BEFORE normalization because XML entities should be handled by XML parser
	xbrlType := DetectXBRLType(data)
	if xbrlType == "inline" || xbrlType == "standalone" {
		// Parse XBRL
		xbrl, err := ParseXBRLAuto(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse XBRL: %w", err)
		}

		// Extract snapshot
		snapshot, err := xbrl.GetSnapshot()
		if err != nil {
			return nil, fmt.Errorf("failed to extract financial snapshot: %w", err)
		}

		// Determine form type from XBRL (10-K, 10-Q, etc.)
		// For now, just return as "10-K/10-Q" - we could extract this from DEI facts
		return &ParsedForm{
			FormType: "XBRL",
			Data:     snapshot,
		}, nil
	}

	// Not XBRL, try ownership forms (Form 4, etc.)
	formType, err := detectFormType(data)
	if err != nil {
		return nil, err
	}

	// Parse based on form type
	// Normalize form type (handle both "SC 13D" and "SCHEDULE 13D" formats)
	normalizedType := formType
	if formType == "SCHEDULE 13D" || formType == "SCHEDULE 13D/A" {
		normalizedType = "SC 13D"
		if formType == "SCHEDULE 13D/A" {
			normalizedType = "SC 13D/A"
		}
	} else if formType == "SCHEDULE 13G" || formType == "SCHEDULE 13G/A" {
		normalizedType = "SC 13G"
		if formType == "SCHEDULE 13G/A" {
			normalizedType = "SC 13G/A"
		}
	}

	switch normalizedType {
	case "4":
		form4, err := Parse(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Form 4: %w", err)
		}
		// Convert to simplified output structure
		return &ParsedForm{
			FormType: "4",
			Data:     form4.ToOutput(),
		}, nil
	case "SC 13D", "SC 13D/A", "SC 13G", "SC 13G/A":
		// Normalize text for Schedule 13 forms (handles non-breaking spaces, HTML entities)
		// This is critical for HTML parsing where &nbsp; appears in item headings
		data = NormalizeText(data)

		// Use auto-detection for 13D/G (handles both XML and HTML)
		sc13, err := ParseSchedule13Auto(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Schedule 13D/G: %w", err)
		}
		return &ParsedForm{
			FormType: normalizedType,
			Data:     sc13,
		}, nil
	default:
		return nil, fmt.Errorf("form type %s not yet supported", formType)
	}
}

// detectFormType examines XML/HTML to determine form type
func detectFormType(data []byte) (string, error) {
	dataStr := string(data)

	// Quick string-based detection for HTML/XHTML forms
	if strings.HasPrefix(strings.TrimSpace(dataStr), "<!DOCTYPE html") ||
		strings.HasPrefix(strings.TrimSpace(dataStr), "<html") {
		// This is HTML/XHTML - check for form type in content
		if strings.Contains(dataStr, "schedule13D") || strings.Contains(dataStr, "SCHEDULE 13D") {
			if strings.Contains(dataStr, "Amendment") {
				return "SC 13D/A", nil
			}
			return "SC 13D", nil
		} else if strings.Contains(dataStr, "schedule13g") || strings.Contains(dataStr, "SCHEDULE 13G") {
			if strings.Contains(dataStr, "Amendment") {
				return "SC 13G/A", nil
			}
			return "SC 13G", nil
		}
		return "", fmt.Errorf("HTML form type not recognized")
	}

	// Try XML parsing for pure XML forms
	type quickCheck struct {
		XMLName        xml.Name
		DocType        string `xml:"documentType"`
		SubmissionType string `xml:"headerData>submissionType"`
	}

	var check quickCheck
	if err := xml.Unmarshal(data, &check); err != nil {
		// If XML parsing fails but it looks like it might be Schedule 13D/G, try that
		if strings.Contains(dataStr, "SCHEDULE 13") || strings.Contains(dataStr, "schedule13") {
			if strings.Contains(dataStr, "13D") {
				return "SC 13D", nil
			}
			return "SC 13G", nil
		}
		return "", fmt.Errorf("failed to parse XML: %w", err)
	}

	// Check root element name
	switch check.XMLName.Local {
	case "ownershipDocument":
		// Forms 3, 4, 5 all use ownershipDocument
		// Differentiate by documentType field
		if check.DocType != "" {
			return check.DocType, nil
		}
		// Default to 4 if not specified
		return "4", nil
	case "informationTable":
		return "13F", nil
	case "edgarSubmission":
		// Could be Schedule 13D/G or other SEC submissions
		// Check the xmlns namespace to distinguish
		if check.XMLName.Space == "http://www.sec.gov/edgar/schedule13D" {
			return check.SubmissionType, nil // "SCHEDULE 13D" or "SCHEDULE 13D/A"
		} else if check.XMLName.Space == "http://www.sec.gov/edgar/schedule13g" {
			return check.SubmissionType, nil // "SCHEDULE 13G" or "SCHEDULE 13G/A"
		}
		return "", fmt.Errorf("edgarSubmission forms with namespace '%s' not yet supported", check.XMLName.Space)
	case "html":
		// XHTML rendered forms (Schedule 13D/G, etc.)
		// Check for namespace declarations to identify form type
		dataStr := string(data)
		if strings.Contains(dataStr, "schedule13D") || strings.Contains(dataStr, "SCHEDULE 13D") {
			if strings.Contains(dataStr, "Amendment") {
				return "SC 13D/A", nil
			}
			return "SC 13D", nil
		} else if strings.Contains(dataStr, "schedule13g") || strings.Contains(dataStr, "SCHEDULE 13G") {
			if strings.Contains(dataStr, "Amendment") {
				return "SC 13G/A", nil
			}
			return "SC 13G", nil
		}
		return "", fmt.Errorf("HTML form type not recognized")
	default:
		return "", fmt.Errorf("unknown form type with root element: %s", check.XMLName.Local)
	}
}
