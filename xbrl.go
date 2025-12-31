package edgar

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// XBRL represents a parsed XBRL instance document (10-K, 10-Q, etc.)
type XBRL struct {
	XMLName  xml.Name  `xml:"xbrl"`
	Contexts []Context `xml:"context"`
	Units    []Unit    `xml:"unit"`
	Facts    []Fact    `xml:"-"` // Populated during parsing
}

// Context defines the dimensional context for facts (period, entity, segments)
type Context struct {
	ID     string `xml:"id,attr"`
	Entity Entity `xml:"entity"`
	Period Period `xml:"period"`
}

// Entity identifies the reporting company
type Entity struct {
	Identifier string `xml:"identifier"`
	Segment    string `xml:"segment,omitempty"`
}

// Period defines the time period for a fact (instant or duration)
type Period struct {
	Instant   string `xml:"instant,omitempty"`   // Point in time (balance sheet)
	StartDate string `xml:"startDate,omitempty"` // Duration start (income statement)
	EndDate   string `xml:"endDate,omitempty"`   // Duration end
}

// Unit defines the measurement unit for a fact (USD, shares, etc.)
type Unit struct {
	ID      string  `xml:"id,attr"`
	Measure string  `xml:"measure"`
	Divide  *Divide `xml:"divide,omitempty"` // For ratios like USD/share
}

// Divide represents a ratio unit (numerator/denominator)
type Divide struct {
	Numerator   string `xml:"unitNumerator>measure"`
	Denominator string `xml:"unitDenominator>measure"`
}

// Fact represents a single XBRL fact (financial data point)
type Fact struct {
	Concept    string // XBRL concept name (e.g., "us-gaap:Cash")
	Value      string // Raw value as string
	ContextRef string // Reference to Context.ID
	UnitRef    string // Reference to Unit.ID
	Decimals   int    // Precision (-3 = thousands, -6 = millions)

	// Derived fields (populated after parsing)
	StandardLabel string   // Standardized concept label (from mappings)
	Period        *Period  // Resolved period from context
	NumericValue  *float64 // Parsed numeric value (nil if non-numeric)
}

// ParseXBRL parses an XBRL instance document from XML bytes
func ParseXBRL(data []byte) (*XBRL, error) {
	var xbrl XBRL
	if err := xml.Unmarshal(data, &xbrl); err != nil {
		return nil, fmt.Errorf("failed to parse XBRL XML: %w", err)
	}

	// Extract facts from the XML tree
	// Note: XBRL facts are dynamic elements (us-gaap:Cash, us-gaap:Revenue, etc.)
	// We need custom parsing to extract them
	if err := extractFacts(&xbrl, data); err != nil {
		return nil, fmt.Errorf("failed to extract facts: %w", err)
	}

	// Resolve contexts and standardize labels
	if err := resolveFacts(&xbrl); err != nil {
		return nil, fmt.Errorf("failed to resolve facts: %w", err)
	}

	return &xbrl, nil
}

// extractFacts parses the XML tree to find all fact elements
// XBRL facts are dynamic elements with namespaces (us-gaap:*, dei:*, etc.)
func extractFacts(xbrl *XBRL, data []byte) error {
	// Create a generic XML decoder to walk the tree
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	var facts []Fact

	for {
		token, err := decoder.Token()
		if err != nil {
			break // End of document
		}

		switch elem := token.(type) {
		case xml.StartElement:
			// Check if this is a fact element (has contextRef attribute)
			contextRef := getAttr(elem.Attr, "contextRef")
			if contextRef == "" {
				continue // Not a fact
			}

			// Parse the fact value
			var value string
			if err := decoder.DecodeElement(&value, &elem); err != nil {
				continue
			}

			// Build the full concept name (namespace:localName)
			conceptName := elem.Name.Local
			if elem.Name.Space != "" {
				// Extract namespace prefix from space (e.g., "http://fasb.org/us-gaap/2023" -> "us-gaap")
				conceptName = getNamespacePrefix(elem.Name.Space) + ":" + elem.Name.Local
			}

			// Parse decimals attribute
			decimals := 0
			if decimalsStr := getAttr(elem.Attr, "decimals"); decimalsStr != "" {
				if decimalsStr != "INF" {
					decimals, _ = strconv.Atoi(decimalsStr)
				}
			}

			fact := Fact{
				Concept:    conceptName,
				Value:      strings.TrimSpace(value),
				ContextRef: contextRef,
				UnitRef:    getAttr(elem.Attr, "unitRef"),
				Decimals:   decimals,
			}

			facts = append(facts, fact)
		}
	}

	xbrl.Facts = facts
	return nil
}

