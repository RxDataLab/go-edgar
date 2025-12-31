package edgar

import (
	"encoding/xml"
	"fmt"
	"io"
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
	switch formType {
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
	default:
		return nil, fmt.Errorf("form type %s not yet supported", formType)
	}
}

// detectFormType examines XML to determine form type
func detectFormType(data []byte) (string, error) {
	// Quick check for root elements
	type quickCheck struct {
		XMLName xml.Name
		DocType string `xml:"documentType"`
	}

	var check quickCheck
	if err := xml.Unmarshal(data, &check); err != nil {
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
		// Could be 10-K, 10-Q, 8-K, etc.
		return "", fmt.Errorf("edgarSubmission forms not yet supported")
	default:
		return "", fmt.Errorf("unknown form type with root element: %s", check.XMLName.Local)
	}
}