// resolveFacts enriches facts with resolved contexts and standardized labels
func resolveFacts(xbrl *XBRL) error {
	// Build context lookup map
	contextMap := make(map[string]*Context)
	for i := range xbrl.Contexts {
		contextMap[xbrl.Contexts[i].ID] = &xbrl.Contexts[i]
	}

	// Resolve each fact
	for i := range xbrl.Facts {
		fact := &xbrl.Facts[i]

		// Resolve context
		if ctx, ok := contextMap[fact.ContextRef]; ok {
			fact.Period = &ctx.Period
		}

		// Get standardized label
		fact.StandardLabel = GetStandardizedLabel(fact.Concept)

		// Parse numeric value
		if val, err := parseNumericValue(fact.Value, fact.Decimals); err == nil {
			fact.NumericValue = &val
		}
	}

	return nil
}

// parseNumericValue converts a string value to float64, applying decimal scaling
func parseNumericValue(value string, decimals int) (float64, error) {
	// Remove commas and whitespace
	cleaned := strings.ReplaceAll(value, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	// Handle empty or non-numeric values
	if cleaned == "" || cleaned == "-" || cleaned == "—" {
		return 0, fmt.Errorf("empty or invalid value")
	}

	// Parse to float
	val, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return 0, err
	}

	// Apply decimal scaling
	// Decimals of -3 means value is in thousands, -6 means millions, etc.
	// We want to return the actual value (not scaled)
	// Example: value="1234" decimals=-3 means 1,234,000
	if decimals < 0 {
		scale := 1.0
		for i := 0; i < -decimals; i++ {
			scale *= 10
		}
		val *= scale
	}

	return val, nil
}

// getAttr gets an attribute value by name
func getAttr(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

// getNamespacePrefix extracts a namespace prefix from a full namespace URI
// Example: "http://fasb.org/us-gaap/2023" -> "us-gaap"
func getNamespacePrefix(namespace string) string {
	// Common namespace patterns
	if strings.Contains(namespace, "us-gaap") {
		return "us-gaap"
	}
	if strings.Contains(namespace, "/dei/") {
		return "dei"
	}
	if strings.Contains(namespace, "xbrli") {
		return "xbrli"
	}

	// Fallback: try to extract from URI structure
	parts := strings.Split(namespace, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "unknown"
}

// Helper methods on Fact

// Float64 returns the numeric value as float64
func (f *Fact) Float64() (float64, error) {
	if f.NumericValue != nil {
		return *f.NumericValue, nil
	}
	return 0, fmt.Errorf("fact %s has no numeric value", f.Concept)
}

// IsInstant returns true if this fact is for a point in time (balance sheet)
func (f *Fact) IsInstant() bool {
	return f.Period != nil && f.Period.Instant != ""
}

// IsDuration returns true if this fact is for a time period (income statement)
func (f *Fact) IsDuration() bool {
	return f.Period != nil && f.Period.StartDate != "" && f.Period.EndDate != ""
}

// GetEndDate returns the end date of the period
func (f *Fact) GetEndDate() (time.Time, error) {
	if f.Period == nil {
		return time.Time{}, fmt.Errorf("fact has no period")
	}

	dateStr := f.Period.EndDate
	if dateStr == "" {
		dateStr = f.Period.Instant
	}

	if dateStr == "" {
		return time.Time{}, fmt.Errorf("fact has no end date or instant")
	}

	return time.Parse("2006-01-02", dateStr)
}

// GetPeriodLabel returns a human-readable period label
func (f *Fact) GetPeriodLabel() string {
	if f.Period == nil {
		return "Unknown"
	}

	if f.Period.Instant != "" {
		return f.Period.Instant
	}

	if f.Period.StartDate != "" && f.Period.EndDate != "" {
		return fmt.Sprintf("%s to %s", f.Period.StartDate, f.Period.EndDate)
	}

	return "Unknown"
}
